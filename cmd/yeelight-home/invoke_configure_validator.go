package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
)

const (
	groupTypeRoom       = 1
	groupTypeDevice     = 2
	groupTypeCustom     = 3
	groupTypeMesh       = 4
	groupTypeScene      = 6
	groupTypeAutomation = 12

	areaRoomLimit        = 200
	groupDeviceLimit     = 500
	sceneActionLimit     = 500
	automationIfLimit    = 9
	automationThenLimit  = 200
	houseAreaLimit       = 300
	houseRoomLimit       = 500
	houseGroupLimit      = 2000
	houseSceneLimit      = 500
	houseAutomationLimit = 500
)

func validateRoomCreatePayload(_ map[string]any, entities api.EntityListResult) string {
	if entities.Counts["room"]+1 > houseRoomLimit {
		return "house_room_limit_exceeded"
	}
	return ""
}

func validateConfigureCreatePayload(entityType string, payload map[string]any, entities api.EntityListResult) string {
	switch entityType {
	case "room":
		return validateRoomCreatePayload(payload, entities)
	case "area":
		return validateAreaCreatePayload(payload, entities)
	case "group":
		return validateGroupCreatePayload(payload, entities)
	case "scene":
		return validateSceneCreatePayload(payload, entities)
	case "automation":
		return validateAutomationCreatePayload(payload, entities)
	default:
		return ""
	}
}

func validateAreaCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["area"]+1 > houseAreaLimit {
		return "house_area_limit_exceeded"
	}
	roomIDs := valueIDList(payload["roomIds"])
	if len(roomIDs) > areaRoomLimit {
		return "area_room_limit_exceeded"
	}
	if parentID := valueIDString(payload["parentId"]); parentID != "" && !entityExists(entities, "area", parentID) {
		return "invalid_area_resource_reference"
	}
	for _, roomID := range roomIDs {
		if !entityExists(entities, "room", roomID) {
			return "invalid_area_resource_reference"
		}
	}
	return ""
}

func validateGroupCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["group"]+1 > houseGroupLimit {
		return "house_group_limit_exceeded"
	}
	roomID := valueIDString(payload["roomId"])
	if roomID == "" || !entityExists(entities, "room", roomID) {
		return "invalid_group_room_reference"
	}
	deviceIDs := valueIDList(payload["deviceIds"])
	if len(deviceIDs) > groupDeviceLimit {
		return "group_device_limit_exceeded"
	}
	for _, deviceID := range deviceIDs {
		if !entityExists(entities, "device", deviceID) {
			return "invalid_group_device_reference"
		}
	}
	return ""
}

func validateSceneCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["scene"]+1 > houseSceneLimit {
		return "house_scene_limit_exceeded"
	}
	details, ok := payload["details"].([]map[string]any)
	if !ok || len(details) == 0 {
		return "invalid_scene_create_payload"
	}
	if len(details) > sceneActionLimit {
		return "scene_action_limit_exceeded"
	}
	for _, detail := range details {
		if reason := normalizeActionParams(detail, "invalid_scene_detail_params"); reason != "" {
			return reason
		}
		if reason := validateResourceReference(detail["typeId"], detail["resId"], entities, "invalid_scene_resource_type", "invalid_scene_resource_reference"); reason != "" {
			return reason
		}
	}
	return ""
}

func validateAutomationCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["automation"]+1 > houseAutomationLimit {
		return "house_automation_limit_exceeded"
	}
	if statusValue, ok := payload["status"]; ok {
		status, valid := valueInt(statusValue)
		if !valid || (status != 0 && status != 1) {
			return "invalid_automation_status"
		}
	}
	repeatType, _ := valueInt(payload["repeatType"])
	if reason := validateAutomationParams(payload["params"], repeatType, entities); reason != "" {
		return reason
	}
	actions, ok := payload["actions"].([]map[string]any)
	if !ok || len(actions) == 0 {
		return "invalid_automation_create_payload"
	}
	if len(actions) > automationThenLimit {
		return "automation_action_limit_exceeded"
	}
	for _, action := range actions {
		if reason := normalizeActionParams(action, "invalid_automation_action_params"); reason != "" {
			return reason
		}
		if reason := validateAutomationActionReference(action["typeId"], action["resId"], entities); reason != "" {
			return reason
		}
	}
	return ""
}

