package trainer

import (
	"testing"
	"time"
)

func TestModernZeroMemoryValuesSurviveRestartAndReview(t *testing.T) {
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	progress := Progress{
		Version: ProgressVersion,
		States: map[string]MemoryState{
			"a": {Attempts: 20, Correct: 20, Stability: 0, Difficulty: 0, IntervalDays: 1, LastSeen: now},
		},
	}
	progress.EnsureDefaults(testKana("a"), now)
	state := progress.States["a"]
	if state.Stability != 0 || state.Difficulty != 0 {
		t.Fatalf("valid modern zero values were treated as absent legacy fields: %+v", state)
	}
	state.Record(true, time.Second, 21, now.Add(time.Minute))
	if state.Difficulty != 0 {
		t.Fatalf("review reset an already learned zero difficulty: %v", state.Difficulty)
	}
}
