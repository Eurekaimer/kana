package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

type screen int

const (
	screenTraining screen = iota
	screenMenu
	screenStats
	screenConfusions
	screenHelp
	screenOptions
)

// ProgressSaver is implemented by persistent storage and test doubles.
type ProgressSaver interface {
	Save(trainer.Progress) error
}

// Options configures a training session without changing durable progress.
type Options struct {
	Mode        kanadata.Mode
	Progressive bool
	Focus       trainer.FocusMode
	Seed        int64
}

// SessionStats summarizes only the current process lifetime.
type SessionStats struct {
	Questions    int
	Correct      int
	TotalLatency time.Duration
}

// Accuracy returns session correctness as a percentage.
func (s SessionStats) Accuracy() float64 {
	if s.Questions == 0 {
		return 0
	}
	return 100 * float64(s.Correct) / float64(s.Questions)
}

// AverageResponse returns the mean response latency.
func (s SessionStats) AverageResponse() time.Duration {
	if s.Questions == 0 {
		return 0
	}
	return s.TotalLatency / time.Duration(s.Questions)
}

// Model is the Bubble Tea state machine.
type Model struct {
	progress  trainer.Progress
	saver     ProgressSaver
	scheduler *trainer.Scheduler
	mode      kanadata.Mode
	focus     trainer.FocusMode
	input     textinput.Model
	current   kanadata.Kana
	shownAt   time.Time
	now       func() time.Time
	screen    screen
	feedback  string
	width     int
	height    int
	session   SessionStats
	saveError error
	fatal     error
}

// NewModel creates a ready-to-run training model.
func NewModel(progress trainer.Progress, saver ProgressSaver, options Options) (Model, error) {
	if options.Mode == "" {
		options.Mode = kanadata.ModeHiragana
	}
	if options.Focus == "" {
		options.Focus = trainer.FocusNormal
	}
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 12
	input.Width = 16
	input.PromptStyle = keyStyle.Copy().Background(cardColor)
	input.TextStyle = cardInputStyle
	input.PlaceholderStyle = dimStyle.Copy().Background(cardColor)
	input.CompletionStyle = dimStyle.Copy().Background(cardColor)
	input.Cursor.Style = keyStyle.Copy().Background(cardColor)
	input.Focus()

	model := Model{
		progress: progress,
		saver:    saver,
		scheduler: trainer.NewScheduler(options.Seed, trainer.SchedulerConfig{
			Progressive: options.Progressive,
			Focus:       options.Focus,
		}),
		mode:   options.Mode,
		focus:  options.Focus,
		input:  input,
		now:    time.Now,
		screen: screenTraining,
		width:  80,
		height: 24,
	}
	if err := model.selectNext(); err != nil {
		return Model{}, err
	}
	return model, nil
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Progress returns the latest durable state after the program exits.
func (m Model) Progress() trainer.Progress {
	return m.progress
}

// Session returns current-session statistics.
func (m Model) Session() SessionStats {
	return m.session
}

// Summary formats a concise plain-text session summary.
func (m Model) Summary() string {
	if m.session.Questions == 0 {
		return ""
	}
	weak := weakestItems(m.progress, 3)
	weakSymbols := ""
	for _, item := range weak {
		if weakSymbols != "" {
			weakSymbols += " "
		}
		weakSymbols += item.Hiragana
	}
	confused := "none"
	if pairs := m.progress.Confusions.Pairs(1); len(pairs) > 0 {
		first, _ := kanadata.ByID(pairs[0].FirstID)
		second, _ := kanadata.ByID(pairs[0].SecondID)
		confused = fmt.Sprintf("%s ↔ %s (%d)", first.Hiragana, second.Hiragana, pairs[0].Count)
	}
	return fmt.Sprintf(
		"Session\n\nQuestions: %d\nCorrect:   %d\nAccuracy:  %.1f%%\n\nWeakest: %s\nMost confused: %s\nAverage response: %.2fs\n",
		m.session.Questions,
		m.session.Correct,
		m.session.Accuracy(),
		weakSymbols,
		confused,
		m.session.AverageResponse().Seconds(),
	)
}

func (m *Model) selectNext() error {
	item, err := m.scheduler.Next(kanadata.Basic, m.progress.States, m.progress.Confusions, m.progress.QuestionIndex)
	if err != nil {
		return err
	}
	m.current = item
	m.input.SetValue("")
	m.feedback = ""
	m.shownAt = m.now()
	return nil
}

func (m Model) currentSymbol() string {
	katakanaVariant := m.progress.QuestionIndex%2 == 1
	return m.current.Symbol(m.mode, katakanaVariant)
}

func (m *Model) cycleMode() {
	switch m.mode {
	case kanadata.ModeHiragana:
		m.mode = kanadata.ModeKatakana
	case kanadata.ModeKatakana:
		m.mode = kanadata.ModeMixed
	default:
		m.mode = kanadata.ModeHiragana
	}
}

func (m *Model) setFocus(focus trainer.FocusMode) {
	m.focus = focus
	m.scheduler.SetFocus(focus)
}
