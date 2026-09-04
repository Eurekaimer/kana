package trainer

import (
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func TestConfusionInferenceAndRecording(t *testing.T) {
	matrix := make(ConfusionMatrix)
	cases := []struct {
		expected string
		answer   string
		resolved string
	}{
		{"ki", "ku", "ku"},
		{"nu", "ne", "ne"},
		{"ki", "asdf", ""},
	}
	for _, tc := range cases {
		expected, _ := kanadata.ByID(tc.expected)
		result := Evaluate(expected, tc.answer, time.Second, kanadata.Basic)
		if result.AnsweredID != tc.resolved {
			t.Fatalf("Evaluate(%s, %q).AnsweredID = %q, want %q", tc.expected, tc.answer, result.AnsweredID, tc.resolved)
		}
		matrix.Record(result.ExpectedID, result.AnsweredID)
	}
	if got := matrix.Count("ki", "ku"); got != 1 {
		t.Fatalf("ki -> ku = %d, want 1", got)
	}
	if got := matrix.Count("nu", "ne"); got != 1 {
		t.Fatalf("nu -> ne = %d, want 1", got)
	}
	if got := matrix.TotalFor("asdf"); got != 0 {
		t.Fatalf("nonsense confusion total = %d", got)
	}
}

func TestConfusionPairsCombineDirections(t *testing.T) {
	matrix := ConfusionMatrix{
		"nu": {"ne": 3},
		"ne": {"nu": 2},
	}
	pairs := matrix.Pairs(2)
	if len(pairs) != 1 || pairs[0].Count != 5 {
		t.Fatalf("pairs = %+v", pairs)
	}
	if matrix.Score("nu") <= 0 {
		t.Fatal("confusion did not increase score")
	}
}
