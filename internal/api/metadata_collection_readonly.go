package api

import (
	"context"
	"net/http"
	"strings"
)

func (client MetadataReadonlyClient) RunSceneList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/r/all", map[string]any{"houseId": houseID}, request.Credentials)
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
			"scenes": projectSceneRows(response["data"]),
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
	response, err := client.call(ctx, http.MethodPost, "/v1/automations/r/list", map[string]any{"houseId": houseID}, request.Credentials)
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
			"automations": projectAutomationRows(response["data"]),
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
	return client.readPath(ctx, request, "device.virtual_count.get", "/v1/device/r/"+pathSegment(houseID)+"/virturlNum", http.MethodPost, nil, map[string]any{"virtualDeviceCount": nil})
}

func (client MetadataReadonlyClient) RunNodeSortedDeviceList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	resType := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["resType"]),
		stringFromAny(request.Parameters["nodeType"]),
		stringFromAny(request.Parameters["typeId"]),
	))
	resID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["resId"]),
		stringFromAny(request.Parameters["nodeId"]),
		stringFromAny(request.Parameters["id"]),
	))
	if resType == "" || resID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "node.sorted_device.list", "node_context_missing")
		result.HouseID = strings.TrimSpace(request.HouseID)
		return result, nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/node/r/"+pathSegment(resType)+"/"+pathSegment(resID)+"/device", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("node sorted device list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: "node.sorted_device.list",
		Data: map[string]any{
			"resType": resType,
			"resId":   resID,
			"devices": projectSortedDeviceRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectSceneRows(data any) []any {
	rows := nestedRowsFromData(data, "list", "rows", "scenes", "userscenes")
	scenes := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		scene := map[string]any{}
		for outputKey, inputKeys := range map[string][]string{
			"id":              {"sceneId", "id"},
			"houseId":         {"houseId"},
			"roomId":          {"roomId"},
			"gatewayDeviceId": {"gatewayDeviceId"},
			"name":            {"name"},
			"img":             {"img"},
			"seq":             {"seq"},
			"roomRank":        {"roomRank"},
			"timeInterval":    {"timeInterval"},
		} {
			copyFirstString(scene, outputKey, item, inputKeys)
		}
		if details, ok := item["details"].([]any); ok {
			scene["actionCount"] = len(details)
		}
		if len(scene) > 0 {
			scenes = append(scenes, scene)
		}
	}
	return scenes
}

func projectAutomationRows(data any) []any {
	rows := nestedRowsFromData(data, "list", "rows", "automations")
	automations := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		automation := map[string]any{}
		for outputKey, inputKeys := range map[string][]string{
			"id":          {"id", "automationId"},
			"houseId":     {"houseId"},
			"name":        {"name"},
			"startTime":   {"startTime"},
			"endTime":     {"endTime"},
			"repeatType":  {"repeatType"},
			"repeatValue": {"repeatValue"},
			"status":      {"status"},
			"version":     {"version"},
			"ruleId":      {"ruleId"},
		} {
			copyFirstString(automation, outputKey, item, inputKeys)
		}
		if actions, ok := item["actions"].([]any); ok {
			automation["actionCount"] = len(actions)
		}
		if len(automation) > 0 {
			automations = append(automations, automation)
		}
	}
	return automations
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
		for outputKey, inputKeys := range map[string][]string{
			"id":              {"deviceId", "id"},
			"did":             {"did"},
			"gatewayDeviceId": {"gatewayDeviceId"},
			"pid":             {"pid"},
			"type":            {"type"},
			"name":            {"alias", "name"},
			"img":             {"img"},
			"capability":      {"capability"},
			"isBind":          {"isBind"},
			"roomId":          {"roomId"},
			"houseId":         {"houseId"},
			"isVirtual":       {"isVirtual"},
			"index":           {"index"},
			"rank":            {"rank"},
		} {
			copyFirstString(device, outputKey, item, inputKeys)
		}
		if len(device) > 0 {
			devices = append(devices, device)
		}
	}
	return devices
}

func copyFirstString(output map[string]any, outputKey string, item map[string]any, inputKeys []string) {
	for _, inputKey := range inputKeys {
		if value := stringFromAny(item[inputKey]); value != "" {
			output[outputKey] = value
			return
		}
	}
}
