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

func TestRetrievabilityDecaysAcrossDays(t *testing.T) {
	reviewed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	state := NewMemoryState()
	state.Record(true, time.Second, 0, reviewed)

	immediate := state.Retrievability(reviewed)
	oneDay := state.Retrievability(reviewed.AddDate(0, 0, 1))
	oneWeek := state.Retrievability(reviewed.AddDate(0, 0, 7))
	if immediate != 1 {
		t.Fatalf("immediate retrieval = %v, want 1", immediate)
	}
	if !(immediate > oneDay && oneDay > oneWeek) {
		t.Fatalf("retention did not decay: immediate=%v day=%v week=%v", immediate, oneDay, oneWeek)
	}
	if oneWeek <= 0 {
		t.Fatalf("week retention = %v, want > 0", oneWeek)
	}
	if got := (MemoryState{}).Retrievability(reviewed); got != 0 {
		t.Fatalf("unseen retrieval = %v, want 0", got)
	}
	if got := state.Retrievability(reviewed.Add(-time.Hour)); got != 1 {
		t.Fatalf("clock-skew retrieval = %v, want 1", got)
	}
}

func TestDelayedCorrectRecallGrowsMoreThanRapidRepeat(t *testing.T) {
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	base := NewMemoryState()
	base.Record(true, 2*time.Second, 0, start)

	rapid := base
	rapid.Record(true, 2*time.Second, 1, start.Add(time.Second))

	delayed := base
	delayed.Record(true, 2*time.Second, 1, start.AddDate(0, 0, 4))

	rapidGain := rapid.Stability - base.Stability
	delayedGain := delayed.Stability - base.Stability
	if delayedGain <= rapidGain {
		t.Fatalf("delayed gain %.4f did not exceed rapid gain %.4f", delayedGain, rapidGain)
	}
	if delayed.IntervalDays <= rapid.IntervalDays {
		t.Fatalf("delayed interval %d did not exceed rapid interval %d", delayed.IntervalDays, rapid.IntervalDays)
	}
	if delayed.Stability > 1 || rapid.Stability > 1 {
		t.Fatalf("stability escaped bounds: rapid=%v delayed=%v", rapid.Stability, delayed.Stability)
	}
}

func TestLapseShortensDayInterval(t *testing.T) {
	start := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	base := NewMemoryState()
	base.Record(true, time.Second, 0, start)
	reviewAt := start.AddDate(0, 0, base.IntervalDays)
	if !base.DueAt.Equal(reviewAt) {
		t.Fatalf("first due = %v, want %v", base.DueAt, reviewAt)
	}

	success := base
	success.Record(true, time.Second, 1, reviewAt)

	lapse := base
	lapse.Record(false, time.Second, 1, reviewAt)

	if lapse.IntervalDays >= success.IntervalDays {
		t.Fatalf("lapse interval %d not shorter than success interval %d", lapse.IntervalDays, success.IntervalDays)
	}
	if !lapse.DueAt.Before(success.DueAt) {
		t.Fatalf("lapse due %v not before success due %v", lapse.DueAt, success.DueAt)
	}
	if want := reviewAt.AddDate(0, 0, lapse.IntervalDays); !lapse.DueAt.Equal(want) {
		t.Fatalf("lapse due = %v, want %v", lapse.DueAt, want)
	}
}

func TestRecentAccuracyUsesBoundedWindows(t *testing.T) {
	state := MemoryState{RecentResults: []bool{false, false, true, true, true}}
	if got := state.RecentAccuracy(); math.Abs(got-0.6) > 0.0001 {
		t.Fatalf("full recent accuracy = %v, want 0.6", got)
	}
	if got := state.recentAccuracy(4); math.Abs(got-0.75) > 0.0001 {
		t.Fatalf("latest-four accuracy = %v, want 0.75", got)
	}
	if got := state.recentAccuracy(0); got != 0 {
		t.Fatalf("empty window accuracy = %v, want 0", got)
	}
	if got := (MemoryState{}).RecentAccuracy(); got != 0 {
		t.Fatalf("unseen recent accuracy = %v, want 0", got)
	}
}

func TestReviewIntervalDaysStaysDeterministicAndBounded(t *testing.T) {
	if got := ReviewIntervalDays(-1, -1); got != int(minimumReviewDays) {
		t.Fatalf("lower bound interval = %d, want %d", got, int(minimumReviewDays))
	}
	if got := ReviewIntervalDays(2, 2); float64(got) < minimumReviewDays || float64(got) > maximumReviewDays {
		t.Fatalf("clamped interval = %d, want between %v and %v", got, minimumReviewDays, maximumReviewDays)
	}
	if ReviewIntervalDays(1, 0) <= ReviewIntervalDays(0.5, 0) {
		t.Fatal("higher stability did not extend the day interval")
	}
	if ReviewIntervalDays(0.6, 0.9) >= ReviewIntervalDays(0.6, 0.1) {
		t.Fatal("higher difficulty did not shorten the day interval")
	}
	first := ReviewIntervalDays(0.6, 0.4)
	second := ReviewIntervalDays(0.6, 0.4)
	if first != second {
		t.Fatal("day interval is not deterministic")
	}
}
