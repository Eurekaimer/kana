package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

func runeKey(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func TestManualMultipleRowsConstrainEntireSession(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlL})
	model = sendKey(model, runeKey("x"))
	model = sendKey(model, runeKey("2")) // ka
	model = sendKey(model, runeKey("4")) // ta
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != screenStudy {
		t.Fatal("manual rows did not open the recognition reference")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	seen := make(map[string]bool)
	for range 80 {
		if model.current.Row != "k" && model.current.Row != "t" {
			t.Fatalf("unselected row escaped into practice: %+v", model.current)
		}
		seen[model.current.ID] = true
		model.input.SetValue(model.current.Primary)
		model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
		updated, _ := model.Update(feedbackDoneMsg{questionIndex: model.progress.QuestionIndex})
		model = updated.(Model)
	}
	for _, item := range kanadata.Basic {
		if (item.Row == "k" || item.Row == "t") && !seen[item.ID] {
			t.Fatalf("selected kana never appeared: %s", item.ID)
		}
	}
}

func TestEmptyRowSelectionRejectedAndCancelKeepsOriginalDeck(t *testing.T) {
	model, _ := newTestModel(t)
	before := model.current.ID
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlL})
	model = sendKey(model, runeKey("x"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != screenRows || model.rowError == "" {
		t.Fatal("empty selection was applied instead of remaining editable")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != screenTraining || model.current.ID != before || len(model.items) != len(kanadata.Basic) {
		t.Fatal("cancelling row selection changed the active question/deck")
	}
}

func TestArrowAndSpaceSelectRowsWithoutTypingAnAnswer(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlL})
	model = sendKey(model, runeKey("x"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyDown})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeySpace})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyDown})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeySpace})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	for _, item := range model.items {
		if item.Row != "k" && item.Row != "s" {
			t.Fatalf("space/arrow selected unintended row: %s", item.Row)
		}
	}
	if len(model.items) != 10 || model.input.Value() != "" {
		t.Fatal("multi-selection failed or navigation leaked into the answer")
	}
}

func TestAutomaticUnlockShowsReferenceBeforeNextRowPractice(t *testing.T) {
	now := time.Unix(1000, 0)
	model, err := NewModel(trainer.NewProgress(kanadata.Basic, now), nil, Options{Level: "beginner", NoSplash: true, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	for _, item := range kanadata.Basic {
		if item.Row != "a" {
			continue
		}
		state := trainer.NewMemoryState()
		attempts := int64(4)
		if item.ID == "a" {
			attempts = 3
		}
		for index := range attempts {
			state.Record(true, time.Second, index, now)
		}
		model.progress.States[item.ID] = state
	}
	model.current, _ = kanadata.ByID("a")
	model.input.SetValue("a")
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ := model.Update(feedbackDoneMsg{questionIndex: model.progress.QuestionIndex})
	model = updated.(Model)
	if model.screen != screenStudy {
		t.Fatal("new row started testing before showing its recognition reference")
	}
	active := model.scheduler.ActiveKana(model.items, model.progress.States, model.progress.Confusions)
	if len(active) != 10 {
		t.Fatalf("expected only a/ka rows after the boundary answer, got %d cards", len(active))
	}
	for _, item := range active {
		if item.Row != "a" && item.Row != "k" {
			t.Fatalf("automatic unlock skipped a row: %s", item.Row)
		}
	}
}

func TestHelpPreservesAnswerAndExcludesReadingTime(t *testing.T) {
	model, _ := newTestModel(t)
	clock := time.Unix(100, 0)
	model.now = func() time.Time { return clock }
	model.shownAt = clock.Add(-time.Second)
	model.current, _ = kanadata.ByID("ki")
	model.input.SetValue("ki")
	model = sendKey(model, runeKey("?"))
	model = sendKey(model, runeKey("5"))
	clock = clock.Add(time.Hour)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != screenTraining || model.input.Value() != "ki" {
		t.Fatal("help navigation destroyed the pending answer")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.session.TotalLatency != time.Second || model.session.Correct != 1 {
		t.Fatalf("help time contaminated result: %+v", model.session)
	}
}

func TestCourseArrowNavigationAndHelpReturn(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlR})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyUp})
	if model.level != "beginner" || model.screen != screenOptions {
		t.Fatal("course arrow did not move from gojuon to beginner")
	}
	model = sendKey(model, runeKey("?"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != screenOptions {
		t.Fatal("help lost its return destination")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyDown})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != screenTraining || model.level != "gojuon" || model.focus != trainer.FocusRandom {
		t.Fatal("gojuon selection did not begin full random practice")
	}
}