func validateAutomationParams(value any, repeatType int, entities api.EntityListResult) string {
	var params map[string]any
	switch typed := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(strings.TrimSpace(typed)), &params); err != nil {
			return "invalid_automation_params"
		}
	case map[string]any:
		params = typed
	default:
		return "invalid_automation_params"
	}
	if strings.TrimSpace(requestString(params["type"])) != "and" {
		return "invalid_automation_params"
	}
	conditions, ok := params["conditions"].([]any)
	if !ok || len(conditions) == 0 {
		return "invalid_automation_params"
	}
	if len(conditions) > automationIfLimit {
		return "automation_condition_limit_exceeded"
	}
	if reason := validateAutomationConditionGroups(params, repeatType); reason != "" {
		return reason
	}
	if reason := validateAutomationConditionReferences(conditions, entities); reason != "" {
		return reason
	}
	return ""
}

func validateAutomationActionReference(typeValue any, idValue any, entities api.EntityListResult) string {
	typeID, ok := valueInt(typeValue)
	if !ok {
		return "invalid_automation_action_resource_type"
	}
	switch typeID {
	case groupTypeDevice, groupTypeMesh, groupTypeScene:
	default:
		return "invalid_automation_action_resource_type"
	}
	entityType, _ := entityTypeForGroupType(typeID)
	if !entityExists(entities, entityType, valueIDString(idValue)) {
		return "invalid_automation_action_reference"
	}
	return ""
}

func validateResourceReference(typeValue any, idValue any, entities api.EntityListResult, typeReason string, referenceReason string) string {
	typeID, ok := valueInt(typeValue)
	if !ok {
		return typeReason
	}
	entityType, ok := entityTypeForGroupType(typeID)
	if !ok {
		return typeReason
	}
	if !entityExists(entities, entityType, valueIDString(idValue)) {
		return referenceReason
	}
	return ""
}

func entityTypeForGroupType(typeID int) (string, bool) {
	switch typeID {
	case groupTypeRoom:
		return "room", true
	case groupTypeDevice:
		return "device", true
	case groupTypeCustom:
		return "group", true
	case groupTypeMesh:
		return "group", true
	case groupTypeScene:
		return "scene", true
	case groupTypeAutomation:
		return "automation", true
	default:
		return "", false
	}
}

func normalizeActionParams(item map[string]any, reason string) string {
	params, ok := item["params"]
	if !ok {
		return reason
	}
	compact, err := compactJSONForRuntime(params)
	if err != nil || strings.TrimSpace(compact) == "" {
		return reason
	}
	item["params"] = compact
	return ""
}

func hasResourceReference(item map[string]any) bool {
	return valueIDString(item["resId"]) != "" || valueIDString(item["deviceId"]) != "" || item["typeId"] != nil
}

func entityExists(entities api.EntityListResult, entityType string, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, entity := range entities.Entities {
		if entity.Type == entityType && entity.ID == id {
			return true
		}
	}
	return false
}

func valueIDList(value any) []string {
	switch typed := value.(type) {
	case []float64:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, fmt.Sprintf("%.0f", item))
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if id := valueIDString(item); id != "" {
				result = append(result, id)
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if id := strings.TrimSpace(item); id != "" {
				result = append(result, id)
			}
		}
		return result
	default:
		if id := valueIDString(value); id != "" {
			return []string{id}
		}
		return nil
	}
}

func valueIDString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed != float64(int64(typed)) {
			return ""
		}
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}

func valueInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case string:
		return requestInt(typed)
	default:
		return 0, false
	}
}
