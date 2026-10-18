package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	kanadata "kana/internal/kana"
	"kana/internal/trainer"
)

// View renders centered, high-contrast application surfaces.
func (m Model) View() string {
	if m.width < 28 || m.height < 14 {
		return ansi.Hardwrap("KANA\n请放大终端至至少 28 × 14。\nCtrl+C 退出", max(1, m.width), true)
	}
	return fitVertically(m.renderPanel(m.contentView()), m.height)
}

func (m Model) contentView() string {
	if m.fatal != nil {
		return errorStyle.Render("kana: " + m.fatal.Error())
	}
	switch m.screen {
	case screenSplash:
		return m.splashView()
	case screenMenu:
		return m.menuView()
	case screenStats:
		return m.statsView()
	case screenConfusions:
		return m.confusionsView()
	case screenHelp:
		return m.helpView()
	case screenOptions:
		return m.optionsView()
	case screenRows:
		return m.rowsView()
	case screenStudy:
		return m.studyView()
	default:
		return m.trainingView()
	}
}

func (m Model) trainingView() string {
	active := m.scheduler.ActiveKana(m.items, m.progress.States, m.progress.Confusions)
	state := m.progress.States[m.current.ID]
	width := m.panelContentWidth()
	header := joinSides(titleStyle.Render("かな  KANA"), dimStyle.Render(fmt.Sprintf("%d / %d", len(active), len(m.items))), width)
	course := levelName(m.level)
	if m.exercise == "anime" {
		course += " / 动漫风语境"
	}
	if m.manualRows {
		course = "五十音自选行"
	}
	feedback := dimStyle.Render("输入罗马音，Enter 确认")
	if m.feedback == "✓" {
		feedback = successStyle.Render("✓  正确")
	} else if m.feedback != "" {
		feedback = errorStyle.Render(m.feedback)
	}
	input := m.input
	input.Width = max(8, min(28, width-10))
	cardWidth := max(12, min(46, width-6))
	cardLines := []string{kanaStyle.Render(m.currentSymbol()), "", input.View(), feedback}
	card := renderOpaque(cardStyle.Width(cardWidth).Padding(0, 2), cardRestoreStyle,
		ansi.Hardwrap(strings.Join(cardLines, "\n"), cardWidth-4, true))
	lines := []string{header, dimStyle.Render(course), "", lipgloss.PlaceHorizontal(width, lipgloss.Center, card)}
	if m.current.Context != "" && (m.exercise == "anime" || m.feedback != "") {
		target := m.current.Hiragana
		if m.mode == kanadata.ModeKatakana || (m.mode == kanadata.ModeMixed && m.katakanaVariant()) {
			target = m.current.Katakana
		}
		context := strings.ReplaceAll(m.current.Context, "【"+m.current.Hiragana+"】", "【"+target+"】")
		context = strings.ReplaceAll(context, "【"+m.current.Katakana+"】", "【"+target+"】")
		lines = append(lines, "", context, dimStyle.Render("只输入【目标词】的罗马音"))
	}
	if m.feedback != "" && m.current.Meaning != "" {
		lines = append(lines, "", successStyle.Render(m.current.Primary+" · "+m.current.Meaning))
		if m.current.ContextMeaning != "" {
			lines = append(lines, m.current.ContextMeaning)
		}
		lines = append(lines, dimStyle.Render(m.current.Source), keyHints("Enter", "下一题"))
	}
	status := fmt.Sprintf("正确率 %.0f%%    本音连对 %d    %s", m.session.Accuracy(), state.CurrentStreak, modeName(m.mode))
	if m.width < 44 {
		status = fmt.Sprintf("正确率 %.0f%%   连对 %d", m.session.Accuracy(), state.CurrentStreak)
	}
	if m.current.Meaning == "" || m.feedback == "" {
		lines = append(lines, "", labelStyle.Render(status), dimStyle.Render(focusName(m.focus)))
	}
	if m.saveError != nil {
		lines = append(lines, errorStyle.Render("保存失败: "+m.saveError.Error()))
	}
	lines = append(lines, keyHints("Ctrl+R", "课程", "?", "手册", "Esc", "菜单"))
	return strings.Join(lines, "\n")
}

