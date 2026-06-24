package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func BuildAreaCreatePayload(houseID string, name string, description string, icon string, parentID string, roomIDs []string) (map[string]any, error) {
	payload, err := baseCreatePayload(houseID, name, description, icon)
	if err != nil {
		return nil, err
	}
	if parentID != "" {
		parsed, err := parseID(parentID, "parent id")
		if err != nil {
			return nil, err
		}
		payload["parentId"] = parsed
	}
	if len(roomIDs) > 0 {
		parsed, err := parseIDs(roomIDs, "room id")
		if err != nil {
			return nil, err
		}
		payload["roomIds"] = parsed
	}
	return payload, nil
}

func BuildGroupCreatePayload(houseID string, name string, roomID string, cid string, deviceIDs []string, description string, icon string) (map[string]any, error) {
	payload, err := baseCreatePayload(houseID, name, description, icon)
	if err != nil {
		return nil, err
	}
	parsedRoomID, err := parseID(roomID, "room id")
	if err != nil {
		return nil, err
	}
	parsedCID, err := parseID(cid, "cid")
	if err != nil {
		return nil, err
	}
	payload["roomId"] = parsedRoomID
	payload["cid"] = parsedCID
	if len(deviceIDs) > 0 {
		parsed, err := parseIDs(deviceIDs, "device id")
		if err != nil {
			return nil, err
		}
		payload["deviceIds"] = parsed
	}
	return payload, nil
}

func BuildSceneCreatePayload(houseID string, name string, description string, icon string, details []map[string]any) (map[string]any, error) {
	payload, err := baseCreatePayload(houseID, name, description, icon)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("scene details are required")
	}
	payload["details"] = details
	return payload, nil
}

func BuildAutomationCreatePayload(houseID string, name string, startTime string, endTime string, repeatType int, repeatValue string, params any, actions []map[string]any, version int, status *int) (map[string]any, error) {
	payload, err := baseCreatePayload(houseID, name, "", "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(startTime) == "" || strings.TrimSpace(endTime) == "" {
		return nil, fmt.Errorf("automation startTime and endTime are required")
	}
	if repeatType < 1 || repeatType > 7 {
		return nil, fmt.Errorf("automation repeatType must be 1..7")
	}
	paramsJSON, err := jsonString(params)
	if err != nil {
		return nil, fmt.Errorf("automation params must be JSON: %w", err)
	}
	if len(actions) == 0 {
		return nil, fmt.Errorf("automation actions are required")
	}
	payload["startTime"] = strings.TrimSpace(startTime)
	payload["endTime"] = strings.TrimSpace(endTime)
	payload["repeatType"] = repeatType
	if value := strings.TrimSpace(repeatValue); value != "" {
		payload["repeatValue"] = value
	}
	payload["params"] = paramsJSON
	payload["actions"] = actions
	if version > 0 {
		payload["version"] = version
	}
	if status != nil {
		payload["status"] = *status
	}
	return payload, nil
}

func baseCreatePayload(houseID string, name string, description string, icon string) (map[string]any, error) {
	parsedHouseID, err := parseID(houseID, "house id")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	payload := map[string]any{
		"houseId": parsedHouseID,
		"name":    strings.TrimSpace(name),
	}
	if value := strings.TrimSpace(description); value != "" {
		payload["desc"] = value
	}
	if value := strings.TrimSpace(icon); value != "" {
		payload["icon"] = value
	}
	return payload, nil
}

func parseID(value string, label string) (float64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be numeric", label)
	}
	return float64(parsed), nil
}

func parseIDs(values []string, label string) ([]float64, error) {
	result := make([]float64, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		parsed, err := parseID(value, label)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

func jsonString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return "", fmt.Errorf("json string is empty")
		}
		var probe any
		if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
			return "", err
		}
		return trimmed, nil
	default:
		return compactJSON(value)
	}
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}
