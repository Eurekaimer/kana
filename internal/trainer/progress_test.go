package trainer

import (
	"math"
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func TestNewProgressAndRecord(t *testing.T) {
	now := time.Unix(100, 0)
	progress := NewProgress(kanadata.Basic, now)
	if len(progress.States) != 46 || progress.Version != ProgressVersion {
		t.Fatalf("new progress = %+v", progress)
	}
	progress.Record(Evaluation{ExpectedID: "ki", AnsweredID: "ku", Correct: false, ResponseTime: 2 * time.Second}, now.Add(time.Second))
	if progress.QuestionIndex != 1 || progress.States["ki"].Wrong != 1 {
		t.Fatalf("recorded progress = %+v", progress)
	}
	if progress.Confusions.Count("ki", "ku") != 1 {
		t.Fatal("record did not update confusion")
	}
}

func TestEnsureDefaults(t *testing.T) {
	now := time.Unix(200, 0)
	progress := Progress{}
	progress.EnsureDefaults(kanadata.Basic, now)
	if len(progress.States) != 46 || progress.Confusions == nil || progress.CreatedAt != now {
		t.Fatalf("repaired progress = %+v", progress)
	}
}

func TestEnsureDefaultsMigratesLegacyProgress(t *testing.T) {
	seen := time.Date(2025, 12, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	legacy := Progress{
		Version:       1,
		QuestionIndex: 42,
		States: map[string]MemoryState{
			"ki": {
				Attempts:        10,
				Correct:         4,
				Wrong:           6,
				CurrentStreak:   1,
				BestStreak:      3,
				LastSeen:        seen,
				LastSeenIndex:   41,
				DueIndex:        57,
				Stability:       0.4,
				ResponseTimeEMA: 2.5,
				RecentResults:   []bool{true, false, true},
			},
			"a": {
				Attempts:      2,
				Correct:       1,
				Wrong:         1,
				LastSeen:      seen,
				DueIndex:      9,
				Stability:     0.3,
				RecentResults: []bool{false, true},
			},
		},
		Confusions: ConfusionMatrix{"ki": {"ku": 3}},
	}

	legacy.EnsureDefaults(testKana("ki", "ku"), now)

	if legacy.Version != ProgressVersion {
		t.Fatalf("version = %d, want %d", legacy.Version, ProgressVersion)
	}
	if legacy.QuestionIndex != 42 || legacy.Confusions.Count("ki", "ku") != 3 {
		t.Fatalf("migration lost progress: %+v", legacy)
	}

	state := legacy.States["ki"]
	if state.Attempts != 10 || state.Correct != 4 || state.Wrong != 6 {
		t.Fatalf("migration lost statistics: %+v", state)
	}
	if state.CurrentStreak != 1 || state.BestStreak != 3 || state.ResponseTimeEMA != 2.5 {
		t.Fatalf("migration lost history: %+v", state)
	}
	if len(state.RecentResults) != 3 || state.DueIndex != 57 || !state.LastSeen.Equal(seen) {
		t.Fatalf("migration lost review fields: %+v", state)
	}
	if wantDifficulty := 0.25 + 0.5*6.0/10.0; math.Abs(state.Difficulty-wantDifficulty) > 0.0001 {
		t.Fatalf("migrated difficulty = %v, want %v", state.Difficulty, wantDifficulty)
	}
	wantInterval := ReviewIntervalDays(0.4, state.Difficulty)
	if state.IntervalDays != wantInterval {
		t.Fatalf("migrated interval = %d, want %d", state.IntervalDays, wantInterval)
	}
	if wantDue := seen.AddDate(0, 0, wantInterval); !state.DueAt.Equal(wantDue) {
		t.Fatalf("migrated due = %v, want %v", state.DueAt, wantDue)
	}

	unseen := legacy.States["ku"]
	if unseen.Stability != DefaultScoringConfig.InitialStability || unseen.Difficulty != defaultDifficulty {
		t.Fatalf("new card defaults = %+v", unseen)
	}
	if unseen.IntervalDays != ReviewIntervalDays(unseen.Stability, unseen.Difficulty) {
		t.Fatalf("new card interval = %d", unseen.IntervalDays)
	}
	if !unseen.DueAt.IsZero() {
		t.Fatalf("unseen due = %v, want zero", unseen.DueAt)
	}

	retained := legacy.States["a"]
	if retained.Attempts != 2 || retained.Correct != 1 || retained.Wrong != 1 || retained.DueIndex != 9 {
		t.Fatalf("migration lost a retained state: %+v", retained)
	}
	if retained.Difficulty == 0 || retained.IntervalDays <= 0 || retained.DueAt.IsZero() {
		t.Fatalf("retained state was not repaired: %+v", retained)
	}
}
