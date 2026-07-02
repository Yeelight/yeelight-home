package semantic

import (
	"math"
	"sort"
	"strings"
	"unicode"

	pinyin "github.com/mozillazg/go-pinyin"
)

type NameMatch struct {
	Matched bool
	Kind    string
	Score   float64
}

type RankedNameMatch[T any] struct {
	Value T
	Name  string
	Match NameMatch
	Index int
}

func ScoreNameMatch(query string, candidate string) NameMatch {
	normalizedQuery := normalizeNameTextBase(query)
	normalizedCandidate := normalizeNameTextBase(candidate)
	if normalizedQuery == "" || normalizedCandidate == "" {
		return NameMatch{}
	}
	if normalizedQuery == normalizedCandidate {
		return NameMatch{Matched: true, Kind: "name", Score: 1}
	}

	aliasedQuery := ApplyNameAliases(normalizedQuery)
	aliasedCandidate := ApplyNameAliases(normalizedCandidate)
	if aliasedQuery == aliasedCandidate {
		return NameMatch{Matched: true, Kind: "alias_name", Score: 0.97}
	}

	matchQuery := aliasedQuery
	matchCandidate := aliasedCandidate
	if len([]rune(matchQuery)) >= 2 && (strings.Contains(matchCandidate, matchQuery) || strings.Contains(matchQuery, matchCandidate)) {
		return NameMatch{Matched: true, Kind: "fuzzy_name", Score: 0.92}
	}
	tokenScore := tokenContainmentNameSimilarity(matchQuery, matchCandidate)
	if tokenScore >= 0.90 {
		return NameMatch{Matched: true, Kind: "token_name", Score: tokenScore}
	}
	phoneticScore := phoneticNameSimilarity(matchQuery, matchCandidate)
	if phoneticScore >= 0.95 {
		return NameMatch{Matched: true, Kind: "phonetic_name", Score: phoneticScore}
	}
	initialScore := phoneticInitialSimilarity(matchQuery, matchCandidate)
	if initialScore >= 0.93 {
		return NameMatch{Matched: true, Kind: "initial_name", Score: initialScore}
	}
	score := editSimilarity(matchQuery, matchCandidate)
	if phoneticScore > score && phoneticScore >= 0.88 {
		return NameMatch{Matched: true, Kind: "phonetic_name", Score: phoneticScore}
	}
	if initialScore > score && initialScore >= 0.90 {
		return NameMatch{Matched: true, Kind: "initial_name", Score: initialScore}
	}
	if score >= minNameCandidateScore(matchQuery, matchCandidate) {
		return NameMatch{Matched: true, Kind: "similar_name", Score: score}
	}
	if phoneticScore > score {
		score = phoneticScore
	}
	if initialScore > score {
		score = initialScore
	}
	return NameMatch{Score: score}
}

func NormalizeNameText(value string) string {
	return ApplyNameAliases(normalizeNameTextBase(value))
}

func NameKeywordMatches(query string, candidate string) bool {
	normalizedQuery := NormalizeNameText(query)
	normalizedCandidate := NormalizeNameText(candidate)
	if normalizedQuery == "" || normalizedCandidate == "" {
		return false
	}
	if strings.Contains(normalizedCandidate, normalizedQuery) || strings.Contains(normalizedQuery, normalizedCandidate) {
		return true
	}
	return ScoreNameMatch(normalizedQuery, normalizedCandidate).Matched
}

