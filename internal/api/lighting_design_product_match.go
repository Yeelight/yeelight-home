package api

import (
	"regexp"
	"sort"
	"strings"
)

const (
	lightingDesignProductMatchHighThreshold = 80
	lightingDesignProductMatchCandidateMax  = 3
)

type lightingDesignProductMatch struct {
	Entry      lightingDesignProductEntry
	Score      int
	Confidence string
	Matched    []string
}

var lightingDesignSpacesRe = regexp.MustCompile(`\s+`)

func lightingDesignResolveSlotProduct(slot map[string]any, baseName string) (lightingDesignProductMatch, []lightingDesignProductMatch) {
	if explicit := lightingDesignExplicitProduct(slot); explicit.Entry.MaterialCode != "" || explicit.Entry.PID > 0 {
		explicit.Score = 100
		explicit.Confidence = "explicit"
		explicit.Matched = []string{"explicit_product_fields"}
		return explicit, nil
	}
	candidates := lightingDesignProductCandidates(slot, baseName)
	if len(candidates) == 0 {
		return lightingDesignProductMatch{}, nil
	}
	best := candidates[0]
	if lightingDesignCanAutoResolveProduct(candidates) {
		return best, candidates
	}
	return lightingDesignProductMatch{}, candidates
}

func lightingDesignCanAutoResolveProduct(candidates []lightingDesignProductMatch) bool {
	if len(candidates) == 0 || candidates[0].Score < lightingDesignProductMatchHighThreshold {
		return false
	}
	if len(candidates) == 1 {
		return true
	}
	return candidates[0].Score-candidates[1].Score >= 25
}

func lightingDesignHasProductIdentity(match lightingDesignProductMatch) bool {
	return match.Entry.MaterialCode != "" || match.Entry.PID > 0
}

func lightingDesignExplicitProduct(slot map[string]any) lightingDesignProductMatch {
	entry := lightingDesignProductEntry{
		MaterialCode: firstNonEmpty(
			stringFromMap(slot, "materialCode"),
			stringFromMap(slot, "skuMaterialCode"),
		),
		PID:         int64FromMap(slot, "pid", 0),
		PCID:        int64FromMap(slot, "pcId", 0),
		ConnectType: intFromMap(slot, "connectType", -1),
		ProductName: firstNonEmpty(stringFromMap(slot, "productName"), stringFromMap(slot, "name")),
		ProductSKU:  firstNonEmpty(stringFromMap(slot, "productSku"), stringFromMap(slot, "sku")),
		ProductSPU:  firstNonEmpty(stringFromMap(slot, "productSpu"), stringFromMap(slot, "spu")),
		Category:    stringFromMap(slot, "category"),
		Series:      stringFromMap(slot, "series"),
	}
	if entry.MaterialCode != "" {
		for _, item := range lightingDesignProductCatalog {
			if strings.EqualFold(item.MaterialCode, entry.MaterialCode) {
				entry = item
				break
			}
		}
	}
	return lightingDesignProductMatch{Entry: entry}
}

