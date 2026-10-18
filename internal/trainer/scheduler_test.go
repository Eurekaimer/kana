package trainer

import (
	"math"
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func testKana(ids ...string) []kanadata.Kana {
	items := make([]kanadata.Kana, 0, len(ids))
	for _, id := range ids {
		item, ok := kanadata.ByID(id)
		if !ok {
			panic("unknown test kana: " + id)
		}
		items = append(items, item)
	}
	return items
}

func TestWeightFavorsWeakKana(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{})
	matrix := make(ConfusionMatrix)
	weak := MemoryState{Attempts: 10, Correct: 2, Wrong: 8, Stability: 0.1, ResponseTimeEMA: 5, RecentResults: []bool{false, false}}
	strong := MemoryState{Attempts: 20, Correct: 20, Stability: 0.95, ResponseTimeEMA: 0.8, RecentResults: []bool{true, true}}
	ki, _ := kanadata.ByID("ki")
	ku, _ := kanadata.ByID("ku")
	if scheduler.Weight(ki, weak, matrix, 100) <= scheduler.Weight(ku, strong, matrix, 100) {
		t.Fatal("weak kana did not receive greater weight")
	}
}

func TestWeightedSelectionIncludesMasteredKana(t *testing.T) {
	items := testKana("ki", "ku", "ke")
	states := map[string]MemoryState{
		"ki": {Attempts: 20, Correct: 4, Wrong: 16, Stability: 0.1, ResponseTimeEMA: 5, RecentResults: []bool{false}},
		"ku": {Attempts: 20, Correct: 20, Stability: 0.95, ResponseTimeEMA: 0.8, RecentResults: []bool{true}},
		"ke": {Attempts: 20, Correct: 15, Wrong: 5, Stability: 0.6, ResponseTimeEMA: 2, RecentResults: []bool{true}},
	}
	scheduler := NewScheduler(4, SchedulerConfig{})
	counts := make(map[string]int)
	for index := range int64(3000) {
		item, err := scheduler.Next(items, states, make(ConfusionMatrix), index)
		if err != nil {
			t.Fatal(err)
		}
		counts[item.ID]++
	}
	if counts["ki"] <= counts["ku"] {
		t.Fatalf("weak=%d, mastered=%d", counts["ki"], counts["ku"])
	}
	if counts["ku"] == 0 {
		t.Fatal("mastered kana starved")
	}
}

func TestAvoidsImmediateDuplicates(t *testing.T) {
	items := testKana("ki", "ku", "ke")
	scheduler := NewScheduler(2, SchedulerConfig{})
	previous := ""
	for index := range int64(100) {
		item, err := scheduler.Next(items, nil, make(ConfusionMatrix), index)
		if err != nil {
			t.Fatal(err)
		}
		if item.ID == previous {
			t.Fatalf("immediate duplicate %q at %d", item.ID, index)
		}
		previous = item.ID
	}
}

func TestWrongAnswerReturnsSoon(t *testing.T) {
	items := testKana("ki", "ku", "ke", "ko")
	scheduler := NewScheduler(8, SchedulerConfig{})
	scheduler.ObserveResult("ki", "ku", false, 0, make(ConfusionMatrix))
	seenAt := int64(-1)
	for index := int64(1); index <= 4; index++ {
		item, err := scheduler.Next(items, nil, make(ConfusionMatrix), index)
		if err != nil {
			t.Fatal(err)
		}
		if item.ID == "ki" {
			seenAt = index
			break
		}
	}
	if seenAt < 1 || seenAt > 4 {
		t.Fatalf("failed kana returned at %d", seenAt)
	}
}

