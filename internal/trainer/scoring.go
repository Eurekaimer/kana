package trainer

import (
	"math"
	"time"
)

const recentResultLimit = 12

// ScoringConfig contains the tunable constants for memory updates and scoring.
type ScoringConfig struct {
	InitialStability     float64
	CorrectGain          float64
	FastBonus            float64
	StreakBonus          float64
	WrongMultiplier      float64
	FastThresholdSeconds float64
	EMAPreviousWeight    float64
	AccuracyWeight       float64
	ResponseWeight       float64
	RecentErrorWeight    float64
	StabilityWeight      float64
	ConfusionWeight      float64
}

// DefaultScoringConfig is conservative with sparse observations and strongly penalizes mistakes.
var DefaultScoringConfig = ScoringConfig{
	InitialStability:     0.20,
	CorrectGain:          0.06,
	FastBonus:            0.04,
	StreakBonus:          0.02,
	WrongMultiplier:      0.55,
	FastThresholdSeconds: 1.5,
	EMAPreviousWeight:    0.8,
	AccuracyWeight:       0.35,
	ResponseWeight:       0.20,
	RecentErrorWeight:    0.20,
	StabilityWeight:      0.15,
	ConfusionWeight:      0.10,
}

// MemoryState stores durable learning state for one kana.
type MemoryState struct {
	Attempts        int       `json:"attempts"`
	Correct         int       `json:"correct"`
	Wrong           int       `json:"wrong"`
	CurrentStreak   int       `json:"current_streak"`
	BestStreak      int       `json:"best_streak"`
	LastSeen        time.Time `json:"last_seen,omitempty"`
	LastCorrect     time.Time `json:"last_correct,omitempty"`
	LastSeenIndex   int64     `json:"last_seen_index"`
	DueIndex        int64     `json:"due_index"`
	Stability       float64   `json:"stability"`
	ResponseTimeEMA float64   `json:"response_time_ema"`
	RecentResults   []bool    `json:"recent_results,omitempty"`
}

// NewMemoryState returns an unseen state with the configured initial stability.
func NewMemoryState() MemoryState {
	return MemoryState{Stability: DefaultScoringConfig.InitialStability}
}

// Record applies one result and returns the next normal review interval in questions.
func (m *MemoryState) Record(correct bool, responseTime time.Duration, questionIndex int64, now time.Time) int {
	cfg := DefaultScoringConfig
	if m.Attempts == 0 && m.Stability == 0 {
		m.Stability = cfg.InitialStability
	}

	seconds := math.Max(responseTime.Seconds(), 0)
	if m.Attempts == 0 {
		m.ResponseTimeEMA = seconds
	} else {
		m.ResponseTimeEMA = cfg.EMAPreviousWeight*m.ResponseTimeEMA + (1-cfg.EMAPreviousWeight)*seconds
	}

	m.Attempts++
	m.LastSeen = now
	m.LastSeenIndex = questionIndex
	m.RecentResults = append(m.RecentResults, correct)
	if len(m.RecentResults) > recentResultLimit {
		m.RecentResults = append([]bool(nil), m.RecentResults[len(m.RecentResults)-recentResultLimit:]...)
	}

	if correct {
		m.Correct++
		m.CurrentStreak++
		if m.CurrentStreak > m.BestStreak {
			m.BestStreak = m.CurrentStreak
		}
		m.LastCorrect = now
		gain := cfg.CorrectGain
		if seconds < cfg.FastThresholdSeconds {
			gain += cfg.FastBonus
		}
		if m.CurrentStreak >= 3 {
			gain += cfg.StreakBonus
		}
		m.Stability = clamp(m.Stability+gain, 0, 1)
	} else {
		m.Wrong++
		m.CurrentStreak = 0
		m.Stability = clamp(m.Stability*cfg.WrongMultiplier, 0, 1)
	}

	interval := ReviewInterval(m.Stability)
	m.DueIndex = questionIndex + int64(interval)
	return interval
}

// SmoothedAccuracy uses a beta prior so sparse success does not imply mastery.
func (m MemoryState) SmoothedAccuracy() float64 {
	return float64(m.Correct+2) / float64(m.Attempts+4)
}

// ResponseScore maps response latency to a normalized weakness value.
func (m MemoryState) ResponseScore() float64 {
	if m.Attempts == 0 {
		return 0.5
	}
	return clamp((m.ResponseTimeEMA-1.5)/4.0, 0, 1)
}

// RecentErrorScore weights newer mistakes more heavily than older ones.
func (m MemoryState) RecentErrorScore() float64 {
	if len(m.RecentResults) == 0 {
		return 0.5
	}
	var errors, total float64
	for i, correct := range m.RecentResults {
		weight := float64(i + 1)
		total += weight
		if !correct {
			errors += weight
		}
	}
	return errors / total
}

// Weakness combines accuracy, latency, recent results, stability, and confusion.
func (m MemoryState) Weakness(confusionScore float64) float64 {
	cfg := DefaultScoringConfig
	return clamp(
		cfg.AccuracyWeight*(1-m.SmoothedAccuracy())+
			cfg.ResponseWeight*m.ResponseScore()+
			cfg.RecentErrorWeight*m.RecentErrorScore()+
			cfg.StabilityWeight*(1-m.Stability)+
			cfg.ConfusionWeight*clamp(confusionScore, 0, 1),
		0, 1,
	)
}

// Mastery is an easy-to-read inverse weakness percentage.
func (m MemoryState) Mastery(confusionScore float64) float64 {
	return 1 - m.Weakness(confusionScore)
}

// ReviewInterval converts stability into a question-count interval.
func ReviewInterval(stability float64) int {
	return 2 + int(math.Pow(clamp(stability, 0, 1), 2)*80)
}

func clamp(value, low, high float64) float64 {
	return math.Max(low, math.Min(high, value))
}
