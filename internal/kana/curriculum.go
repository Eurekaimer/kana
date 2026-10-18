package kana

import (
	"fmt"
	"strings"
)

// Content provenance. The drills are original; levels are editorial estimates
// because JLPT does not publish official vocabulary lists.
const (
	SourceVocabulary = "原创练习；等级为编辑估计，非官方真题"
	SourceAnime      = "原创动漫风例句，非作品台词"

	JLPTFAQURL        = "https://www.jlpt.jp/e/faq/index.html"
	JLPTLevelsURL     = "https://www.jlpt.jp/e/about/levelsummary.html"
	JLPTGuidelinesURL = "https://www.jlpt.jp/e/policy.html"
)

var (
	gradedLevels = []string{"n5", "n4", "n3", "n2", "n1"}

	// Levels is the supported level order for menus.
	Levels = append([]string{"beginner", "gojuon"}, gradedLevels...)
	// Exercises is the supported exercise order for menus.
	Exercises = []string{"reading", "anime"}
)

// Deck returns an independent copy of the cards for one level and exercise.
// The two gojuon levels share Basic; anime drills begin at N5.
func Deck(level, exercise string) ([]Kana, error) {
	level = strings.ToLower(strings.TrimSpace(level))
	exercise = strings.ToLower(strings.TrimSpace(exercise))
	if exercise != "reading" && exercise != "anime" {
		return nil, fmt.Errorf("kana: unknown exercise %q", exercise)
	}

	switch level {
	case "beginner", "gojuon":
		if exercise == "anime" {
			return nil, fmt.Errorf("kana: %s level has no anime drills; choose n5 or above", level)
		}
		return cloneCards(Basic), nil
	}

	words, ok := wordDecks[level]
	if !ok {
		return nil, fmt.Errorf("kana: unknown level %q", level)
	}
	if exercise == "anime" {
		return cloneCards(animeDecks[level]), nil
	}
	return cloneCards(words), nil
}

// All returns every card for progress initialization and statistics lookups.
func All() []Kana {
	return cloneCards(allCards)
}

// ByID returns any card by stable ID. The index is built once, so lookups are
// allocation-free.
func ByID(id string) (Kana, bool) {
	item, ok := cardIndex[id]
	return item, ok
}

var (
	allCards  = collectAll()
	cardIndex = indexCards(allCards)
)

func collectAll() []Kana {
	cards := make([]Kana, 0, len(Basic)+(len(gradedLevels)*16))
	cards = append(cards, Basic...)
	for _, level := range gradedLevels {
		cards = append(cards, wordDecks[level]...)
		cards = append(cards, animeDecks[level]...)
	}
	return cards
}

func indexCards(cards []Kana) map[string]Kana {
	index := make(map[string]Kana, len(cards))
	for _, card := range cards {
		if _, exists := index[card.ID]; exists {
			panic("kana: duplicate card ID " + card.ID)
		}
		index[card.ID] = card
	}
	return index
}

func cloneCards(cards []Kana) []Kana {
	cloned := make([]Kana, len(cards))
	copy(cloned, cards)
	for index := range cloned {
		if cloned[index].Accepted != nil {
			cloned[index].Accepted = append([]string(nil), cloned[index].Accepted...)
		}
	}
	return cloned
}

func wordCard(level, slug, hiragana, katakana, primary string, accepted []string, meaning, sentence, translation string, nativeKatakana bool) Kana {
	return Kana{
		ID:             level + ".word." + slug,
		Hiragana:       hiragana,
		Katakana:       katakana,
		Primary:        primary,
		Accepted:       accepted,
		Row:            level,
		Meaning:        meaning,
		Context:        sentence,
		ContextMeaning: translation,
		Source:         SourceVocabulary,
		NativeKatakana: nativeKatakana,
	}
}

func animeCard(level, slug, hiragana, katakana, primary string, accepted []string, meaning, sentence, translation string, nativeKatakana bool) Kana {
	return Kana{
		ID:             level + ".context." + slug,
		Hiragana:       hiragana,
		Katakana:       katakana,
		Primary:        primary,
		Accepted:       accepted,
		Row:            level + "-anime",
		Meaning:        meaning,
		Context:        sentence,
		ContextMeaning: translation,
		Source:         SourceAnime,
		NativeKatakana: nativeKatakana,
	}
}

