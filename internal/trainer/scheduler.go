package trainer

import (
	"errors"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	kanadata "kana/internal/kana"
)

const (
	minimumWeight       = 0.02
	unlockAttempts      = 4
	unlockRecentWindow  = 4
	unlockStreak        = 3
	unlockRowAccuracy   = 0.85
	defaultWeakLimit    = 12
	meaningfulConfusion = 2
)

// FocusMode changes the candidate pool without changing the memory model.
type FocusMode string

const (
	FocusNormal    FocusMode = "normal"
	FocusWeak      FocusMode = "weak"
	FocusConfusion FocusMode = "confusion"
	FocusRandom    FocusMode = "random"
)

// SchedulerConfig controls progression and focused practice.
type SchedulerConfig struct {
	Progressive bool
	Focus       FocusMode
	WeakLimit   int
	// Now supplies wall-clock time for retention weighting. Nil means time.Now.
	Now func() time.Time
}

type reinforcement struct {
	ID    string
	Due   int64
	Stage int
}

// Scheduler performs weighted adaptive selection with deterministic randomness.
type Scheduler struct {
	rng               *rand.Rand
	config            SchedulerConfig
	reinforcements    []reinforcement
	history           []string
	selectedID        string
	selectedStage     int
	selectedReinforce bool
}

// NewScheduler creates a scheduler. A fixed seed makes tests and experiments repeatable.
// An unset config.Now defaults to time.Now.
func NewScheduler(seed int64, config SchedulerConfig) *Scheduler {
	if config.WeakLimit <= 0 {
		config.WeakLimit = defaultWeakLimit
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Scheduler{rng: rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15)), config: config}
}

// SetFocus changes the candidate emphasis while preserving scheduler history.
func (s *Scheduler) SetFocus(focus FocusMode) {
	s.config.Focus = focus
}

// SetProgressive enables or disables row-by-row unlocking.
func (s *Scheduler) SetProgressive(progressive bool) {
	s.config.Progressive = progressive
}

// Config returns the active scheduler settings.
func (s *Scheduler) Config() SchedulerConfig {
	return s.config
}

// ActiveKana returns the currently unlocked and focus-filtered candidate set.
func (s *Scheduler) ActiveKana(items []kanadata.Kana, states map[string]MemoryState, matrix ConfusionMatrix) []kanadata.Kana {
	active := items
	if s.config.Progressive {
		active = progressiveKana(items, states)
	}

	switch s.config.Focus {
	case FocusWeak:
		return weakestKana(active, states, matrix, s.config.WeakLimit)
	case FocusConfusion:
		focused := confusedKana(active, matrix)
		if len(focused) >= 2 {
			return focused
		}
		return weakestKana(active, states, matrix, s.config.WeakLimit)
	case FocusRandom:
		return active
	default:
		return active
	}
}

// Weight returns the base scheduling weight for one candidate.
func (s *Scheduler) Weight(item kanadata.Kana, state MemoryState, matrix ConfusionMatrix, questionIndex int64) float64 {
	return s.weight(item, state, matrix, questionIndex, s.currentTime())
}

// weight is Weight with an explicit timestamp so one selection uses one clock read.
func (s *Scheduler) weight(item kanadata.Kana, state MemoryState, matrix ConfusionMatrix, questionIndex int64, now time.Time) float64 {
	weakness := state.Weakness(matrix.Score(item.ID))
	weight := minimumWeight + weakness*weakness
	if state.Attempts == 0 {
		weight *= 2.2
	}
	if questionIndex >= state.DueIndex {
		overdue := questionIndex - state.DueIndex
		weight *= 1.5 + math.Min(float64(overdue)*0.12, 2.5)
	} else {
		weight *= 0.45
	}
	if state.Difficulty > 0 {
		weight *= 1 + 0.35*clamp(state.Difficulty, 0, 1)
	}
	if state.Attempts > 0 && !state.LastSeen.IsZero() {
		dueAt := state.DueAt
		if dueAt.IsZero() {
			days := state.IntervalDays
			if days <= 0 {
				days = ReviewIntervalDays(state.Stability, state.Difficulty)
			}
			dueAt = state.LastSeen.AddDate(0, 0, days)
		}
		retention := state.Retrievability(now)
		if !now.Before(dueAt) {
			overdueDays := now.Sub(dueAt).Hours() / 24
			weight *= 1.35 + math.Min(overdueDays*0.15, 1.5) + 0.5*(1-retention)
		} else {
			weight *= 0.55 + 0.45*retention
		}
	}
	if matrix.TotalFor(item.ID) >= meaningfulConfusion {
		weight *= 1.25 + 0.5*matrix.Score(item.ID)
	}
	return math.Max(weight, minimumWeight)
}

