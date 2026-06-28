package main

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func entityListResponse(request contract.Request, result api.EntityListResult) contract.Response {
	entities := make([]any, 0, len(result.Entities))
	for _, entity := range result.Entities {
		entities = append(entities, entitySummaryMap(entity))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到 %d 个实体。", result.Total),
		Result: map[string]any{
			"region":   result.Region,
			"houseId":  result.HouseID,
			"total":    result.Total,
			"counts":   result.Counts,
			"entities": entities,
		},
		Warnings: result.Warnings,
		TraceID:  "entity-list-readonly",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(result),
			"cacheHits": 0,
		},
	}
}

func entityGetResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary, matchedBy string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到实体：%s。", entity.Name),
		Result: map[string]any{
			"region":    result.Region,
			"houseId":   result.HouseID,
			"entity":    entitySummaryMap(entity),
			"matchedBy": matchedBy,
		},
		Warnings: result.Warnings,
		TraceID:  "entity-get-readonly",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(result),
			"cacheHits": 0,
		},
	}
}

func entityGetClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
	preview := make([]any, 0, len(candidates))
	for index, candidate := range candidates {
		if index >= 5 {
			break
		}
		preview = append(preview, entitySummaryMap(candidate))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请明确要查看的家庭实体，例如房间、设备、情景或自动化名称。",
		Clarification: map[string]any{
			"reason":               reason,
			"target":               target.toMap(),
			"candidates":           preview,
			"supportedEntityTypes": []string{"home", "room", "gateway", "group", "device", "scene", "automation"},
		},
		Warnings: []string{},
		TraceID:  "entity-get-clarification",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func entityCapabilitiesResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已返回 %s 的当前可确认能力边界。", entity.Name),
		Result: map[string]any{
			"region":           result.Region,
			"houseId":          result.HouseID,
			"entity":           entitySummaryMap(entity),
			"capabilitySource": "entity.list_projection",
			"schemaStatus":     "not_connected",
			"operations": map[string]any{
				"read":  []string{"entity.get", "entity.list", "entity.capabilities"},
				"write": []string{},
			},
			"limitations": []string{
				"当前 Runtime 尚未接入设备实例级 product schema adapter。",
				"不会根据静态物模型推断设备控制能力。",
				"写操作只通过对应语义 intent 执行，不接受由模型拼接的原始 payload。",
			},
		},
		Warnings: []string{"设备实例级能力需要后续接入 getProductSchema(pid, houseId, deviceId) 等价只读接口后才能返回。"},
		TraceID:  "entity-capabilities-readonly",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(result),
			"cacheHits": 0,
		},
	}
}

func entityDeviceCapabilitiesResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary, capabilities api.DeviceCapabilitiesResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已返回 %s 的当前可确认设备能力。", entity.Name),
		Result: map[string]any{
			"region":           result.Region,
			"houseId":          result.HouseID,
			"entity":           entitySummaryMap(entity),
			"capabilitySource": capabilities.CapabilitySource,
			"schemaStatus":     capabilities.SchemaStatus,
			"deviceSchema":     deviceCapabilityMap(capabilities.Device),
			"operations": map[string]any{
				"read":  []string{"entity.get", "entity.list", "entity.capabilities"},
				"write": []string{},
			},
			"limitations": []string{
				"当前仅返回设备实例级 schema 的安全摘要。",
				"写操作只通过对应语义 intent 执行，不接受由模型拼接的原始 payload。",
				"不会根据静态物模型或设备名称推断额外控制能力。",
			},
		},
		Warnings: []string{"设备 schema 已接入只读摘要，但普通写操作仍未启用。"},
		TraceID:  "entity-capabilities-readonly",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(result) + 1,
			"cacheHits": 0,
		},
	}
}

func entityCapabilitiesFallbackResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary, warning string) contract.Response {
	response := entityCapabilitiesResponse(request, result, entity)
	if warning != "" {
		response.Warnings = append(response.Warnings, warning)
	}
	return response
}

func entityCapabilitiesClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
	response := entityGetClarificationResponse(request, reason, target, candidates, apiCalls)
	response.TraceID = "entity-capabilities-clarification"
	return response
}

func entitySummaryMap(entity api.EntitySummary) map[string]any {
	item := map[string]any{
		"type": entity.Type,
		"id":   entity.ID,
		"name": entity.Name,
	}
	if entity.HouseID != "" {
		item["houseId"] = entity.HouseID
	}
	if entity.RoomID != "" {
		item["roomId"] = entity.RoomID
	}
	if entity.Online != nil {
		item["online"] = *entity.Online
	}
	if entity.Status != "" {
		item["status"] = entity.Status
	}
	return item
}

func deviceCapabilityMap(device api.DeviceCapability) map[string]any {
	item := map[string]any{
		"id": device.ID,
	}
	addString(item, "name", device.Name)
	addString(item, "pid", device.PID)
	addString(item, "pcId", device.PCID)
	addString(item, "cid", device.CID)
	addString(item, "category", device.Category)
	addString(item, "roomId", device.RoomID)
	addString(item, "nodeType", device.NodeType)
	if len(device.Properties) > 0 {
		item["properties"] = propertyCapabilityList(device.Properties)
	}
	if len(device.Components) > 0 {
		components := make([]any, 0, len(device.Components))
		for _, component := range device.Components {
			components = append(components, componentCapabilityMap(component))
		}
		item["components"] = components
	}
	if len(device.Events) > 0 {
		item["events"] = eventCapabilityList(device.Events)
	}
	if len(device.Actions) > 0 {
		item["actions"] = actionCapabilityList(device.Actions)
	}
	return item
}

func componentCapabilityMap(component api.ComponentCapability) map[string]any {
	item := map[string]any{}
	addString(item, "id", component.ID)
	addString(item, "index", component.Index)
	addString(item, "name", component.Name)
	addString(item, "type", component.Type)
	addString(item, "category", component.Category)
	if len(component.Properties) > 0 {
		item["properties"] = propertyCapabilityList(component.Properties)
	}
	if len(component.Events) > 0 {
		item["events"] = eventCapabilityList(component.Events)
	}
	if len(component.Actions) > 0 {
		item["actions"] = actionCapabilityList(component.Actions)
	}
	return item
}

func propertyCapabilityList(properties []api.PropertyCapability) []any {
	items := make([]any, 0, len(properties))
	for _, property := range properties {
		item := map[string]any{"id": property.ID}
		addString(item, "description", property.Description)
		addString(item, "access", property.Access)
		addString(item, "format", property.Format)
		addString(item, "unit", property.Unit)
		addString(item, "type", property.Type)
		if property.Range != nil {
			item["range"] = map[string]any{
				"min":  property.Range.Min,
				"max":  property.Range.Max,
				"step": property.Range.Step,
			}
		}
		if len(property.ValueList) > 0 {
			values := make([]any, 0, len(property.ValueList))
			for _, value := range property.ValueList {
				valueItem := map[string]any{"code": value.Code}
				addString(valueItem, "desc", value.Desc)
				values = append(values, valueItem)
			}
			item["valueList"] = values
		}
		if len(property.Operators) > 0 {
			item["operators"] = property.Operators
		}
		items = append(items, item)
	}
	return items
}

func eventCapabilityList(events []api.EventCapability) []any {
	items := make([]any, 0, len(events))
	for _, event := range events {
		item := map[string]any{}
		addString(item, "id", event.ID)
		addString(item, "typeId", event.TypeID)
		addString(item, "name", event.Name)
		if len(event.Params) > 0 {
			item["params"] = propertyCapabilityList(event.Params)
		}
		items = append(items, item)
	}
	return items
}

