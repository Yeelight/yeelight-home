package main

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func entityListResponse(request contract.Request, result api.EntityListResult) contract.Response {
	filteredEntities, filteredCounts, filtersApplied := filterEntityListForRequest(request, result.Entities)
	entities := make([]any, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entities = append(entities, entitySummaryMap(entity))
	}
	total := len(filteredEntities)
	counts := result.Counts
	if filtersApplied {
		counts = filteredCounts
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到 %d 个实体。", total),
		Result: map[string]any{
			semantic.FieldRegion:   result.Region,
			semantic.FieldHouseID:  result.HouseID,
			semantic.FieldTotal:    total,
			semantic.FieldCounts:   counts,
			semantic.FieldEntities: entities,
		},
		Warnings: result.Warnings,
		TraceID:  "entity-list-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(result),
			semantic.FieldCacheHits: 0,
		},
	}
}

func filterEntityListForRequest(request contract.Request, entities []api.EntitySummary) ([]api.EntitySummary, map[string]int, bool) {
	if !entityListRequestHasFilters(request) {
		return entities, nil, false
	}
	target := entityGetTargetFromRequest(request)
	filtered := make([]api.EntitySummary, 0, len(entities))
	for _, entity := range entities {
		if target.entityType != "" && !entityTypeMatches(target.entityType, entity.Type) {
			continue
		}
		if (target.roomID != "" || target.roomName != "") && !entityRoomMatches(target, entity, entities) {
			continue
		}
		filtered = append(filtered, entity)
	}
	if target.name != "" && target.entityType != "room" {
		ranked := semantic.RankNameMatches(target.name, filtered, func(entity api.EntitySummary) string {
			return entity.Name
		})
		nameFiltered := make([]api.EntitySummary, 0, len(ranked))
		seen := map[string]bool{}
		for _, item := range ranked {
			nameFiltered = append(nameFiltered, item.Value)
			seen[item.Value.ID] = true
		}
		for _, entity := range filtered {
			if seen[entity.ID] {
				continue
			}
			if semantic.NameKeywordMatches(target.name, entity.Name) {
				nameFiltered = append(nameFiltered, entity)
			}
		}
		filtered = nameFiltered
	}
	return filtered, countEntitiesByType(filtered), true
}

func entityListRequestHasFilters(request contract.Request) bool {
	if len(request.Targets) > 0 {
		return true
	}
	if firstRequestString(request.Parameters,
		semantic.FieldEntityType,
		semantic.FieldType,
		semantic.FieldID,
		semantic.FieldEntityID,
		semantic.FieldRoomID,
		semantic.FieldTargetRoomID,
		semantic.FieldRoomName,
		semantic.FieldTargetRoomName,
		semantic.FieldName,
		semantic.FieldEntityName,
		semantic.FieldDeviceName,
		semantic.FieldGroupName,
		semantic.FieldSceneName,
		semantic.FieldAutomationName,
		semantic.FieldAreaName,
	) != "" {
		return true
	}
	return false
}

func countEntitiesByType(entities []api.EntitySummary) map[string]int {
	counts := map[string]int{}
	for _, entity := range entities {
		counts[entity.Type]++
	}
	return counts
}

func entityGetResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary, matchedBy string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到实体：%s。", entity.Name),
		Result: map[string]any{
			semantic.FieldRegion:    result.Region,
			semantic.FieldHouseID:   result.HouseID,
			semantic.FieldEntity:    entitySummaryMap(entity),
			semantic.FieldMatchedBy: matchedBy,
		},
		Warnings: result.Warnings,
		TraceID:  "entity-get-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(result),
			semantic.FieldCacheHits: topologyCacheHits(result),
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
	userMessage := "请明确要查看的家庭实体，例如房间、设备、情景或自动化名称。"
	if len(preview) > 0 {
		if reason == "ambiguous_target" {
			userMessage = "找到多个可能目标，请从候选中选择一个。"
		} else {
			userMessage = "未能高置信确认目标，但找到了可能候选，请选择要查看的实体。"
		}
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     userMessage,
		Clarification: map[string]any{
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"home", "room", "gateway", "group", "device", "scene", "automation"},
		},
		Warnings: []string{},
		TraceID:  "entity-get-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
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
			semantic.FieldRegion:           result.Region,
			semantic.FieldHouseID:          result.HouseID,
			semantic.FieldEntity:           entitySummaryMap(entity),
			semantic.FieldCapabilitySource: "entity.list_projection",
			semantic.FieldSchemaStatus:     "not_connected",
			semantic.FieldOperations: map[string]any{
				semantic.FieldRead:  []string{"entity.get", "entity.list", "entity.capabilities"},
				semantic.FieldWrite: []string{},
			},
			semantic.FieldLimitations: []string{
				"当前 Runtime 尚不能返回该设备实例的完整能力详情。",
				"不会根据静态物模型推断设备控制能力。",
				"写操作必须通过对应 Runtime intent 执行，并依赖 Runtime 校验和写后验证。",
			},
		},
		Warnings: []string{"设备实例级能力详情需要 Runtime 返回可验证的只读能力证据后才能展示。"},
		TraceID:  "entity-capabilities-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(result),
			semantic.FieldCacheHits: topologyCacheHits(result),
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
			semantic.FieldRegion:           result.Region,
			semantic.FieldHouseID:          result.HouseID,
			semantic.FieldEntity:           entitySummaryMap(entity),
			semantic.FieldCapabilitySource: capabilities.CapabilitySource,
			semantic.FieldSchemaStatus:     capabilities.SchemaStatus,
			semantic.FieldDeviceSchema:     deviceCapabilityMap(capabilities.Device),
			semantic.FieldOperations: map[string]any{
				semantic.FieldRead:  []string{"entity.get", "entity.list", "entity.capabilities"},
				semantic.FieldWrite: []string{},
			},
			semantic.FieldLimitations: []string{
				"当前仅返回设备实例级 schema 的安全摘要。",
				"写操作只通过对应语义 intent 执行，不接受由模型拼接的原始 payload。",
				"不会根据静态物模型或设备名称推断额外控制能力。",
			},
		},
		Warnings: []string{"设备 schema 已接入只读摘要，但普通写操作仍未启用。"},
		TraceID:  "entity-capabilities-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(result) + 1,
			semantic.FieldCacheHits: topologyCacheHits(result),
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
		semantic.FieldEntityType: entity.Type,
		semantic.FieldEntityID:   entity.ID,
		semantic.FieldType:       entity.Type,
		semantic.FieldID:         entity.ID,
		semantic.FieldName:       entity.Name,
	}
	if entity.HouseID != "" {
		item[semantic.FieldHouseID] = entity.HouseID
	}
	if entity.RoomID != "" {
		item[semantic.FieldRoomID] = entity.RoomID
	}
	if entity.GatewayDeviceID != "" {
		item[semantic.FieldGatewayDeviceID] = entity.GatewayDeviceID
	}
	if entity.Online != nil {
		item[semantic.FieldOnline] = *entity.Online
	}
	if entity.Status != "" {
		item[semantic.FieldStatus] = entity.Status
	}
	return item
}

func deviceCapabilityMap(device api.DeviceCapability) map[string]any {
	item := map[string]any{
		semantic.FieldID: device.ID,
	}
	addString(item, semantic.FieldName, device.Name)
	addString(item, semantic.FieldCapabilityProductID, device.ProductID)
	addString(item, semantic.FieldProductCategoryID, device.CategoryID)
	addString(item, semantic.FieldCategory, device.Category)
	addString(item, semantic.FieldRoomID, device.RoomID)
	addString(item, semantic.FieldNodeType, device.NodeType)
	if len(device.Properties) > 0 {
		item[semantic.FieldProperties] = propertyCapabilityList(device.Properties)
	}
	if len(device.Components) > 0 {
		components := make([]any, 0, len(device.Components))
		for _, component := range device.Components {
			components = append(components, componentCapabilityMap(component))
		}
		item[semantic.FieldComponents] = components
	}
	if len(device.Events) > 0 {
		item[semantic.FieldEvents] = eventCapabilityList(device.Events)
	}
	if len(device.Actions) > 0 {
		item[semantic.FieldActions] = actionCapabilityList(device.Actions)
	}
	return item
}

