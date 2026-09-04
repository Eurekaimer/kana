package trainer

import (
	"math"
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func TestWeakStatsOrderingAndLimit(t *testing.T) {
	progress := NewProgress(testKana("ki", "ku"), time.Unix(1, 0))
	progress.States["ki"] = MemoryState{Attempts: 10, Correct: 2, Wrong: 8, Stability: 0.1, ResponseTimeEMA: 5}
	progress.States["ku"] = MemoryState{Attempts: 10, Correct: 9, Wrong: 1, Stability: 0.9, ResponseTimeEMA: 1}
	statistics := WeakStats(progress, testKana("ki", "ku"), 1)
	if len(statistics) != 1 || statistics[0].Kana.ID != "ki" {
		t.Fatalf("weak statistics = %+v", statistics)
	}
}

func TestOverall(t *testing.T) {
	progress := NewProgress(kanadata.Basic, time.Unix(1, 0))
	progress.States["ki"] = MemoryState{Attempts: 2, Correct: 1, Wrong: 1, ResponseTimeEMA: 2}
	progress.States["ku"] = MemoryState{Attempts: 1, Correct: 1, ResponseTimeEMA: 1}
	overall := Overall(progress)
	if overall.Attempts != 3 || overall.Correct != 2 || math.Abs(overall.Accuracy-66.6667) > 0.001 {
		t.Fatalf("overall = %+v", overall)
	}
	if overall.AverageResponse != 1666666666*time.Nanosecond {
		t.Fatalf("average response = %v", overall.AverageResponse)
	}
}