func TestVocabularyApostropheAnswerAndFeedbackWaitsForEnter(t *testing.T) {
	model, err := NewModel(trainer.NewProgress(kanadata.All(), time.Unix(1, 0)), nil, Options{Level: "n5", Exercise: "reading", NoSplash: true, Seed: 2})
	if err != nil {
		t.Fatal(err)
	}
	model.current, _ = kanadata.ByID("n5.word.konya")
	model = sendKey(model, runeKey("kon'ya"))
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.session.Correct != 1 || command != nil || model.feedback == "" {
		t.Fatal("valid apostrophe answer failed or reading feedback auto-dismissed")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.feedback != "" || model.input.Value() != "" || model.session.Questions != 1 {
		t.Fatal("Enter failed to continue vocabulary practice")
	}
}

func TestSplashSkipDoesNotSubmitOrReactivateOnStaleTick(t *testing.T) {
	model, err := NewModel(trainer.NewProgress(kanadata.Basic, time.Unix(1, 0)), nil, Options{Level: "beginner"})
	if err != nil {
		t.Fatal(err)
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	updated, command := model.Update(splashTickMsg{})
	model = updated.(Model)
	if model.screen != screenOptions || model.session.Questions != 0 || command != nil {
		t.Fatal("splash skip leaked Enter or stale tick restarted animation")
	}
}

func TestEverySurfaceFitsTerminalAndCanScrollToEnd(t *testing.T) {
	model, _ := newTestModel(t)
	for _, size := range [][2]int{{28, 14}, {40, 18}, {80, 24}, {120, 36}} {
		model.width, model.height = size[0], size[1]
		model.openRows()
		for _, target := range []screen{screenTraining, screenMenu, screenStats, screenConfusions, screenHelp, screenOptions, screenRows, screenStudy, screenSplash} {
			model.screen, model.scroll = target, 0
			for _, offset := range []int{0, model.maxScroll()} {
				model.scroll = offset
				view := model.View()
				lines := strings.Split(view, "\n")
				if len(lines) > model.height {
					t.Fatalf("screen %d height %d exceeds %d", target, len(lines), model.height)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > model.width {
						t.Fatalf("screen %d line width %d exceeds %d: %q", target, ansi.StringWidth(line), model.width, line)
					}
				}
			}
		}
	}
}

func TestFeedbackKeepsAdvertisedNavigationAvailable(t *testing.T) {
	for _, scenario := range []struct {
		key    tea.KeyType
		target screen
	}{
		{tea.KeyCtrlR, screenOptions},
		{tea.KeyCtrlL, screenRows},
		{tea.KeyCtrlS, screenStudy},
	} {
		model, _ := newTestModel(t)
		model.input.SetValue(model.current.Primary)
		model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
		model = sendKey(model, tea.KeyMsg{Type: scenario.key})
		if model.screen != scenario.target || model.session.Questions != 1 {
			t.Fatalf("feedback blocked navigation %v or duplicated submission", scenario.key)
		}
	}
}

func TestVimCourseAndRowNavigation(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlR})
	model = sendKey(model, runeKey("k"))
	model = sendKey(model, runeKey("k"))
	if model.level != "beginner" {
		t.Fatal("k did not move up and stop at the first course")
	}
	model = sendKey(model, runeKey("j"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.screen != screenTraining || model.level != "gojuon" {
		t.Fatal("j did not select full gojuon practice")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlL})
	model = sendKey(model, runeKey("x"))
	model = sendKey(model, runeKey("j"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeySpace})
	model = sendKey(model, runeKey("j"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeySpace})
	model = sendKey(model, runeKey("k"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeySpace})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if len(model.items) != 5 || model.input.Value() != "" {
		t.Fatal("Vim row selection failed or navigation leaked into the answer")
	}
	for _, item := range model.items {
		if item.Row != "s" {
			t.Fatalf("Vim row navigation selected an unintended row: %s", item.Row)
		}
	}
}

func TestVimHelpScrollingPreservesPendingAnswer(t *testing.T) {
	model, _ := newTestModel(t)
	model.width, model.height = 40, 18
	model.input.SetValue("ki")
	model = sendKey(model, runeKey("?"))
	before := model.View()
	model = sendKey(model, runeKey("j"))
	if model.View() == before {
		t.Fatal("j did not scroll the visible help content")
	}
	model = sendKey(model, runeKey("k"))
	if model.View() != before {
		t.Fatal("k did not restore the previous help viewport")
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.screen != screenTraining || model.input.Value() != "ki" {
		t.Fatal("Vim help navigation destroyed the pending answer")
	}
}

func TestVimFeedbackScrollingDoesNotAdvanceQuestion(t *testing.T) {
	model, err := NewModel(trainer.NewProgress(kanadata.All(), time.Unix(1, 0)), nil, Options{Level: "n1", Exercise: "anime", NoSplash: true})
	if err != nil {
		t.Fatal(err)
	}
	model.width, model.height = 40, 18
	model.input.SetValue("wrong")
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	before := model.View()
	model = sendKey(model, runeKey("j"))
	if model.View() == before {
		t.Fatal("j did not scroll the visible feedback")
	}
	model = sendKey(model, runeKey("k"))
	if model.View() != before || model.feedback == "" || model.session.Questions != 1 {
		t.Fatal("Vim scrolling dismissed feedback or advanced the question")
	}
}

func TestJKRemainAnswerLetters(t *testing.T) {
	model, _ := newTestModel(t)
	model.current, _ = kanadata.ByID("ka")
	model = sendKey(model, runeKey("k"))
	model = sendKey(model, runeKey("a"))
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.session.Correct != 1 {
		t.Fatal("k was intercepted instead of accepting the ka answer")
	}
	model, _ = newTestModel(t)
	if err := model.setCourse("n1", "reading"); err != nil {
		t.Fatal(err)
	}
	model.current, _ = kanadata.ByID("n1.word.jirenma")
	for _, character := range "jirenma" {
		model = sendKey(model, runeKey(string(character)))
	}
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.session.Correct != 1 {
		t.Fatal("j was intercepted instead of accepting the jirenma answer")
	}
}

func TestVimNavigationIgnoresPastedOrModifiedKeys(t *testing.T) {
	model, _ := newTestModel(t)
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyCtrlR})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}, Paste: true})
	model = sendKey(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}, Alt: true})
	if model.level != "gojuon" {
		t.Fatal("pasted or Alt-modified letters unexpectedly changed the course")
	}
}
