package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
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
		entityType:     "area",
		entityLabel:    "区域",
		invalidReason:  "invalid_area_create_payload",
		acceptedFields: []string{"parameters.houseId", "parameters.name", "parameters.roomIds", "parameters.parentId"},
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"区域名不存在时才创建",
			"创建后通过区域列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			return api.BuildAreaCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, "description", "desc"),
				firstRequestString(request.Parameters, "icon"),
				firstRequestString(request.Parameters, "parentId", "parent_id"),
				requestStringList(request.Parameters["roomIds"], request.Parameters["roomId"]),
			)
		},
	}
}

func groupCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:     "group",
		entityLabel:    "设备组",
		invalidReason:  "invalid_group_create_payload",
		acceptedFields: []string{"parameters.houseId", "parameters.name", "parameters.roomId", "parameters.cid", "parameters.deviceIds"},
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"设备组名不存在时才创建",
			"房间、组件和成员设备必须属于当前家庭且适合加入该设备组",
			"创建后通过设备组列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			return api.BuildGroupCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, "roomId", "room_id"),
				firstRequestString(request.Parameters, "cid", "componentId", "component_id"),
				requestStringList(request.Parameters["deviceIds"], request.Parameters["deviceId"]),
				firstRequestString(request.Parameters, "description", "desc"),
				firstRequestString(request.Parameters, "icon"),
			)
		},
	}
}

func sceneCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:     "scene",
		entityLabel:    "情景",
		invalidReason:  "invalid_scene_create_payload",
		acceptedFields: []string{"parameters.houseId", "parameters.name", "parameters.details"},
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"情景名不存在时才创建",
			"情景动作资源必须属于当前家庭",
			"创建后通过情景列表按名称验证",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			details, ok := requestMapList(request.Parameters["details"])
			if !ok {
				detail, ok := sceneSingleDetail(request)
				if !ok {
					return nil, fmt.Errorf("scene details are required")
				}
				details = []map[string]any{detail}
			}
			return api.BuildSceneCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, "description", "desc"),
				firstRequestString(request.Parameters, "icon"),
				details,
			)
		},
	}
}

func automationCreateSpec() configureCreateSpec {
	return configureCreateSpec{
		entityType:     "automation",
		entityLabel:    "自动化",
		invalidReason:  "invalid_automation_create_payload",
		acceptedFields: []string{"parameters.houseId", "parameters.name", "parameters.startTime", "parameters.endTime", "parameters.repeatType", "parameters.params", "parameters.actions"},
		preconditions: []string{
			"提交前重新读取家庭实体列表",
			"自动化名不存在时才创建",
			"条件结构和动作资源需通过 owner-reviewed 自动化校验器",
			"当前版本仅生成计划，真实提交保持阻断",
		},
		buildPayload: func(request contract.Request, houseID string) (map[string]any, error) {
			repeatType, ok := requestInt(request.Parameters["repeatType"])
			if !ok {
				return nil, fmt.Errorf("repeatType is required")
			}
			version, _ := requestInt(request.Parameters["version"])
			var statusPtr *int
			if status, ok := requestInt(request.Parameters["status"]); ok {
				statusPtr = &status
			}
			actions, ok := requestMapList(request.Parameters["actions"])
			if !ok {
				return nil, fmt.Errorf("actions are required")
			}
			return api.BuildAutomationCreatePayload(
				houseID,
				configureName(request),
				firstRequestString(request.Parameters, "startTime", "start_time"),
				firstRequestString(request.Parameters, "endTime", "end_time"),
				repeatType,
				firstRequestString(request.Parameters, "repeatValue", "repeat_value"),
				request.Parameters["params"],
				actions,
				version,
				statusPtr,
			)
		},
	}
}

func configureName(request contract.Request) string {
	return firstRequestString(request.Parameters, "name", "areaName", "groupName", "sceneName", "automationName")
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

func sceneSingleDetail(request contract.Request) (map[string]any, bool) {
	resID := firstRequestString(request.Parameters, "resId", "deviceId", "entityId")
	params := request.Parameters["params"]
	if resID == "" || params == nil {
		return nil, false
	}
	compact, err := compactJSONForRuntime(params)
	if err != nil {
		return nil, false
	}
	detail := map[string]any{
		"typeId":  requestNumberOrDefault(request.Parameters["typeId"], 2),
		"resId":   requestNumberOrString(resID),
		"resName": firstNonEmptyString(firstRequestString(request.Parameters, "resName", "deviceName", "entityName"), resID),
		"action":  requestNumberOrDefault(request.Parameters["action"], 0),
		"rank":    requestNumberOrDefault(request.Parameters["rank"], 0),
		"params":  compact,
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
