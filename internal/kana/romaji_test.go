package kana

import "testing"

func TestMatches(t *testing.T) {
	shi, _ := ByID("shi")
	for _, answer := range []string{"shi", "si", " SHI ", "Si"} {
		if !Matches(shi, answer) {
			t.Errorf("Matches(shi, %q) = false", answer)
		}
	}
	for _, answer := range []string{"", "sh", "shii", "ji", "し"} {
		if Matches(shi, answer) {
			t.Errorf("Matches(shi, %q) = true", answer)
		}
	}
}

func TestResolveRomaji(t *testing.T) {
	item, ok := ResolveRomaji(" KU ", Basic)
	if !ok || item.ID != "ku" {
		t.Fatalf("ResolveRomaji(ku) = %+v, %v", item, ok)
	}
	if _, ok := ResolveRomaji("asdf", Basic); ok {
		t.Fatal("nonsense answer resolved to kana")
	}
}

func TestNormalizeAndAmbiguousRomaji(t *testing.T) {
	if got := NormalizeAnswer("  KI "); got != "ki" {
		t.Fatalf("NormalizeAnswer = %q", got)
	}
	duplicate := []Kana{
		{ID: "first", Accepted: []string{"x"}},
		{ID: "second", Accepted: []string{"x"}},
	}
	if _, ok := ResolveRomaji("x", duplicate); ok {
		t.Fatal("ambiguous answer resolved")
	}
}
