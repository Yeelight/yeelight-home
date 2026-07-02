package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunSceneList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/r/all", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("scene list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "scene.list",
		Data: map[string]any{
			semantic.FieldScenes: projectSceneRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunAutomationList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/automations/r/list", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("automation list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "automation.list",
		Data: map[string]any{
			semantic.FieldAutomations: projectAutomationRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunDeviceVirtualCountGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.virtual_count.get", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "device.virtual_count.get", "/v1/device/r/"+pathSegment(houseID)+"/virturlNum", http.MethodPost, nil, map[string]any{semantic.FieldVirtualDeviceCount: nil})
}

func (client MetadataReadonlyClient) RunNodeSortedDeviceList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	targetType := strings.TrimSpace(stringFromAny(request.Parameters[semantic.FieldTargetType]))
	resType := strings.TrimSpace(firstNonEmpty(
		resourceTypeIDString(targetType),
		stringFromAny(request.Parameters["nodeType"]),
	))
	resID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldTargetID]),
		stringFromAny(request.Parameters[semantic.FieldNodeID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
	))
	if resType == "" || resID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "node.sorted_device.list", "node_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.callWithHouseHeader(ctx, http.MethodPost, "/v1/node/r/"+pathSegment(resType)+"/"+pathSegment(resID)+"/device", nil, request.Credentials, houseID)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, "", "node.sorted_device.list", response), nil
	}
	devices := projectSortedDeviceRows(response["data"])
	if resType == "1" {
		if enriched, err := client.enrichSortedDeviceRows(ctx, houseID, devices, request.Credentials); err == nil {
			devices = enriched
		}
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "node.sorted_device.list",
		Data: map[string]any{
			semantic.FieldTargetType: firstNonEmpty(semantic.ResourceTypeName(resType), targetType),
			semantic.FieldTargetID:   resID,
			semantic.FieldDevices:    devices,
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectSceneRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.SceneRowContainers()...)
	scenes := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		scene := map[string]any{}
		copyResponseStringMappings(scene, item, semantic.SceneSummaryMappings())
		if details, ok := item[semantic.FieldDetails].([]any); ok {
			scene[semantic.FieldActionCount] = len(details)
		}
		if len(scene) > 0 {
			scenes = append(scenes, scene)
		}
	}
	return scenes
}

func projectAutomationRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.AutomationRowContainers()...)
	automations := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		automation := map[string]any{}
		copyResponseStringMappings(automation, item, semantic.AutomationSummaryMappings())
		start := strings.TrimSpace(firstAnyString(item, semantic.FieldStartTime))
		end := strings.TrimSpace(firstAnyString(item, semantic.FieldEndTime))
		if start != "" || end != "" {
			window := map[string]any{}
			if start != "" {
				window[semantic.FieldStart] = start
			}
			if end != "" {
				window[semantic.FieldEnd] = end
			}
			automation[semantic.FieldActiveWindow] = window
		}
		if repeat := publicRepeat(item); repeat != nil {
			automation[semantic.FieldRepeat] = repeat
		}
		if params := readableAutomationConditions(firstNonNil(item[semantic.InternalAutomationParamsField()], item[semantic.FieldSet])); params != nil {
			putAutomationConditions(automation, params)
		}
		if actions := editableActionList(firstNonNil(item[semantic.FieldActions], item[semantic.FieldDetails])); len(actions) > 0 {
			automation[semantic.FieldActions] = actions
			automation[semantic.FieldActionCount] = len(actions)
		} else if actions, ok := item[semantic.FieldActions].([]any); ok {
			automation[semantic.FieldActionCount] = len(actions)
		}
		if len(automation) > 0 {
			automations = append(automations, automation)
		}
	}
	return automations
}

func readableAutomationConditions(value any) any {
	parsed := editableJSONValue(value)
	switch parsed.(type) {
	case map[string]any:
		return parsed
	default:
		return nil
	}
}

func projectAutomationPage(data any) map[string]any {
	page := map[string]any{
		semantic.FieldEntries: projectAutomationRows(data),
	}
	if total := firstAutomationPageValue(data, semantic.FieldTotal); total != nil {
		page[semantic.FieldTotal] = sanitizeCloudData(total)
	}
	return page
}

func firstAutomationPageValue(data any, keys ...string) any {
	item, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			return value
		}
	}
	for _, container := range semantic.AutomationRowContainers() {
		if nested, ok := item[container].(map[string]any); ok {
			for _, key := range keys {
				if value, ok := nested[key]; ok && value != nil {
					return value
				}
			}
		}
	}
	return nil
}

func projectAutomationRuleRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.FieldRules, "rules", "list", "rows")
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		rule := map[string]any{}
		for _, key := range []string{
			semantic.FieldID,
			semantic.FieldName,
			semantic.FieldHouseID,
			semantic.FieldStatus,
			semantic.FieldValid,
			semantic.FieldVersion,
			semantic.FieldGatewayDeviceID,
			semantic.FieldTargetID,
			semantic.FieldTargetType,
		} {
			if value, ok := item[key]; ok {
				rule[key] = sanitizeCloudData(value)
			}
		}
		if value, ok := item[semantic.FieldCreateTime]; ok {
			rule[semantic.FieldCreatedAt] = sanitizeCloudData(value)
		}
		if value, ok := item["updateTime"]; ok {
			rule[semantic.FieldUpdatedAt] = sanitizeCloudData(value)
		}
		if set, ok := item[semantic.FieldSet].(map[string]any); ok {
			if conditionGroup, ok := set["if"].(map[string]any); ok {
				putAutomationRuleConditions(rule, conditionGroup)
			}
			if actionGroup, ok := set["then"].(map[string]any); ok {
				if actions := publicAutomationRuleActions(actionGroup); len(actions) > 0 {
					rule[semantic.FieldActions] = actions
					rule[semantic.FieldActionCount] = len(actions)
				}
			}
		}
		if len(rule) > 0 {
			result = append(result, rule)
		}
	}
	return result
}

