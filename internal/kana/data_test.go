package kana

import "testing"

func TestBasicDataset(t *testing.T) {
	if got := len(Basic); got != 46 {
		t.Fatalf("len(Basic) = %d, want 46", got)
	}

	ids := make(map[string]bool, len(Basic))
	for _, item := range Basic {
		if item.ID == "" || item.Hiragana == "" || item.Katakana == "" || item.Primary == "" || item.Row == "" {
			t.Fatalf("incomplete kana: %+v", item)
		}
		if ids[item.ID] {
			t.Fatalf("duplicate ID %q", item.ID)
		}
		ids[item.ID] = true
		if len(item.Accepted) == 0 || item.Accepted[0] != item.Primary {
			t.Fatalf("%s must accept its primary romanization first", item.ID)
		}
	}
}

func TestByIDUnknown(t *testing.T) {
	if _, ok := ByID("missing"); ok {
		t.Fatal("unknown ID resolved")
	}
}

func TestSymbol(t *testing.T) {
	item, ok := ByID("ki")
	if !ok {
		t.Fatal("ki not found")
	}
	if got := item.Symbol(ModeHiragana, true); got != "き" {
		t.Fatalf("hiragana symbol = %q", got)
	}
	if got := item.Symbol(ModeKatakana, false); got != "キ" {
		t.Fatalf("katakana symbol = %q", got)
	}
	if got := item.Symbol(ModeMixed, true); got != "キ" {
		t.Fatalf("mixed katakana symbol = %q", got)
	}
}
