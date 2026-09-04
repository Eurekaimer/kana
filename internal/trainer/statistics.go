package trainer

import (
	"sort"
	"time"

	kanadata "kana/internal/kana"
)

// KanaStat combines one kana with its calculated learning metrics.
type KanaStat struct {
	Kana    kanadata.Kana
	Memory  MemoryState
	Mastery float64
}

// OverallStats summarizes durable attempts across all kana.
type OverallStats struct {
	Attempts        int
	Correct         int
	Accuracy        float64
	AverageResponse time.Duration
}

// WeakStats returns kana ordered from weakest to strongest.
func WeakStats(progress Progress, items []kanadata.Kana, limit int) []KanaStat {
	statistics := make([]KanaStat, 0, len(items))
	for _, item := range items {
		memory := progress.States[item.ID]
		statistics = append(statistics, KanaStat{
			Kana:    item,
			Memory:  memory,
			Mastery: memory.Mastery(progress.Confusions.Score(item.ID)),
		})
	}
	sort.SliceStable(statistics, func(i, j int) bool {
		return statistics[i].Mastery < statistics[j].Mastery
	})
	if limit > 0 && len(statistics) > limit {
		statistics = statistics[:limit]
	}
	return statistics
}

// Overall calculates aggregate accuracy and an attempt-weighted response average.
func Overall(progress Progress) OverallStats {
	var result OverallStats
	weightedResponse := 0.0
	for _, memory := range progress.States {
		result.Attempts += memory.Attempts
		result.Correct += memory.Correct
		weightedResponse += memory.ResponseTimeEMA * float64(memory.Attempts)
	}
	if result.Attempts > 0 {
		result.Accuracy = 100 * float64(result.Correct) / float64(result.Attempts)
		result.AverageResponse = time.Duration(weightedResponse / float64(result.Attempts) * float64(time.Second))
	}
	return result
}