func putAutomationRuleConditions(target map[string]any, conditionGroup map[string]any) {
	if rows, ok := conditionGroup[semantic.FieldConditions].([]any); ok {
		filtered := make([]any, 0, len(rows))
		for _, raw := range rows {
			item, ok := raw.(map[string]any)
			if !ok {
				filtered = append(filtered, raw)
				continue
			}
			if strings.TrimSpace(stringFromAny(item[semantic.FieldType])) == "time" {
				putAutomationRuleTimeWindow(target, item)
				continue
			}
			filtered = append(filtered, publicAutomationRuleConditionRow(item))
		}
		copied := map[string]any{}
		for key, value := range conditionGroup {
			copied[key] = value
		}
		copied[semantic.FieldConditions] = filtered
		conditionGroup = copied
	}
	putAutomationConditions(target, conditionGroup)
}

func publicAutomationRuleConditionRow(row map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range row {
		switch key {
		case semantic.InternalRepeatTypeField(), semantic.InternalRepeatValueField(), "weekdays", "repeat_value", "repeat_type":
			continue
		default:
			result[key] = value
		}
	}
	return result
}

func putAutomationRuleTimeWindow(target map[string]any, row map[string]any) {
	start := strings.TrimSpace(firstAnyString(row, "after", semantic.FieldStart, semantic.FieldStartTime))
	end := strings.TrimSpace(firstAnyString(row, "before", semantic.FieldEnd, semantic.FieldEndTime))
	if start != "" || end != "" {
		window := map[string]any{}
		if start != "" {
			window[semantic.FieldStart] = start
		}
		if end != "" {
			window[semantic.FieldEnd] = end
		}
		target[semantic.FieldActiveWindow] = window
	}
	if repeat := publicRepeat(map[string]any{
		semantic.InternalRepeatTypeField():  firstNonNil(row[semantic.InternalRepeatTypeField()], row["repeat_type"]),
		semantic.InternalRepeatValueField(): firstNonNil(row[semantic.InternalRepeatValueField()], row["weekdays"], row["repeat_value"]),
	}); repeat != nil {
		target[semantic.FieldRepeat] = repeat
	}
}

func publicAutomationRuleActions(actionGroup map[string]any) []any {
	rows := rowsFromData(firstNonNil(actionGroup["nodes"], actionGroup[semantic.FieldActions], actionGroup[semantic.FieldItems]))
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		action := map[string]any{}
		if targetID := firstNonNil(item[semantic.FieldID], item[semantic.FieldTargetID]); targetID != nil {
			action[semantic.FieldTargetID] = sanitizeCloudData(targetID)
		}
		if targetType := firstNonNil(item["nt"], item[semantic.FieldTargetType]); targetType != nil {
			if name := semantic.ResourceTypeName(targetType); name != "" {
				action[semantic.FieldTargetType] = name
			} else {
				action[semantic.FieldTargetType] = sanitizeCloudData(targetType)
			}
		}
		for _, key := range []string{semantic.FieldDelay, semantic.FieldDuration, semantic.FieldRank} {
			if value, ok := item[key]; ok {
				action[key] = sanitizeCloudData(value)
			}
		}
		if set, ok := item[semantic.FieldSet].(map[string]any); ok {
			action[semantic.FieldSet] = semantic.ToPublicLightSet(set)
		}
		if len(action) > 0 {
			result = append(result, action)
		}
	}
	return result
}

func resourceTypeIDString(targetType string) string {
	if typeID, ok := semantic.TargetTypeID(targetType, semantic.ResourceMeshGroup); ok {
		return stringFromAny(typeID)
	}
	return ""
}

func projectSortedDeviceRows(data any) []any {
	rows := rowsFromData(data)
	devices := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		device := map[string]any{}
		copyResponseStringMappings(device, item, semantic.SortedDeviceSummaryMappings())
		projectSortedDeviceIdentity(device, item)
		if len(device) > 0 {
			devices = append(devices, device)
		}
	}
	return devices
}

func projectSortedDeviceIdentity(device map[string]any, item map[string]any) {
	targetID := firstAnyString(item,
		semantic.FieldTargetID,
		semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID),
		semantic.FieldDeviceID,
		semantic.FieldID,
	)
	if targetID != "" {
		device[semantic.FieldTargetID] = targetID
		if _, ok := device[semantic.FieldID]; !ok {
			device[semantic.FieldID] = targetID
		}
	}
	targetType := firstAnyString(item, semantic.FieldTargetType)
	if targetType == "" {
		targetType = cloudResourceTypeName(item)
	}
	if targetType == "" && firstAnyString(item, semantic.FieldDeviceID) != "" {
		targetType = semantic.ResourceTypeName(semantic.ResourceDevice)
	}
	if targetType != "" {
		device[semantic.FieldTargetType] = targetType
	}
}

func copyResponseStringMappings(output map[string]any, item map[string]any, mappings []semantic.ResponseFieldMapping) {
	for _, mapping := range mappings {
		copyFirstString(output, mapping.Public, item, mapping.Internal)
	}
}

func copyFirstString(output map[string]any, outputKey string, item map[string]any, inputKeys []string) {
	for _, inputKey := range inputKeys {
		if value := stringFromAny(item[inputKey]); value != "" {
			output[outputKey] = value
			return
		}
	}
}
