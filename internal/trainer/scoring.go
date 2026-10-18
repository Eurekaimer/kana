package trainer

import (
	"math"
	"time"
)

const (
	recentResultLimit = 12
	defaultDifficulty = 0.40

	// Retention half-life grows with bounded stability and shrinks with difficulty.
	// These constants keep the curve conservative: a brand-new card retains
	// roughly six days at neutral difficulty, while a strong card retains weeks.
	retentionBaseHalfLifeDays      = 4.0
	retentionStabilityHalfLifeDays = 14.0
	retentionDifficultyPenalty     = 0.35

	minimumReviewDays = 1.0
	maximumReviewDays = 30.0
)

// ScoringConfig contains the tunable constants for memory updates and scoring.
type ScoringConfig struct {
	InitialStability          float64
	CorrectGain               float64
	FastBonus                 float64
	StreakBonus               float64
	SpacingBonus              float64
	WrongMultiplier           float64
	FastThresholdSeconds      float64
	DifficultyCorrectLearning float64
	DifficultyWrongLearning   float64
	EMAPreviousWeight         float64
	AccuracyWeight            float64
	ResponseWeight            float64
	RecentErrorWeight         float64
	StabilityWeight           float64
	ConfusionWeight           float64
}

// DefaultScoringConfig is conservative with sparse observations and strongly penalizes mistakes.
var DefaultScoringConfig = ScoringConfig{
	InitialStability:          0.20,
	CorrectGain:               0.06,
	FastBonus:                 0.04,
	StreakBonus:               0.02,
	SpacingBonus:              0.15,
	WrongMultiplier:           0.55,
	FastThresholdSeconds:      1.5,
	DifficultyCorrectLearning: 0.06,
	DifficultyWrongLearning:   0.20,
	EMAPreviousWeight:         0.8,
	AccuracyWeight:            0.35,
	ResponseWeight:            0.20,
	RecentErrorWeight:         0.20,
	StabilityWeight:           0.15,
	ConfusionWeight:           0.10,
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
	Difficulty      float64   `json:"difficulty"`
	IntervalDays    int       `json:"interval_days"`
	DueAt           time.Time `json:"due_at,omitempty"`
	ResponseTimeEMA float64   `json:"response_time_ema"`
	RecentResults   []bool    `json:"recent_results,omitempty"`
}

// NewMemoryState returns an unseen state with the configured initial stability.
func NewMemoryState() MemoryState {
	state := MemoryState{
		Stability:  DefaultScoringConfig.InitialStability,
		Difficulty: defaultDifficulty,
	}
	state.IntervalDays = ReviewIntervalDays(state.Stability, state.Difficulty)
	return state
}

// Record applies one result and returns the next normal review interval in questions.
// Stability remains the bounded [0,1] mastery signal. Difficulty and the redundant
// due time describe wall-clock retention without changing existing mastery weights.
func (m *MemoryState) Record(correct bool, responseTime time.Duration, questionIndex int64, now time.Time) int {
	cfg := DefaultScoringConfig
	if m.Attempts == 0 && m.Stability == 0 {
		m.Stability = cfg.InitialStability
	}
	if m.Difficulty == 0 && m.IntervalDays <= 0 {
		if m.Attempts == 0 {
			m.Difficulty = defaultDifficulty
		} else {
			// Repair a legacy state from its durable accuracy evidence.
			m.Difficulty = clamp(0.25+0.5*float64(m.Wrong)/float64(m.Attempts), 0, 1)
		}
	}

	elapsedKnown := m.Attempts > 0 && !m.LastSeen.IsZero()
	retention := 1.0
	if elapsedKnown {
		retention = m.Retrievability(now)
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
		if elapsedKnown {
			// Successful recall after a longer absence is stronger evidence than
			// repeating a card in quick succession.
			gain += cfg.SpacingBonus * (1 - retention)
		}
		m.Stability = clamp(m.Stability+gain, 0, 1)
		// Difficulty moves toward zero on correct answers; slow successes count
		// as weaker evidence than fast ones.
		learning := cfg.DifficultyCorrectLearning
		if seconds >= cfg.FastThresholdSeconds {
			learning *= 0.5
		}
		m.Difficulty = clamp(m.Difficulty*(1-learning), 0, 1)
	} else {
		m.Wrong++
		m.CurrentStreak = 0
		m.Stability = clamp(m.Stability*cfg.WrongMultiplier, 0, 1)
		// A lapse moves difficulty toward one, shortening future retention.
		m.Difficulty = clamp(m.Difficulty+(1-m.Difficulty)*cfg.DifficultyWrongLearning, 0, 1)
	}

	interval := ReviewInterval(m.Stability)
	m.DueIndex = questionIndex + int64(interval)
	m.IntervalDays = ReviewIntervalDays(m.Stability, m.Difficulty)
	if !now.IsZero() {
		m.DueAt = now.AddDate(0, 0, m.IntervalDays)
	}
	return interval
}

