package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

type feedbackDoneMsg struct {
	questionIndex int64
}

// Update handles keyboard input, resize events, and feedback timing.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		return m, nil
	case feedbackDoneMsg:
		if m.feedback != "" && message.questionIndex == m.progress.QuestionIndex {
			if err := m.selectNext(); err != nil {
				m.fatal = err
				return m, tea.Quit
			}
		}
		return m, nil
	case tea.KeyMsg:
		if message.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.feedback != "" {
			if message.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.screen != screenTraining {
			return m.updatePanel(message)
		}
		return m.updateTraining(message)
	}

	var command tea.Cmd
	m.input, command = m.input.Update(message)
	return m, command
}

func (m Model) updateTraining(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlR {
		m.screen = screenOptions
		return m, nil
	}
	if key.Type == tea.KeyEsc {
		if m.input.Value() != "" {
			m.input.SetValue("")
		} else {
			m.screen = screenMenu
		}
		return m, nil
	}
	if key.Type == tea.KeyTab && m.input.Value() == "" {
		m.cycleMode()
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.input.Value() == "" {
			return m, nil
		}
		return m.submitAnswer()
	}
	if m.input.Value() == "" {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "?":
			m.screen = screenHelp
			return m, nil
		}
	}
	if key.Type == tea.KeyRunes && !isASCIIAnswer(key.Runes) {
		return m, nil
	}
	var command tea.Cmd
	m.input, command = m.input.Update(key)
	return m, command
}

func (m Model) updatePanel(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.screen = screenTraining
		m.shownAt = m.now()
		return m, nil
	}
	if key.String() == "q" {
		return m, tea.Quit
	}

	switch m.screen {
	case screenMenu:
		switch key.String() {
		case "s":
			m.screen = screenStats
		case "c":
			m.screen = screenConfusions
		case "w":
			m.setFocus(trainer.FocusWeak)
			m.screen = screenTraining
			if err := m.selectNext(); err != nil {
				m.fatal = err
				return m, tea.Quit
			}
		case "r":
			m.screen = screenOptions
		case "?":
			m.screen = screenHelp
		case "tab":
			m.cycleMode()
		}
	case screenStats:
		if key.String() == "c" {
			m.screen = screenConfusions
		}
	case screenConfusions:
		if key.String() == "s" {
			m.screen = screenStats
		}
	case screenOptions:
		changed := true
		switch key.String() {
		case "tab":
			m.cycleMode()
		case "p":
			config := m.scheduler.Config()
			m.scheduler.SetProgressive(!config.Progressive)
		case "a":
			m.scheduler.SetProgressive(false)
			m.setFocus(trainer.FocusNormal)
		case "n":
			m.setFocus(trainer.FocusNormal)
		case "w":
			m.setFocus(trainer.FocusWeak)
		case "c":
			m.setFocus(trainer.FocusConfusion)
		default:
			changed = false
		}
		if changed {
			m.screen = screenTraining
			if err := m.selectNext(); err != nil {
				m.fatal = err
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m Model) submitAnswer() (tea.Model, tea.Cmd) {
	now := m.now()
	responseTime := now.Sub(m.shownAt)
	if responseTime < 0 {
		responseTime = 0
	}
	evaluation := trainer.Evaluate(m.current, m.input.Value(), responseTime, kanadata.Basic)
	questionIndex := m.progress.QuestionIndex
	m.progress.Record(evaluation, now)
	m.scheduler.ObserveResult(evaluation.ExpectedID, evaluation.AnsweredID, evaluation.Correct, questionIndex, m.progress.Confusions)
	m.session.Questions++
	m.session.TotalLatency += responseTime
	if evaluation.Correct {
		m.session.Correct++
		m.feedback = "✓"
	} else {
		m.feedback = "✗  " + m.current.Primary
	}
	if m.saver != nil {
		m.saveError = m.saver.Save(m.progress)
	}
	indexAfterAnswer := m.progress.QuestionIndex
	return m, tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return feedbackDoneMsg{questionIndex: indexAfterAnswer}
	})
}

func isASCIIAnswer(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, character := range runes {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')) {
			return false
		}
	}
	return true
}