var (
	wordDecks = map[string][]Kana{
		"n5": n5Words,
		"n4": n4Words,
		"n3": n3Words,
		"n2": n2Words,
		"n1": n1Words,
	}
	animeDecks = map[string][]Kana{
		"n5": n5Anime,
		"n4": n4Anime,
		"n3": n3Anime,
		"n2": n2Anime,
		"n1": n1Anime,
	}
)

// N5 vocabulary: everyday actions and loanwords, split six hiragana and six
// natively katakana words.
var n5Words = []Kana{
	wordCard("n5", "taberu", "たべる", "タベル", "taberu", []string{"taberu"}, "吃",
		"「朝ごはんは、いつもここで【たべる】んだ」", "“早饭我总是在这里吃。”", false),
	wordCard("n5", "nomu", "のむ", "ノム", "nomu", []string{"nomu"}, "喝",
		"「のどが渇いたね。何か【のむ】？」", "“口渴了呢。要喝点什么吗？”", false),
	wordCard("n5", "gakkou", "がっこう", "ガッコウ", "gakkou", []string{"gakkou", "gakkoo"}, "学校",
		"「明日は早く起きて【がっこう】へ行くよ」", "“明天要早起去学校哦。”", false),
	wordCard("n5", "konya", "こんや", "コンヤ", "kon'ya", []string{"kon'ya", "konya"}, "今晚",
		"「【こんや】は星がよく見えるね」", "“今晚星星看得真清楚呢。”", false),
	wordCard("n5", "sensei", "せんせい", "センセイ", "sensei", []string{"sensei", "sensee"}, "老师",
		"「【せんせい】に聞けば、きっと分かるよ」", "“去问老师的话，一定会明白的。”", false),
	wordCard("n5", "ookii", "おおきい", "オオキイ", "ookii", []string{"ookii"}, "大的",
		"「この箱は思ったより【おおきい】な」", "“这个箱子比想象中大啊。”", false),

	wordCard("n5", "kamera", "かめら", "カメラ", "kamera", []string{"kamera"}, "相机",
		"「この【カメラ】で、あの景色を撮りたい」", "“想用这台相机拍下那片风景。”", true),
	wordCard("n5", "terebi", "てれび", "テレビ", "terebi", []string{"terebi"}, "电视",
		"「【テレビ】の音が大きすぎない？」", "“电视的声音是不是太大了？”", true),
	wordCard("n5", "pan", "ぱん", "パン", "pan", []string{"pan"}, "面包",
		"「朝は【パン】とコーヒーでいい」", "“早上有面包和咖啡就够了。”", true),
	wordCard("n5", "koohii", "こおひい", "コーヒー", "koohii", []string{"koohii"}, "咖啡",
		"「熱い【コーヒー】を一杯もらえる？」", "“能给我一杯热咖啡吗？”", true),
	wordCard("n5", "basu", "ばす", "バス", "basu", []string{"basu"}, "公交车",
		"「【バス】が来るまで、あと五分だ」", "“距离公交来还有五分钟。”", true),
	wordCard("n5", "hoteru", "ほてる", "ホテル", "hoteru", []string{"hoteru"}, "酒店",
		"「駅の前の【ホテル】に泊まることにした」", "“决定住在车站前的酒店。”", true),
}

