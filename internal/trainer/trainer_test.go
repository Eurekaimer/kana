package trainer

import (
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func TestEvaluate(t *testing.T) {
	ki, _ := kanadata.ByID("ki")

	correct := Evaluate(ki, " KI ", 800*time.Millisecond, kanadata.Basic)
	if !correct.Correct || correct.AnsweredID != "ki" || correct.RawAnswer != "ki" {
		t.Fatalf("correct evaluation = %+v", correct)
	}

	wrong := Evaluate(ki, "ku", 2*time.Second, kanadata.Basic)
	if wrong.Correct || wrong.AnsweredID != "ku" {
		t.Fatalf("wrong evaluation = %+v", wrong)
	}

	nonsense := Evaluate(ki, "asdf", time.Second, kanadata.Basic)
	if nonsense.Correct || nonsense.AnsweredID != "" {
		t.Fatalf("nonsense evaluation = %+v", nonsense)
	}
}