func TestCorrectReinforcementGetsFollowUp(t *testing.T) {
	items := testKana("ki", "ku", "ke")
	scheduler := NewScheduler(5, SchedulerConfig{})
	scheduler.reinforcements = []reinforcement{{ID: "ki", Due: 1, Stage: 1}}
	item, err := scheduler.Next(items, nil, make(ConfusionMatrix), 1)
	if err != nil || item.ID != "ki" {
		t.Fatalf("reinforcement selection = %v, %v", item, err)
	}
	scheduler.ObserveResult("ki", "", true, 1, make(ConfusionMatrix))
	if len(scheduler.reinforcements) != 1 || scheduler.reinforcements[0].Stage != 2 {
		t.Fatalf("follow-up queue = %+v", scheduler.reinforcements)
	}
	if due := scheduler.reinforcements[0].Due; due < 6 || due > 11 {
		t.Fatalf("follow-up due = %d", due)
	}
}

func TestProgressiveUnlocking(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})
	if got := len(scheduler.ActiveKana(kanadata.Basic, nil, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("initial active count = %d, want 5", got)
	}
	states := make(map[string]MemoryState)
	for _, item := range kanadata.Basic[:5] {
		states[item.ID] = MemoryState{
			Attempts:      20,
			Correct:       19,
			Wrong:         1,
			CurrentStreak: 19,
			BestStreak:    19,
			Stability:     0.8,
			RecentResults: []bool{true, true, true, true},
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 10 {
		t.Fatalf("unlocked active count = %d, want 10", got)
	}
}

func TestConfusionFocus(t *testing.T) {
	matrix := ConfusionMatrix{"nu": {"ne": 2}}
	scheduler := NewScheduler(1, SchedulerConfig{Focus: FocusConfusion})
	active := scheduler.ActiveKana(kanadata.Basic, nil, matrix)
	if len(active) != 2 || active[0].ID != "nu" || active[1].ID != "ne" {
		t.Fatalf("confusion focus = %+v", active)
	}
}

func TestSchedulerConfigurationAndEmptyPool(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{})
	scheduler.SetFocus(FocusWeak)
	scheduler.SetProgressive(true)
	config := scheduler.Config()
	if config.Focus != FocusWeak || !config.Progressive || config.WeakLimit != defaultWeakLimit {
		t.Fatalf("config = %+v", config)
	}
	if _, err := scheduler.Next(nil, nil, make(ConfusionMatrix), 0); err == nil {
		t.Fatal("empty candidate pool succeeded")
	}
}

func TestRepeatedConfusionSchedulesContrast(t *testing.T) {
	matrix := ConfusionMatrix{"nu": {"ne": 2}}
	scheduler := NewScheduler(1, SchedulerConfig{})
	scheduler.ObserveResult("nu", "ne", false, 10, matrix)
	foundExpected := false
	foundContrast := false
	for _, entry := range scheduler.reinforcements {
		if entry.ID == "nu" && entry.Stage == 1 {
			foundExpected = true
		}
		if entry.ID == "ne" && entry.Stage == 0 {
			foundContrast = true
		}
	}
	if !foundExpected || !foundContrast {
		t.Fatalf("reinforcements = %+v", scheduler.reinforcements)
	}
}

func TestProgressiveUnlockingBoundary(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})

	atThreshold := make(map[string]MemoryState)
	for index, item := range kanadata.Basic[:5] {
		if index < 3 {
			atThreshold[item.ID] = MemoryState{
				Attempts: 4, Correct: 3, Wrong: 1,
				CurrentStreak: 3, BestStreak: 3, Stability: 0.5,
				RecentResults: []bool{false, true, true, true},
			}
		} else {
			atThreshold[item.ID] = MemoryState{
				Attempts: 4, Correct: 4,
				CurrentStreak: 4, BestStreak: 4, Stability: 0.5,
				RecentResults: []bool{true, true, true, true},
			}
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, atThreshold, make(ConfusionMatrix))); got != 10 {
		t.Fatalf("row at 85%% recent accuracy = %d active, want 10", got)
	}

	belowThreshold := make(map[string]MemoryState)
	for index, item := range kanadata.Basic[:5] {
		if index < 4 {
			belowThreshold[item.ID] = MemoryState{
				Attempts: 4, Correct: 3, Wrong: 1,
				CurrentStreak: 3, BestStreak: 3, Stability: 0.5,
				RecentResults: []bool{false, true, true, true},
			}
		} else {
			belowThreshold[item.ID] = MemoryState{
				Attempts: 4, Correct: 4,
				CurrentStreak: 4, BestStreak: 4, Stability: 0.5,
				RecentResults: []bool{true, true, true, true},
			}
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, belowThreshold, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("row below 85%% recent accuracy = %d active, want 5", got)
	}
}

