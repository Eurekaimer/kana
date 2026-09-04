package kana

// Kana describes one basic gojuon symbol and its accepted romanizations.
type Kana struct {
	ID         string   `json:"id"`
	Hiragana   string   `json:"hiragana"`
	Katakana   string   `json:"katakana"`
	Primary    string   `json:"primary"`
	Accepted   []string `json:"accepted"`
	Row        string   `json:"row"`
	Confusable []string `json:"confusable,omitempty"`
}

// Mode determines which script is shown in questions.
type Mode string

const (
	ModeHiragana Mode = "hiragana"
	ModeKatakana Mode = "katakana"
	ModeMixed    Mode = "mixed"
)

// Rows is the progression order used when unlocking kana.
var Rows = []string{"a", "k", "s", "t", "n", "h", "m", "y", "r", "w", "nn"}

// Basic contains the 46 modern basic kana in gojuon order.
var Basic = []Kana{
	{ID: "a", Hiragana: "あ", Katakana: "ア", Primary: "a", Accepted: []string{"a"}, Row: "a"},
	{ID: "i", Hiragana: "い", Katakana: "イ", Primary: "i", Accepted: []string{"i"}, Row: "a"},
	{ID: "u", Hiragana: "う", Katakana: "ウ", Primary: "u", Accepted: []string{"u"}, Row: "a"},
	{ID: "e", Hiragana: "え", Katakana: "エ", Primary: "e", Accepted: []string{"e"}, Row: "a"},
	{ID: "o", Hiragana: "お", Katakana: "オ", Primary: "o", Accepted: []string{"o"}, Row: "a"},
	{ID: "ka", Hiragana: "か", Katakana: "カ", Primary: "ka", Accepted: []string{"ka"}, Row: "k"},
	{ID: "ki", Hiragana: "き", Katakana: "キ", Primary: "ki", Accepted: []string{"ki"}, Row: "k"},
	{ID: "ku", Hiragana: "く", Katakana: "ク", Primary: "ku", Accepted: []string{"ku"}, Row: "k"},
	{ID: "ke", Hiragana: "け", Katakana: "ケ", Primary: "ke", Accepted: []string{"ke"}, Row: "k"},
	{ID: "ko", Hiragana: "こ", Katakana: "コ", Primary: "ko", Accepted: []string{"ko"}, Row: "k"},
	{ID: "sa", Hiragana: "さ", Katakana: "サ", Primary: "sa", Accepted: []string{"sa"}, Row: "s"},
	{ID: "shi", Hiragana: "し", Katakana: "シ", Primary: "shi", Accepted: []string{"shi", "si"}, Row: "s"},
	{ID: "su", Hiragana: "す", Katakana: "ス", Primary: "su", Accepted: []string{"su"}, Row: "s"},
	{ID: "se", Hiragana: "せ", Katakana: "セ", Primary: "se", Accepted: []string{"se"}, Row: "s"},
	{ID: "so", Hiragana: "そ", Katakana: "ソ", Primary: "so", Accepted: []string{"so"}, Row: "s"},
	{ID: "ta", Hiragana: "た", Katakana: "タ", Primary: "ta", Accepted: []string{"ta"}, Row: "t"},
	{ID: "chi", Hiragana: "ち", Katakana: "チ", Primary: "chi", Accepted: []string{"chi", "ti"}, Row: "t"},
	{ID: "tsu", Hiragana: "つ", Katakana: "ツ", Primary: "tsu", Accepted: []string{"tsu", "tu"}, Row: "t"},
	{ID: "te", Hiragana: "て", Katakana: "テ", Primary: "te", Accepted: []string{"te"}, Row: "t"},
	{ID: "to", Hiragana: "と", Katakana: "ト", Primary: "to", Accepted: []string{"to"}, Row: "t"},
	{ID: "na", Hiragana: "な", Katakana: "ナ", Primary: "na", Accepted: []string{"na"}, Row: "n"},
	{ID: "ni", Hiragana: "に", Katakana: "ニ", Primary: "ni", Accepted: []string{"ni"}, Row: "n"},
	{ID: "nu", Hiragana: "ぬ", Katakana: "ヌ", Primary: "nu", Accepted: []string{"nu"}, Row: "n"},
	{ID: "ne", Hiragana: "ね", Katakana: "ネ", Primary: "ne", Accepted: []string{"ne"}, Row: "n"},
	{ID: "no", Hiragana: "の", Katakana: "ノ", Primary: "no", Accepted: []string{"no"}, Row: "n"},
	{ID: "ha", Hiragana: "は", Katakana: "ハ", Primary: "ha", Accepted: []string{"ha"}, Row: "h"},
	{ID: "hi", Hiragana: "ひ", Katakana: "ヒ", Primary: "hi", Accepted: []string{"hi"}, Row: "h"},
	{ID: "fu", Hiragana: "ふ", Katakana: "フ", Primary: "fu", Accepted: []string{"fu", "hu"}, Row: "h"},
	{ID: "he", Hiragana: "へ", Katakana: "ヘ", Primary: "he", Accepted: []string{"he"}, Row: "h"},
	{ID: "ho", Hiragana: "ほ", Katakana: "ホ", Primary: "ho", Accepted: []string{"ho"}, Row: "h"},
	{ID: "ma", Hiragana: "ま", Katakana: "マ", Primary: "ma", Accepted: []string{"ma"}, Row: "m"},
	{ID: "mi", Hiragana: "み", Katakana: "ミ", Primary: "mi", Accepted: []string{"mi"}, Row: "m"},
	{ID: "mu", Hiragana: "む", Katakana: "ム", Primary: "mu", Accepted: []string{"mu"}, Row: "m"},
	{ID: "me", Hiragana: "め", Katakana: "メ", Primary: "me", Accepted: []string{"me"}, Row: "m"},
	{ID: "mo", Hiragana: "も", Katakana: "モ", Primary: "mo", Accepted: []string{"mo"}, Row: "m"},
	{ID: "ya", Hiragana: "や", Katakana: "ヤ", Primary: "ya", Accepted: []string{"ya"}, Row: "y"},
	{ID: "yu", Hiragana: "ゆ", Katakana: "ユ", Primary: "yu", Accepted: []string{"yu"}, Row: "y"},
	{ID: "yo", Hiragana: "よ", Katakana: "ヨ", Primary: "yo", Accepted: []string{"yo"}, Row: "y"},
	{ID: "ra", Hiragana: "ら", Katakana: "ラ", Primary: "ra", Accepted: []string{"ra"}, Row: "r"},
	{ID: "ri", Hiragana: "り", Katakana: "リ", Primary: "ri", Accepted: []string{"ri"}, Row: "r"},
	{ID: "ru", Hiragana: "る", Katakana: "ル", Primary: "ru", Accepted: []string{"ru"}, Row: "r"},
	{ID: "re", Hiragana: "れ", Katakana: "レ", Primary: "re", Accepted: []string{"re"}, Row: "r"},
	{ID: "ro", Hiragana: "ろ", Katakana: "ロ", Primary: "ro", Accepted: []string{"ro"}, Row: "r"},
	{ID: "wa", Hiragana: "わ", Katakana: "ワ", Primary: "wa", Accepted: []string{"wa"}, Row: "w"},
	{ID: "wo", Hiragana: "を", Katakana: "ヲ", Primary: "wo", Accepted: []string{"wo"}, Row: "w"},
	{ID: "n", Hiragana: "ん", Katakana: "ン", Primary: "n", Accepted: []string{"n", "nn"}, Row: "nn"},
}

// ByID returns a kana by stable ID.
func ByID(id string) (Kana, bool) {
	for _, item := range Basic {
		if item.ID == id {
			return item, true
		}
	}
	return Kana{}, false
}

// Symbol returns the glyph for a mode. Mixed mode uses the supplied variant bit.
func (k Kana) Symbol(mode Mode, katakanaVariant bool) string {
	if mode == ModeKatakana || (mode == ModeMixed && katakanaVariant) {
		return k.Katakana
	}
	return k.Hiragana
}
