package kana

import (
	"reflect"
	"strings"
	"testing"
)

func gradedTestLevels() []string {
	return []string{"n5", "n4", "n3", "n2", "n1"}
}

func basicIDSet() map[string]bool {
	ids := make(map[string]bool, len(Basic))
	for _, item := range Basic {
		ids[item.ID] = true
	}
	return ids
}

func markedTarget(t *testing.T, text string) string {
	t.Helper()
	if strings.Count(text, "【") != 1 || strings.Count(text, "】") != 1 {
		t.Fatalf("context must mark exactly one target: %q", text)
	}
	start := strings.Index(text, "【")
	end := strings.Index(text, "】")
	if end <= start+len("【") {
		t.Fatalf("context has an empty target: %q", text)
	}
	return text[start+len("【") : end]
}

func hasLatin(text string) bool {
	for _, character := range text {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') {
			return true
		}
	}
	return false
}

func TestBasicDecks(t *testing.T) {
	for _, level := range []string{"beginner", "gojuon"} {
		deck, err := Deck(level, "reading")
		if err != nil {
			t.Fatalf("Deck(%q, reading): %v", level, err)
		}
		if len(deck) != len(Basic) {
			t.Fatalf("Deck(%q, reading) has %d cards, want %d", level, len(deck), len(Basic))
		}
		for index := range Basic {
			if deck[index].ID != Basic[index].ID {
				t.Fatalf("Deck(%q, reading)[%d] = %q, want %q", level, index, deck[index].ID, Basic[index].ID)
			}
		}
	}
}

func TestDeckErrorsUnknownCombinations(t *testing.T) {
	cases := []struct {
		level    string
		exercise string
	}{
		{"n6", "reading"},
		{"", "reading"},
		{"n5", "listening"},
		{"n5", ""},
		{"beginner", "anime"},
		{"gojuon", "anime"},
	}
	for _, testCase := range cases {
		if _, err := Deck(testCase.level, testCase.exercise); err == nil {
			t.Fatalf("Deck(%q, %q) accepted an unsupported combination", testCase.level, testCase.exercise)
		}
	}
}

func TestDeckNormalizesLevelAndExercise(t *testing.T) {
	upper, err := Deck(" N5 ", " Reading ")
	if err != nil {
		t.Fatalf("Deck normalized input: %v", err)
	}
	lower, err := Deck("n5", "reading")
	if err != nil {
		t.Fatalf("Deck lower input: %v", err)
	}
	if len(upper) != len(lower) {
		t.Fatalf("normalized deck length = %d, want %d", len(upper), len(lower))
	}
	for index := range lower {
		if upper[index].ID != lower[index].ID {
			t.Fatalf("normalized card %d = %q, want %q", index, upper[index].ID, lower[index].ID)
		}
	}
}

func TestGradedVocabularyDecks(t *testing.T) {
	basic := basicIDSet()
	for _, level := range gradedTestLevels() {
		deck, err := Deck(level, "reading")
		if err != nil {
			t.Fatalf("Deck(%q, reading): %v", level, err)
		}
		if len(deck) < 12 {
			t.Fatalf("Deck(%q, reading) has %d cards, want at least 12", level, len(deck))
		}
		seen := make(map[string]bool, len(deck))
		hiraganaCards := 0
		katakanaCards := 0
		for _, card := range deck {
			if seen[card.ID] {
				t.Fatalf("duplicate vocabulary ID %q", card.ID)
			}
			seen[card.ID] = true
			if basic[card.ID] {
				t.Fatalf("vocabulary card reused basic ID %q", card.ID)
			}
			if !strings.HasPrefix(card.ID, level+".word.") {
				t.Fatalf("vocabulary ID %q is not namespaced for %s", card.ID, level)
			}
			if card.Hiragana == "" || card.Katakana == "" || card.Primary == "" {
				t.Fatalf("vocabulary card %q is incomplete: %+v", card.ID, card)
			}
			if len(card.Accepted) == 0 || card.Accepted[0] != card.Primary {
				t.Fatalf("vocabulary card %q must accept its primary first", card.ID)
			}
			if card.Row != level {
				t.Fatalf("vocabulary card %q row = %q, want %q", card.ID, card.Row, level)
			}
			if card.Meaning == "" || card.Context == "" || card.ContextMeaning == "" {
				t.Fatalf("vocabulary card %q is missing teaching text", card.ID)
			}
			if hasLatin(card.Meaning) || hasLatin(card.ContextMeaning) {
				t.Fatalf("vocabulary card %q leaks romaji in a Chinese hint", card.ID)
			}

			if card.NativeKatakana {
				katakanaCards++
			} else {
				hiraganaCards++
			}
			expectedTarget := card.Hiragana
			if card.NativeKatakana {
				expectedTarget = card.Katakana
			}
			if target := markedTarget(t, card.Context); target != expectedTarget {
				t.Fatalf("vocabulary card %q marks %q, want %q", card.ID, target, expectedTarget)
			}
		}
		if hiraganaCards < 6 || katakanaCards < 6 {
			t.Fatalf("Deck(%q, reading) balance = %d hiragana / %d katakana, want at least 6 each", level, hiraganaCards, katakanaCards)
		}
	}
}

