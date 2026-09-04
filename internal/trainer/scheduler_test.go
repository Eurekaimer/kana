package trainer

import (
	"testing"

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
		states[item.ID] = MemoryState{Attempts: 20, Correct: 19, Wrong: 1, Stability: 0.8}
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
