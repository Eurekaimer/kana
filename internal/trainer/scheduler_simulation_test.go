package trainer

import (
	"testing"
	"time"
)

func TestSchedulerLearningSimulations(t *testing.T) {
	t.Run("weak focus moves after recovery", func(t *testing.T) {
		items := testKana("a", "i", "u", "e", "o", "ki", "ku", "ke", "ko", "shi")
		states := make(map[string]MemoryState)
		for _, item := range items {
			states[item.ID] = NewMemoryState()
		}
		matrix := make(ConfusionMatrix)
		scheduler := NewScheduler(21, SchedulerConfig{})
		phaseCounts := [3]map[string]int{make(map[string]int), make(map[string]int), make(map[string]int)}
		now := time.Unix(1000, 0)

		for index := range int64(600) {
			item, err := scheduler.Next(items, states, matrix, index)
			if err != nil {
				t.Fatal(err)
			}
			phase := int(index / 200)
			phaseCounts[phase][item.ID]++
			correct := !(item.ID == "ki" && index < 200)
			response := 800 * time.Millisecond
			answeredID := ""
			if !correct {
				response = 4 * time.Second
				answeredID = "ku"
				matrix.Record(item.ID, answeredID)
			}
			state := states[item.ID]
			state.Record(correct, response, index, now.Add(time.Duration(index)*time.Second))
			states[item.ID] = state
			scheduler.ObserveResult(item.ID, answeredID, correct, index, matrix)
		}

		t.Logf("phase 1 (ki fails): %v", phaseCounts[0])
		t.Logf("phase 2 (ki recovers): %v", phaseCounts[1])
		t.Logf("phase 3 (focus shifts): %v", phaseCounts[2])
		if phaseCounts[0]["ki"] <= 30 {
			t.Fatalf("failed ki was not strongly reinforced: %d", phaseCounts[0]["ki"])
		}
		if phaseCounts[2]["ki"] >= phaseCounts[0]["ki"] {
			t.Fatalf("focus did not move after recovery: first=%d third=%d", phaseCounts[0]["ki"], phaseCounts[2]["ki"])
		}
	})

	t.Run("learned confusion reinforces pair", func(t *testing.T) {
		items := testKana("na", "ni", "nu", "ne", "no", "ha", "hi", "fu")
		states := make(map[string]MemoryState)
		for _, item := range items {
			states[item.ID] = MemoryState{Attempts: 20, Correct: 18, Wrong: 2, Stability: 0.75, ResponseTimeEMA: 1.2, RecentResults: []bool{true, true}}
		}
		matrix := make(ConfusionMatrix)
		matrix["nu"] = map[string]int{"ne": 7}
		matrix["ne"] = map[string]int{"nu": 5}
		scheduler := NewScheduler(33, SchedulerConfig{})
		counts := make(map[string]int)
		for index := range int64(800) {
			item, err := scheduler.Next(items, states, matrix, index)
			if err != nil {
				t.Fatal(err)
			}
			counts[item.ID]++
		}
		t.Logf("confused-pair distribution: %v", counts)
		pairAverage := float64(counts["nu"]+counts["ne"]) / 2
		otherAverage := float64(800-counts["nu"]-counts["ne"]) / float64(len(items)-2)
		if pairAverage <= otherAverage*1.15 {
			t.Fatalf("confused pair not sufficiently emphasized: pair=%.1f others=%.1f", pairAverage, otherAverage)
		}
	})

	t.Run("mastered item remains visible", func(t *testing.T) {
		items := testKana("a", "i", "u", "e", "o")
		states := make(map[string]MemoryState)
		for _, item := range items {
			states[item.ID] = MemoryState{Attempts: 30, Correct: 20, Wrong: 10, Stability: 0.5, ResponseTimeEMA: 2, RecentResults: []bool{true}}
		}
		states["a"] = MemoryState{Attempts: 100, Correct: 98, Wrong: 2, Stability: 0.98, ResponseTimeEMA: 0.7, RecentResults: []bool{true, true, true}}
		scheduler := NewScheduler(55, SchedulerConfig{})
		counts := make(map[string]int)
		for index := range int64(1000) {
			item, err := scheduler.Next(items, states, make(ConfusionMatrix), index)
			if err != nil {
				t.Fatal(err)
			}
			counts[item.ID]++
		}
		t.Logf("mastery distribution: %v", counts)
		if counts["a"] == 0 {
			t.Fatal("mastered a starved")
		}
		if counts["a"] >= counts["i"] {
			t.Fatalf("mastered a was not deprioritized: %v", counts)
		}
	})
}
