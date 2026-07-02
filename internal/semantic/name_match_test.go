package semantic

import "testing"

func TestScoreNameMatchExactAndNormalized(t *testing.T) {
	match := ScoreNameMatch("客 厅-的灯", "客厅灯")
	if !match.Matched || match.Kind != "name" || match.Score != 1 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("exact match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchPhoneticHomophone(t *testing.T) {
	match := ScoreNameMatch("客廷", "客厅")
	if !match.Matched || match.Kind != "phonetic_name" || match.Score < 0.95 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("phonetic match should auto accept when unique: %#v", match)
	}
}

func TestScoreNameMatchFullPinyinQuery(t *testing.T) {
	match := ScoreNameMatch("keting", "客厅")
	if !match.Matched || match.Kind != "phonetic_name" || match.Score < 0.95 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique full pinyin match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchAliasSynonym(t *testing.T) {
	match := ScoreNameMatch("人在感应器", "人在传感器")
	if !match.Matched || match.Kind != "alias_name" || match.Score < 0.96 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique alias match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchProductAlias(t *testing.T) {
	match := ScoreNameMatch("主卧爱思筒射灯组", "主卧S系列筒射灯组")
	if !match.Matched || match.Kind != "alias_name" || match.Score < 0.96 {
		t.Fatalf("match = %#v", match)
	}
}

func TestScoreNameMatchUnorderedMixedTokens(t *testing.T) {
	match := ScoreNameMatch("RGBW色彩灯", "light-色彩灯通用固件 - RGBW-264193-01")
	if !match.Matched || match.Kind != "token_name" || match.Score < 0.90 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique token match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchMixedTokensWithChineseTypo(t *testing.T) {
	match := ScoreNameMatch("RGBW 色采灯", "light-色彩灯通用固件 - RGBW-264193-01")
	if !match.Matched || match.Kind != "token_name" || match.Score < 0.90 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique mixed-token typo match should auto accept: %#v", match)
	}
}

func TestNameKeywordMatchesShortGenericWord(t *testing.T) {
	if !NameKeywordMatches("灯", "light-色彩灯通用固件 - RGBW-264193-01") {
		t.Fatal("short list keyword should match contained device name text")
	}
}

func TestScoreNameMatchGatewayAliasMixedTokens(t *testing.T) {
	match := ScoreNameMatch("6.9寸智慧屏网关", "gateway-6.9寸智慧屏-17000007-01")
	if !match.Matched || match.Kind != "token_name" || match.Score < 0.90 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique gateway token match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchPinyinInitials(t *testing.T) {
	match := ScoreNameMatch("kt", "客厅")
	if !match.Matched || match.Kind != "initial_name" || match.Score < 0.93 {
		t.Fatalf("match = %#v", match)
	}
	if !NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("unique initial match should auto accept: %#v", match)
	}
}

func TestScoreNameMatchShortInitialDoesNotMatch(t *testing.T) {
	match := ScoreNameMatch("k", "客厅")
	if match.Matched {
		t.Fatalf("match = %#v", match)
	}
}

func TestScoreNameMatchLatinOnlyDoesNotUsePhonetic(t *testing.T) {
	match := ScoreNameMatch("rgb", "rgbw")
	if match.Matched && match.Kind == "phonetic_name" {
		t.Fatalf("latin-only terms must not be promoted by pinyin scoring: %#v", match)
	}
}

func TestScoreNameMatchPhoneticSubstring(t *testing.T) {
	match := ScoreNameMatch("离佳测试", "全覆盖离家测试")
	if !match.Matched || match.Kind != "phonetic_name" || match.Score < 0.90 {
		t.Fatalf("match = %#v", match)
	}
	if NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("phonetic substring should be a candidate but not auto accepted for writes: %#v", match)
	}
}

func TestScoreNameMatchWeakShortEditDoesNotMatch(t *testing.T) {
	match := ScoreNameMatch("客房", "客厅")
	if match.Matched {
		t.Fatalf("match = %#v", match)
	}
	if NameMatchAutoAccept(match, NameMatch{}) {
		t.Fatalf("weak short edit should not auto accept: %#v", match)
	}
}

func TestNameMatchAutoAcceptRequiresSeparation(t *testing.T) {
	top := NameMatch{Matched: true, Kind: "phonetic_name", Score: 0.96}
	second := NameMatch{Matched: true, Kind: "phonetic_name", Score: 0.94}
	if NameMatchAutoAccept(top, second) {
		t.Fatalf("close phonetic candidates should require clarification")
	}
}

func TestFullPinyinMatchRequiresSeparation(t *testing.T) {
	candidates := []string{"客厅", "客庭"}
	ranked := RankNameMatches("keting", candidates, func(value string) string { return value })
	if len(ranked) != 2 {
		t.Fatalf("ranked = %#v", ranked)
	}
	if NameMatchAutoAccept(ranked[0].Match, ranked[1].Match) {
		t.Fatalf("same-pinyin candidates should require clarification: %#v", ranked)
	}
}

func TestNameMatchAutoAcceptRequiresAliasSeparation(t *testing.T) {
	top := NameMatch{Matched: true, Kind: "alias_name", Score: 0.97}
	second := NameMatch{Matched: true, Kind: "alias_name", Score: 0.97}
	if NameMatchAutoAccept(top, second) {
		t.Fatalf("close alias candidates should require clarification")
	}
}

func TestNameMatchAutoAcceptRequiresInitialSeparation(t *testing.T) {
	top := NameMatch{Matched: true, Kind: "initial_name", Score: 0.93}
	second := NameMatch{Matched: true, Kind: "initial_name", Score: 0.90}
	if NameMatchAutoAccept(top, second) {
		t.Fatalf("close initial candidates should require clarification")
	}
}

func TestRankNameMatchesSortsByConfidence(t *testing.T) {
	candidates := []string{"客厅", "客房", "卧室"}
	ranked := RankNameMatches("客廷", candidates, func(value string) string { return value })
	if len(ranked) != 1 {
		t.Fatalf("ranked = %#v", ranked)
	}
	if ranked[0].Value != "客厅" || ranked[0].Match.Kind != "phonetic_name" {
		t.Fatalf("ranked = %#v", ranked)
	}
	if !NameMatchAutoAccept(ranked[0].Match, NameMatch{}) {
		t.Fatalf("top candidate should be confidently accepted: %#v", ranked)
	}
}

func TestSuggestNameMatchesReturnsLowConfidenceCandidateWithoutAutoMatch(t *testing.T) {
	candidates := []string{"格栅灯", "筒灯", "射灯"}
	ranked := RankNameMatches("格栏灯", candidates, func(value string) string { return value })
	if len(ranked) != 0 {
		t.Fatalf("low confidence typo should not auto-match: %#v", ranked)
	}

	suggestions := SuggestNameMatches("格栏灯", candidates, func(value string) string { return value }, 3)
	if len(suggestions) != 1 || suggestions[0].Value != "格栅灯" {
		t.Fatalf("suggestions = %#v", suggestions)
	}
	if suggestions[0].Match.Matched {
		t.Fatalf("suggestion must not be treated as matched: %#v", suggestions[0].Match)
	}
}
