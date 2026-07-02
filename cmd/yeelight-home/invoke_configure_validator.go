package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
	roomIDs := valueIDList(payload[semantic.FieldRoomIDs])
	if len(roomIDs) > areaRoomLimit {
		return "area_room_limit_exceeded"
	}
	if parentID := valueIDString(payload[semantic.FieldParentID]); parentID != "" && !entityExists(entities, "area", parentID) {
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
	if reason := resolveGroupCreateReferences(payload, entities); reason != "" {
		return reason
	}
	roomID := valueIDString(payload[semantic.FieldRoomID])
	if roomID == "" || !entityExists(entities, "room", roomID) {
		return "invalid_group_room_reference"
	}
	deviceIDs := valueIDList(payload[semantic.FieldDeviceIDs])
	if len(deviceIDs) == 0 {
		return "missing_group_members"
	}
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

func resolveGroupCreateReferences(payload map[string]any, entities api.EntityListResult) string {
	roomID := valueIDString(payload[semantic.FieldRoomID])
	if roomID == "" {
		roomName := firstNonEmptyString(
			requestString(payload[semantic.FieldRoomName]),
			requestString(payload[semantic.FieldTargetRoomName]),
		)
		if roomName != "" {
			match, candidates, _ := findEntity(entityGetTarget{name: roomName, entityType: "room"}, entities.Entities)
			if match.ID != "" && len(candidates) == 1 {
				roomID = match.ID
				payload[semantic.FieldRoomID] = requestNumberOrString(match.ID)
			} else if len(candidates) > 0 {
				return "ambiguous_group_room_reference"
			}
		}
	}
	deviceNames := requestStringList(payload[semantic.FieldDeviceNames])
	if len(deviceNames) == 0 {
		return ""
	}
	deviceIDs := valueIDList(payload[semantic.FieldDeviceIDs])
	for _, deviceName := range deviceNames {
		target := entityGetTarget{name: deviceName, entityType: "device", roomID: roomID}
		match, candidates, _ := findEntity(target, entities.Entities)
		if match.ID != "" && len(candidates) == 1 {
			deviceIDs = append(deviceIDs, match.ID)
			continue
		}
		if len(candidates) > 0 {
			return "ambiguous_group_device_reference"
		}
		return "invalid_group_device_reference"
	}
	payload[semantic.FieldDeviceIDs] = stringListAsRequestIDs(deviceIDs)
	delete(payload, semantic.FieldDeviceNames)
	return ""
}

func groupCreateReferenceCandidateMaps(payload map[string]any, entities api.EntityListResult, reason string) []map[string]any {
	switch reason {
	case "ambiguous_group_room_reference":
		roomName := firstNonEmptyString(
			requestString(payload[semantic.FieldRoomName]),
			requestString(payload[semantic.FieldTargetRoomName]),
		)
		_, candidates, _ := findEntity(entityGetTarget{name: roomName, entityType: "room"}, entities.Entities)
		return entityCandidateMaps(candidates)
	case "ambiguous_group_device_reference", "invalid_group_device_reference":
		roomID := valueIDString(payload[semantic.FieldRoomID])
		roomName := firstNonEmptyString(
			requestString(payload[semantic.FieldRoomName]),
			requestString(payload[semantic.FieldTargetRoomName]),
		)
		if roomID == "" && roomName != "" {
			roomMatch, roomCandidates, _ := findEntity(entityGetTarget{name: roomName, entityType: "room"}, entities.Entities)
			if roomMatch.ID != "" && len(roomCandidates) == 1 {
				roomID = roomMatch.ID
			}
		}
		for _, deviceName := range requestStringList(payload[semantic.FieldDeviceNames]) {
			target := entityGetTarget{name: deviceName, entityType: "device", roomID: roomID, roomName: roomName}
			match, candidates, _ := findEntity(target, entities.Entities)
			if match.ID != "" && len(candidates) == 1 {
				continue
			}
			return entityCandidateMaps(candidates)
		}
	}
	return nil
}

func entityCandidateMaps(candidates []api.EntitySummary) []map[string]any {
	if len(candidates) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, entitySummaryMap(candidate))
	}
	return result
}

func validateSceneCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["scene"]+1 > houseSceneLimit {
		return "house_scene_limit_exceeded"
	}
	details, ok := payload[semantic.FieldDetails].([]map[string]any)
	if !ok || len(details) == 0 {
		return "invalid_scene_create_payload"
	}
	if len(details) > sceneActionLimit {
		return "scene_action_limit_exceeded"
	}
	for _, detail := range details {
		if reason := resolveActionResourceName(detail, entities, "ambiguous_scene_resource_reference"); reason != "" {
			return reason
		}
		if reason := normalizeSceneDetail(detail, entities, "invalid_scene_detail_params"); reason != "" {
			return reason
		}
		if reason := validateResourceReference(
			detail[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)],
			detail[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)],
			entities,
			"invalid_scene_resource_type",
			"invalid_scene_resource_reference",
		); reason != "" {
			return reason
		}
	}
	return ""
}

