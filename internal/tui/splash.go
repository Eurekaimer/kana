package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const splashFrames = 22

var kanaLogo = []string{
	"██╗  ██╗ █████╗ ███╗   ██╗ █████╗ ",
	"██║ ██╔╝██╔══██╗████╗  ██║██╔══██╗",
	"█████╔╝ ███████║██╔██╗ ██║███████║",
	"██╔═██╗ ██╔══██║██║╚██╗██║██╔══██║",
	"██║  ██╗██║  ██║██║ ╚████║██║  ██║",
	"╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝",
}

type splashTickMsg struct{}

func splashTick() tea.Cmd {
	return tea.Tick(55*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
}

func (m Model) splashView() string {
	width := m.panelContentWidth()
	logo := titleStyle.Render("K A N A")
	if width >= 36 {
		lines := make([]string, len(kanaLogo))
		sweep := m.splashFrame * 42 / splashFrames
		for row, line := range kanaLogo {
			var rendered strings.Builder
			for column, character := range []rune(line) {
				style := dimStyle
				if column <= sweep {
					style = titleStyle
				}
				if column >= sweep-2 && column <= sweep {
					style = kanaStyle
				}
				rendered.WriteString(style.Render(string(character)))
			}
			lines[row] = rendered.String()
		}
		logo = strings.Join(lines, "\n")
	}
	filled := min(22, m.splashFrame)
	bar := titleStyle.Render(strings.Repeat("━", filled)) + dimStyle.Render(strings.Repeat("─", 22-filled))
	lines := []string{
		lipgloss.PlaceHorizontal(width, lipgloss.Center, logo),
		"",
		lipgloss.PlaceHorizontal(width, lipgloss.Center, titleStyle.Render("かな · 一字ずつ、少しずつ。")),
		lipgloss.PlaceHorizontal(width, lipgloss.Center, dimStyle.Render("一次一字，慢慢记住。")),
		"",
		lipgloss.PlaceHorizontal(width, lipgloss.Center, bar),
		"",
		lipgloss.PlaceHorizontal(width, lipgloss.Center, keyHints("Enter", "选择课程", "Esc", "跳过")),
	}
	return strings.Join(lines, "\n")
}
