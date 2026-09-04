package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

const progressFilename = "progress.json"

// Store loads and atomically saves progress at one explicit path.
type Store struct {
	path string
	now  func() time.Time
}

// New returns a progress store for path.
func New(path string) *Store {
	return &Store{path: path, now: time.Now}
}

// DefaultPath follows XDG_DATA_HOME and falls back to ~/.local/share.
func DefaultPath() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "kana", progressFilename), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "kana", progressFilename), nil
}

// Path returns the configured progress file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads progress. Missing or corrupt files are replaced with a clean state.
// A corrupt file is preserved beside the new file for manual recovery.
func (s *Store) Load(items []kanadata.Kana) (trainer.Progress, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		progress := trainer.NewProgress(items, s.now())
		if err := s.Save(progress); err != nil {
			return trainer.Progress{}, err
		}
		return progress, nil
	}
	if err != nil {
		return trainer.Progress{}, fmt.Errorf("read progress: %w", err)
	}

	var progress trainer.Progress
	if err := json.Unmarshal(data, &progress); err != nil {
		backup := s.path + ".corrupt-" + s.now().UTC().Format("20060102T150405")
		if renameErr := os.Rename(s.path, backup); renameErr != nil {
			return trainer.Progress{}, fmt.Errorf("preserve corrupt progress: %w", renameErr)
		}
		progress = trainer.NewProgress(items, s.now())
		if saveErr := s.Save(progress); saveErr != nil {
			return trainer.Progress{}, saveErr
		}
		return progress, nil
	}
	progress.EnsureDefaults(items, s.now())
	return progress, nil
}

// Save writes indented JSON through a same-directory temporary file and rename.
func (s *Store) Save(progress trainer.Progress) error {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".progress-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary progress: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure temporary progress: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(progress); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode progress: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync progress: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close progress: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("commit progress: %w", err)
	}
	committed = true

	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open data directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync data directory: %w", err)
	}
	return nil
}

// Reset removes saved progress. Callers are responsible for confirmation.
func (s *Store) Reset() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove progress: %w", err)
	}
	return nil
}
