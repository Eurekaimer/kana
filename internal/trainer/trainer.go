package trainer

import (
	"time"

	kanadata "kana/internal/kana"
)

// Evaluation is the frontend-independent result of one submitted answer.
type Evaluation struct {
	ExpectedID   string
	AnsweredID   string
	RawAnswer    string
	Correct      bool
	ResponseTime time.Duration
}

// Evaluate checks an answer and identifies a uniquely implied wrong kana.
func Evaluate(expected kanadata.Kana, answer string, responseTime time.Duration, items []kanadata.Kana) Evaluation {
	result := Evaluation{
		ExpectedID:   expected.ID,
		RawAnswer:    kanadata.NormalizeAnswer(answer),
		Correct:      kanadata.Matches(expected, answer),
		ResponseTime: responseTime,
	}
	if result.Correct {
		result.AnsweredID = expected.ID
		return result
	}
	if answered, ok := kanadata.ResolveRomaji(answer, items); ok {
		result.AnsweredID = answered.ID
	}
	return result
}