func lightingDesignProductCandidates(slot map[string]any, baseName string) []lightingDesignProductMatch {
	queryText := lightingDesignNormalizeProductText(strings.Join(lightingDesignSlotProductWords(slot, baseName), " "))
	if queryText == "" {
		return nil
	}
	results := make([]lightingDesignProductMatch, 0, len(lightingDesignProductCatalog))
	for _, entry := range lightingDesignProductCatalog {
		score, matched := lightingDesignScoreProduct(entry, queryText)
		if score <= 0 {
			continue
		}
		confidence := "low"
		if score >= lightingDesignProductMatchHighThreshold {
			confidence = "high"
		} else if score >= 55 {
			confidence = "medium"
		}
		results = append(results, lightingDesignProductMatch{
			Entry:      entry,
			Score:      score,
			Confidence: confidence,
			Matched:    matched,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].Entry.Priority != results[j].Entry.Priority {
				return results[i].Entry.Priority > results[j].Entry.Priority
			}
			return results[i].Entry.MaterialCode < results[j].Entry.MaterialCode
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > lightingDesignProductMatchCandidateMax {
		return results[:lightingDesignProductMatchCandidateMax]
	}
	return results
}

func lightingDesignSlotProductWords(slot map[string]any, baseName string) []string {
	words := []string{baseName}
	for _, key := range []string{
		"name",
		"type",
		"category",
		"color",
		"installStyle",
		"beamAngle",
		"series",
		"productName",
		"productSku",
		"productSpu",
		"productShortName",
		"model",
		"modelNo",
		"notes",
		"description",
	} {
		if value := stringFromMap(slot, key); value != "" {
			words = append(words, value)
		}
	}
	return words
}

func lightingDesignScoreProduct(entry lightingDesignProductEntry, queryText string) (int, []string) {
	score := 0
	matched := []string{}
	for _, alias := range entry.Aliases {
		if lightingDesignContainsToken(queryText, alias) {
			score += 60
			matched = append(matched, alias)
			break
		}
	}
	for _, pair := range []struct {
		text  string
		score int
	}{
		{entry.ProductName, 35},
		{entry.ProductSKU, 28},
		{entry.ProductSPU, 24},
		{entry.Category, 30},
		{entry.Series, 20},
		{entry.MaterialCode, 100},
	} {
		if pair.text == "" {
			continue
		}
		if lightingDesignContainsToken(queryText, pair.text) {
			score += pair.score
			matched = append(matched, pair.text)
		}
	}
	if lightingDesignProductCategoryCompatible(queryText, entry) && !lightingDesignContainsToken(queryText, entry.Category) {
		score += 18
		matched = append(matched, "compatible:"+entry.Category)
	}
	for _, token := range lightingDesignDisambiguationTokens(entry) {
		if lightingDesignContainsToken(queryText, token.word) {
			score += token.score
			matched = append(matched, token.word)
		}
	}
	if len(matched) == 0 {
		return 0, nil
	}
	score -= lightingDesignProductConstraintPenalty(entry, queryText)
	if score <= 0 {
		return 0, nil
	}
	return score, uniqueStrings(matched)
}

func lightingDesignProductConstraintPenalty(entry lightingDesignProductEntry, queryText string) int {
	entryText := lightingDesignProductEntryText(entry)
	penalty := 0
	for _, word := range []string{"黑色", "白色", "深空灰", "星空灰", "墨灰", "暖白", "陶瓷白", "晶墨灰", "云灰银"} {
		if lightingDesignContainsToken(queryText, word) && !lightingDesignContainsToken(entryText, word) {
			penalty += 35
		}
	}
	for _, word := range []string{"嵌入式", "明装", "吸顶", "墙面", "磁吸", "无边框", "窄边框"} {
		if lightingDesignContainsToken(queryText, word) && !lightingDesignContainsToken(entryText, word) {
			penalty += 25
		}
	}
	for _, word := range []string{"8°", "8度", "15°", "15度", "24°", "24度", "32°", "32度", "36°", "36度", "60°", "60度"} {
		if lightingDesignContainsToken(queryText, word) && !lightingDesignContainsToken(entryText, word) {
			penalty += 40
		}
	}
	for _, word := range []string{"35开孔", "55开孔", "65开孔", "75开孔", "80开孔", "3寸", "5头", "6头", "10头", "12头"} {
		if lightingDesignContainsToken(queryText, word) && !lightingDesignContainsToken(entryText, word) {
			penalty += 25
		}
	}
	if (lightingDesignContainsToken(queryText, "爱思系列") || lightingDesignContainsToken(queryText, "S系列")) && !lightingDesignContainsToken(entryText, "S系列") {
		penalty += 45
	}
	return penalty
}

func lightingDesignProductEntryText(entry lightingDesignProductEntry) string {
	words := []string{
		entry.ProductName,
		entry.ProductSKU,
		entry.ProductSPU,
		entry.Category,
		entry.Series,
	}
	words = append(words, entry.Aliases...)
	words = append(words, entry.DesignRoles...)
	words = append(words, entry.DesignKeywords...)
	words = append(words, entry.CapabilityTags...)
	words = append(words, lightingDesignProductAttributeWords(entry.DesignAttributes)...)
	return lightingDesignNormalizeProductText(strings.Join(words, " "))
}

func lightingDesignDisambiguationTokens(entry lightingDesignProductEntry) []struct {
	word  string
	score int
} {
	words := []string{entry.ProductName, entry.ProductSKU, entry.ProductSPU, entry.Category, entry.Series}
	words = append(words, entry.DesignKeywords...)
	words = append(words, lightingDesignProductAttributeWords(entry.DesignAttributes)...)
	text := lightingDesignNormalizeProductText(strings.Join(words, " "))
	tokens := []struct {
		word  string
		score int
	}{}
	for _, word := range []string{"黑色", "白色", "深空灰", "星空灰", "墨灰", "晶墨灰", "汉玉白", "丝墨青", "方形", "圆形", "明装", "嵌入式", "无边框", "窄边框", "磁吸", "轨道", "格栅", "筒灯", "射灯", "筒射灯", "青空灯", "吸顶灯", "36°", "36度", "32°", "32度", "24°", "24度", "60°", "60度", "75开孔", "55开孔", "35开孔", "3寸", "5头", "6头", "10头", "12头", "12W", "15W"} {
		if strings.Contains(text, lightingDesignNormalizeProductText(word)) {
			tokens = append(tokens, struct {
				word  string
				score int
			}{word: word, score: 10})
		}
	}
	for _, word := range []string{"S系列", "爱思系列", "E系列", "E20", "M20", "D系列", "P20", "Nightingale", "夙夜"} {
		if strings.Contains(text, lightingDesignNormalizeProductText(word)) {
			tokens = append(tokens, struct {
				word  string
				score int
			}{word: word, score: 14})
		}
	}
	return tokens
}

func lightingDesignProductAttributeWords(attributes lightingDesignProductAttributes) []string {
	words := []string{}
	words = append(words, attributes.Colors...)
	words = append(words, attributes.InstallStyles...)
	words = append(words, attributes.BeamAngles...)
	words = append(words, attributes.Openings...)
	words = append(words, attributes.Sizes...)
	words = append(words, attributes.HeadCounts...)
	words = append(words, attributes.Wattages...)
	words = append(words, attributes.Shapes...)
	return words
}

func lightingDesignContainsToken(haystack string, needle string) bool {
	normalized := lightingDesignNormalizeProductText(needle)
	if normalized == "" {
		return false
	}
	return strings.Contains(haystack, normalized)
}

func lightingDesignProductCategoryCompatible(queryText string, entry lightingDesignProductEntry) bool {
	category := lightingDesignNormalizeProductText(entry.Category)
	if category == "" {
		return false
	}
	if lightingDesignContainsToken(queryText, entry.Category) {
		return true
	}
	if lightingDesignContainsToken(queryText, "射灯") && category == lightingDesignNormalizeProductText("筒射灯") {
		return true
	}
	if lightingDesignContainsToken(queryText, "筒灯") && category == lightingDesignNormalizeProductText("筒射灯") {
		return true
	}
	if lightingDesignContainsToken(queryText, "传感器") && category == lightingDesignNormalizeProductText("人在传感器") {
		return true
	}
	return false
}

func lightingDesignNormalizeProductText(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	replacements := []struct {
		old string
		new string
	}{
		{" ", ""},
		{"\t", ""},
		{"-", ""},
		{"_", ""},
		{"（", "("},
		{"）", ")"},
		{"°", "度"},
		{"爱思", "s"},
		{"ｓ", "s"},
		{"系列", "系列"},
	}
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.old, replacement.new)
	}
	text = lightingDesignSpacesRe.ReplaceAllString(text, "")
	return text
}