func TestGradedAnimeDecks(t *testing.T) {
	basic := basicIDSet()
	for _, level := range gradedTestLevels() {
		deck, err := Deck(level, "anime")
		if err != nil {
			t.Fatalf("Deck(%q, anime): %v", level, err)
		}
		if len(deck) < 4 {
			t.Fatalf("Deck(%q, anime) has %d cards, want at least 4", level, len(deck))
		}
		seen := make(map[string]bool, len(deck))
		for _, card := range deck {
			if seen[card.ID] {
				t.Fatalf("duplicate context ID %q", card.ID)
			}
			seen[card.ID] = true
			if basic[card.ID] {
				t.Fatalf("context card reused basic ID %q", card.ID)
			}
			if !strings.HasPrefix(card.ID, level+".context.") {
				t.Fatalf("context ID %q is not namespaced for %s", card.ID, level)
			}
			if card.Hiragana == "" || card.Katakana == "" || card.Primary == "" {
				t.Fatalf("context card %q is incomplete: %+v", card.ID, card)
			}
			if len(card.Accepted) == 0 || card.Accepted[0] != card.Primary {
				t.Fatalf("context card %q must accept its primary first", card.ID)
			}
			if card.Row != level+"-anime" {
				t.Fatalf("context card %q row = %q", card.ID, card.Row)
			}
			if card.Meaning == "" || card.ContextMeaning == "" {
				t.Fatalf("context card %q is missing teaching text", card.ID)
			}
			if hasLatin(card.Meaning) || hasLatin(card.ContextMeaning) {
				t.Fatalf("context card %q leaks romaji in a Chinese hint", card.ID)
			}
			for _, accepted := range card.Accepted {
				if strings.Contains(strings.ToLower(card.ContextMeaning), accepted) {
					t.Fatalf("context card %q leaks accepted answer %q", card.ID, accepted)
				}
			}
			expectedTarget := card.Hiragana
			if card.NativeKatakana {
				expectedTarget = card.Katakana
			}
			if target := markedTarget(t, card.Context); target != expectedTarget {
				t.Fatalf("context card %q marks %q, want %q", card.ID, target, expectedTarget)
			}
		}
	}
}

func TestDeckAcceptedAnswersAreUnambiguous(t *testing.T) {
	type namedDeck struct {
		name  string
		cards []Kana
	}
	decks := make([]namedDeck, 0, 12)
	for _, level := range []string{"beginner", "gojuon"} {
		deck, err := Deck(level, "reading")
		if err != nil {
			t.Fatalf("Deck(%q, reading): %v", level, err)
		}
		decks = append(decks, namedDeck{name: level + "/reading", cards: deck})
	}
	for _, level := range gradedTestLevels() {
		for _, exercise := range []string{"reading", "anime"} {
			deck, err := Deck(level, exercise)
			if err != nil {
				t.Fatalf("Deck(%q, %q): %v", level, exercise, err)
			}
			decks = append(decks, namedDeck{name: level + "/" + exercise, cards: deck})
		}
	}

	for _, deck := range decks {
		seen := make(map[string]string, len(deck.cards)*2)
		for _, card := range deck.cards {
			for _, accepted := range card.Accepted {
				if normalized := NormalizeAnswer(accepted); normalized != accepted {
					t.Fatalf("%s: accepted answer %q is not normalized", deck.name, accepted)
				}
				if previous, exists := seen[accepted]; exists {
					t.Fatalf("%s: %q is accepted by both %q and %q", deck.name, accepted, previous, card.ID)
				}
				seen[accepted] = card.ID
			}
		}
		for _, card := range deck.cards {
			for _, accepted := range card.Accepted {
				resolved, ok := ResolveRomaji(accepted, deck.cards)
				if !ok || resolved.ID != card.ID {
					t.Fatalf("%s: ResolveRomaji(%q) = %q, %v; want %q", deck.name, accepted, resolved.ID, ok, card.ID)
				}
			}
		}
	}
}

