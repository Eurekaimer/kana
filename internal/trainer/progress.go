package trainer

import (
	"time"

	kanadata "kana/internal/kana"
)

const ProgressVersion = 1

// Progress is the complete session-independent learning state.
type Progress struct {
	Version       int                    `json:"version"`
	QuestionIndex int64                  `json:"question_index"`
	States        map[string]MemoryState `json:"states"`
	Confusions    ConfusionMatrix        `json:"confusions"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// NewProgress initializes state for every supported kana.
func NewProgress(items []kanadata.Kana, now time.Time) Progress {
	progress := Progress{
		Version:    ProgressVersion,
		States:     make(map[string]MemoryState, len(items)),
		Confusions: make(ConfusionMatrix),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	for _, item := range items {
		progress.States[item.ID] = NewMemoryState()
	}
	return progress
}

// EnsureDefaults repairs absent fields from older or partially written data.
func (p *Progress) EnsureDefaults(items []kanadata.Kana, now time.Time) {
	p.Version = ProgressVersion
	if p.States == nil {
		p.States = make(map[string]MemoryState, len(items))
	}
	for _, item := range items {
		if _, ok := p.States[item.ID]; !ok {
			p.States[item.ID] = NewMemoryState()
		}
	}
	if p.Confusions == nil {
		p.Confusions = make(ConfusionMatrix)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
}

// Record applies an evaluation to memory and learned confusions.
func (p *Progress) Record(evaluation Evaluation, now time.Time) {
	state, ok := p.States[evaluation.ExpectedID]
	if !ok {
		state = NewMemoryState()
	}
	state.Record(evaluation.Correct, evaluation.ResponseTime, p.QuestionIndex, now)
	p.States[evaluation.ExpectedID] = state
	if !evaluation.Correct {
		p.Confusions.Record(evaluation.ExpectedID, evaluation.AnsweredID)
	}
	p.QuestionIndex++
	p.UpdatedAt = now
}