func normalizeNameTextBase(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		switch r {
		case '的', '_', '-':
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func RankNameMatches[T any](query string, candidates []T, nameOf func(T) string) []RankedNameMatch[T] {
	normalizedQuery := strings.TrimSpace(query)
	if normalizedQuery == "" {
		return nil
	}
	ranked := make([]RankedNameMatch[T], 0, len(candidates))
	for index, candidate := range candidates {
		name := strings.TrimSpace(nameOf(candidate))
		match := ScoreNameMatch(normalizedQuery, name)
		if match.Matched {
			ranked = append(ranked, RankedNameMatch[T]{
				Value: candidate,
				Name:  name,
				Match: match,
				Index: index,
			})
		}
	}
	sort.SliceStable(ranked, func(i int, j int) bool {
		if ranked[i].Match.Score != ranked[j].Match.Score {
			return ranked[i].Match.Score > ranked[j].Match.Score
		}
		if NameMatchKindRank(ranked[i].Match.Kind) != NameMatchKindRank(ranked[j].Match.Kind) {
			return NameMatchKindRank(ranked[i].Match.Kind) > NameMatchKindRank(ranked[j].Match.Kind)
		}
		return ranked[i].Index < ranked[j].Index
	})
	return ranked
}

func SuggestNameMatches[T any](query string, candidates []T, nameOf func(T) string, limit int) []RankedNameMatch[T] {
	normalizedQuery := normalizeNameTextBase(query)
	if normalizedQuery == "" || limit <= 0 {
		return nil
	}
	suggestions := make([]RankedNameMatch[T], 0, len(candidates))
	for index, candidate := range candidates {
		name := strings.TrimSpace(nameOf(candidate))
		match := ScoreNameMatch(normalizedQuery, name)
		if match.Matched || match.Score < minNameSuggestionScore(normalizedQuery, normalizeNameTextBase(name)) {
			continue
		}
		suggestions = append(suggestions, RankedNameMatch[T]{
			Value: candidate,
			Name:  name,
			Match: match,
			Index: index,
		})
	}
	sort.SliceStable(suggestions, func(i int, j int) bool {
		if suggestions[i].Match.Score != suggestions[j].Match.Score {
			return suggestions[i].Match.Score > suggestions[j].Match.Score
		}
		return suggestions[i].Index < suggestions[j].Index
	})
	if len(suggestions) > limit {
		return suggestions[:limit]
	}
	return suggestions
}

func NameMatchAutoAccept(top NameMatch, second NameMatch) bool {
	if !top.Matched {
		return false
	}
	switch top.Kind {
	case "name":
		return true
	case "alias_name":
		return top.Score >= 0.96 && top.Score-second.Score >= 0.08
	case "fuzzy_name":
		return top.Score-second.Score >= 0.06
	case "token_name":
		return top.Score >= 0.90 && top.Score-second.Score >= 0.08
	case "phonetic_name":
		return top.Score >= 0.95 && top.Score-second.Score >= 0.08
	case "initial_name":
		return top.Score >= 0.93 && top.Score-second.Score >= 0.12
	}
	if top.Score >= 0.86 {
		return top.Score-second.Score >= 0.12
	}
	return false
}

func NameMatchKindRank(kind string) int {
	switch kind {
	case "name":
		return 6
	case "alias_name":
		return 5
	case "phonetic_name":
		return 4
	case "fuzzy_name":
		return 3
	case "token_name":
		return 3
	case "initial_name":
		return 2
	case "similar_name":
		return 1
	default:
		return 0
	}
}

func minNameCandidateScore(query string, candidate string) float64 {
	maxLen := math.Max(float64(len([]rune(query))), float64(len([]rune(candidate))))
	switch {
	case maxLen <= 2:
		return 0.75
	case maxLen <= 4:
		return 0.75
	default:
		return 0.80
	}
}

func minNameSuggestionScore(query string, candidate string) float64 {
	maxLen := math.Max(float64(len([]rune(query))), float64(len([]rune(candidate))))
	switch {
	case maxLen <= 2:
		return 0.70
	case maxLen <= 4:
		return 0.62
	default:
		return 0.58
	}
}

func phoneticNameSimilarity(left string, right string) float64 {
	leftPhonetic, leftHasHan := normalizedPinyin(left)
	rightPhonetic, rightHasHan := normalizedPinyin(right)
	if !leftHasHan && !rightHasHan {
		return 0
	}
	if leftPhonetic == "" || rightPhonetic == "" {
		return 0
	}
	if leftPhonetic == rightPhonetic {
		return 0.96
	}
	if len([]rune(left)) >= 2 && (strings.Contains(rightPhonetic, leftPhonetic) || strings.Contains(leftPhonetic, rightPhonetic)) {
		return 0.94
	}
	return editSimilarity(leftPhonetic, rightPhonetic) * 0.95
}

func tokenContainmentNameSimilarity(query string, candidate string) float64 {
	queryTokens := significantNameTokens(query)
	if len(queryTokens) < 2 {
		return 0
	}
	candidateNormalized := normalizeNameTextBase(candidate)
	if candidateNormalized == "" {
		return 0
	}
	candidateTokens := significantNameTokens(candidate)
	matched := 0
	for _, token := range queryTokens {
		if nameTokenMatchesCandidate(token, candidateNormalized, candidateTokens) {
			matched++
		}
	}
	if matched != len(queryTokens) {
		return 0
	}
	coverage := float64(len([]rune(strings.Join(queryTokens, "")))) / float64(len([]rune(candidateNormalized)))
	if coverage >= 0.45 {
		return 0.91
	}
	return 0.90
}

func nameTokenMatchesCandidate(queryToken string, candidateNormalized string, candidateTokens []string) bool {
	if strings.Contains(candidateNormalized, queryToken) {
		return true
	}
	if len([]rune(queryToken)) < 2 {
		return false
	}
	for _, candidateToken := range candidateTokens {
		if strings.Contains(candidateToken, queryToken) || strings.Contains(queryToken, candidateToken) {
			return true
		}
		if phoneticNameSimilarity(queryToken, candidateToken) >= 0.90 {
			return true
		}
		if editSimilarity(queryToken, candidateToken) >= minNameCandidateScore(queryToken, candidateToken) {
			return true
		}
	}
	return false
}

func significantNameTokens(value string) []string {
	normalized := normalizeNameTextBase(value)
	tokens := make([]string, 0)
	var builder strings.Builder
	var currentKind string
	flush := func() {
		token := builder.String()
		builder.Reset()
		if len([]rune(token)) >= 2 {
			tokens = append(tokens, token)
		}
	}
	for _, r := range normalized {
		kind := ""
		switch {
		case unicode.Is(unicode.Han, r):
			kind = "han"
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			kind = "latin"
		default:
			if builder.Len() > 0 {
				flush()
			}
			currentKind = ""
			continue
		}
		if currentKind != "" && currentKind != kind && builder.Len() > 0 {
			flush()
		}
		currentKind = kind
		builder.WriteRune(unicode.ToLower(r))
	}
	if builder.Len() > 0 {
		flush()
	}
	return tokens
}

func phoneticInitialSimilarity(left string, right string) float64 {
	leftInitials := normalizedPinyinInitials(left)
	rightInitials := normalizedPinyinInitials(right)
	if leftInitials == "" || rightInitials == "" {
		return 0
	}
	if len([]rune(leftInitials)) < 2 || len([]rune(rightInitials)) < 2 {
		return 0
	}
	if leftInitials == rightInitials {
		return 0.93
	}
	if strings.Contains(rightInitials, leftInitials) || strings.Contains(leftInitials, rightInitials) {
		return 0.90
	}
	return editSimilarity(leftInitials, rightInitials) * 0.88
}

func normalizedPinyin(value string) (string, bool) {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	var builder strings.Builder
	hasChinese := false
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			parts := pinyin.SinglePinyin(r, args)
			if len(parts) > 0 && parts[0] != "" {
				builder.WriteString(parts[0])
				hasChinese = true
				continue
			}
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String(), hasChinese
}

func normalizedPinyinInitials(value string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	var builder strings.Builder
	hasChinese := false
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			parts := pinyin.SinglePinyin(r, args)
			if len(parts) > 0 && parts[0] != "" {
				builder.WriteRune([]rune(parts[0])[0])
				hasChinese = true
				continue
			}
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	if !hasChinese && len([]rune(builder.String())) < 2 {
		return ""
	}
	return builder.String()
}

func editSimilarity(left string, right string) float64 {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	maxLen := len(leftRunes)
	if len(rightRunes) > maxLen {
		maxLen = len(rightRunes)
	}
	if maxLen == 0 {
		return 0
	}
	distance := levenshteinDistance(leftRunes, rightRunes)
	score := 1 - float64(distance)/float64(maxLen)
	if score < 0 {
		return 0
	}
	return score
}

func levenshteinDistance(left []rune, right []rune) int {
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range left {
		current[0] = i + 1
		for j, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = minInt(
				current[j]+1,
				previous[j+1]+1,
				previous[j]+cost,
			)
		}
		previous, current = current, previous
	}
	return previous[len(right)]
}

func minInt(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}
