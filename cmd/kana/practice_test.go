package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	kanadata "kana/internal/kana"
)

func TestInvalidPracticeOptionsDoNotCreateProgress(t *testing.T) {
	for _, args := range [][]string{
		{"--level", "n6", "--stats"},
		{"--level", "gojuon", "--exercise", "anime", "--stats"},
		{"--rows", "a,unknown", "--stats"},
		{"--rows", "a,", "--stats"},
		{"--rows", "k", "--level", "n2", "--stats"},
		{"--all", "--rows", "k", "--stats"},
		{"--random", "--weak", "--stats"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("XDG_DATA_HOME", home)
			if err := run(args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid combination accepted")
			}
			if _, err := os.Stat(filepath.Join(home, "kana", "progress.json")); !os.IsNotExist(err) {
				t.Fatalf("invalid invocation touched progress: %v", err)
			}
		})
	}
}

func TestStatisticsRespectSelectedRowsAndGradedDeck(t *testing.T) {
	for _, scenario := range []struct {
		args  []string
		level string
		rows  map[string]bool
	}{
		{[]string{"--stats", "--rows", " k, S,k ", "--mixed"}, "gojuon", map[string]bool{"k": true, "s": true}},
		{[]string{"--stats", "--level", "N2", "--katakana"}, "n2", nil},
		{[]string{"--stats", "--level", "n1", "--exercise", "anime"}, "n1", nil},
	} {
		t.Run(strings.Join(scenario.args, " "), func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			var output bytes.Buffer
			if err := run(scenario.args, strings.NewReader(""), &output, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			exercise := "reading"
			if scenario.level == "n1" {
				exercise = "anime"
			}
			deck, err := kanadata.Deck(scenario.level, exercise)
			if err != nil {
				t.Fatal(err)
			}
			allowed := make(map[string]bool)
			for _, item := range deck {
				if scenario.rows == nil || scenario.rows[item.Row] {
					allowed[item.Hiragana] = true
					allowed[item.Katakana] = true
				}
			}
			questions := 0
			for _, line := range strings.Split(output.String(), "\n") {
				fields := strings.Fields(line)
				if len(fields) < 3 {
					continue
				}
				if _, err := strconv.Atoi(strings.TrimSuffix(fields[0], ".")); err != nil {
					continue
				}
				if !allowed[fields[1]] {
					t.Fatalf("statistics leaked a question outside chosen deck: %s", line)
				}
				questions++
			}
			if questions != min(12, len(allowed)/2) {
				t.Fatalf("statistics omitted expected questions: %d", questions)
			}
		})
	}
}
