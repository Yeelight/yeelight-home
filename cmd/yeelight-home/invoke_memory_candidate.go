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
	candidate := memoryPreferenceCandidate{
		scopeType:       firstNonEmptyString(firstRequestString(request.Parameters, "scopeType"), "home"),
		scopeRef:        firstRequestString(request.Parameters, "scopeRef"),
		preferenceType:  firstRequestString(request.Parameters, "preferenceType", "type"),
		preferenceValue: firstRequestString(request.Parameters, "preferenceValue", "value"),
		kind:            firstNonEmptyString(firstRequestString(request.Parameters, "kind"), "explicit"),
		evidence:        firstRequestString(request.Parameters, "evidence"),
	}
	if candidate.preferenceType != "" && candidate.preferenceValue != "" {
		return candidate
	}
	return memoryPreferenceFromUtterance(request, candidate)
}

func memoryPreferenceFromUtterance(request contract.Request, candidate memoryPreferenceCandidate) memoryPreferenceCandidate {
	utterance := strings.TrimSpace(request.Utterance)
	if utterance == "" || !memoryUtteranceAllowsExtraction(utterance) {
		return candidate
	}
	if candidate.preferenceType == "" {
		candidate.preferenceType = inferMemoryPreferenceType(utterance)
	}
	if candidate.scopeRef == "" {
		candidate.scopeRef = inferMemoryScopeRef(utterance)
	}
	if candidate.scopeType == "" || candidate.scopeType == "home" {
		candidate.scopeType = inferMemoryScopeType(candidate.scopeRef)
	}
	if candidate.preferenceValue == "" {
		candidate.preferenceValue = inferMemoryPreferenceValue(utterance)
	}
	if candidate.evidence == "" && candidate.preferenceValue != "" {
		candidate.evidence = "用户明确要求记住：" + utterance
	}
	return candidate
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
