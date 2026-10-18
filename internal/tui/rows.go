package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	kanadata "kana/internal/kana"
)

const rowKeys = "1234567890-"

func validRow(row string) bool {
	for _, candidate := range kanadata.Rows {
		if candidate == row {
			return true
		}
	}
	return false
}

func (m *Model) openRows() {
	m.rowsReturn = m.screen
	m.rows = make(map[string]bool, len(kanadata.Rows))
	if m.level == "beginner" || m.level == "gojuon" {
		for _, item := range m.scheduler.ActiveKana(m.items, m.progress.States, m.progress.Confusions) {
			m.rows[item.Row] = true
		}
	}
	m.rowError = ""
	m.openPanel(screenRows)
}

func (m *Model) applyRows() error {
	items := make([]kanadata.Kana, 0, len(kanadata.Basic))
	for _, item := range kanadata.Basic {
		if m.rows[item.Row] {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return fmt.Errorf("至少选择一行；数字键切换，可同时选择多行")
	}
	m.level, m.exercise, m.items = "gojuon", "reading", items
	m.manualRows = true
	m.scheduler.SetProgressive(false)
	m.activeCount = 0
	return m.selectNext()
}

func (m Model) updateRows(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up":
		m.rowCursor = max(0, m.rowCursor-1)
		m.keepVisible(m.rowCursor + 3)
	case "down":
		m.rowCursor = min(len(kanadata.Rows)-1, m.rowCursor+1)
		m.keepVisible(m.rowCursor + 3)
	case " ":
		row := kanadata.Rows[m.rowCursor]
		m.rows[row] = !m.rows[row]
		m.rowError = ""
		m.keepVisible(m.rowCursor + 3)
	case "enter":
		if err := m.applyRows(); err != nil {
			m.rowError = err.Error()
			m.scroll = m.maxScroll()
			return m, nil
		}
		m.openPanel(screenStudy)
		m.pausedAt = m.now()
	case "a":
		for _, row := range kanadata.Rows {
			m.rows[row] = true
		}
		m.rowError = ""
	case "x":
		clear(m.rows)
		m.rowError = ""
	case "p":
		if err := m.setCourse("beginner", "reading"); err != nil {
			m.fatal = err
			return m, tea.Quit
		}
		m.openPanel(screenStudy)
		m.pausedAt = m.now()
	default:
		if len(key.Runes) == 1 {
			if index := strings.IndexRune(rowKeys, key.Runes[0]); index >= 0 {
				row := kanadata.Rows[index]
				m.rows[row] = !m.rows[row]
				m.rowError = ""
				m.rowCursor = index
				m.keepVisible(index + 3)
			}
		}
	}
	return m, nil
}

func (m Model) rowsView() string {
	lines := []string{dimStyle.Render("可多选任意行；未达标也能手动加入下一行。")}
	for index, row := range kanadata.Rows {
		mark := "[ ]"
		if m.rows[row] {
			mark = titleStyle.Render("[✓]")
		}
		var symbols []string
		for _, item := range kanadata.Basic {
			if item.Row == row {
				symbols = append(symbols, item.Symbol(m.mode, false))
			}
		}
		line := "  " + keyLine(string(rowKeys[index]), mark+" "+strings.Join(symbols, " "))
		if index == m.rowCursor {
			line = selectedStyle.Width(m.panelContentWidth()).Render("› " + keyLine(string(rowKeys[index]), mark+" "+strings.Join(symbols, " ")))
		}
		lines = append(lines, line)
	}
	if m.rowError != "" {
		lines = append(lines, errorStyle.Render(m.rowError))
	}
	lines = append(lines, keyHints("a", "全选", "x", "清空", "p", "恢复自动解锁"))
	return m.decoratedPanel("五十音 · 自选行", lines, keyHints("↑↓/jk", "移动", "Space", "勾选", "Enter", "开始", "Esc", "取消"))
}

func (m Model) studyView() string {
	if m.level != "beginner" && m.level != "gojuon" {
		return m.decoratedPanel("认读提示", []string{"进阶识读：输入卡片单词的罗马音。", "语境模式：只读【目标词】，不要输入整句。", "答题后显示中文释义；Enter 手动继续。"}, keyHints("Enter", "开始", "?", "手册"))
	}
	active := m.scheduler.ActiveKana(m.items, m.progress.States, m.progress.Confusions)
	lines := []string{dimStyle.Render("先看字形与读音，再 Enter 开始回忆。")}
	for _, row := range kanadata.Rows {
		var cells []string
		for _, item := range active {
			if item.Row == row {
				cells = append(cells, fmt.Sprintf("%s %s %-3s", item.Hiragana, item.Katakana, item.Primary))
			}
		}
		if len(cells) > 0 {
			lines = append(lines, strings.Join(cells, "  "))
		}
	}
	if m.scheduler.Config().Progressive {
		lines = append(lines, "", "每字至少练习 4 次、最近连续答对 3 次，", "整行近期正确率达 85% 后自动加入下一行。", "新行解锁时会再次显示本页；g 可随时手选行。")
	}
	return m.decoratedPanel("五十音 · 认读对照", lines, keyHints("Enter", "开始练习", "g", "自选行", "?", "手册"))
}