// currentTime returns the injected clock, falling back safely for zero configs.
func (s *Scheduler) currentTime() time.Time {
	if s.config.Now != nil {
		return s.config.Now()
	}
	return time.Now()
}

// Next selects a kana while normally excluding the previous two questions.
func (s *Scheduler) Next(items []kanadata.Kana, states map[string]MemoryState, matrix ConfusionMatrix, questionIndex int64) (kanadata.Kana, error) {
	candidates := s.ActiveKana(items, states, matrix)
	if len(candidates) == 0 {
		return kanadata.Kana{}, errors.New("no active kana")
	}

	allowed := s.avoidRecent(candidates)
	if s.config.Focus == FocusRandom {
		// Unbiased sampling ignores weights and pending reinforcements. History
		// still excludes recent cards, and answers are recorded normally for a
		// later adaptive session.
		selected := allowed[s.rng.IntN(len(allowed))]
		s.selectedID = selected.ID
		s.selectedStage = 0
		s.selectedReinforce = false
		s.remember(selected.ID)
		return selected, nil
	}
	if item, entry, ok := s.dueReinforcement(allowed, questionIndex); ok {
		s.selectedID = item.ID
		s.selectedStage = entry.Stage
		s.selectedReinforce = entry.Stage > 0
		s.remember(item.ID)
		return item, nil
	}

	weights := make([]float64, len(allowed))
	total := 0.0
	now := s.currentTime()
	for i, item := range allowed {
		weights[i] = s.weight(item, states[item.ID], matrix, questionIndex, now)
		total += weights[i]
	}
	pick := s.rng.Float64() * total
	selected := allowed[len(allowed)-1]
	for i, weight := range weights {
		pick -= weight
		if pick <= 0 {
			selected = allowed[i]
			break
		}
	}
	s.selectedID = selected.ID
	s.selectedStage = 0
	s.selectedReinforce = false
	s.remember(selected.ID)
	return selected, nil
}

// ObserveResult schedules failure, follow-up, and contrastive reviews.
// Call it after recording the confusion in matrix.
func (s *Scheduler) ObserveResult(expectedID, answeredID string, correct bool, questionIndex int64, matrix ConfusionMatrix) {
	if !correct {
		s.removeReinforcement(expectedID)
		delay := int64(2 + s.rng.IntN(3))
		if s.selectedReinforce || s.recentFailurePressure(expectedID) {
			delay = int64(1 + s.rng.IntN(2))
		}
		s.reinforcements = append(s.reinforcements, reinforcement{ID: expectedID, Due: questionIndex + delay, Stage: 1})
		if answeredID != "" && matrix.Count(expectedID, answeredID)+matrix.Count(answeredID, expectedID) >= meaningfulConfusion {
			s.scheduleContrast(answeredID, questionIndex+int64(2+s.rng.IntN(4)))
		}
		return
	}
	if s.selectedID == expectedID && s.selectedReinforce && s.selectedStage == 1 {
		s.removeReinforcement(expectedID)
		s.reinforcements = append(s.reinforcements, reinforcement{ID: expectedID, Due: questionIndex + int64(5+s.rng.IntN(6)), Stage: 2})
	}
}

func (s *Scheduler) avoidRecent(candidates []kanadata.Kana) []kanadata.Kana {
	if len(candidates) < 2 || len(s.history) == 0 {
		return candidates
	}
	blocked := map[string]bool{s.history[len(s.history)-1]: true}
	if len(candidates) > 3 && len(s.history) > 1 {
		blocked[s.history[len(s.history)-2]] = true
	}
	allowed := make([]kanadata.Kana, 0, len(candidates))
	for _, item := range candidates {
		if !blocked[item.ID] {
			allowed = append(allowed, item)
		}
	}
	if len(allowed) == 0 {
		return candidates
	}
	return allowed
}

