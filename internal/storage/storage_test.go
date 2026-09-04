package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

func TestDefaultPathUsesXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/kana-xdg")
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/kana-xdg/kana/progress.json"; path != want {
		t.Fatalf("DefaultPath() = %q, want %q", path, want)
	}
}

func TestStorePath(t *testing.T) {
	if got := New("/tmp/progress.json").Path(); got != "/tmp/progress.json" {
		t.Fatalf("Path() = %q", got)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/kana-home")
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/kana-home/.local/share/kana/progress.json"; path != want {
		t.Fatalf("DefaultPath() = %q, want %q", path, want)
	}
}

func TestSaveLoadAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "progress.json")
	store := New(path)
	now := time.Unix(100, 0)
	progress := trainer.NewProgress(kanadata.Basic, now)
	progress.QuestionIndex = 42
	state := progress.States["ki"]
	state.Attempts = 7
	state.Correct = 5
	progress.States["ki"] = state
	progress.Confusions.Record("ki", "ku")
	if err := store.Save(progress); err != nil {
		t.Fatal(err)
	}

	restartedStore := New(path)
	loaded, err := restartedStore.Load(kanadata.Basic)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.QuestionIndex != 42 || loaded.States["ki"].Attempts != 7 || loaded.Confusions.Count("ki", "ku") != 1 {
		t.Fatalf("loaded progress = %+v", loaded)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n  \"version\"") {
		t.Fatal("progress JSON is not human-readable")
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("progress permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestLoadCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kana", "progress.json")
	store := New(path)
	progress, err := store.Load(kanadata.Basic)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.States) != 46 {
		t.Fatalf("state count = %d", len(progress.States))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing progress was not created: %v", err)
	}
}

func TestCorruptProgressIsPreservedAndRecovered(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "progress.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(path)
	store.now = func() time.Time { return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC) }
	progress, err := store.Load(kanadata.Basic)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.States) != 46 {
		t.Fatalf("recovered state count = %d", len(progress.States))
	}
	backup := path + ".corrupt-20260904T120000"
	if data, err := os.ReadFile(backup); err != nil || string(data) != "{not json" {
		t.Fatalf("corrupt backup = %q, %v", data, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("replacement progress missing: %v", err)
	}
}

func TestReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(path)
	if err := store.Reset(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("progress still exists: %v", err)
	}
	if err := store.Reset(); err != nil {
		t.Fatalf("second reset should be harmless: %v", err)
	}
}
