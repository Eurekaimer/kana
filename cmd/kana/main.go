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
	all := flags.Bool("all", false, "disable progressive unlocking")
	weak := flags.Bool("weak", false, "train the weakest kana")
	confusion := flags.Bool("confusion", false, "train learned confusion pairs")
	stats := flags.Bool("stats", false, "print statistics and exit")
	reset := flags.Bool("reset", false, "reset saved progress after confirmation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if boolCount(*hiragana, *katakana, *mixed) > 1 {
		return errors.New("choose only one of --hiragana, --katakana, or --mixed")
	}
	if boolCount(*weak, *confusion) > 1 {
		return errors.New("choose only one of --weak or --confusion")
	}
	if *stats && *reset {
		return errors.New("--stats and --reset cannot be used together")
	}

	path, err := storage.DefaultPath()
	if err != nil {
		return err
	}
	store := storage.New(path)
	if *reset {
		return resetProgress(store, input, output)
	}
	progress, err := store.Load(kanadata.Basic)
	if err != nil {
		return err
	}
	mode := selectedMode(*katakana, *mixed)
	if *stats {
		printStatistics(output, progress, mode, path)
		return nil
	}
	focus := trainer.FocusNormal
	if *weak {
		focus = trainer.FocusWeak
	} else if *confusion {
		focus = trainer.FocusConfusion
	}
	model, err := tui.NewModel(progress, store, tui.Options{
		Mode:        mode,
		Progressive: !*all,
		Focus:       focus,
		Seed:        time.Now().UnixNano(),
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

func printStatistics(output io.Writer, progress trainer.Progress, mode kanadata.Mode, path string) {
	overall := trainer.Overall(progress)
	fmt.Fprintln(output, "kana statistics")
	fmt.Fprintln(output)
	fmt.Fprintf(output, "Questions: %d\nCorrect:   %d\nAccuracy:  %.1f%%\nAverage response: %.2fs\n", overall.Attempts, overall.Correct, overall.Accuracy, overall.AverageResponse.Seconds())
	fmt.Fprintln(output, "\nWeak Kana")
	for index, statistic := range trainer.WeakStats(progress, kanadata.Basic, 12) {
		average := "—"
		if statistic.Memory.Attempts > 0 {
			average = fmt.Sprintf("%.1fs", statistic.Memory.ResponseTimeEMA)
		}
		fmt.Fprintf(
			output,
			"%2d. %s  %-3s  mastery %3.0f%%  errors %d  avg %s\n",
			index+1,
			statistic.Kana.Symbol(mode, false),
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
