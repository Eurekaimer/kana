package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

type recordingSaver struct {
	calls int
	last  trainer.Progress
}

func (s *recordingSaver) Save(progress trainer.Progress) error {
	s.calls++
	s.last = progress
	return nil
}

func newTestModel(t *testing.T) (Model, *recordingSaver) {
	t.Helper()
	saver := &recordingSaver{}
	progress := trainer.NewProgress(kanadata.Basic, time.Unix(1, 0))
	model, err := NewModel(progress, saver, Options{Mode: kanadata.ModeHiragana, Level: "gojuon", Seed: 1, NoSplash: true})
	if err != nil {
		t.Fatal(err)
	}
	model.now = func() time.Time { return time.Unix(3, 0) }
	model.shownAt = time.Unix(2, 0)
	return model, saver
}

func sendKey(model Model, key tea.KeyMsg) Model {
	updated, _ := model.Update(key)
	return updated.(Model)
}

func TestSubmitCorrectAnswerSavesAndAdvances(t *testing.T) {
	model, saver := newTestModel(t)
	model.input.SetValue(strings.ToUpper(model.current.Primary))
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if command == nil {
		t.Fatal("submission did not schedule feedback completion")
	}
	if model.feedback != "✓" || model.session.Correct != 1 || model.session.Questions != 1 {
		t.Fatalf("submitted model = %+v", model)
	}
	if saver.calls != 1 || saver.last.QuestionIndex != 1 {
		t.Fatalf("save calls=%d progress=%+v", saver.calls, saver.last)
	}
	updated, _ = model.Update(feedbackDoneMsg{questionIndex: 1})
	model = updated.(Model)
	if model.feedback != "" || model.input.Value() != "" {
		t.Fatalf("feedback did not advance: %+v", model)
	}
}

func TestWrongAnswerShowsCorrection(t *testing.T) {
	model, _ := newTestModel(t)
	model.current, _ = kanadata.ByID("ki")
	model.input.SetValue("ku")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.feedback != "✗  ki" || model.progress.Confusions.Count("ki", "ku") != 1 {
		t.Fatalf("wrong feedback = %q, confusions=%v", model.feedback, model.progress.Confusions)
	}
}

func TestAnswerLettersDoNotTriggerCommands(t *testing.T) {
	model, _ := newTestModel(t)
	model.current, _ = kanadata.ByID("shi")
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if model.screen != screenTraining || model.input.Value() != "s" {
		t.Fatalf("s was treated as command: screen=%v input=%q", model.screen, model.input.Value())
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if model.screen != screenStats {
		t.Fatalf("menu s did not open stats: %v", model.screen)
	}
}

func TestTabCyclesModeOnlyWithEmptyAnswer(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyTab})
	if model.mode != kanadata.ModeKatakana {
		t.Fatalf("mode = %s", model.mode)
	}
	model.input.SetValue("k")
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyTab})
	if model.mode != kanadata.ModeKatakana {
		t.Fatalf("tab changed mode while typing: %s", model.mode)
	}
}

func TestPanelNavigationAndAnswerFiltering(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if model.screen != screenHelp {
		t.Fatalf("help screen = %v", model.screen)
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if model.input.Value() != "" {
		t.Fatalf("non-letter accepted: %q", model.input.Value())
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K', 'I'}})
	if model.input.Value() != "KI" {
		t.Fatalf("uppercase paste rejected: %q", model.input.Value())
	}
}