func (m Model) menuView() string {
	return m.decoratedPanel("菜单", []string{
		keyLine("r", "选择等级与练习模式"),
		keyLine("s", "薄弱项目 / Weak Kana"),
		keyLine("c", "易混淆项目"),
		keyLine("w", "集中练习薄弱项"),
		keyLine("g", "自选多个行"),
		keyLine("t", "五十音认读对照"),
		keyLine("?", "详细学习手册"),
		keyLine("q", "保存并退出"),
	}, keyHints("Esc", "继续练习"))
}

func (m Model) statsView() string {
	lines := make([]string, 0, 12)
	for index, item := range weakestItems(m.progress, m.items, 12) {
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

func (m Model) optionsView() string {
	lines := make([]string, 0, 14)
	for index, level := range kanadata.Levels {
		name := levelName(level)
		if m.manualRows && level == "gojuon" {
			name = "五十音 · 当前自选行"
		}
		line := "  " + fmt.Sprintf("%d  %s", index+1, name)
		if level == m.level {
			line = selectedStyle.Width(m.panelContentWidth()).Render("› " + fmt.Sprintf("%d  %s", index+1, name))
		}
		lines = append(lines, line)
	}
	lines = append(lines, "",
		keyLine("g", "自选行（支持多选）"),
		keyLine("e", exerciseName(m.exercise)),
		keyLine("Tab", modeName(m.mode)),
		keyLine("n/r", "自适应 / 随机"),
		keyLine("w/c", "薄弱 / 易混淆"),
		dimStyle.Render("当前："+focusName(m.focus)),
	)
	return m.decoratedPanel("选择课程", lines, keyHints("↑↓/jk/1–7", "选择", "Enter", "开始", "?", "手册"))
}

func weakestItems(progress trainer.Progress, source []kanadata.Kana, limit int) []kanadata.Kana {
	items := append([]kanadata.Kana(nil), source...)
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
	body := []string{header, dimStyle.Render(strings.Repeat("─", max(1, width)))}
	body = append(body, lines...)
	body = append(body, footer)
	return strings.Join(body, "\n")
}

func (m Model) renderPanel(content string) string {
	width := m.panelContentWidth()
	lines := strings.Split(ansi.Hardwrap(content, width, true), "\n")
	available := max(1, m.height-6)
	if len(lines) > available {
		available = max(1, available-1)
		offset := min(m.scroll, max(0, len(lines)-available))
		end := min(len(lines), offset+available)
		lines = append(lines[offset:end:end], dimStyle.Render(fmt.Sprintf("↑↓/jk %d–%d/%d", offset+1, end, len(lines))))
	}
	panel := renderOpaque(
		frameStyle.Width(width+4),
		panelRestoreStyle,
		strings.Join(lines, "\n"),
	)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, panel)
}

func (m Model) maxScroll() int {
	lines := strings.Count(ansi.Hardwrap(m.contentView(), m.panelContentWidth(), true), "\n") + 1
	available := max(1, m.height-6)
	if lines <= available {
		return 0
	}
	return max(0, lines-max(1, available-1))
}

func (m *Model) keepVisible(line int) {
	available := max(1, m.height-7)
	if line < m.scroll {
		m.scroll = line
	} else if line >= m.scroll+available {
		m.scroll = line - available + 1
	}
}

func (m Model) panelContentWidth() int {
	return max(18, min(70, m.width-10))
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
		return "片假名"
	case kanadata.ModeMixed:
		return "混合"
	default:
		return "平假名"
	}
}

func levelName(level string) string {
	switch level {
	case "beginner":
		return "入门 · 五十音逐行解锁"
	case "gojuon":
		return "五十音 · 46 音全量随机"
	default:
		return strings.ToUpper(level) + " · 进阶词汇识读"
	}
}

func exerciseName(exercise string) string {
	if exercise == "anime" {
		return "动漫风语境"
	}
	return "假名 / 词汇识读"
}

func focusName(focus trainer.FocusMode) string {
	switch focus {
	case trainer.FocusRandom:
		return "均匀随机"
	case trainer.FocusWeak:
		return "薄弱项目优先"
	case trainer.FocusConfusion:
		return "易混淆对比"
	default:
		return "自适应间隔复习"
	}
}