// N4 vocabulary: familiar daily situations with more complex kana.
var n4Words = []Kana{
	wordCard("n4", "okureru", "おくれる", "オクレル", "okureru", []string{"okureru"}, "迟到；赶不上",
		"「寝坊して、電車に【おくれる】ところだった」", "“睡过头，差点赶不上电车。”", false),
	wordCard("n4", "tanomu", "たのむ", "タノム", "tanomu", []string{"tanomu"}, "拜托；请求",
		"「無理を【たのむ】つもりはない」", "“不打算勉强拜托你。”", false),
	wordCard("n4", "undou", "うんどう", "ウンドウ", "undou", []string{"undou", "undoo"}, "运动",
		"「毎朝少しだけ【うんどう】している」", "“每天早上都会稍微运动一下。”", false),
	wordCard("n4", "shinpai", "しんぱい", "シンパイ", "shinpai", []string{"shinpai", "sinpai"}, "担心",
		"「何も【しんぱい】しなくていいよ」", "“什么都不用担心哦。”", false),
	wordCard("n4", "hakkiri", "はっきり", "ハッキリ", "hakkiri", []string{"hakkiri"}, "清楚地；明确地",
		"「好きかどうか、【はっきり】言ってほしい」", "“希望你清楚地说出是否喜欢。”", false),
	wordCard("n4", "kyoumi", "きょうみ", "キョウミ", "kyoumi", []string{"kyoumi", "kyoomi"}, "兴趣",
		"「その話には少し【きょうみ】がある」", "“我对那件事有点兴趣。”", false),

	wordCard("n4", "arubaito", "あるばいと", "アルバイト", "arubaito", []string{"arubaito"}, "打工",
		"「放課後はカフェで【アルバイト】している」", "“放学后在咖啡馆打工。”", true),
	wordCard("n4", "apaato", "あぱあと", "アパート", "apaato", []string{"apaato"}, "公寓",
		"「大学の近くに【アパート】を借りた」", "“在大学附近租了公寓。”", true),
	wordCard("n4", "pasokon", "ぱそこん", "パソコン", "pasokon", []string{"pasokon"}, "电脑",
		"「新しい【パソコン】の設定が終わった」", "“新电脑的设置完成了。”", true),
	wordCard("n4", "suupaa", "すうぱあ", "スーパー", "suupaa", []string{"suupaa"}, "超市",
		"「帰りに【スーパー】で野菜を買う」", "“回家路上在超市买蔬菜。”", true),
	wordCard("n4", "resutoran", "れすとらん", "レストラン", "resutoran", []string{"resutoran"}, "餐厅",
		"「記念日に【レストラン】を予約した」", "“纪念日预订了餐厅。”", true),
	wordCard("n4", "nyuusu", "にゅうす", "ニュース", "nyuusu", []string{"nyuusu"}, "新闻",
		"「朝の【ニュース】でその事件を知った」", "“早上看新闻知道了那起事件。”", true),
}

// N3 vocabulary: bridging everyday and abstract topics.
var n3Words = []Kana{
	wordCard("n3", "kikkake", "きっかけ", "キッカケ", "kikkake", []string{"kikkake"}, "契机",
		"「あの失敗が、学び直す【きっかけ】になった」", "“那次失败成了重新学习的契机。”", false),
	wordCard("n3", "shaberu", "しゃべる", "シャベル", "shaberu", []string{"shaberu"}, "聊天；说话",
		"「彼はいつも軽い調子で【しゃべる】」", "“他总是用轻快的语气聊天。”", false),
	wordCard("n3", "tsutaeru", "つたえる", "ツタエル", "tsutaeru", []string{"tsutaeru", "tutaeru"}, "传达",
		"「この気持ちを、ちゃんと【つたえる】つもりだ」", "“打算好好传达这份心情。”", false),
	wordCard("n3", "keiken", "けいけん", "ケイケン", "keiken", []string{"keiken", "keeken"}, "经验",
		"「一度でも【けいけん】したことは忘れない」", "“哪怕经历过一次的事就不会忘。”", false),
	wordCard("n3", "anshin", "あんしん", "アンシン", "anshin", []string{"anshin", "ansin"}, "安心",
		"「君が無事だと聞いて【あんしん】した」", "“听说你平安无事，我就放心了。”", false),
	wordCard("n3", "mayou", "まよう", "マヨウ", "mayou", []string{"mayou", "mayoo"}, "犹豫；迷路",
		"「どちらを選ぶか、少し【まよう】」", "“选哪一个，有点犹豫。”", false),

	wordCard("n3", "imeeji", "いめえじ", "イメージ", "imeeji", []string{"imeeji"}, "印象",
		"「この色は、少し冷たい【イメージ】がある」", "“这个颜色给人一种有点冷的印象。”", true),
	wordCard("n3", "guruupu", "ぐるうぷ", "グループ", "guruupu", []string{"guruupu"}, "小组",
		"「三人ずつ【グループ】に分かれてください」", "“请每三人分成一组。”", true),
	wordCard("n3", "chansu", "ちゃんす", "チャンス", "chansu", []string{"chansu"}, "机会",
		"「この【チャンス】を逃したくない」", "“不想错过这个机会。”", true),
	wordCard("n3", "sutoresu", "すとれす", "ストレス", "sutoresu", []string{"sutoresu"}, "压力",
		"「【ストレス】がたまると、歌いたくなる」", "“压力积攒起来时，就想唱歌。”", true),
	wordCard("n3", "borantia", "ぼらんてぃあ", "ボランティア", "borantia", []string{"borantia"}, "志愿者",
		"「週末は公園で【ボランティア】をしている」", "“周末在公园做志愿者。”", true),
	wordCard("n3", "intabyuu", "いんたびゅう", "インタビュー", "intabyuu", []string{"intabyuu"}, "采访",
		"「駅前で通行人に【インタビュー】をした」", "“在车站前采访了路人。”", true),
}

