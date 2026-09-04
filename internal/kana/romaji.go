package kana

import "strings"

// NormalizeAnswer applies the deliberately strict answer normalization rules.
func NormalizeAnswer(answer string) string {
	return strings.ToLower(strings.TrimSpace(answer))
}

// Matches reports whether an answer exactly matches an accepted romanization.
func Matches(item Kana, answer string) bool {
	normalized := NormalizeAnswer(answer)
	for _, accepted := range item.Accepted {
		if normalized == accepted {
			return true
		}
	}
	return false
}

// ResolveRomaji returns the single kana identified by an accepted answer.
// Ambiguous or unknown romanizations intentionally return no result.
func ResolveRomaji(answer string, items []Kana) (Kana, bool) {
	normalized := NormalizeAnswer(answer)
	var match Kana
	found := false
	for _, item := range items {
		for _, accepted := range item.Accepted {
			if normalized != accepted {
				continue
			}
			if found && match.ID != item.ID {
				return Kana{}, false
			}
			match = item
			found = true
			break
		}
	}
	return match, found
}
