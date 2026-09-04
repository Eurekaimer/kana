package trainer

import (
	"math"
	"sort"
)

// ConfusionMatrix counts inferred mistakes from the expected kana to the answered kana.
type ConfusionMatrix map[string]map[string]int

// ConfusionPair combines both directions of a learned confusion.
type ConfusionPair struct {
	FirstID  string
	SecondID string
	Count    int
}

// Record adds an inferred wrong answer. Unknown and self answers are ignored.
func (m ConfusionMatrix) Record(expectedID, answeredID string) {
	if expectedID == "" || answeredID == "" || expectedID == answeredID {
		return
	}
	if m[expectedID] == nil {
		m[expectedID] = make(map[string]int)
	}
	m[expectedID][answeredID]++
}

// Count returns the directional count for one confusion.
func (m ConfusionMatrix) Count(expectedID, answeredID string) int {
	return m[expectedID][answeredID]
}

// TotalFor returns all incoming and outgoing mistakes involving a kana.
func (m ConfusionMatrix) TotalFor(id string) int {
	total := 0
	for answered, count := range m[id] {
		if answered != id {
			total += count
		}
	}
	for expected, answers := range m {
		if expected != id {
			total += answers[id]
		}
	}
	return total
}

// Score maps accumulated confusion evidence to [0,1].
func (m ConfusionMatrix) Score(id string) float64 {
	return 1 - math.Exp(-float64(m.TotalFor(id))/5)
}

// Pairs returns combined bidirectional pairs ordered by descending count.
func (m ConfusionMatrix) Pairs(minimum int) []ConfusionPair {
	pairs := make(map[[2]string]int)
	for expected, answers := range m {
		for answered, count := range answers {
			if expected == answered {
				continue
			}
			key := [2]string{expected, answered}
			if key[1] < key[0] {
				key[0], key[1] = key[1], key[0]
			}
			pairs[key] += count
		}
	}

	result := make([]ConfusionPair, 0, len(pairs))
	for pair, count := range pairs {
		if count >= minimum {
			result = append(result, ConfusionPair{FirstID: pair[0], SecondID: pair[1], Count: count})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		if result[i].FirstID != result[j].FirstID {
			return result[i].FirstID < result[j].FirstID
		}
		return result[i].SecondID < result[j].SecondID
	})
	return result
}