func TestProgressiveUnlockingRequiresFourAttempts(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})
	states := make(map[string]MemoryState)
	for index, item := range kanadata.Basic[:5] {
		if index == 0 {
			states[item.ID] = MemoryState{
				Attempts: 3, Correct: 3,
				CurrentStreak: 3, BestStreak: 3, Stability: 0.8,
				RecentResults: []bool{true, true, true},
			}
			continue
		}
		states[item.ID] = MemoryState{
			Attempts: 4, Correct: 4,
			CurrentStreak: 4, BestStreak: 4, Stability: 0.8,
			RecentResults: []bool{true, true, true, true},
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("row with a three-attempt kana = %d active, want 5", got)
	}
}

func TestProgressiveUnlockingRequiresStreak(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})
	states := make(map[string]MemoryState)
	for index, item := range kanadata.Basic[:5] {
		if index == 0 {
			states[item.ID] = MemoryState{
				Attempts: 4, Correct: 3, Wrong: 1,
				CurrentStreak: 2, BestStreak: 2, Stability: 0.4,
				RecentResults: []bool{true, false, true, true},
			}
			continue
		}
		states[item.ID] = MemoryState{
			Attempts: 4, Correct: 4,
			CurrentStreak: 4, BestStreak: 4, Stability: 0.8,
			RecentResults: []bool{true, true, true, true},
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("row with a two-answer streak = %d active, want 5", got)
	}
}

func TestProgressiveUnlockingRecoversAfterCorrection(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})
	states := make(map[string]MemoryState)
	history := make([]bool, 12)
	history[0], history[1] = true, true
	for _, item := range kanadata.Basic[:5] {
		states[item.ID] = MemoryState{
			Attempts: 12, Correct: 2, Wrong: 10,
			CurrentStreak: 0, BestStreak: 2, Stability: 0.3,
			RecentResults: append([]bool(nil), history...),
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("uncorrected row = %d active, want 5", got)
	}

	now := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	recordRound := func(round int) {
		for index, item := range kanadata.Basic[:5] {
			state := states[item.ID]
			state.Record(true, time.Second, int64(round*5+index), now.AddDate(0, 0, round))
			states[item.ID] = state
		}
	}
	for round := range 3 {
		recordRound(round)
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("partially corrected row = %d active, want 5", got)
	}
	recordRound(3)
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 10 {
		t.Fatalf("corrected row = %d active, want 10", got)
	}
}

func TestProgressiveUnlockingDoesNotSkipRows(t *testing.T) {
	scheduler := NewScheduler(1, SchedulerConfig{Progressive: true})
	states := make(map[string]MemoryState)
	for _, item := range kanadata.Basic {
		switch item.Row {
		case kanadata.Rows[0]:
			states[item.ID] = MemoryState{
				Attempts: 4, Correct: 2, Wrong: 2,
				CurrentStreak: 1, BestStreak: 1, Stability: 0.4,
				RecentResults: []bool{true, false, false, true},
			}
		case kanadata.Rows[1]:
			states[item.ID] = MemoryState{
				Attempts: 4, Correct: 4,
				CurrentStreak: 4, BestStreak: 4, Stability: 0.8,
				RecentResults: []bool{true, true, true, true},
			}
		}
	}
	if got := len(scheduler.ActiveKana(kanadata.Basic, states, make(ConfusionMatrix))); got != 5 {
		t.Fatalf("unlocked row skipped a weak first row: %d active, want 5", got)
	}
}

