package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

// View renders centered, high-contrast application surfaces.
func (m Model) View() string {
	var content string
	if m.fatal != nil {
		content = errorStyle.Render("kana: " + m.fatal.Error())
	} else {
		switch m.screen {
		case screenMenu:
			content = m.menuView()
		case screenStats:
			content = m.statsView()
		case screenConfusions:
			content = m.confusionsView()
		case screenHelp:
			content = m.helpView()
		case screenOptions:
			content = m.optionsView()
		default:
			content = m.trainingView()
		}
	}
	return fitVertically(content, m.height)
}

func (m Model) trainingView() string {
	active := m.scheduler.ActiveKana(kanadata.Basic, m.progress.States, m.progress.Confusions)
	state := m.progress.States[m.current.ID]
	weak := weakestItems(m.progress, 3)
	weakSymbols := make([]string, 0, len(weak))
	for _, item := range weak {
		weakSymbols = append(weakSymbols, item.Symbol(m.mode, false))
	}

	innerWidth := m.panelContentWidth()
	header := joinSides(
		titleStyle.Render("かな  KANA"),
		dimStyle.Render(fmt.Sprintf("%d / 46", len(active))),
		innerWidth,
	)
	feedback := dimStyle.Render("type the romaji")
	if m.feedback == "✓" {
		feedback = successStyle.Render("✓  correct")
	} else if m.feedback != "" {
		feedback = errorStyle.Render(m.feedback)
	}
	cardWidth := min(34, max(14, innerWidth-6))
	card := renderOpaque(
		cardStyle.Width(cardWidth),
		cardRestoreStyle,
		strings.Join([]string{
			kanaStyle.Copy().Bold(true).Render(m.currentSymbol()),
			"",
			m.input.View(),
			feedback,
		}, "\n"),
	)
	status := strings.Join([]string{
		labelStyle.Render("ACC"),
		fmt.Sprintf("%.0f%%", m.session.Accuracy()),
		dimStyle.Render("│"),
		labelStyle.Render("STREAK"),
		fmt.Sprintf("%d", state.CurrentStreak),
		dimStyle.Render("│"),
		labelStyle.Render("MODE"),
		modeName(m.mode),
	}, " ")
	weakLine := labelStyle.Render("WEAK") + "  " + strings.Join(weakSymbols, "  ")
	if m.focus != trainer.FocusNormal {
		weakLine += dimStyle.Render("  ·  " + string(m.focus) + " focus")
	}
	lines := []string{
		header,
		"",
		lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, card),
		"",
		lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, status),
		lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, weakLine),
	}
	if m.saveError != nil {
		lines = append(lines, errorStyle.Render("Save error: "+m.saveError.Error()))
	}
	lines = append(lines, "", keyHints("Esc", "menu", "?", "help", "q", "quit"))
	return m.renderPanel(strings.Join(lines, "\n"))
}

func (m Model) menuView() string {
	return m.decoratedPanel("Menu", []string{
		keyLine("s", "Weak kana"),
		keyLine("c", "Confusions"),
		keyLine("w", "Train weakest"),
		keyLine("r", "Training options"),
		keyLine("?", "Help"),
		keyLine("q", "Quit"),
	}, keyHints("Esc", "back"))
}

func (m Model) statsView() string {
	lines := make([]string, 0, 12)
	for index, item := range weakestItems(m.progress, 12) {
		state := m.progress.States[item.ID]
		average := "—"
		if state.Attempts > 0 {
			average = fmt.Sprintf("%.1fs", state.ResponseTimeEMA)
		}
		if m.width < 58 {
			lines = append(lines, fmt.Sprintf(
				"%2d  %s  %-3s  %3.0f%%  %d err  %s",
				index+1,
				item.Symbol(m.mode, false),
				item.Primary,
				100*state.Mastery(m.progress.Confusions.Score(item.ID)),
				state.Wrong,
				average,
			))
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"%2d  %s  %-3s   mastery %3.0f%%   errors %d   avg %s",
			index+1,
			item.Symbol(m.mode, false),
			item.Primary,
			100*state.Mastery(m.progress.Confusions.Score(item.ID)),
			state.Wrong,
			average,
		))
	}
	return m.decoratedPanel("Weak Kana", lines, keyHints("c", "confusions", "Esc", "back", "q", "quit"))
}

