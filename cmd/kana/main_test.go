package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kanadata "kana/internal/kana"
)

func TestStatsCommand(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	var output, errors bytes.Buffer
	if err := run([]string{"--stats", "--katakana"}, strings.NewReader(""), &output, &errors); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"kana statistics", "Weak Kana", "Most Confused", "ア", "Data:"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("statistics missing %q:\n%s", expected, output.String())
		}
	}
	if _, err := os.Stat(filepath.Join(dataHome, "kana", "progress.json")); err != nil {
		t.Fatalf("stats did not initialize progress: %v", err)
	}
}

func TestResetRequiresConfirmation(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	path := filepath.Join(dataHome, "kana", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("progress"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"--reset"}, strings.NewReader("n\n"), &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("declined reset removed progress")
	}
	output.Reset()
	if err := run([]string{"--reset"}, strings.NewReader("yes\n"), &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("confirmed reset left progress: %v", err)
	}
	if !strings.Contains(output.String(), "Progress reset") {
		t.Fatalf("reset output = %q", output.String())
	}
}

func TestFlagValidation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	for _, arguments := range [][]string{
		{"--hiragana", "--katakana", "--stats"},
		{"--weak", "--confusion", "--stats"},
		{"--reset", "--stats"},
		{"unexpected"},
	} {
		if err := run(arguments, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("run(%v) succeeded", arguments)
		}
	}
}

func TestHelpers(t *testing.T) {
	if got := selectedMode(false, false); got != kanadata.ModeHiragana {
		t.Fatalf("default mode = %s", got)
	}
	if got := selectedMode(true, false); got != kanadata.ModeKatakana {
		t.Fatalf("katakana mode = %s", got)
	}
	if got := selectedMode(false, true); got != kanadata.ModeMixed {
		t.Fatalf("mixed mode = %s", got)
	}
	if got := boolCount(true, false, true); got != 2 {
		t.Fatalf("boolCount = %d", got)
	}
}
