package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	panelColor  = lipgloss.Color("#151B24")
	cardColor   = lipgloss.Color("#202A38")
	textColor   = lipgloss.Color("#F8FAFC")
	accentColor = lipgloss.Color("#38BDF8")
	mutedColor  = lipgloss.Color("#A8B3C7")
	goodColor   = lipgloss.Color("#4ADE80")
	badColor    = lipgloss.Color("#FB7185")
	borderColor = lipgloss.Color("#64748B")

	panelRestoreStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(panelColor)
	cardRestoreStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(cardColor)
	cardInputStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(cardColor).
			Bold(true)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)
	kanaStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)
	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(goodColor)
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(badColor)
	dimStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)
	frameStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(panelColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)
	cardStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(cardColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			Align(lipgloss.Center)
)

func renderOpaque(style, restore lipgloss.Style, content string) string {
	if restorePrefix := stylePrefix(restore); restorePrefix != "" {
		content = strings.ReplaceAll(content, "\x1b[0m", "\x1b[0m"+restorePrefix)
	}
	return style.Render(content)
}

func stylePrefix(style lipgloss.Style) string {
	const marker = "\uE000"

	prefix, _, found := strings.Cut(style.Render(marker), marker)
	if !found {
		return ""
	}
	return prefix
}
