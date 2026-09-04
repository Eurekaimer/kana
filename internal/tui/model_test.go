package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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
	model, err := NewModel(progress, saver, Options{Mode: kanadata.ModeHiragana, Seed: 1})
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

func TestViewAndSummary(t *testing.T) {
	model, _ := newTestModel(t)
	model.width = 24
	model.height = 8
	view := model.View()
	for _, expected := range []string{model.currentSymbol(), ">", "╭", "╰", "ACC", "WEAK"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("training view missing %q:\n%s", expected, view)
		}
	}
	model.session = SessionStats{Questions: 2, Correct: 1, TotalLatency: 3 * time.Second}
	summary := model.Summary()
	for _, expected := range []string{"Questions: 2", "Accuracy:  50.0%", "Average response: 1.50s"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("summary missing %q:\n%s", expected, summary)
		}
	}
}

func TestFocusOptionUpdatesScheduler(t *testing.T) {
	model, _ := newTestModel(t)
	model.screen = screenOptions
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if model.focus != trainer.FocusWeak || model.scheduler.Config().Focus != trainer.FocusWeak || model.screen != screenTraining {
		t.Fatalf("weak focus not applied: %+v", model.scheduler.Config())
	}
}

func TestSessionStatsEmptyAndModelAccessors(t *testing.T) {
	model, _ := newTestModel(t)
	if model.Init() == nil {
		t.Fatal("Init did not return cursor blink command")
	}
	if model.Session().Accuracy() != 0 || model.Session().AverageResponse() != 0 {
		t.Fatalf("empty session = %+v", model.Session())
	}
	if model.Progress().Version != trainer.ProgressVersion {
		t.Fatalf("progress version = %d", model.Progress().Version)
	}
	if model.Summary() != "" {
		t.Fatalf("empty summary = %q", model.Summary())
	}
}

func TestResizeAndPanelViews(t *testing.T) {
	model, _ := newTestModel(t)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 32, Height: 10})
	model = updated.(Model)
	if model.width != 32 || model.height != 10 {
		t.Fatalf("resize = %dx%d", model.width, model.height)
	}
	screens := map[screen]string{
		screenMenu:       "Train weakest",
		screenStats:      "Weak Kana",
		screenConfusions: "Most Confused",
		screenHelp:       "Backspace",
		screenOptions:    "Progression",
	}
	for target, expected := range screens {
		model.screen = target
		if view := model.View(); !strings.Contains(view, expected) {
			t.Fatalf("screen %d missing %q:\n%s", target, expected, view)
		}
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

func TestOptionsToggleProgressionAndFocus(t *testing.T) {
	model, _ := newTestModel(t)
	model.screen = screenOptions
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !model.scheduler.Config().Progressive {
		t.Fatal("progression was not toggled on")
	}
	model.screen = screenOptions
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if model.focus != trainer.FocusConfusion {
		t.Fatalf("focus = %s", model.focus)
	}
	model.screen = screenOptions
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if model.focus != trainer.FocusNormal || model.scheduler.Config().Progressive {
		t.Fatalf("all option = %+v, focus=%s", model.scheduler.Config(), model.focus)
	}
}

func TestOpaqueHighContrastTheme(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
	})

	panelPrefix := stylePrefix(panelRestoreStyle)
	if panelPrefix == "" {
		t.Fatal("panel restore style emitted no true-color prefix")
	}
	nested := titleStyle.Render("KANA")
	opaque := renderOpaque(panelRestoreStyle, panelRestoreStyle, nested)
	if !strings.Contains(opaque, "\x1b[0m"+panelPrefix) {
		t.Fatalf("nested reset did not restore panel background: %q", opaque)
	}

	if frameStyle.GetBackground() != panelColor {
		t.Fatalf("frame background = %v", frameStyle.GetBackground())
	}
	if cardStyle.GetBackground() != cardColor {
		t.Fatalf("card background = %v", cardStyle.GetBackground())
	}
	if panelColor == cardColor {
		t.Fatal("theme surfaces are not visually distinct")
	}

	model, _ := newTestModel(t)
	model.width = 120
	model.height = 40
	view := model.View()
	topPaddingLine, framedRegion, padded := strings.Cut(view, "\n")
	if !padded {
		t.Fatal("training view has no vertical padding")
	}
	if strings.Contains(topPaddingLine, "\x1b[48;") {
		t.Fatalf("vertical padding has an opaque background: %q", topPaddingLine)
	}
	framedRegion = strings.TrimLeft(framedRegion, "\n")
	if !strings.Contains(framedRegion, panelPrefix) {
		t.Fatalf("framed region missing panel background prefix %q", panelPrefix)
	}
	cardPrefix := stylePrefix(cardRestoreStyle)
	if cardPrefix == "" || !strings.Contains(framedRegion, cardPrefix) {
		t.Fatalf("framed region missing card background prefix %q", cardPrefix)
	}
}