func TestRandomFocusIsUnbiasedAndIgnoresReinforcement(t *testing.T) {
	items := testKana("ki", "ku", "ke")
	states := map[string]MemoryState{
		"ki": {Attempts: 20, Correct: 1, Wrong: 19, Stability: 0.05, ResponseTimeEMA: 5, RecentResults: []bool{false, false, false}},
		"ku": {Attempts: 20, Correct: 20, Stability: 1, ResponseTimeEMA: 0.6, RecentResults: []bool{true, true, true}},
		"ke": {Attempts: 20, Correct: 20, Stability: 1, ResponseTimeEMA: 0.6, RecentResults: []bool{true, true, true}},
	}
	scheduler := NewScheduler(11, SchedulerConfig{Focus: FocusRandom})
	scheduler.reinforcements = []reinforcement{{ID: "ki", Due: 0, Stage: 1}}

	counts := make(map[string]int)
	for index := range int64(3000) {
		item, err := scheduler.Next(items, states, make(ConfusionMatrix), index)
		if err != nil {
			t.Fatal(err)
		}
		counts[item.ID]++
	}
	const expected = 1000.0
	for _, item := range items {
		if math.Abs(float64(counts[item.ID])-expected) > 150 {
			t.Fatalf("random focus was biased: %v", counts)
		}
	}
}

func TestRandomFocusStillSchedulesForAdaptiveMode(t *testing.T) {
	items := testKana("ki", "ku", "ke")
	matrix := make(ConfusionMatrix)
	matrix.Record("ki", "ku")
	scheduler := NewScheduler(13, SchedulerConfig{Focus: FocusRandom})
	scheduler.ObserveResult("ki", "ku", false, 0, matrix)
	if len(scheduler.reinforcements) == 0 {
		t.Fatal("random focus did not record the failed review")
	}

	scheduler.SetFocus(FocusNormal)
	item, err := scheduler.Next(items, nil, matrix, 10)
	if err != nil || item.ID != "ki" {
		t.Fatalf("adaptive follow-up = %v, %v; want ki", item.ID, err)
	}
}

func TestWeightUsesInjectedClockForDayDue(t *testing.T) {
	reviewed := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	state := NewMemoryState()
	state.Record(true, time.Second, 4, reviewed)
	item, ok := kanadata.ByID("ki")
	if !ok {
		t.Fatal("missing ki")
	}
	matrix := make(ConfusionMatrix)
	questionIndex := state.DueIndex + 100

	notDue := NewScheduler(1, SchedulerConfig{Now: func() time.Time {
		return state.DueAt.Add(-time.Hour)
	}})
	overdue := NewScheduler(1, SchedulerConfig{Now: func() time.Time {
		return state.DueAt.AddDate(0, 0, 3)
	}})
	if overdue.Weight(item, state, matrix, questionIndex) <= notDue.Weight(item, state, matrix, questionIndex) {
		t.Fatal("wall-clock overdue card did not receive greater weight")
	}
}

func TestDifficultyRaisesWeight(t *testing.T) {
	item, ok := kanadata.ByID("ki")
	if !ok {
		t.Fatal("missing ki")
	}
	easy := MemoryState{Attempts: 8, Correct: 7, Wrong: 1, Stability: 0.7, Difficulty: 0.1, ResponseTimeEMA: 1, RecentResults: []bool{true}}
	hard := easy
	hard.Difficulty = 0.9
	scheduler := NewScheduler(1, SchedulerConfig{})
	if scheduler.Weight(item, hard, make(ConfusionMatrix), 10) <= scheduler.Weight(item, easy, make(ConfusionMatrix), 10) {
		t.Fatal("harder card did not receive greater weight")
	}
}