func componentCapabilityMap(component api.ComponentCapability) map[string]any {
	item := map[string]any{}
	addString(item, semantic.FieldID, component.ID)
	addString(item, semantic.FieldIndex, component.Index)
	addString(item, semantic.FieldName, component.Name)
	addString(item, semantic.FieldType, component.Type)
	addString(item, semantic.FieldCategory, component.Category)
	if len(component.Properties) > 0 {
		item[semantic.FieldProperties] = propertyCapabilityList(component.Properties)
	}
	if len(component.Events) > 0 {
		item[semantic.FieldEvents] = eventCapabilityList(component.Events)
	}
	if len(component.Actions) > 0 {
		item[semantic.FieldActions] = actionCapabilityList(component.Actions)
	}
	return item
}

func propertyCapabilityList(properties []api.PropertyCapability) []any {
	items := make([]any, 0, len(properties))
	for _, property := range properties {
		item := map[string]any{semantic.FieldID: semantic.PropertyName(property.ID)}
		addString(item, semantic.FieldDescription, property.Description)
		addString(item, semantic.FieldAccess, property.Access)
		addString(item, semantic.FieldFormat, property.Format)
		addString(item, semantic.FieldUnit, property.Unit)
		addString(item, semantic.FieldType, property.Type)
		if property.Range != nil {
			item[semantic.FieldRange] = map[string]any{
				semantic.FieldMin:  property.Range.Min,
				semantic.FieldMax:  property.Range.Max,
				semantic.FieldStep: property.Range.Step,
			}
		}
		if len(property.ValueList) > 0 {
			values := make([]any, 0, len(property.ValueList))
			for _, value := range property.ValueList {
				valueItem := map[string]any{semantic.FieldCode: value.Code}
				addString(valueItem, semantic.FieldDescription, value.Desc)
				values = append(values, valueItem)
			}
			item[semantic.FieldValueList] = values
		}
		if len(property.Operators) > 0 {
			item[semantic.FieldOperators] = property.Operators
		}
		items = append(items, item)
	}
	return items
}

func eventCapabilityList(events []api.EventCapability) []any {
	items := make([]any, 0, len(events))
	for _, event := range events {
		item := map[string]any{}
		addString(item, semantic.FieldID, event.ID)
		addString(item, semantic.FieldEventTypeID, event.TypeID)
		addString(item, semantic.FieldName, event.Name)
		if len(event.Params) > 0 {
			item[semantic.FieldInputs] = propertyCapabilityList(event.Params)
		}
		items = append(items, item)
	}
	return items
}

