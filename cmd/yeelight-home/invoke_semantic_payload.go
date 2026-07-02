package main

import "github.com/yeelight/yeelight-home/internal/semantic"

func normalizeSceneActionRows(value any) ([]map[string]any, bool) {
	return semantic.NormalizeSceneActions(value)
}

func normalizeAutomationActionRows(value any) ([]map[string]any, bool) {
	return semantic.NormalizeAutomationActions(value)
}

func normalizePanelActionRows(value any) ([]any, bool) {
	return semantic.NormalizePanelActions(value)
}

func normalizeTargetBinding(value map[string]any, groupTypeID int, typeField string) map[string]any {
	return semantic.NormalizeTargetBinding(value, groupTypeID, typeField)
}

func normalizeLightActionParams(value any) any {
	return semantic.NormalizeLightParams(value)
}

func normalizeAutomationParamsFromRequest(parameters map[string]any) any {
	return semantic.NormalizeAutomationParamsFromRequest(parameters)
}

func semanticTargetTypeID(value string, groupTypeID int) (int, bool) {
	return semantic.TargetTypeID(value, groupTypeID)
}

func semanticParameterPaths(fields ...string) []string {
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		result = append(result, semantic.ParameterPath(field))
	}
	return result
}

func semanticParameterArrayPath(arrayField string, field string) string {
	return semantic.ParameterPath(semantic.ArrayField(arrayField), field)
}

func firstPresent(values map[string]any, keys ...string) (any, bool) {
	return semantic.FirstPresent(values, keys...)
}