func TestAllAndByID(t *testing.T) {
	total := len(Basic)
	for _, level := range gradedTestLevels() {
		words, err := Deck(level, "reading")
		if err != nil {
			t.Fatalf("Deck(%q, reading): %v", level, err)
		}
		anime, err := Deck(level, "anime")
		if err != nil {
			t.Fatalf("Deck(%q, anime): %v", level, err)
		}
		total += len(words) + len(anime)
	}

	all := All()
	if len(all) != total {
		t.Fatalf("All() has %d cards, want %d", len(all), total)
	}
	seen := make(map[string]bool, len(all))
	for _, want := range all {
		if seen[want.ID] {
			t.Fatalf("duplicate card ID %q in All()", want.ID)
		}
		seen[want.ID] = true
		got, ok := ByID(want.ID)
		if !ok {
			t.Fatalf("ByID(%q) not found", want.ID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ByID(%q) = %+v, want %+v", want.ID, got, want)
		}
	}
	for _, id := range []string{"", "missing", "n5.word.missing"} {
		if _, ok := ByID(id); ok {
			t.Fatalf("ByID(%q) unexpectedly found a card", id)
		}
	}
}

func TestRomanizationConventions(t *testing.T) {
	cases := []struct {
		id       string
		accepted []string
		rejected []string
	}{
		{
			id:       "n5.word.gakkou",
			accepted: []string{"gakkou", "gakkoo"},
			rejected: []string{"gakko", "gakkouu"},
		},
		{
			id:       "n5.word.konya",
			accepted: []string{"kon'ya", "konya"},
			rejected: []string{"konnya"},
		},
		{
			id:       "n5.word.koohii",
			accepted: []string{"koohii"},
			rejected: []string{"kohii", "koohi"},
		},
		{
			id:       "n4.word.hakkiri",
			accepted: []string{"hakkiri"},
			rejected: []string{"hakiri", "hakkiri-"},
		},
		{
			id:       "n4.word.kyoumi",
			accepted: []string{"kyoumi", "kyoomi"},
			rejected: []string{"kyomi"},
		},
		{
			id:       "n3.word.tsutaeru",
			accepted: []string{"tsutaeru", "tutaeru"},
			rejected: []string{"tsutairu"},
		},
		{
			id:       "n2.word.magirawashii",
			accepted: []string{"magirawashii", "magirawasii"},
			rejected: []string{"magirawashi"},
		},
		{
			id:       "n2.context.tameratta",
			accepted: []string{"tameratta"},
			rejected: []string{"tamerata"},
		},
		{
			id:       "n1.word.jirenma",
			accepted: []string{"jirenma", "zirenma"},
			rejected: []string{"jirema"},
		},
		{
			id:       "n1.word.nouhau",
			accepted: []string{"nouhau", "noohau"},
			rejected: []string{"nohau"},
		},
	}

	for _, testCase := range cases {
		card, ok := ByID(testCase.id)
		if !ok {
			t.Fatalf("ByID(%q) not found", testCase.id)
		}
		for _, accepted := range testCase.accepted {
			if !Matches(card, accepted) {
				t.Fatalf("%q should accept %q", testCase.id, accepted)
			}
			if !Matches(card, " "+strings.ToUpper(accepted)+" ") {
				t.Fatalf("%q should accept normalized %q", testCase.id, accepted)
			}
		}
		for _, rejected := range testCase.rejected {
			if Matches(card, rejected) {
				t.Fatalf("%q should reject %q", testCase.id, rejected)
			}
		}
	}

}