func actionCapabilityList(actions []api.ActionCapability) []any {
	items := make([]any, 0, len(actions))
	for _, action := range actions {
		item := map[string]any{"id": action.ID}
		if len(action.Params) > 0 {
			item["params"] = propertyCapabilityList(action.Params)
		}
		items = append(items, item)
	}
	return items
}

func addString(item map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		item[key] = strings.TrimSpace(value)
	}
}

func entityListAPICalls(result api.EntityListResult) int {
	if result.APICalls > 0 {
		return result.APICalls
	}
	if result.HouseID == "" {
		return 1
	}
	return api.HouseScopedEntityListCallCount()
}

func requestHouseID(request contract.Request) string {
	if value, ok := request.Parameters["houseId"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := request.HomeRef["id"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return ""
}

type entityGetTarget struct {
	id         string
	name       string
	entityType string
}

func (target entityGetTarget) toMap() map[string]any {
	result := map[string]any{}
	if target.id != "" {
		result["id"] = target.id
	}
	if target.name != "" {
		result["name"] = target.name
	}
	if target.entityType != "" {
		result["entityType"] = target.entityType
	}
	return result
}

func entityGetTargetFromRequest(request contract.Request) entityGetTarget {
	target := entityGetTarget{
		id: firstRequestString(request.Parameters,
			"entityId", "entityID", "deviceId", "deviceID", "roomId", "roomID", "areaId", "areaID",
			"groupId", "groupID", "sceneId", "sceneID", "automationId", "automationID", "gatewayId", "gatewayID", "id",
		),
		name:       firstRequestString(request.Parameters, "entityName", "deviceName", "roomName", "areaName", "groupName", "sceneName", "automationName", "gatewayName", "name"),
		entityType: firstRequestString(request.Parameters, "entityType", "type"),
	}
	if target.entityType == "" {
		target.entityType = entityTypeFromTargetParameters(request.Parameters)
	}
	if len(request.Targets) == 0 {
		return target
	}
	firstTarget := request.Targets[0]
	return entityGetTarget{
		id:         firstNonEmptyString(firstRequestString(firstTarget, "id", "entityId", "entityID"), target.id),
		name:       firstNonEmptyString(firstRequestString(firstTarget, "name", "entityName"), target.name),
		entityType: firstNonEmptyString(firstRequestString(firstTarget, "entityType", "type"), target.entityType),
	}
}

func entityTypeFromTargetParameters(parameters map[string]any) string {
	for _, candidate := range []struct {
		keys       []string
		entityType string
	}{
		{[]string{"deviceId", "deviceID", "gatewayId", "gatewayID"}, "device"},
		{[]string{"roomId", "roomID"}, "room"},
		{[]string{"areaId", "areaID"}, "area"},
		{[]string{"groupId", "groupID"}, "group"},
		{[]string{"sceneId", "sceneID"}, "scene"},
		{[]string{"automationId", "automationID"}, "automation"},
	} {
		for _, key := range candidate.keys {
			if firstRequestString(parameters, key) != "" {
				return candidate.entityType
			}
		}
	}
	return ""
}

func findEntity(target entityGetTarget, entities []api.EntitySummary) (api.EntitySummary, []api.EntitySummary, string) {
	candidates := make([]api.EntitySummary, 0)
	if target.id != "" {
		for _, entity := range entities {
			if entity.ID == target.id && entityTypeMatches(target.entityType, entity.Type) {
				return entity, []api.EntitySummary{entity}, "id"
			}
		}
		return api.EntitySummary{}, candidates, "id"
	}
	for _, entity := range entities {
		if entity.Name == target.name && entityTypeMatches(target.entityType, entity.Type) {
			candidates = append(candidates, entity)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], candidates, "name"
	}
	return api.EntitySummary{}, candidates, "name"
}

func entityTypeMatches(expected string, actual string) bool {
	return expected == "" || expected == actual
}

func firstRequestString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
