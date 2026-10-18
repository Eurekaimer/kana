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
	screenSplash
	screenRows
	screenStudy
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
	Level       string
	Exercise    string
	NoSplash    bool
	Rows        []string
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
	progress    trainer.Progress
	saver       ProgressSaver
	scheduler   *trainer.Scheduler
	mode        kanadata.Mode
	focus       trainer.FocusMode
	input       textinput.Model
	current     kanadata.Kana
	shownAt     time.Time
	now         func() time.Time
	screen      screen
	feedback    string
	width       int
	height      int
	session     SessionStats
	saveError   error
	fatal       error
	level       string
	exercise    string
	items       []kanadata.Kana
	splashFrame int
	helpPage    int
	scroll      int
	pausedAt    time.Time
	helpReturn  screen
	rows        map[string]bool
	rowError    string
	activeCount int
	manualRows  bool
	rowsReturn  screen
	rowCursor   int
}

// NewModel creates a ready-to-run training model.
func NewModel(progress trainer.Progress, saver ProgressSaver, options Options) (Model, error) {
	if options.Mode == "" {
		options.Mode = kanadata.ModeHiragana
	}
	if options.Level == "" {
		options.Level = "gojuon"
		if options.Progressive {
			options.Level = "beginner"
		}
	}
	if options.Focus == "" {
		options.Focus = trainer.FocusNormal
		if options.Level == "gojuon" {
			options.Focus = trainer.FocusRandom
		}
	}
	if options.Exercise == "" {
		options.Exercise = "reading"
	}
	items, err := kanadata.Deck(options.Level, options.Exercise)
	if err != nil {
		return Model{}, err
	}
	progress.EnsureDefaults(items, time.Now())
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 80
	input.Width = 28
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
			Progressive: options.Level == "beginner",
			Focus:       options.Focus,
		}),
		mode:     options.Mode,
		focus:    options.Focus,
		input:    input,
		now:      time.Now,
		screen:   screenTraining,
		width:    80,
		height:   24,
		level:    options.Level,
		exercise: options.Exercise,
		items:    items,
	}
	if err := model.selectNext(); err != nil {
		return Model{}, err
	}
	if len(options.Rows) > 0 {
		model.rows = make(map[string]bool, len(options.Rows))
		for _, row := range options.Rows {
			if !validRow(row) {
				return Model{}, fmt.Errorf("unknown kana row %q", row)
			}
			model.rows[row] = true
		}
		if err := model.applyRows(); err != nil {
			return Model{}, err
		}
	}
	if !options.NoSplash {
		model.screen = screenSplash
		model.pausedAt = model.now()
	}
	return model, nil
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	if m.screen == screenSplash {
		return tea.Batch(textinput.Blink, splashTick())
	}
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
	weak := weakestItems(m.progress, m.items, 3)
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
	item, err := m.scheduler.Next(m.items, m.progress.States, m.progress.Confusions, m.progress.QuestionIndex)
	if err != nil {
		return err
	}
	m.current = item
	m.input.SetValue("")
	m.feedback = ""
	m.shownAt = m.now()
	m.pausedAt = time.Time{}
	active := m.scheduler.ActiveKana(m.items, m.progress.States, m.progress.Confusions)
	if m.level == "beginner" && len(active) > m.activeCount {
		m.screen = screenStudy
		m.pausedAt = m.now()
	}
	m.activeCount = len(active)
	return nil
}

func (m Model) currentSymbol() string {
	return m.current.Symbol(m.mode, m.katakanaVariant())
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

func (m Model) katakanaVariant() bool {
	if m.level != "beginner" && m.level != "gojuon" {
		return m.current.NativeKatakana
	}
	return m.progress.QuestionIndex%2 == 1
}

func (m *Model) openPanel(target screen) {
	if target == screenHelp {
		m.helpReturn = m.screen
	}
	if m.screen == screenTraining {
		m.pausedAt = m.now()
	}
	m.screen = target
	m.scroll = 0
}

func (m *Model) resumeTraining() {
	if !m.pausedAt.IsZero() {
		m.shownAt = m.shownAt.Add(m.now().Sub(m.pausedAt))
	}
	m.pausedAt = time.Time{}
	m.screen = screenTraining
	m.scroll = 0
}

func (m *Model) setCourse(level, exercise string) error {
	items, err := kanadata.Deck(level, exercise)
	if err != nil {
		return err
	}
	m.level, m.exercise, m.items = level, exercise, items
	m.activeCount = 0
	m.rows = nil
	m.manualRows = false
	m.scheduler.SetProgressive(level == "beginner")
	if level == "gojuon" {
		m.setFocus(trainer.FocusRandom)
	} else {
		m.setFocus(trainer.FocusNormal)
	}
	return m.selectNext()
}