// N2 vocabulary: practical words for everyday and workplace contexts.
var n2Words = []Kana{
	wordCard("n2", "tamerau", "ためらう", "タメラウ", "tamerau", []string{"tamerau"}, "犹豫",
		"「返事を【ためらう】理由が分からない」", "“不明白你犹豫着不回答的理由。”", false),
	wordCard("n2", "magirawashii", "まぎらわしい", "マギラワシイ", "magirawashii", []string{"magirawashii", "magirawasii"}, "容易混淆",
		"「名前が似ていて【まぎらわしい】」", "“名字很像，容易混淆。”", false),
	wordCard("n2", "unagasu", "うながす", "ウナガス", "unagasu", []string{"unagasu"}, "促使",
		"「静かな口調で、彼は発言を【うながす】」", "“他用平静的语气促使大家发言。”", false),
	wordCard("n2", "oginau", "おぎなう", "オギナウ", "oginau", []string{"oginau"}, "弥补",
		"「経験の不足を努力で【おぎなう】」", "“用努力弥补经验的不足。”", false),
	wordCard("n2", "kuwawaru", "くわわる", "クワワル", "kuwawaru", []string{"kuwawaru"}, "加入",
		"「来月から新しい仲間が【くわわる】」", "“下个月会有新伙伴加入。”", false),
	wordCard("n2", "nerai", "ねらい", "ネライ", "nerai", []string{"nerai"}, "目的；意图",
		"「この質問の【ねらい】は何だろう」", "“这个问题的用意是什么呢？”", false),

	wordCard("n2", "konsensasu", "こんせんさす", "コンセンサス", "konsensasu", []string{"konsensasu"}, "共识",
		"「まずチーム内で【コンセンサス】を作ろう」", "“先在小队里形成共识吧。”", true),
	wordCard("n2", "meritto", "めりっと", "メリット", "meritto", []string{"meritto"}, "优点",
		"「この方法の【メリット】を説明してほしい」", "“希望说明这个方法的优点。”", true),
	wordCard("n2", "demeritto", "でめりっと", "デメリット", "demeritto", []string{"demeritto"}, "缺点",
		"「便利だが、【デメリット】もある」", "“虽然方便，但也有缺点。”", true),
	wordCard("n2", "purosesu", "ぷろせす", "プロセス", "purosesu", []string{"purosesu"}, "过程",
		"「結果より【プロセス】が大事だ」", "“过程比结果更重要。”", true),
	wordCard("n2", "kea", "けあ", "ケア", "kea", []string{"kea"}, "护理；照顾",
		"「入院中の母の【ケア】をしている」", "“在照顾住院的母亲。”", true),
	wordCard("n2", "niizu", "にいず", "ニーズ", "niizu", []string{"niizu"}, "需求",
		"「利用者の【ニーズ】を先に調べる」", "“先调查使用者的需求。”", true),
}