func (m Model) confusionsView() string {
	lines := make([]string, 0, 12)
	pairs := m.progress.Confusions.Pairs(1)
	if len(pairs) == 0 {
		lines = append(lines, dimStyle.Render("No learned confusions yet."))
	}
	for index, pair := range pairs {
		if index >= 12 {
			break
		}
		first, firstOK := kanadata.ByID(pair.FirstID)
		second, secondOK := kanadata.ByID(pair.SecondID)
		if !firstOK || !secondOK {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"%s  %s  %s    %s",
			first.Symbol(m.mode, false),
			titleStyle.Render("↔"),
			second.Symbol(m.mode, false),
			labelStyle.Render(fmt.Sprintf("%d mistakes", pair.Count)),
		))
	}
	return m.decoratedPanel("Most Confused", lines, keyHints("s", "weak kana", "Esc", "back", "q", "quit"))
}

func (m Model) helpView() string {
	return m.decoratedPanel("Help", []string{
		keyLine("a-z", "Type romaji"),
		keyLine("Backspace", "Delete"),
		keyLine("Enter", "Submit"),
		keyLine("Tab", "Switch script"),
		keyLine("Ctrl+R", "Training options"),
		keyLine("Esc", "Clear / menu / back"),
		keyLine("?", "Help"),
		keyLine("q", "Quit with empty answer"),
		keyLine("Ctrl+C", "Quit immediately"),
	}, keyHints("Esc", "back"))
}

func (m Model) optionsView() string {
	config := m.scheduler.Config()
	progression := "all kana"
	if config.Progressive {
		progression = "progressive rows"
	}
	lines := []string{
		labelStyle.Render("CURRENT"),
		fmt.Sprintf("Script       %s", modeName(m.mode)),
		fmt.Sprintf("Progression  %s", progression),
		fmt.Sprintf("Focus        %s", m.focus),
		"",
		keyLine("Tab", "Cycle script"),
		keyLine("p", "Toggle progression"),
		keyLine("a", "All kana, normal focus"),
		keyLine("n", "Normal adaptive focus"),
		keyLine("w", "Weakest kana"),
		keyLine("c", "Confused pairs"),
	}
	return m.decoratedPanel("Training Options", lines, keyHints("Esc", "keep settings"))
}

func weakestItems(progress trainer.Progress, limit int) []kanadata.Kana {
	items := append([]kanadata.Kana(nil), kanadata.Basic...)
	sort.SliceStable(items, func(i, j int) bool {
		left := progress.States[items[i].ID].Weakness(progress.Confusions.Score(items[i].ID))
		right := progress.States[items[j].ID].Weakness(progress.Confusions.Score(items[j].ID))
		return left > right
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (m Model) decoratedPanel(title string, lines []string, footer string) string {
	width := m.panelContentWidth()
	header := joinSides(titleStyle.Render(title), dimStyle.Render("KANA"), width)
	body := []string{header, dimStyle.Render(strings.Repeat("─", max(1, width))), ""}
	body = append(body, lines...)
	body = append(body, "", footer)
	return m.renderPanel(strings.Join(body, "\n"))
}

func (m Model) renderPanel(content string) string {
	if m.width < 24 {
		return renderOpaque(panelRestoreStyle, panelRestoreStyle, content)
	}
	panel := renderOpaque(
		frameStyle.Width(m.panelContentWidth()+4),
		panelRestoreStyle,
		content,
	)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, panel)
}

func (m Model) panelContentWidth() int {
	return max(16, min(64, m.width-8))
}

func joinSides(left, right string, width int) string {
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func keyLine(key, description string) string {
	gap := max(2, 7-lipgloss.Width(key))
	return keyStyle.Render(key) + strings.Repeat(" ", gap) + description
}

func keyHints(parts ...string) string {
	hints := make([]string, 0, len(parts)/2)
	for index := 0; index+1 < len(parts); index += 2 {
		hints = append(hints, keyStyle.Render(parts[index])+" "+dimStyle.Render(parts[index+1]))
	}
	return strings.Join(hints, dimStyle.Render("  ·  "))
}

func fitVertically(content string, height int) string {
	lineCount := strings.Count(content, "\n") + 1
	if height <= lineCount {
		return content
	}
	top := (height - lineCount) / 2
	return strings.Repeat("\n", top) + content
}

func modeName(mode kanadata.Mode) string {
	switch mode {
	case kanadata.ModeKatakana:
		return "Katakana"
	case kanadata.ModeMixed:
		return "Mixed"
	default:
		return "Hiragana"
	}
}
