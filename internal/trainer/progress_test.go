package trainer

import (
	"testing"
	"time"

	kanadata "kana/internal/kana"
)

func TestNewProgressAndRecord(t *testing.T) {
	now := time.Unix(100, 0)
	progress := NewProgress(kanadata.Basic, now)
	if len(progress.States) != 46 || progress.Version != ProgressVersion {
		t.Fatalf("new progress = %+v", progress)
	}
	progress.Record(Evaluation{ExpectedID: "ki", AnsweredID: "ku", Correct: false, ResponseTime: 2 * time.Second}, now.Add(time.Second))
	if progress.QuestionIndex != 1 || progress.States["ki"].Wrong != 1 {
		t.Fatalf("recorded progress = %+v", progress)
	}
	if progress.Confusions.Count("ki", "ku") != 1 {
		t.Fatal("record did not update confusion")
	}
}

func TestEnsureDefaults(t *testing.T) {
	now := time.Unix(200, 0)
	progress := Progress{}
	progress.EnsureDefaults(kanadata.Basic, now)
	if len(progress.States) != 46 || progress.Confusions == nil || progress.CreatedAt != now {
		t.Fatalf("repaired progress = %+v", progress)
	}
}
