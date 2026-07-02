package api

import (
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func looksLikeNaturalLightingDesign(payload map[string]any) bool {
	for _, key := range semantic.NaturalLightingDesignFields() {
		if _, ok := payload[key]; ok {
			return true
		}
	}
	return false
}

func lightingDesignMetaName(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= 14 {
		return value
	}
	return string([]rune(value)[:14])
}

func mapListFromAny(value any) ([]map[string]any, bool) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		result = append(result, typed)
	}
	return result, true
}

func mapFromAny(value any) (map[string]any, bool) {
	item, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	return item, true
}

func anyListFromMap(values map[string]any, key string) []any {
	items, ok := values[key].([]any)
	if !ok {
		return nil
	}
	return items
}

func lightingDesignStringListFromAny(value any) ([]string, bool) {
	if items, ok := value.([]string); ok {
		result := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				return nil, false
			}
			result = append(result, item)
		}
		return result, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := lightingDesignStringFromAny(item)
		if value == "" {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return lightingDesignStringFromAny(values[key])
}

func intFromMap(values map[string]any, key string, fallback int) int {
	if parsed, ok := lightingDesignIntFromAny(values[key]); ok {
		return parsed
	}
	return fallback
}

func int64FromMap(values map[string]any, key string, fallback int64) int64 {
	if parsed, ok := lightingDesignIntFromAny(values[key]); ok {
		return int64(parsed)
	}
	return fallback
}

func boolFromMap(values map[string]any, key string) bool {
	parsed, ok := lightingDesignBoolFromAny(values[key])
	return ok && parsed
}

func lightingDesignBoolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	}
	return false, false
}

func lightingDesignIntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func lightingDesignStringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	}
	return ""
}

func copyLightingDesignDeepMap(source map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range source {
		result[key] = copyLightingDesignDeepValue(value)
	}
	return result
}

func copyLightingDesignDeepValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return copyLightingDesignDeepMap(typed)
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, copyLightingDesignDeepValue(item))
		}
		return result
	default:
		return typed
	}
}