// repair fills fields introduced after version 1 from durable statistics. It
// never rewrites attempts, accuracy, streaks, response times, or question dues.
func (m *MemoryState) repair() {
	if m.Stability == 0 && m.Attempts == 0 {
		m.Stability = DefaultScoringConfig.InitialStability
	}
	if m.Difficulty == 0 && m.IntervalDays <= 0 {
		if m.Attempts == 0 {
			m.Difficulty = defaultDifficulty
		} else {
			m.Difficulty = clamp(0.25+0.5*float64(m.Wrong)/float64(m.Attempts), 0, 1)
		}
	}
	m.Stability = clamp(m.Stability, 0, 1)
	m.Difficulty = clamp(m.Difficulty, 0, 1)
	if m.IntervalDays <= 0 {
		m.IntervalDays = ReviewIntervalDays(m.Stability, m.Difficulty)
	}
	if m.DueAt.IsZero() && !m.LastSeen.IsZero() {
		m.DueAt = m.LastSeen.AddDate(0, 0, m.IntervalDays)
	}
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

// RecentAccuracy is the correctness ratio in the bounded recent-result window.
// It returns zero when no result has been recorded, so callers can decide
// whether that means "unknown" or "not ready".
func (m MemoryState) RecentAccuracy() float64 {
	return m.recentAccuracy(len(m.RecentResults))
}

// recentAccuracy returns the correctness ratio of the newest limit results.
// Slicing the existing window avoids allocations on every unlock check.
func (m MemoryState) recentAccuracy(limit int) float64 {
	if limit <= 0 || len(m.RecentResults) == 0 {
		return 0
	}
	start := len(m.RecentResults) - limit
	if start < 0 {
		start = 0
	}
	correct := 0
	for _, result := range m.RecentResults[start:] {
		if result {
			correct++
		}
	}
	return float64(correct) / float64(len(m.RecentResults)-start)
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

// Retrievability estimates the probability that this kana is still recalled at
// now. It uses exponential forgetting whose half-life grows with stability and
// shrinks with difficulty. Unseen cards return zero because they have no memory
// to retain yet; a clock before LastSeen returns one.
func (m MemoryState) Retrievability(now time.Time) float64 {
	if m.LastSeen.IsZero() {
		return 0
	}
	if !now.After(m.LastSeen) {
		return 1
	}
	halfLife := m.retentionHalfLifeDays()
	if halfLife <= 0 {
		return 0
	}
	elapsedDays := now.Sub(m.LastSeen).Hours() / 24
	return clamp(math.Exp(-math.Ln2*elapsedDays/halfLife), 0, 1)
}

// retentionHalfLifeDays is the transparent forgetting curve used by
// Retrievability. Difficulty above the neutral point shortens retention; the
// result stays positive because both inputs are clamped to [0,1].
func (m MemoryState) retentionHalfLifeDays() float64 {
	stability := clamp(m.Stability, 0, 1)
	difficulty := clamp(m.Difficulty, 0, 1)
	return (retentionBaseHalfLifeDays + retentionStabilityHalfLifeDays*stability) *
		(1 - retentionDifficultyPenalty*difficulty)
}

// ReviewIntervalDays converts stability and difficulty into a deterministic
// calendar-day interval clamped to [1,30]. It deliberately contains no
// randomness so a card's due time only changes after a recorded review.
func ReviewIntervalDays(stability, difficulty float64) int {
	days := (1 + 21*math.Pow(clamp(stability, 0, 1), 2)) *
		(1.15 - 0.30*clamp(difficulty, 0, 1))
	if days < minimumReviewDays {
		days = minimumReviewDays
	}
	if days > maximumReviewDays {
		days = maximumReviewDays
	}
	return int(math.Round(days))
}

func clamp(value, low, high float64) float64 {
	return math.Max(low, math.Min(high, value))
}