// N1 vocabulary: abstract and lower-frequency words, still spelled in kana.
var n1Words = []Kana{
	wordCard("n1", "aimai", "あいまい", "アイマイ", "aimai", []string{"aimai"}, "模糊；含糊",
		"「【あいまい】な態度では、誰も納得しない」", "“态度含糊的话，谁都不会信服。”", false),
	wordCard("n1", "ikidoori", "いきどおり", "イキドオリ", "ikidoori", []string{"ikidoori"}, "愤怒",
		"「その知らせに、静かな【いきどおり】を覚えた」", "“听到那个消息，我感到了无声的愤怒。”", false),
	wordCard("n1", "orosoka", "おろそか", "オロソカ", "orosoka", []string{"orosoka"}, "疏忽；马虎",
		"「準備を【おろそか】にしてはいけない」", "“不能疏忽准备。”", false),
	wordCard("n1", "kokorogakeru", "こころがける", "ココロガケル", "kokorogakeru", []string{"kokorogakeru"}, "留心；顾及",
		"「毎日、相手の話を聞くことを【こころがける】」", "“每天都留心倾听对方说话。”", false),
	wordCard("n1", "samatageru", "さまたげる", "サマタゲル", "samatageru", []string{"samatageru"}, "妨碍",
		"「過去が君の未来を【さまたげる】ことはない」", "“过去不会妨碍你的未来。”", false),
	wordCard("n1", "tsunezune", "つねづね", "ツネズネ", "tsunezune", []string{"tsunezune"}, "平时；经常",
		"「【つねづね】思っていたことを、今伝えたい」", "“平时一直想着的事，现在想告诉你。”", false),

	wordCard("n1", "ebidensu", "えびでんす", "エビデンス", "ebidensu", []string{"ebidensu"}, "证据",
		"「推測ではなく、【エビデンス】を示してほしい」", "“希望拿出证据，而不是推测。”", true),
	wordCard("n1", "konseputo", "こんせぷと", "コンセプト", "konseputo", []string{"konseputo"}, "概念；理念",
		"「店の【コンセプト】を一言で説明する」", "“用一句话说明店铺的理念。”", true),
	wordCard("n1", "jirenma", "じれんま", "ジレンマ", "jirenma", []string{"jirenma", "zirenma"}, "两难",
		"「正しさと優しさの間で【ジレンマ】を抱える」", "“在正确与温柔之间陷入两难。”", true),
	wordCard("n1", "nouhau", "のうはう", "ノウハウ", "nouhau", []string{"nouhau", "noohau"}, "诀窍；技术",
		"「長年積み重ねた【ノウハウ】がある」", "“有长年积累的诀窍。”", true),
	wordCard("n1", "puresshaa", "ぷれっしゃあ", "プレッシャー", "puresshaa", []string{"puresshaa"}, "压力",
		"「大事な場面ほど【プレッシャー】を感じる」", "“越是重要的场合越感到压力。”", true),
	wordCard("n1", "apuroochi", "あぷろおち", "アプローチ", "apuroochi", []string{"apuroochi"}, "方法；接近方式",
		"「別の【アプローチ】から考えてみよう」", "“换个方法思考看看吧。”", true),
}

// Anime-style context drills. Each sentence is original and marks only the
// target whose reading is asked; the Chinese translation never contains romaji.
var n5Anime = []Kana{
	animeCard("n5", "matte", "まって", "マッテ", "matte", []string{"matte"}, "等着",
		"「ここで【まって】いて。すぐ戻るから」", "“在这里等着。我马上回来。”", false),
	animeCard("n5", "tomodachi", "ともだち", "トモダチ", "tomodachi", []string{"tomodachi"}, "朋友",
		"「【ともだち】を信じるって、そういうことだろ？」", "“相信朋友，就是这么回事吧？”", false),
	animeCard("n5", "ganbare", "がんばれ", "ガンバレ", "ganbare", []string{"ganbare"}, "加油",
		"「あと一歩だ。【がんばれ】！」", "“就差一步了。加油！”", false),
	animeCard("n5", "arigatou", "ありがとう", "アリガトウ", "arigatou", []string{"arigatou", "arigatoo"}, "谢谢",
		"「こんな夜に来てくれて、【ありがとう】」", "“这么晚还过来，谢谢你。”", false),
}

