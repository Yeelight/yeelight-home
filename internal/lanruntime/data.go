package lanruntime

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func resolveTargetFromData(data any, target Target) (Target, error) {
	if strings.TrimSpace(target.ID) != "" {
		return target, nil
	}
	name := strings.TrimSpace(target.Name)
	if name == "" {
		return Target{}, unsupported("target id or name is required")
	}
	matches := []Target{}
	for _, item := range collectObjectMaps(data) {
		candidate := targetFromMap(item)
		if candidate.ID == "" || !strings.EqualFold(candidate.Name, name) {
			continue
		}
		if target.Type != "" && candidate.Type != "" && !strings.EqualFold(candidate.Type, target.Type) {
			continue
		}
		if target.Room != "" && candidate.Room != "" && !strings.EqualFold(candidate.Room, target.Room) {
			continue
		}
		matches = append(matches, candidate)
	}
	if len(matches) == 0 {
		return Target{}, &Error{Kind: ErrorRejected, Stage: "resolve", Message: "target was not found in gateway topology"}
	}
	if len(matches) > 1 {
		return Target{}, &Error{Kind: ErrorRejected, Stage: "resolve", Message: "target name is ambiguous; provide an exact id or room"}
	}
	resolved := matches[0]
	resolved.HouseID = target.HouseID
	if resolved.Type == "" {
		resolved.Type = target.Type
	}
	return resolved, nil
}

func projectTargets(data any, houseID string) []Target {
	result := []Target{}
	seen := map[string]bool{}
	for _, item := range collectObjectMaps(data) {
		target := targetFromMap(item)
		if target.ID == "" || target.Name == "" || seen[target.ID] {
			continue
		}
		target.HouseID = houseID
		seen[target.ID] = true
		result = append(result, target)
	}
	return result
}

func extractPropertyValue(data any, target Target, property string) (any, bool) {
	for _, item := range collectObjectMaps(data) {
		candidate := targetFromMap(item)
		if target.ID != "" && candidate.ID != "" && candidate.ID != target.ID {
			continue
		}
		if value, ok := propertyFromMap(item, property); ok {
			return value, true
		}
	}
	return nil, false
}

func collectObjectMaps(value any) []map[string]any {
	result := []map[string]any{}
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			result = append(result, typed)
			for _, child := range typed {
				visit(child)
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
	return result
}

func targetFromMap(item map[string]any) Target {
	return Target{
		ID:   firstMapString(item, "nodeId", "deviceId", "entityId", "id"),
		Name: firstMapString(item, "nodeName", "deviceName", "entityName", "name"),
		Type: firstMapString(item, "nodeType", "deviceType", "entityType", "type"),
		Room: firstMapString(item, "roomName", "room"),
	}
}

func propertyFromMap(item map[string]any, property string) (any, bool) {
	for _, source := range []map[string]any{item, asMap(item["properties"]), asMap(item["state"]), asMap(item["status"]), asMap(item["attrs"])} {
		for key, value := range source {
			if propertyNamesMatch(key, property) {
				return value, true
			}
		}
	}
	return nil, false
}

func propertyNamesMatch(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if strings.EqualFold(left, right) {
		return true
	}
	leftID, leftOK := semantic.PropertyID(left)
	rightID, rightOK := semantic.PropertyID(right)
	if leftOK && rightOK {
		return leftID == rightID
	}
	if leftOK {
		return leftID == right
	}
	if rightOK {
		return rightID == left
	}
	return false
}

func firstMapString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(item[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func valuesMatch(actual, expected any) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}
	if actualNumber, ok := numberValue(actual); ok {
		if expectedNumber, expectedOK := numberValue(expected); expectedOK {
			return math.Abs(actualNumber-expectedNumber) < 0.001
		}
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(actual)), strings.TrimSpace(fmt.Sprint(expected)))
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case json.Number:
		parsed, err := strconv.ParseFloat(string(typed), 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