func lightingDesignProductAttrs(match lightingDesignProductMatch, candidates []lightingDesignProductMatch) map[string]any {
	result := map[string]any{}
	if lightingDesignHasProductIdentity(match) {
		putIfNonEmpty(result, "productMatchConfidence", match.Confidence)
		if match.Score > 0 {
			result["productMatchScore"] = match.Score
		}
		if len(match.Matched) > 0 {
			result["productMatchedWords"] = match.Matched
		}
		lightingDesignCopyProductEntry(result, match.Entry)
	}
	if len(candidates) > 0 {
		result["productCandidates"] = lightingDesignProductCandidateSummaries(candidates)
	}
	return result
}

func lightingDesignCopyProductEntry(output map[string]any, entry lightingDesignProductEntry) {
	putIfNonEmpty(output, "materialCode", entry.MaterialCode)
	if entry.PID > 0 {
		output["pid"] = entry.PID
	}
	if entry.PCID > 0 {
		output["pcId"] = entry.PCID
	}
	if entry.ConnectType >= -1 {
		output["connectType"] = entry.ConnectType
	}
	putIfNonEmpty(output, "productName", entry.ProductName)
	putIfNonEmpty(output, "productSku", entry.ProductSKU)
	putIfNonEmpty(output, "productSpu", entry.ProductSPU)
	putIfNonEmpty(output, "category", entry.Category)
	putIfNonEmpty(output, "series", entry.Series)
	if entry.Priority > 0 {
		output["catalogPriority"] = entry.Priority
	}
	if len(entry.AdjustableProperties) > 0 {
		output["adjustableProperties"] = entry.AdjustableProperties
	}
	if len(entry.PropertyEvents) > 0 {
		output["propertyEvents"] = entry.PropertyEvents
	}
	if len(entry.SensorEvents) > 0 {
		output["sensorEvents"] = entry.SensorEvents
	}
	if len(lightingDesignProductAttributeWords(entry.DesignAttributes)) > 0 {
		output["designAttributes"] = entry.DesignAttributes
	}
	if len(entry.DesignRoles) > 0 {
		output["designRoles"] = entry.DesignRoles
	}
	if len(entry.DesignKeywords) > 0 {
		output["designKeywords"] = entry.DesignKeywords
	}
	if len(entry.CapabilityTags) > 0 {
		output["capabilityTags"] = entry.CapabilityTags
	}
}

func lightingDesignProductCandidateSummaries(matches []lightingDesignProductMatch) []any {
	result := make([]any, 0, len(matches))
	for _, match := range matches {
		item := map[string]any{
			"score":      match.Score,
			"confidence": match.Confidence,
		}
		if len(match.Matched) > 0 {
			item["matchedWords"] = match.Matched
		}
		lightingDesignCopyProductEntry(item, match.Entry)
		result = append(result, item)
	}
	return result
}

func putIfNonEmpty(values map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		values[key] = strings.TrimSpace(value)
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
