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
		m.width, m.height = message.Width, message.Height
		m.scroll = 0
		return m, nil
	case splashTickMsg:
		if m.screen != screenSplash {
			return m, nil
		}
		m.splashFrame++
		if m.splashFrame >= splashFrames {
			m.openPanel(screenOptions)
			return m, nil
		}
		return m, splashTick()
	case feedbackDoneMsg:
		if m.screen == screenTraining && m.feedback != "" && message.questionIndex == m.progress.QuestionIndex {
			return m.nextQuestion()
		}
		return m, nil
	case tea.KeyMsg:
		if message.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.screen == screenSplash {
			if message.String() == "q" {
				return m, tea.Quit
			}
			m.openPanel(screenOptions)
			return m, nil
		}
		if message.String() == "?" && m.screen != screenHelp {
			m.openPanel(screenHelp)
			return m, nil
		}
		// Keep j/k available for romaji while editing; elsewhere reuse arrow handling.
		if (m.screen != screenTraining || m.feedback != "") &&
			message.Type == tea.KeyRunes && !message.Alt && !message.Paste && len(message.Runes) == 1 {
			switch message.Runes[0] {
			case 'j':
				message = tea.KeyMsg{Type: tea.KeyDown}
			case 'k':
				message = tea.KeyMsg{Type: tea.KeyUp}
			}
		}
		if m.screen == screenOptions && (message.Type == tea.KeyUp || message.Type == tea.KeyDown) {
			return m.updateOptions(message)
		}
		if m.screen == screenRows && (message.Type == tea.KeyUp || message.Type == tea.KeyDown) {
			return m.updateRows(message)
		}
		switch message.String() {
		case "up", "pgup":
			m.scroll = max(0, m.scroll-scrollStep(message))
			return m, nil
		case "down", "pgdown":
			// Bound the offset so a long key hold can always be reversed.
			m.scroll = min(m.scroll+scrollStep(message), m.maxScroll())
			return m, nil
		case "home":
			m.scroll = 0
			return m, nil
		case "end":
			m.scroll = m.maxScroll()
			return m, nil
		}
		if m.screen != screenTraining {
			return m.updatePanel(message)
		}
		if message.Type == tea.KeyCtrlR || message.Type == tea.KeyCtrlL || message.Type == tea.KeyCtrlS {
			return m.updateTraining(message)
		}
		if m.feedback != "" {
			switch message.String() {
			case "q":
				return m, tea.Quit
			case "enter", " ":
				return m.nextQuestion()
			case "esc":
				m.openPanel(screenMenu)
			}
			return m, nil
		}
		return m.updateTraining(message)
	}
	var command tea.Cmd
	m.input, command = m.input.Update(message)
	return m, command
}

func scrollStep(key tea.KeyMsg) int {
	if key.Type == tea.KeyPgUp || key.Type == tea.KeyPgDown {
		return 6
	}
	return 1
}

func (m Model) updateTraining(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlR:
		m.openPanel(screenOptions)
		return m, nil
	case tea.KeyCtrlL:
		m.openRows()
		return m, nil
	case tea.KeyCtrlS:
		m.openPanel(screenStudy)
		return m, nil
	case tea.KeyEsc:
		if m.input.Value() != "" {
			m.input.SetValue("")
		} else {
			m.openPanel(screenMenu)
		}
		return m, nil
	case tea.KeyTab:
		if m.input.Value() == "" {
			m.cycleMode()
			m.shownAt = m.now()
		}
		return m, nil
	case tea.KeyEnter:
		if m.input.Value() != "" {
			return m.submitAnswer()
		}
		return m, nil
	}
	if key.String() == "q" && m.input.Value() == "" {
		return m, tea.Quit
	}
	if key.Type == tea.KeyRunes && !isASCIIAnswer(key.Runes) {
		return m, nil
	}
	var command tea.Cmd
	m.input, command = m.input.Update(key)
	return m, command
}