func actionCapabilityList(actions []api.ActionCapability) []any {
	items := make([]any, 0, len(actions))
	for _, action := range actions {
		item := map[string]any{semantic.FieldID: action.ID}
		if len(action.Params) > 0 {
			item[semantic.FieldInputs] = propertyCapabilityList(action.Params)
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
	if topologyCacheHits(result) > 0 {
		return 0
	}
	if result.APICalls > 0 {
		return result.APICalls
	}
	if result.HouseID == "" {
		return 1
	}
	return api.HouseScopedEntityListCallCount()
}

func requestHouseID(request contract.Request) string {
	if value, ok := request.Parameters[semantic.FieldHouseID].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := request.HomeRef[semantic.FieldID].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return ""
}

type entityGetTarget struct {
	id         string
	name       string
	entityType string
	roomID     string
	roomName   string
}

func (target entityGetTarget) toMap() map[string]any {
	result := map[string]any{}
	if target.id != "" {
		result[semantic.FieldID] = target.id
	}
	if target.name != "" {
		result[semantic.FieldName] = target.name
	}
	if target.entityType != "" {
		result[semantic.FieldEntityType] = target.entityType
	}
	if target.roomID != "" {
		result[semantic.FieldRoomID] = target.roomID
	}
	if target.roomName != "" {
		result[semantic.FieldRoomName] = target.roomName
	}
	return result
}

func entityGetTargetFromRequest(request contract.Request) entityGetTarget {
	targetEntityType := firstRequestString(request.Parameters, semantic.FieldEntityType, semantic.FieldTargetType, semantic.FieldNodeType, semantic.FieldType)
	if targetEntityType == "" {
		targetEntityType = entityTypeFromIntent(request.Intent)
	}
	if targetEntityType == "" {
		targetEntityType = entityTypeFromTargetParameters(request.Parameters)
	}
	targetID := firstRequestString(request.Parameters, entityIDKeysForType(targetEntityType)...)
	targetID = firstNonEmptyString(targetID, firstRequestString(request.Parameters, semantic.FieldTargetID, semantic.FieldNodeID))
	roomID := firstRequestString(request.Parameters, semantic.FieldTargetRoomID, semantic.FieldRoomID)
	if targetEntityType == "room" {
		targetID = firstNonEmptyString(targetID, roomID)
	}
	targetName := firstRequestString(request.Parameters, entityNameKeysForType(targetEntityType)...)
	targetName = firstNonEmptyString(targetName, firstRequestString(request.Parameters, semantic.FieldTargetName))
	roomName := firstRequestString(request.Parameters, semantic.FieldTargetRoomName, semantic.FieldRoomName)
	if targetEntityType == "room" {
		targetName = firstNonEmptyString(targetName, roomName)
	}
	target := entityGetTarget{
		id:         targetID,
		name:       targetName,
		entityType: targetEntityType,
		roomID:     roomID,
		roomName:   roomName,
	}
	if target.entityType == "" {
		target.entityType = entityTypeFromTargetParameters(request.Parameters)
	}
	if target.entityType == "" && target.name == "" && target.id == "" {
		target.name = roomName
		target.entityType = "room"
	}
	if len(request.Targets) == 0 {
		return target
	}
	firstTarget, roomTarget := selectEntityTarget(request.Targets, target.entityType)
	if firstTarget == nil {
		firstTarget = request.Targets[0]
	}
	if roomTarget != nil {
		target.roomID = firstNonEmptyString(firstRequestString(roomTarget, semantic.FieldID, semantic.FieldEntityID, semantic.FieldRoomID), target.roomID)
		target.roomName = firstNonEmptyString(firstRequestString(roomTarget, semantic.FieldName, semantic.FieldEntityName, semantic.FieldRoomName), target.roomName)
	}
	return entityGetTarget{
		id:         firstNonEmptyString(firstRequestString(firstTarget, semantic.FieldID, semantic.FieldEntityID), target.id),
		name:       firstNonEmptyString(firstRequestString(firstTarget, semantic.FieldName, semantic.FieldEntityName), target.name),
		entityType: firstNonEmptyString(firstRequestString(firstTarget, semantic.FieldEntityType, semantic.FieldType), target.entityType),
		roomID:     firstNonEmptyString(firstRequestString(firstTarget, semantic.FieldRoomID, semantic.FieldTargetRoomID), target.roomID),
		roomName:   firstNonEmptyString(firstRequestString(firstTarget, semantic.FieldRoomName, semantic.FieldTargetRoomName), target.roomName),
	}
}

func selectEntityTarget(targets []map[string]any, preferredType string) (map[string]any, map[string]any) {
	var roomTarget map[string]any
	var firstNonRoom map[string]any
	for _, target := range targets {
		entityType := firstRequestString(target, semantic.FieldEntityType, semantic.FieldType)
		if entityType == "room" {
			if roomTarget == nil {
				roomTarget = target
			}
			continue
		}
		if firstNonRoom == nil {
			firstNonRoom = target
		}
		if preferredType != "" && entityType == preferredType {
			return target, roomTarget
		}
	}
	if preferredType == "room" && roomTarget != nil {
		return roomTarget, roomTarget
	}
	return firstNonRoom, roomTarget
}

func entityTypeFromIntent(intent string) string {
	switch intent {
	case "device.detail.get",
		"device.attr.list",
		"device.energy.summary",
		"device.weather.get",
		"device.storage.get",
		"upgrade.file.list",
		"upgrade.progress.get",
		"upgrade.file.batch_list",
		"node.property_config.get",
		"panel.get",
		"panel.button.type.get",
		"screen.control.list",
		"knob.get",
		"state.query",
		"light.power.set",
		"light.brightness.set",
		"light.brightness.adjust",
		"light.color_temperature.set",
		"light.color_temperature.adjust",
		"light.color.set",
		"lighting.experience.apply":
		return "device"
	case "room.detail.get":
		return "room"
	case "scene.scoped.list":
		return "room"
	case "area.detail.get":
		return "area"
	case "group.detail.get", "meshgroup.detail.get":
		return "group"
	case "scene.detail.get":
		return "scene"
	case "automation.detail.get":
		return "automation"
	case "gateway.detail.get", "gateway.thread.get", "gateway.scene_relation.list", "diagnose.gateway":
		return "gateway"
	case "scene.execute", "scene.test":
		return "scene"
	case "automation.enable", "automation.disable", "automation.explain", "diagnose.automation":
		return "automation"
	case "diagnose.device":
		return "device"
	case "diagnose.scene":
		return "scene"
	case "entity.get", "entity.capabilities":
		return ""
	default:
		return ""
	}
}

func entityNameKeysForType(entityType string) []string {
	switch entityType {
	case "device":
		return []string{semantic.FieldDeviceName, semantic.FieldPanelName, semantic.FieldKnobName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName, semantic.FieldGatewayName, semantic.FieldAreaName, semantic.FieldGroupName, semantic.FieldSceneName, semantic.FieldAutomationName}
	case "area":
		return []string{semantic.FieldAreaName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName}
	case "group":
		return []string{semantic.FieldGroupName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName}
	case "scene":
		return []string{semantic.FieldSceneName, semantic.FieldEntityName, semantic.FieldName}
	case "automation":
		return []string{semantic.FieldAutomationName, semantic.FieldEntityName, semantic.FieldName}
	case "gateway":
		return []string{semantic.FieldGatewayName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName, semantic.FieldDeviceName}
	case "room":
		return []string{semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName}
	case "home":
		return []string{semantic.FieldHomeName, semantic.FieldHouseName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName}
	default:
		return []string{semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldDeviceName, semantic.FieldAreaName, semantic.FieldGroupName, semantic.FieldSceneName, semantic.FieldAutomationName, semantic.FieldGatewayName, semantic.FieldName}
	}
}

func entityIDKeysForType(entityType string) []string {
	switch entityType {
	case "device":
		return []string{semantic.FieldDeviceID, semantic.FieldPanelID, semantic.FieldKnobID, semantic.FieldEntityID, semantic.FieldID}
	case "area":
		return []string{semantic.FieldAreaID, semantic.FieldEntityID, semantic.FieldID}
	case "group":
		return []string{semantic.FieldGroupID, semantic.FieldEntityID, semantic.FieldID}
	case "scene":
		return []string{semantic.FieldSceneID, semantic.FieldEntityID, semantic.FieldID}
	case "automation":
		return []string{semantic.FieldAutomationID, semantic.FieldEntityID, semantic.FieldID}
	case "gateway":
		return []string{semantic.FieldGatewayID, semantic.FieldDeviceID, semantic.FieldEntityID, semantic.FieldID}
	case "room":
		return []string{semantic.FieldRoomID, semantic.FieldEntityID, semantic.FieldID}
	case "home":
		return []string{semantic.FieldHouseID, semantic.FieldEntityID, semantic.FieldID}
	default:
		return []string{semantic.FieldEntityID, semantic.FieldDeviceID, semantic.FieldAreaID, semantic.FieldGroupID, semantic.FieldSceneID, semantic.FieldAutomationID, semantic.FieldGatewayID, semantic.FieldID}
	}
}

func entityTypeFromTargetParameters(parameters map[string]any) string {
	hasNamedDeviceTarget := firstRequestString(parameters, semantic.FieldDeviceName, semantic.FieldGatewayName, semantic.FieldPanelName, semantic.FieldKnobName) != ""
	for _, candidate := range []struct {
		keys       []string
		entityType string
	}{
		{[]string{semantic.FieldDeviceID, semantic.FieldGatewayID}, "device"},
		{[]string{semantic.FieldRoomID, semantic.FieldTargetRoomID}, "room"},
		{[]string{semantic.FieldAreaID}, "area"},
		{[]string{semantic.FieldGroupID}, "group"},
		{[]string{semantic.FieldSceneID}, "scene"},
		{[]string{semantic.FieldAutomationID}, "automation"},
	} {
		for _, key := range candidate.keys {
			if candidate.entityType == "room" && hasNamedDeviceTarget {
				continue
			}
			if firstRequestString(parameters, key) != "" {
				return candidate.entityType
			}
		}
	}
	return ""
}

func findEntity(target entityGetTarget, entities []api.EntitySummary) (api.EntitySummary, []api.EntitySummary, string) {
	if target.id != "" {
		for _, entity := range entities {
			if entity.ID == target.id && entityTypeMatches(target.entityType, entity.Type) && entityRoomMatches(target, entity, entities) {
				return entity, []api.EntitySummary{entity}, "id"
			}
		}
		return api.EntitySummary{}, nil, "id"
	}
	ranked := rankedEntityNameMatches(target, entities)
	if len(ranked) == 0 {
		return api.EntitySummary{}, suggestedEntityNameCandidates(target, entities), "name"
	}
	if ranked[0].Match.Kind == "name" {
		exact := entityMatchCandidatesByKind(ranked, "name")
		if len(exact) == 1 {
			return exact[0], exact, "name"
		}
		return api.EntitySummary{}, exact, "name"
	}
	second := semantic.NameMatch{}
	if len(ranked) > 1 {
		second = ranked[1].Match
	}
	if semantic.NameMatchAutoAccept(ranked[0].Match, second) {
		return ranked[0].Value, []api.EntitySummary{ranked[0].Value}, ranked[0].Match.Kind
	}
	return api.EntitySummary{}, entityMatchCandidates(ranked), ranked[0].Match.Kind
}

func entityTypeMatches(expected string, actual string) bool {
	if expected == "" || expected == actual {
		return true
	}
	return expected == "gateway" && actual == "device"
}

func entityRoomMatches(target entityGetTarget, entity api.EntitySummary, entities []api.EntitySummary) bool {
	if strings.TrimSpace(target.roomID) == "" && strings.TrimSpace(target.roomName) == "" {
		return true
	}
	if isWholeHomeScopeName(target.roomName) {
		return true
	}
	if entity.Type == "room" {
		if target.roomID != "" {
			return entity.ID == target.roomID
		}
		return semantic.NameMatchAutoAccept(semantic.ScoreNameMatch(target.roomName, entity.Name), semantic.NameMatch{})
	}
	if target.roomID != "" {
		return entity.RoomID == target.roomID
	}
	roomID := roomIDByName(target.roomName, entities)
	return roomID != "" && entity.RoomID == roomID
}

func isWholeHomeScopeName(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "全屋", "整屋", "全家", "whole", "whole home", "home":
		return true
	default:
		return false
	}
}

func roomIDByName(roomName string, entities []api.EntitySummary) string {
	match, candidates, _ := findEntity(entityGetTarget{name: roomName, entityType: "room"}, entities)
	if match.ID != "" && len(candidates) == 1 {
		return match.ID
	}
	return ""
}

func rankedEntityNameMatches(target entityGetTarget, entities []api.EntitySummary) []semantic.RankedNameMatch[api.EntitySummary] {
	if strings.TrimSpace(target.name) == "" {
		return nil
	}
	candidates := make([]api.EntitySummary, 0)
	for _, entity := range entities {
		if !entityTypeMatches(target.entityType, entity.Type) || !entityRoomMatches(target, entity, entities) {
			continue
		}
		entityCopy := entity
		candidates = append(candidates, entityCopy)
	}
	return semantic.RankNameMatches(target.name, candidates, func(entity api.EntitySummary) string {
		return entity.Name
	})
}

func suggestedEntityNameCandidates(target entityGetTarget, entities []api.EntitySummary) []api.EntitySummary {
	if strings.TrimSpace(target.name) == "" {
		return nil
	}
	candidates := make([]api.EntitySummary, 0)
	for _, entity := range entities {
		if !entityTypeMatches(target.entityType, entity.Type) || !entityRoomMatches(target, entity, entities) {
			continue
		}
		entityCopy := entity
		candidates = append(candidates, entityCopy)
	}
	ranked := semantic.SuggestNameMatches(target.name, candidates, func(entity api.EntitySummary) string {
		return entity.Name
	}, 5)
	return entityMatchCandidates(ranked)
}

func entityMatchCandidates(ranked []semantic.RankedNameMatch[api.EntitySummary]) []api.EntitySummary {
	candidates := make([]api.EntitySummary, 0, len(ranked))
	for _, item := range ranked {
		candidates = append(candidates, item.Value)
	}
	return candidates
}

func entityMatchCandidatesByKind(ranked []semantic.RankedNameMatch[api.EntitySummary], kind string) []api.EntitySummary {
	candidates := make([]api.EntitySummary, 0)
	for _, item := range ranked {
		if item.Match.Kind == kind {
			candidates = append(candidates, item.Value)
		}
	}
	return candidates
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