var n4Anime = []Kana{
	animeCard("n4", "shinpai", "しんぱい", "シンパイ", "shinpai", []string{"shinpai", "sinpai"}, "担心",
		"「【しんぱい】しないで。ちゃんと連絡するから」", "“别担心。我会好好联系的。”", false),
	animeCard("n4", "maniau", "まにあう", "マニアウ", "maniau", []string{"maniau"}, "赶得上",
		"「走ればまだ【まにあう】かもしれない」", "“跑过去的话或许还赶得上。”", false),
	animeCard("n4", "yakusoku", "やくそく", "ヤクソク", "yakusoku", []string{"yakusoku"}, "约定",
		"「【やくそく】は守る。それが俺のルールだ」", "“约定就要遵守。这是我的原则。”", false),
	animeCard("n4", "chansu", "ちゃんす", "チャンス", "chansu", []string{"chansu"}, "机会",
		"「今が【チャンス】だ。行け！」", "“现在就是机会。上啊！”", true),
}

var n3Anime = []Kana{
	animeCard("n3", "kikkake", "きっかけ", "キッカケ", "kikkake", []string{"kikkake"}, "契机",
		"「あの出会いが、すべての【きっかけ】だった」", "“那次相遇，是一切的开端。”", false),
	animeCard("n3", "mayotta", "まよった", "マヨッタ", "mayotta", []string{"mayotta"}, "犹豫了",
		"「どっちを選ぶか、少し【まよった】」", "“选哪边，我犹豫了一下。”", false),
	animeCard("n3", "sutoresu", "すとれす", "ストレス", "sutoresu", []string{"sutoresu"}, "压力",
		"「【ストレス】を溜め込む前に、話してくれよ」", "“在压力积攒起来之前，跟我说说吧。”", true),
	animeCard("n3", "himitsu", "ひみつ", "ヒミツ", "himitsu", []string{"himitsu"}, "秘密",
		"「これは、私たちだけの【ひみつ】だからね」", "“这是只属于我们的秘密哦。”", false),
}

var n2Anime = []Kana{
	animeCard("n2", "tameratta", "ためらった", "タメラッタ", "tameratta", []string{"tameratta"}, "犹豫了",
		"「一瞬【ためらった】けど、もう決めた」", "“虽然犹豫了一瞬，但已经决定了。”", false),
	animeCard("n2", "konsensasu", "こんせんさす", "コンセンサス", "konsensasu", []string{"konsensasu"}, "共识",
		"「まずはチームの【コンセンサス】を取ろう」", "“先取得小队的共识吧。”", true),
	animeCard("n2", "unagashita", "うながした", "ウナガシタ", "unagashita", []string{"unagashita"}, "促使了",
		"「彼の言葉が、私を前に進むよう【うながした】」", "“他的话促使我向前迈进。”", false),
	animeCard("n2", "magirawashii", "まぎらわしい", "マギラワシイ", "magirawashii", []string{"magirawashii", "magirawasii"}, "容易混淆",
		"「似た名前で【まぎらわしい】から、気をつけて」", "“名字相似容易混淆，小心点。”", false),
}

var n1Anime = []Kana{
	animeCard("n1", "aimai", "あいまい", "アイマイ", "aimai", []string{"aimai"}, "模糊；含糊",
		"「【あいまい】な返事は、もう聞きたくない」", "“含糊的答复，我不想再听了。”", false),
	animeCard("n1", "jirenma", "じれんま", "ジレンマ", "jirenma", []string{"jirenma", "zirenma"}, "两难",
		"「守るべきか、進むべきか。【ジレンマ】だな」", "“该守护还是该前进。真是两难啊。”", true),
	animeCard("n1", "samatageru", "さまたげる", "サマタゲル", "samatageru", []string{"samatageru"}, "妨碍",
		"「過去が君の未来を【さまたげる】ことはない」", "“过去不会妨碍你的未来。”", false),
	animeCard("n1", "kokorogakete", "こころがけて", "ココロガケテ", "kokorogakete", []string{"kokorogakete"}, "留心着",
		"「相手の話を聞くことを【こころがけて】いる」", "“我一直留心倾听对方说话。”", false),
}