func (m Model) updatePanel(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyEsc || (m.screen == screenHelp && key.String() == "?") {
		if m.screen == screenHelp && m.helpReturn != screenTraining {
			m.screen, m.scroll = m.helpReturn, 0
		} else if m.screen == screenRows {
			if m.rowsReturn == screenTraining {
				m.resumeTraining()
			} else {
				m.screen, m.scroll = m.rowsReturn, 0
			}
		} else {
			m.resumeTraining()
		}
		return m, nil
	}
	if key.String() == "q" {
		return m, tea.Quit
	}
	switch m.screen {
	case screenMenu:
		switch key.String() {
		case "s":
			m.openPanel(screenStats)
		case "c":
			m.openPanel(screenConfusions)
		case "w":
			m.setFocus(trainer.FocusWeak)
			return m.nextQuestion()
		case "r":
			m.openPanel(screenOptions)
		case "g":
			m.openRows()
		case "t":
			m.openPanel(screenStudy)
		}
	case screenStats:
		if key.String() == "c" {
			m.openPanel(screenConfusions)
		}
	case screenConfusions:
		if key.String() == "s" {
			m.openPanel(screenStats)
		}
	case screenHelp:
		page := m.helpPage
		switch key.String() {
		case "right", "tab":
			page = (page + 1) % len(helpPages)
		case "left", "shift+tab":
			page = (page + len(helpPages) - 1) % len(helpPages)
		case "1", "2", "3", "4", "5":
			page = int(key.Runes[0] - '1')
		}
		if page != m.helpPage {
			m.helpPage, m.scroll = page, 0
		}
	case screenOptions:
		return m.updateOptions(key)
	case screenRows:
		return m.updateRows(key)
	case screenStudy:
		if key.Type == tea.KeyEnter {
			m.resumeTraining()
		} else if key.String() == "g" {
			m.openRows()
		} else if key.Type == tea.KeyTab {
			m.cycleMode()
		}
	}
	return m, nil
}

func (m Model) updateOptions(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	level, exercise := m.level, m.exercise
	switch key.String() {
	case "up", "down":
		index := 0
		for i, candidate := range kanadata.Levels {
			if candidate == level {
				index = i
				break
			}
		}
		if key.Type == tea.KeyUp {
			index = max(0, index-1)
		} else {
			index = min(len(kanadata.Levels)-1, index+1)
		}
		level = kanadata.Levels[index]
		if level == "beginner" || level == "gojuon" {
			exercise = "reading"
		}
	case "enter":
		if m.level == "beginner" {
			m.openPanel(screenStudy)
		} else {
			m.resumeTraining()
		}
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7":
		level = kanadata.Levels[int(key.Runes[0]-'1')]
		if level == "beginner" || level == "gojuon" {
			exercise = "reading"
		}
	case "g":
		m.openRows()
		return m, nil
	case "e":
		if exercise == "reading" {
			exercise = "anime"
			if level == "beginner" || level == "gojuon" {
				level = "n5"
			}
		} else {
			exercise = "reading"
		}
	case "tab":
		m.cycleMode()
		return m, nil
	case "p":
		level, exercise = "beginner", "reading"
	case "a":
		level, exercise = "gojuon", "reading"
	case "n", "r", "w", "c":
		focus := trainer.FocusNormal
		switch key.String() {
		case "r":
			focus = trainer.FocusRandom
		case "w":
			focus = trainer.FocusWeak
		case "c":
			focus = trainer.FocusConfusion
		}
		m.setFocus(focus)
		if err := m.selectNext(); err != nil {
			m.fatal = err
			return m, tea.Quit
		}
		m.openPanel(screenOptions)
		m.pausedAt = m.now()
		return m, nil
	default:
		return m, nil
	}
	if err := m.setCourse(level, exercise); err != nil {
		m.fatal = err
		return m, tea.Quit
	}
	m.openPanel(screenOptions)
	m.pausedAt = m.now()
	for index, candidate := range kanadata.Levels {
		if candidate == m.level {
			m.keepVisible(index + 2)
			break
		}
	}
	return m, nil
}

func (m Model) nextQuestion() (tea.Model, tea.Cmd) {
	m.screen, m.scroll = screenTraining, 0
	if err := m.selectNext(); err != nil {
		m.fatal = err
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) submitAnswer() (tea.Model, tea.Cmd) {
	now := m.now()
	responseTime := max(time.Duration(0), now.Sub(m.shownAt))
	evaluation := trainer.Evaluate(m.current, m.input.Value(), responseTime, m.items)
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
	m.scroll = 0
	if m.current.Meaning != "" {
		return m, nil // Vocabulary/context corrections stay visible until Enter.
	}
	delay := 650 * time.Millisecond
	if !evaluation.Correct {
		delay = 2 * time.Second
	}
	indexAfterAnswer := m.progress.QuestionIndex
	return m, tea.Tick(delay, func(time.Time) tea.Msg {
		return feedbackDoneMsg{questionIndex: indexAfterAnswer}
	})
}

func isASCIIAnswer(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, character := range runes {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || character == '\'' || character == '-' || character == ' ') {
			return false
		}
	}
	return true
}