func validateAutomationCreatePayload(payload map[string]any, entities api.EntityListResult) string {
	if entities.Counts["automation"]+1 > houseAutomationLimit {
		return "house_automation_limit_exceeded"
	}
	if statusValue, ok := payload[semantic.FieldStatus]; ok {
		status, valid := valueInt(statusValue)
		if !valid || (status != 0 && status != 1) {
			return "invalid_automation_status"
		}
	}
	repeatType, _ := valueInt(payload[semantic.InternalRepeatTypeField()])
	if reason := validateAutomationParams(payload[semantic.InternalAutomationParamsField()], repeatType, entities); reason != "" {
		return reason
	}
	actions, ok := payload[semantic.FieldActions].([]map[string]any)
	if !ok || len(actions) == 0 {
		return "invalid_automation_create_payload"
	}
	if len(actions) > automationThenLimit {
		return "automation_action_limit_exceeded"
	}
	for index, action := range actions {
		if reason := resolveActionResourceName(action, entities, "ambiguous_automation_action_reference"); reason != "" {
			return reason
		}
		if reason := normalizeActionParams(action, "invalid_automation_action_params"); reason != "" {
			return reason
		}
		if _, ok := valueInt(action[semantic.FieldRank]); !ok {
			action[semantic.FieldRank] = index
		}
		if reason := validateAutomationActionReference(
			action[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)],
			action[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)],
			entities,
		); reason != "" {
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
	if strings.TrimSpace(requestString(params[semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType)])) != "and" {
		return "invalid_automation_params"
	}
	conditions, ok := params[semantic.FieldConditions].([]any)
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

func resolveActionResourceName(item map[string]any, entities api.EntityListResult, ambiguousReason string) string {
	idKey := semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)
	if valueIDString(item[idKey]) != "" {
		return ""
	}
	typeID, ok := valueInt(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)])
	if !ok {
		return ""
	}
	entityType, ok := entityTypeForGroupType(typeID)
	if !ok {
		return ""
	}
	targetName := strings.TrimSpace(requestString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)]))
	if targetName == "" {
		return ""
	}
	match, candidates, _ := findEntity(entityGetTarget{name: targetName, entityType: entityType}, entities.Entities)
	if match.ID != "" && len(candidates) == 1 {
		item[idKey] = requestNumberOrString(match.ID)
		item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)] = match.Name
		return ""
	}
	if len(candidates) > 0 {
		return ambiguousReason
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
	params, ok := item[semantic.InternalActionParamsField()]
	if !ok {
		typeID, typeOK := valueInt(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)])
		if typeOK && typeID == groupTypeScene {
			item[semantic.InternalActionParamsField()] = "{}"
			return ""
		}
		return reason
	}
	compact, err := compactJSONForRuntime(params)
	if err != nil || strings.TrimSpace(compact) == "" {
		return reason
	}
	item[semantic.InternalActionParamsField()] = compact
	return ""
}

func normalizeSceneDetail(item map[string]any, entities api.EntityListResult, reason string) string {
	if reason := normalizeActionParams(item, reason); reason != "" {
		return reason
	}
	if _, ok := valueInt(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)]); !ok {
		return reason
	}
	if valueIDString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)]) == "" {
		return reason
	}
	if _, ok := valueInt(item[semantic.FieldAction]); !ok {
		item[semantic.FieldAction] = 0
	}
	if _, ok := valueInt(item[semantic.FieldRank]); !ok {
		item[semantic.FieldRank] = 0
	}
	if strings.TrimSpace(requestString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)])) == "" {
		if entity, ok := findEntityForSceneDetail(
			entities,
			item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)],
			item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)],
		); ok && strings.TrimSpace(entity.Name) != "" {
			item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)] = entity.Name
		} else {
			item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)] = valueIDString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)])
		}
	}
	return ""
}

func findEntityForSceneDetail(entities api.EntityListResult, typeValue any, idValue any) (api.EntitySummary, bool) {
	typeID, ok := valueInt(typeValue)
	if !ok {
		return api.EntitySummary{}, false
	}
	entityType, ok := entityTypeForGroupType(typeID)
	if !ok {
		return api.EntitySummary{}, false
	}
	id := valueIDString(idValue)
	for _, entity := range entities.Entities {
		if entity.Type == entityType && entity.ID == id {
			return entity, true
		}
	}
	return api.EntitySummary{}, false
}

func hasResourceReference(item map[string]any) bool {
	return valueIDString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)]) != "" ||
		valueIDString(item[semantic.FieldDeviceID]) != "" ||
		item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)] != nil
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
