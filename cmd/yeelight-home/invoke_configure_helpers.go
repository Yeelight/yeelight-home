package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

type configureCreateSpec struct {
	entityType     string
	entityLabel    string
	invalidReason  string
	acceptedFields []string
	preconditions  []string
	buildPayload   func(contract.Request, string) (map[string]any, error)
}

func areaCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:    "area",
		entityLabel:   "区域",
		invalidReason: "invalid_area_create_payload",
		acceptedFields: semanticParameterPaths(
			semantic.FieldHouseID,
			semantic.FieldName,
			semantic.FieldRoomIDs,
			semantic.FieldParentID,
		),
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"区域名不存在时才创建",
			"创建后通过区域列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			return api.BuildAreaCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, semantic.FieldDescription),
				firstRequestString(request.Parameters, semantic.FieldIcon),
				firstRequestString(request.Parameters, semantic.FieldParentID),
				requestStringList(request.Parameters[semantic.FieldRoomIDs], request.Parameters[semantic.FieldRoomID]),
			)
		},
	}
}

func groupCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:    "group",
		entityLabel:   "设备组",
		invalidReason: "invalid_group_create_payload",
		acceptedFields: semanticParameterPaths(
			semantic.FieldHouseID,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldGroupCategory,
			semantic.FieldGroupCapability,
			semantic.FieldDeviceIDs,
			semantic.FieldDeviceNames,
		),
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"设备组名不存在时才创建",
			"房间、组件和成员设备必须属于当前家庭且适合加入该设备组",
			"创建后通过设备组列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			componentID, _ := semantic.GroupCapabilityComponentID(request.Parameters)
			payload, err := api.BuildGroupCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, semantic.FieldRoomID),
				requestString(componentID),
				requestStringList(request.Parameters[semantic.FieldDeviceIDs], request.Parameters[semantic.FieldDeviceID]),
				firstRequestString(request.Parameters, semantic.FieldDescription),
				firstRequestString(request.Parameters, semantic.FieldIcon),
			)
			if err != nil {
				return nil, err
			}
			if roomName := firstRequestString(request.Parameters, semantic.FieldRoomName, semantic.FieldTargetRoomName); roomName != "" {
				payload[semantic.FieldRoomName] = roomName
			}
			if names := requestStringList(request.Parameters[semantic.FieldDeviceNames]); len(names) > 0 {
				payload[semantic.FieldDeviceNames] = names
			}
			return payload, nil
		},
	}
}

func sceneCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:    "scene",
		entityLabel:   "情景",
		invalidReason: "invalid_scene_create_payload",
		acceptedFields: append(semanticParameterPaths(
			semantic.FieldHouseID,
			semantic.FieldName,
			semantic.FieldDescription,
			semantic.FieldIcon,
			semantic.FieldActions,
		),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldCustom),
		),
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"情景名不存在时才创建",
			"情景动作资源必须属于当前家庭",
			"创建后通过情景列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			details, ok := normalizeSceneActionRows(request.Parameters[semantic.FieldActions])
			if !ok {
				return nil, fmt.Errorf("scene actions are required")
			}
			return api.BuildSceneCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, semantic.FieldDescription),
				firstRequestString(request.Parameters, semantic.FieldIcon),
				details,
			)
		},
	}
}

func automationCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:    "automation",
		entityLabel:   "自动化",
		invalidReason: "invalid_automation_create_payload",
		acceptedFields: append(semanticParameterPaths(
			semantic.FieldHouseID,
			semantic.FieldName,
			semantic.FieldActiveWindow,
			semantic.FieldRepeat,
			semantic.FieldTrigger,
			semantic.FieldConditions,
			semantic.FieldActions,
		),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet),
		),
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"自动化名不存在时才创建",
			"条件结构和动作资源需通过 owner-reviewed 自动化校验器",
			"创建后通过自动化列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			schedule, ok := semantic.AutomationScheduleFromRequest(request.Parameters)
			if !ok {
				return nil, fmt.Errorf("repeat is required")
			}
			version, _ := requestInt(request.Parameters[semantic.FieldVersion])
			var statusPtr *int
			if status, ok := requestInt(request.Parameters[semantic.FieldStatus]); ok {
				statusPtr = &status
			}
			actions, ok := normalizeAutomationActionRows(request.Parameters[semantic.FieldActions])
			if !ok {
				return nil, fmt.Errorf("actions are required")
			}
			return api.BuildAutomationCreatePayload(
				houseID,
				configureName(request),
				schedule.StartTime,
				schedule.EndTime,
				schedule.RepeatType,
				schedule.RepeatValue,
				normalizeAutomationParamsFromRequest(request.Parameters),
				actions,
				version,
				statusPtr,
			)
		},
	}
}

func configureName(request contract.Request) string {
	return firstRequestString(request.Parameters, semantic.FieldName)
}

func requestStringList(values ...any) []string {
	result := []string{}
	for _, value := range values {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				result = append(result, requestString(item))
			}
		case []string:
			result = append(result, typed...)
		case nil:
		default:
			result = append(result, requestString(typed))
		}
	}
	compacted := make([]string, 0, len(result))
	for _, value := range result {
		if strings.TrimSpace(value) != "" {
			compacted = append(compacted, strings.TrimSpace(value))
		}
	}
	return compacted
}

func requestMapList(value any) ([]map[string]any, bool) {
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

func requestMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return typed
}

func sceneSingleDetail(request contract.Request) (map[string]any, bool) {
	resID := firstRequestString(request.Parameters, semantic.FieldTargetID, semantic.FieldDeviceID)
	params, _ := firstPresent(request.Parameters, semantic.FieldSet)
	if resID == "" || params == nil {
		return nil, false
	}
	params = normalizeLightActionParams(params)
	compact, err := compactJSONForRuntime(params)
	if err != nil {
		return nil, false
	}
	typeID := requestNumberOrDefault(request.Parameters[semantic.FieldTargetTypeID], semantic.ResourceDevice)
	if targetType := firstRequestString(request.Parameters, semantic.FieldTargetType, semantic.FieldEntityType); targetType != "" {
		if parsed, ok := semanticTargetTypeID(targetType, groupTypeMesh); ok {
			typeID = parsed
		}
	}
	detail := map[string]any{
		semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType): typeID,
		semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID):   requestNumberOrString(resID),
		semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName): firstNonEmptyString(firstRequestString(request.Parameters, semantic.FieldTargetName), resID),
		semantic.FieldAction:                 requestNumberOrDefault(request.Parameters[semantic.FieldAction], 0),
		semantic.FieldRank:                   requestNumberOrDefault(request.Parameters[semantic.FieldRank], 0),
		semantic.InternalActionParamsField(): compact,
	}
	return detail, true
}

func requestString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}

func requestBool(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1", "yes", "y", "on":
				return true
			case "false", "0", "no", "n", "off":
				return false
			}
		case float64:
			return typed != 0
		case int:
			return typed != 0
		}
	}
	return false
}

func requestInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case string:
		var result int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &result); err != nil {
			return 0, false
		}
		return result, true
	default:
		return 0, false
	}
}

func requestNumberOrDefault(value any, fallback int) any {
	if parsed, ok := requestInt(value); ok {
		return parsed
	}
	return fallback
}

func requestNumberOrString(value string) any {
	if parsed, ok := requestInt(value); ok {
		return parsed
	}
	return value
}

func compactJSONForRuntime(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return "", err
		}
		data, err := json.Marshal(decoded)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
