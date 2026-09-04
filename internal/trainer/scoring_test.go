package trainer

import (
	"math"
	"testing"
	"time"
)

func TestSmoothedAccuracy(t *testing.T) {
	cases := []struct {
		state MemoryState
		want  float64
	}{
		{MemoryState{}, 0.5},
		{MemoryState{Attempts: 1, Correct: 1}, 0.6},
		{MemoryState{Attempts: 6, Correct: 4}, 0.6},
	}
	for _, tc := range cases {
		if got := tc.state.SmoothedAccuracy(); got != tc.want {
			t.Errorf("SmoothedAccuracy() = %v, want %v", got, tc.want)
		}
	}
}

func TestRecordUpdatesEMAAndStability(t *testing.T) {
	now := time.Unix(100, 0)
	state := NewMemoryState()
	state.Record(true, time.Second, 1, now)
	if state.ResponseTimeEMA != 1 {
		t.Fatalf("first EMA = %v, want 1", state.ResponseTimeEMA)
	}
	if math.Abs(state.Stability-0.30) > 0.0001 {
		t.Fatalf("fast correct stability = %v, want 0.30", state.Stability)
	}

	state.Record(true, 2*time.Second, 2, now.Add(time.Second))
	if math.Abs(state.ResponseTimeEMA-1.2) > 0.0001 {
		t.Fatalf("second EMA = %v, want 1.2", state.ResponseTimeEMA)
	}
	beforeWrong := state.Stability
	state.Record(false, time.Second, 3, now.Add(2*time.Second))
	if math.Abs(state.Stability-beforeWrong*0.55) > 0.0001 {
		t.Fatalf("wrong stability = %v, want %v", state.Stability, beforeWrong*0.55)
	}
	if state.CurrentStreak != 0 || state.Wrong != 1 {
		t.Fatalf("wrong counters = %+v", state)
	}
}

func TestReviewIntervalIncreases(t *testing.T) {
	previous := 0
	for _, stability := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
		interval := ReviewInterval(stability)
		if interval <= previous {
			t.Fatalf("interval did not increase at %.1f: %d <= %d", stability, interval, previous)
		}
		previous = interval
	}
}

func TestRecentErrorsRaiseWeakness(t *testing.T) {
	strong := MemoryState{Attempts: 10, Correct: 9, Stability: 0.8, ResponseTimeEMA: 1, RecentResults: []bool{true, true, true}}
	weak := strong
	weak.RecentResults = []bool{true, false, false}
	if weak.Weakness(0) <= strong.Weakness(0) {
		t.Fatal("recent mistakes did not raise weakness")
	}
}

func TestResponseAndMasteryBounds(t *testing.T) {
	if got := (MemoryState{}).ResponseScore(); got != 0.5 {
		t.Fatalf("unseen response score = %v", got)
	}
	fast := MemoryState{Attempts: 1, Correct: 1, Stability: 1, ResponseTimeEMA: 0.5, RecentResults: []bool{true}}
	slow := MemoryState{Attempts: 1, Stability: 0, ResponseTimeEMA: 10, RecentResults: []bool{false}}
	if fast.ResponseScore() != 0 || slow.ResponseScore() != 1 {
		t.Fatalf("response scores: fast=%v slow=%v", fast.ResponseScore(), slow.ResponseScore())
	}
	if fast.Mastery(0) <= slow.Mastery(1) {
		t.Fatalf("mastery ordering: fast=%v slow=%v", fast.Mastery(0), slow.Mastery(1))
	}
	if got := clamp(-1, 0, 1); got != 0 {
		t.Fatalf("lower clamp = %v", got)
	}
	if got := clamp(2, 0, 1); got != 1 {
		t.Fatalf("upper clamp = %v", got)
	}
}

func TestRecentHistoryIsBounded(t *testing.T) {
	state := NewMemoryState()
	for index := range int64(recentResultLimit + 5) {
		state.Record(index%2 == 0, time.Second, index, time.Unix(index, 0))
	}
	if len(state.RecentResults) != recentResultLimit {
		t.Fatalf("recent result count = %d", len(state.RecentResults))
	}
}
