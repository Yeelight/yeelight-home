package main

import (
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
)

type memoryPreferenceCandidate struct {
	scopeType       string
	scopeRef        string
	preferenceType  string
	preferenceValue string
	kind            string
	evidence        string
}

func memoryPreferenceFromRequest(request contract.Request) memoryPreferenceCandidate {
	candidates := memoryPreferencesFromRequest(request)
	if len(candidates) == 0 {
		return memoryPreferenceCandidate{}
	}
	return candidates[0]
}

func memoryPreferencesFromRequest(request contract.Request) []memoryPreferenceCandidate {
	candidate := memoryPreferenceCandidate{
		scopeType:       firstNonEmptyString(firstRequestString(request.Parameters, "scopeType"), "home"),
		scopeRef:        firstRequestString(request.Parameters, "scopeRef"),
		preferenceType:  firstRequestString(request.Parameters, "preferenceType", "type"),
		preferenceValue: firstRequestString(request.Parameters, "preferenceValue", "value"),
		kind:            firstNonEmptyString(firstRequestString(request.Parameters, "kind"), "explicit"),
		evidence:        firstRequestString(request.Parameters, "evidence"),
	}
	if candidate.preferenceType != "" && candidate.preferenceValue != "" {
		return []memoryPreferenceCandidate{candidate}
	}
	return memoryPreferencesFromUtterance(request, candidate)
}

func memoryPreferenceFromUtterance(request contract.Request, candidate memoryPreferenceCandidate) memoryPreferenceCandidate {
	candidates := memoryPreferencesFromUtterance(request, candidate)
	if len(candidates) == 0 {
		return candidate
	}
	return candidates[0]
}

func memoryPreferencesFromUtterance(request contract.Request, candidate memoryPreferenceCandidate) []memoryPreferenceCandidate {
	utterance := strings.TrimSpace(request.Utterance)
	if utterance == "" || !memoryUtteranceAllowsExtraction(utterance) {
		if candidate.preferenceType != "" || candidate.preferenceValue != "" {
			return []memoryPreferenceCandidate{candidate}
		}
		return nil
	}
	if candidate.scopeRef == "" {
		candidate.scopeRef = inferMemoryScopeRef(utterance)
	}
	if candidate.scopeType == "" || candidate.scopeType == "home" {
		candidate.scopeType = inferMemoryScopeType(candidate.scopeRef)
	}
	if candidate.evidence == "" {
		candidate.evidence = "用户明确要求记住：" + utterance
	}
	if candidate.preferenceType != "" && candidate.preferenceValue == "" {
		candidate.preferenceValue = inferMemoryPreferenceValue(utterance)
	}
	if candidate.preferenceType != "" && candidate.preferenceValue != "" {
		return []memoryPreferenceCandidate{candidate}
	}
	candidates := explicitPreferenceCandidatesFromUtterance(utterance, candidate)
	if len(candidates) > 0 {
		return candidates
	}
	candidate.preferenceType = inferMemoryPreferenceType(utterance)
	candidate.preferenceValue = inferMemoryPreferenceValue(utterance)
	if candidate.preferenceType == "" || candidate.preferenceValue == "" {
		return nil
	}
	return []memoryPreferenceCandidate{candidate}
}

func explicitPreferenceCandidatesFromUtterance(utterance string, base memoryPreferenceCandidate) []memoryPreferenceCandidate {
	candidates := []memoryPreferenceCandidate{}
	add := func(preferenceType string, preferenceValue string) {
		for _, existing := range candidates {
			if existing.preferenceType == preferenceType && existing.preferenceValue == preferenceValue {
				return
			}
		}
		candidate := base
		candidate.preferenceType = preferenceType
		candidate.preferenceValue = preferenceValue
		candidates = append(candidates, candidate)
	}
	for _, marker := range []string{"柔和暖光", "暖光", "暖白", "偏暖", "暖一点", "温暖"} {
		if strings.Contains(utterance, marker) {
			add("color_temperature", marker)
			break
		}
	}
	for _, marker := range []string{"冷光", "冷白", "偏冷", "冷一点"} {
		if strings.Contains(utterance, marker) {
			add("color_temperature", marker)
			break
		}
	}
	for _, marker := range []string{"不要太亮", "别太亮", "不要刺眼", "别刺眼", "暗一点", "调暗", "太亮"} {
		if strings.Contains(utterance, marker) {
			add("brightness", marker)
			break
		}
	}
	for _, marker := range []string{"亮一点", "太暗"} {
		if strings.Contains(utterance, marker) {
			add("brightness", marker)
			break
		}
	}
	for _, marker := range []string{"不要彩光", "不喜欢彩色", "别用彩光"} {
		if strings.Contains(utterance, marker) {
			add("color", marker)
			break
		}
	}
	return candidates
}

func memoryUtteranceAllowsExtraction(value string) bool {
	for _, marker := range []string{"记住", "以后默认", "以后都", "以后帮我", "我喜欢", "我偏好", "我不喜欢", "不要推荐"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func inferMemoryPreferenceType(value string) string {
	for _, candidate := range []struct {
		markers []string
		value   string
	}{
		{[]string{"亮度", "太亮", "暗"}, "brightness"},
		{[]string{"色温", "暖光", "冷光", "暖白", "冷白"}, "color_temperature"},
		{[]string{"颜色", "彩光", "红色", "蓝色", "绿色"}, "color"},
		{[]string{"氛围", "场景", "模式"}, "ambience"},
		{[]string{"推荐", "建议"}, "recommendation"},
		{[]string{"名字", "叫法", "别名"}, "alias"},
	} {
		for _, marker := range candidate.markers {
			if strings.Contains(value, marker) {
				return candidate.value
			}
		}
	}
	return "preference"
}

func inferMemoryScopeRef(value string) string {
	for _, marker := range []string{"客厅", "卧室", "主卧", "次卧", "书房", "餐厅", "厨房", "卫生间", "阳台", "玄关", "走廊", "灯光区"} {
		if strings.Contains(value, marker) {
			return marker
		}
	}
	return ""
}

func inferMemoryScopeType(scopeRef string) string {
	if strings.TrimSpace(scopeRef) == "" {
		return "home"
	}
	return "room"
}

func inferMemoryPreferenceValue(value string) string {
	trimmed := strings.TrimSpace(value)
	for _, marker := range []string{"偏好", "喜欢", "默认", "记住"} {
		if _, after, ok := strings.Cut(trimmed, marker); ok {
			after = strings.TrimSpace(strings.TrimPrefix(after, "："))
			after = strings.TrimSpace(strings.TrimPrefix(after, ":"))
			if after != "" {
				return after
			}
		}
	}
	return trimmed
}
