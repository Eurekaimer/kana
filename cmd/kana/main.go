package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	kanadata "kana/internal/kana"
	"kana/internal/storage"
	"kana/internal/trainer"
	"kana/internal/tui"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "kana:", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, errorOutput io.Writer) error {
	flags := flag.NewFlagSet("kana", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	hiragana := flags.Bool("hiragana", false, "train hiragana")
	katakana := flags.Bool("katakana", false, "train katakana")
	mixed := flags.Bool("mixed", false, "mix hiragana and katakana")
	all := flags.Bool("all", false, "practice all 46 kana without row unlocking")
	weak := flags.Bool("weak", false, "train the weakest kana")
	confusion := flags.Bool("confusion", false, "train learned confusion pairs")
	stats := flags.Bool("stats", false, "print statistics and exit")
	reset := flags.Bool("reset", false, "reset saved progress after confirmation")
	level := flags.String("level", "beginner", "beginner, gojuon, n5, n4, n3, n2, n1")
	exercise := flags.String("exercise", "reading", "reading or anime (N5–N1)")
	random := flags.Bool("random", false, "uniform random practice instead of adaptive review")
	rowsFlag := flags.String("rows", "", "practice selected rows: a,k,s,t,n,h,m,y,r,w,nn")
	noSplash := flags.Bool("no-splash", false, "skip startup animation and course picker")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if boolCount(*hiragana, *katakana, *mixed) > 1 {
		return errors.New("choose only one of --hiragana, --katakana, or --mixed")
	}
	if boolCount(*weak, *confusion, *random) > 1 {
		return errors.New("choose only one of --weak, --confusion, or --random")
	}
	if *stats && *reset {
		return errors.New("--stats and --reset cannot be used together")
	}
	*level = strings.ToLower(strings.TrimSpace(*level))
	*exercise = strings.ToLower(strings.TrimSpace(*exercise))
	if *all {
		if *level != "beginner" && *level != "gojuon" {
			return errors.New("--all only applies to basic gojuon practice")
		}
		*level = "gojuon"
	}
	rows, err := parseRows(*rowsFlag)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		if *all || (*level != "beginner" && *level != "gojuon") || *exercise != "reading" {
			return errors.New("--rows requires basic reading practice and cannot be combined with --all")
		}
		*level = "gojuon"
	}
	items, err := kanadata.Deck(*level, *exercise)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		selected := make(map[string]bool, len(rows))
		for _, row := range rows {
			selected[row] = true
		}
		filtered := items[:0]
		for _, item := range items {
			if selected[item.Row] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	path, err := storage.DefaultPath()
	if err != nil {
		return err
	}
	store := storage.New(path)
	if *reset {
		return resetProgress(store, input, output)
	}
	progress, err := store.Load(kanadata.All())
	if err != nil {
		return err
	}
	mode := selectedMode(*katakana, *mixed)
	if *stats {
		printStatistics(output, progress, mode, path, items)
		return nil
	}
	focus := trainer.FocusNormal
	if *weak {
		focus = trainer.FocusWeak
	} else if *confusion {
		focus = trainer.FocusConfusion
	} else if *random || *level == "gojuon" {
		focus = trainer.FocusRandom
	}
	model, err := tui.NewModel(progress, store, tui.Options{
		Mode:        mode,
		Progressive: *level == "beginner",
		Focus:       focus,
		Seed:        time.Now().UnixNano(),
		Level:       *level,
		Exercise:    *exercise,
		Rows:        rows,
		NoSplash:    *noSplash || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb",
	})
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(input),
		tea.WithOutput(output),
	).Run()
	if err != nil {
		return err
	}
	if finished, ok := finalModel.(tui.Model); ok {
		fmt.Fprint(output, finished.Summary())
	}
	return nil
}

func selectedMode(katakana, mixed bool) kanadata.Mode {
	if katakana {
		return kanadata.ModeKatakana
	}
	if mixed {
		return kanadata.ModeMixed
	}
	return kanadata.ModeHiragana
}

func resetProgress(store *storage.Store, input io.Reader, output io.Writer) error {
	fmt.Fprintf(output, "Delete progress at %s? [y/N]: ", store.Path())
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		fmt.Fprintln(output, "Reset cancelled.")
		return nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(output, "Reset cancelled.")
		return nil
	}
	if err := store.Reset(); err != nil {
		return err
	}
	fmt.Fprintln(output, "Progress reset.")
	return nil
}

func printStatistics(output io.Writer, progress trainer.Progress, mode kanadata.Mode, path string, items []kanadata.Kana) {
	overall := trainer.Overall(progress)
	fmt.Fprintln(output, "kana statistics")
	fmt.Fprintln(output)
	fmt.Fprintf(output, "Questions: %d\nCorrect:   %d\nAccuracy:  %.1f%%\nAverage response: %.2fs\n", overall.Attempts, overall.Correct, overall.Accuracy, overall.AverageResponse.Seconds())
	fmt.Fprintln(output, "\nWeak Kana")
	for index, statistic := range trainer.WeakStats(progress, items, 12) {
		average := "—"
		if statistic.Memory.Attempts > 0 {
			average = fmt.Sprintf("%.1fs", statistic.Memory.ResponseTimeEMA)
		}
		fmt.Fprintf(
			output,
			"%2d. %s  %-3s  mastery %3.0f%%  errors %d  avg %s\n",
			index+1,
			statistic.Kana.Symbol(mode, statistic.Kana.NativeKatakana),
			statistic.Kana.Primary,
			100*statistic.Mastery,
			statistic.Memory.Wrong,
			average,
		)
	}
	fmt.Fprintln(output, "\nMost Confused")
	pairs := progress.Confusions.Pairs(1)
	if len(pairs) == 0 {
		fmt.Fprintln(output, "none")
	}
	for _, pair := range pairs {
		first, firstOK := kanadata.ByID(pair.FirstID)
		second, secondOK := kanadata.ByID(pair.SecondID)
		if firstOK && secondOK {
			fmt.Fprintf(output, "%s ↔ %s  %d\n", first.Symbol(mode, false), second.Symbol(mode, false), pair.Count)
		}
	}
	fmt.Fprintf(output, "\nData: %s\n", path)
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func parseRows(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	var rows []string
	seen := make(map[string]bool)
	for _, token := range strings.Split(value, ",") {
		row := strings.ToLower(strings.TrimSpace(token))
		valid := false
		for _, candidate := range kanadata.Rows {
			if candidate == row {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("unknown row %q; choose a,k,s,t,n,h,m,y,r,w,nn", token)
		}
		if !seen[row] {
			rows = append(rows, row)
			seen[row] = true
		}
	}
	return rows, nil
}