func (s *Scheduler) dueReinforcement(candidates []kanadata.Kana, questionIndex int64) (kanadata.Kana, reinforcement, bool) {
	candidateByID := make(map[string]kanadata.Kana, len(candidates))
	for _, item := range candidates {
		candidateByID[item.ID] = item
	}
	best := -1
	for i, entry := range s.reinforcements {
		if entry.Due > questionIndex {
			continue
		}
		if _, ok := candidateByID[entry.ID]; !ok {
			continue
		}
		if best == -1 || entry.Due < s.reinforcements[best].Due {
			best = i
		}
	}
	if best == -1 {
		return kanadata.Kana{}, reinforcement{}, false
	}
	entry := s.reinforcements[best]
	s.reinforcements = append(s.reinforcements[:best], s.reinforcements[best+1:]...)
	return candidateByID[entry.ID], entry, true
}

func (s *Scheduler) remember(id string) {
	s.history = append(s.history, id)
	if len(s.history) > 4 {
		s.history = append([]string(nil), s.history[len(s.history)-4:]...)
	}
}

func (s *Scheduler) removeReinforcement(id string) {
	kept := s.reinforcements[:0]
	for _, entry := range s.reinforcements {
		if entry.ID != id {
			kept = append(kept, entry)
		}
	}
	s.reinforcements = kept
}

func (s *Scheduler) scheduleContrast(id string, due int64) {
	for _, entry := range s.reinforcements {
		if entry.ID == id && entry.Stage == 0 {
			return
		}
	}
	s.reinforcements = append(s.reinforcements, reinforcement{ID: id, Due: due})
}

func (s *Scheduler) recentFailurePressure(id string) bool {
	count := 0
	for _, entry := range s.reinforcements {
		if entry.ID == id && entry.Stage == 1 {
			count++
		}
	}
	return count > 0
}

// progressiveKana unlocks rows strictly in order. A row unlocks when every kana
// in it has at least four attempts and a current streak of at least three
// correct answers, and the row's mean latest-four accuracy is at least 85%.
// Old mistakes leave the four-answer window, so corrections restore an unlock
// without erasing lifetime statistics.
func progressiveKana(items []kanadata.Kana, states map[string]MemoryState) []kanadata.Kana {
	unlockedRows := 1
	for rowIndex := range len(kanadata.Rows) - 1 {
		row := kanadata.Rows[rowIndex]
		count := 0
		accuracy := 0.0
		ready := true
		for _, item := range items {
			if item.Row != row {
				continue
			}
			state := states[item.ID]
			count++
			accuracy += state.recentAccuracy(unlockRecentWindow)
			if state.Attempts < unlockAttempts || state.CurrentStreak < unlockStreak {
				ready = false
			}
		}
		if count == 0 || !ready || accuracy/float64(count) < unlockRowAccuracy {
			break
		}
		unlockedRows++
	}
	allowedRows := make(map[string]bool, unlockedRows)
	for _, row := range kanadata.Rows[:unlockedRows] {
		allowedRows[row] = true
	}
	active := make([]kanadata.Kana, 0, len(items))
	for _, item := range items {
		if allowedRows[item.Row] {
			active = append(active, item)
		}
	}
	return active
}

func weakestKana(items []kanadata.Kana, states map[string]MemoryState, matrix ConfusionMatrix, limit int) []kanadata.Kana {
	result := append([]kanadata.Kana(nil), items...)
	sort.SliceStable(result, func(i, j int) bool {
		return states[result[i].ID].Weakness(matrix.Score(result[i].ID)) > states[result[j].ID].Weakness(matrix.Score(result[j].ID))
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func confusedKana(items []kanadata.Kana, matrix ConfusionMatrix) []kanadata.Kana {
	included := make(map[string]bool)
	for _, pair := range matrix.Pairs(meaningfulConfusion) {
		included[pair.FirstID] = true
		included[pair.SecondID] = true
	}
	result := make([]kanadata.Kana, 0, len(included))
	for _, item := range items {
		if included[item.ID] {
			result = append(result, item)
		}
	}
	return result
}
