package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	panelColor  = lipgloss.Color("#191F2C")
	cardColor   = lipgloss.Color("#293449")
	textColor   = lipgloss.Color("#EDF0F5")
	accentColor = lipgloss.Color("#B5CAEA")
	mutedColor  = lipgloss.Color("#A4ADC0")
	goodColor   = lipgloss.Color("#A8CBA4")
	badColor    = lipgloss.Color("#EEA6AE")
	borderColor = lipgloss.Color("#9FA6C8")

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
			Border(lipgloss.DoubleBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)
	cardStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(cardColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			Align(lipgloss.Center)
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(cardColor)
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
