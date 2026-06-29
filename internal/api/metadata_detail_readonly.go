package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (client MetadataReadonlyClient) RunDeviceDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["id"])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.detail.get", "device_context_missing"), nil
	}
	return client.readPath(ctx, request, "device.detail.get", "/v1/device/"+deviceID+"/r/detail", http.MethodPost, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) RunDeviceAttrList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["id"])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.attr.list", "device_context_missing"), nil
	}
	return client.readPath(ctx, request, "device.attr.list", "/v1/device/r/attrs", http.MethodPost, map[string]any{"ids": deviceID}, map[string]any{"attrs": nil})
}

func (client MetadataReadonlyClient) RunDeviceList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/device/r/all", map[string]any{"houseId": houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("device list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "device.list",
		Data: map[string]any{
			"devices":    projectDeviceRows(response["data"]),
			"meshgroups": projectMeshgroupRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunRoomDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	roomID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["roomId"]), stringFromAny(request.Parameters["id"])))
	if roomID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "room.detail.get", "room_context_missing"), nil
	}
	return client.readPath(ctx, request, "room.detail.get", "/v1/room/"+roomID+"/r/detail", http.MethodPost, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) RunRoomList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "room.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/room/r/all", map[string]any{"houseId": houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("room list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "room.list",
		Data: map[string]any{
			"rooms": projectRoomRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunRoomSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "room.search", "house_context_missing"), nil
	}
	fuzzyName := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["fuzzyName"]),
		stringFromAny(request.Parameters["name"]),
		stringFromAny(request.Parameters["keyword"]),
		stringFromAny(request.Parameters["query"]),
	))
	if fuzzyName == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "room.search", "room_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	body := map[string]any{
		"fuzzyName": fuzzyName,
		"pageNo":    positiveInt(firstNonNil(request.Parameters["pageNo"], request.Parameters["page"]), 1),
		"pageSize":  positiveInt(firstNonNil(request.Parameters["pageSize"], request.Parameters["size"], request.Parameters["limit"]), 20),
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/room/"+pathSegment(houseID)+"/r/fuzzy", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("room search", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "room.search",
		Data: map[string]any{
			"rooms": projectRoomRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunAreaDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "area.detail.get", "house_context_missing"), nil
	}
	areaID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["areaId"]), stringFromAny(request.Parameters["id"]), stringFromAny(request.Parameters["entityId"])))
	if areaID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "area.detail.get", "area_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	return client.readPath(ctx, request, "area.detail.get", "/v2/thing/manage/house/"+pathSegment(houseID)+"/area/"+pathSegment(areaID)+"/r/info", http.MethodGet, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) RunHomeDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.detail.get", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "home.detail.get", "/v1/house/"+houseID+"/r/info", http.MethodGet, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) RunHomeStatGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.stat.get", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "home.stat.get", "/v1/house/"+pathSegment(houseID)+"/r/stat", http.MethodPost, nil, map[string]any{"stat": nil})
}

func (client MetadataReadonlyClient) RunGroupStructureList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.structure.list", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "group.structure.list", "/v1/group/r/all", http.MethodPost, map[string]any{"houseId": houseID}, map[string]any{"groups": nil})
}

func (client MetadataReadonlyClient) RunGroupList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.list", "house_context_missing"), nil
	}
	pageNo := positiveInt(firstNonNil(request.Parameters["pageNo"], request.Parameters["page"]), 1)
	pageSize := positiveInt(firstNonNil(request.Parameters["pageSize"], request.Parameters["size"], request.Parameters["limit"]), 100)
	response, err := client.call(ctx, http.MethodGet, fmt.Sprintf("/v2/thing/manage/house/%s/group/r/info/%d/%d", pathSegment(houseID), pageNo, pageSize), nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("group list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "group.list",
		Data: map[string]any{
			"groups": projectGroupRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGroupSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.search", "house_context_missing"), nil
	}
	fuzzyName := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["fuzzyName"]),
		stringFromAny(request.Parameters["name"]),
		stringFromAny(request.Parameters["keyword"]),
		stringFromAny(request.Parameters["query"]),
	))
	if fuzzyName == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "group.search", "group_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	pageNo := positiveInt(firstNonNil(request.Parameters["pageNo"], request.Parameters["page"]), 1)
	pageSize := positiveInt(firstNonNil(request.Parameters["pageSize"], request.Parameters["size"], request.Parameters["limit"]), 100)
	response, err := client.call(ctx, http.MethodGet, fmt.Sprintf("/v2/thing/manage/house/%s/group/r/info/%d/%d", pathSegment(houseID), pageNo, pageSize), nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("group search", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "group.search",
		Data: map[string]any{
			"groups": filterProjectedRowsByName(projectGroupRows(response["data"]), fuzzyName),
			"query":  map[string]any{"name": fuzzyName, "pageNo": pageNo, "pageSize": pageSize},
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGroupDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.detail.get", "house_context_missing"), nil
	}
	groupID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["groupId"]), stringFromAny(request.Parameters["id"])))
	if groupID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "group.detail.get", "group_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	return client.readPath(ctx, request, "group.detail.get", "/v2/thing/manage/house/"+pathSegment(houseID)+"/group/"+pathSegment(groupID)+"/r/info", http.MethodGet, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) RunSceneDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	sceneID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["sceneId"]), stringFromAny(request.Parameters["id"])))
	if sceneID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.detail.get", "scene_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/"+sceneID+"/r/detail", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, "scene.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, "scene.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   strings.TrimSpace(request.DeviceID),
		Capability: "scene.detail.get",
		Data:       sceneDetailData(response["data"], sceneID),
		RawShape:   responseDataType(response),
		APICalls:   1,
		Warnings:   []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunSceneSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.search", "house_context_missing"), nil
	}
	keyword := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["name"]),
		stringFromAny(request.Parameters["keyword"]),
		stringFromAny(request.Parameters["query"]),
		stringFromAny(request.Parameters["fuzzyName"]),
	))
	if keyword == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "scene.search", "scene_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	body := map[string]any{}
	for _, key := range []string{"name", "fuzzyName", "keyword", "query", "pageNo", "pageSize", "sort", "order", "orderBy"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	if body["name"] == nil {
		body["name"] = keyword
	}
	if body["pageNo"] == nil {
		body["pageNo"] = 1
	}
	if body["pageSize"] == nil {
		body["pageSize"] = 20
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/"+houseID+"/r/fuzzy", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, request.DeviceID, "scene.search", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "scene.search",
		Data: map[string]any{
			"scenes": filterProjectedRowsByName(projectSceneRows(response["data"]), keyword),
			"query":  map[string]any{"name": keyword, "pageNo": body["pageNo"], "pageSize": body["pageSize"]},
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunAutomationSupportedList(ctx context.Context, request MetadataReadonlyRequest, v2 bool) (MetadataReadonlyResult, error) {
	capability := "automation.supported.list"
	path := "/v1/automations/r/supported"
	key := "supported"
	if v2 {
		capability = "automation.supported.v2.list"
		path = "/v1/automations/r/supported/v2"
		key = "supportedV2"
	}
	return client.readPath(ctx, request, capability, path, http.MethodPost, map[string]any{}, map[string]any{key: nil})
}

func (client MetadataReadonlyClient) RunAutomationRuleList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.rule.list", "house_context_missing"), nil
	}
	body := map[string]any{"houseId": houseID}
	for _, key := range []string{"gatewayDeviceId", "name", "status", "valid"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	return client.readPath(ctx, request, "automation.rule.list", "/v1/rule/r/list", http.MethodPost, body, map[string]any{"rules": nil})
}

func (client MetadataReadonlyClient) RunAutomationListPage(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.list.page", "house_context_missing"), nil
	}
	pageNo, pageSize := readonlyPage(request.Parameters, 1, 20)
	return client.readPath(ctx, request, "automation.list.page", "/v1/automations/"+pathSegment(houseID)+"/r/list/"+pageNo+"/"+pageSize, http.MethodGet, nil, map[string]any{"automations": nil})
}

func projectRoomRows(data any) []any {
	rows := rowsFromData(data)
	rooms := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		room := map[string]any{}
		for outputKey, inputKeys := range map[string][]string{
			"id":              {"roomId", "id"},
			"houseId":         {"houseId"},
			"name":            {"name"},
			"capability":      {"capability"},
			"img":             {"img", "icon"},
			"gatewayDeviceId": {"gatewayDeviceId"},
			"seq":             {"seq"},
			"rank":            {"rank"},
		} {
			for _, inputKey := range inputKeys {
				if value := stringFromAny(item[inputKey]); value != "" {
					room[outputKey] = value
					break
				}
			}
		}
		if ids := stringListFromAny(firstNonNil(item["deviceIds"], item["devices"])); len(ids) > 0 {
			room["deviceCount"] = len(ids)
			room["deviceIds"] = ids
		}
		if ids := stringListFromAny(item["gatewayDeviceIds"]); len(ids) > 0 {
			room["gatewayDeviceIds"] = ids
		}
		if len(room) > 0 {
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func stringListFromAny(value any) []string {
	switch typed := value.(type) {
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringFromAny(item); text != "" {
				items = append(items, text)
			}
		}
		return items
	case []string:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				items = append(items, text)
			}
		}
		return items
	default:
		return nil
	}
}

func projectGroupRows(data any) []any {
	rows := rowsFromData(data)
	groups := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		group := map[string]any{}
		for outputKey, inputKeys := range map[string][]string{
			"id":      {"userGroupId", "groupId", "id"},
			"houseId": {"houseId"},
			"name":    {"name", "nane", "groupName"},
			"rank":    {"rank"},
			"icon":    {"icon"},
			"img":     {"img"},
		} {
			for _, inputKey := range inputKeys {
				if value := stringFromAny(item[inputKey]); value != "" {
					group[outputKey] = value
					break
				}
			}
		}
		if ids := stringListFromAny(item["roomIds"]); len(ids) > 0 {
			group["roomCount"] = len(ids)
			group["roomIds"] = ids
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func filterProjectedRowsByName(rows []any, name string) []any {
	keyword := strings.TrimSpace(name)
	if keyword == "" {
		return rows
	}
	filtered := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if strings.Contains(stringFromAny(item["name"]), keyword) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func projectDeviceRows(data any) []any {
	rows := nestedRowsFromData(data, "devices", "rows", "list")
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
			"name":            {"name", "alias", "remark"},
			"img":             {"img"},
			"seq":             {"seq"},
			"position":        {"position"},
			"houseId":         {"houseId"},
			"roomId":          {"roomId"},
			"capability":      {"capability"},
			"roomRank":        {"roomRank"},
			"isBind":          {"isBind"},
			"isVirtual":       {"isVirtual"},
			"connectType":     {"connectType"},
			"typeName":        {"typeName"},
		} {
			for _, inputKey := range inputKeys {
				if value := stringFromAny(item[inputKey]); value != "" {
					device[outputKey] = value
					break
				}
			}
		}
		if ids := stringListFromAny(item["roomIds"]); len(ids) > 0 {
			device["roomIds"] = ids
		}
		if ids := stringListFromAny(item["deviceIds"]); len(ids) > 0 {
			device["childDeviceCount"] = len(ids)
			device["deviceIds"] = ids
		}
		if len(device) > 0 {
			devices = append(devices, device)
		}
	}
	return devices
}

func projectMeshgroupRows(data any) []any {
	rows := nestedRowsFromData(data, "meshgroups", "meshGroups")
	groups := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		group := map[string]any{}
		for outputKey, inputKeys := range map[string][]string{
			"id":      {"meshGroupId", "meshgroupId", "groupId", "id"},
			"houseId": {"houseId"},
			"name":    {"name", "groupName"},
			"roomId":  {"roomId"},
		} {
			for _, inputKey := range inputKeys {
				if value := stringFromAny(item[inputKey]); value != "" {
					group[outputKey] = value
					break
				}
			}
		}
		if ids := stringListFromAny(item["deviceIds"]); len(ids) > 0 {
			group["deviceCount"] = len(ids)
			group["deviceIds"] = ids
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func nestedRowsFromData(data any, keys ...string) []any {
	switch typed := data.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range keys {
			if rows, ok := typed[key].([]any); ok {
				return rows
			}
		}
		return rowsFromData(typed)
	default:
		return []any{}
	}
}

func (client MetadataReadonlyClient) RunAutomationDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.detail.get", "house_context_missing"), nil
	}
	automationID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["automationId"]),
		stringFromAny(request.Parameters["automationID"]),
		stringFromAny(request.Parameters["id"]),
		stringFromAny(request.Parameters["entityId"]),
	))
	if automationID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "automation.detail.get", "automation_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v2/thing/manage/house/"+pathSegment(houseID)+"/automation/"+pathSegment(automationID)+"/r/info", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, "automation.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, "automation.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		DeviceID:   strings.TrimSpace(request.DeviceID),
		Capability: "automation.detail.get",
		Data:       automationDetailData(response["data"], automationID),
		RawShape:   responseDataType(response),
		APICalls:   1,
		Warnings:   []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunSensorList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "sensor.list", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "sensor.list", "/v1/device/r/sensors", http.MethodPost, map[string]any{"houseId": houseID}, map[string]any{"sensors": nil})
}

func (client MetadataReadonlyClient) RunSensorEventList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "sensor.event.list", "house_context_missing"), nil
	}
	body := map[string]any{"houseId": houseID}
	for _, key := range []string{"deviceId", "sensorId", "eventId", "name", "status", "valid"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	return client.readPath(ctx, request, "sensor.event.list", "/v1/sensor/r/events", http.MethodPost, body, map[string]any{"events": nil})
}

func (client MetadataReadonlyClient) RunDeviceEnergySummary(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["id"])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.energy.summary", "device_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "device.energy.summary", "/v1/energy/devices/"+pathSegment(deviceID)+"/r/summary", http.MethodGet, nil, map[string]any{"summary": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunDeviceWeatherGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["id"])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.weather.get", "device_context_missing"), nil
	}
	queryType := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["queryType"]), stringFromAny(request.Parameters["type"])))
	if queryType == "" {
		queryType = "default"
	}
	body := map[string]any{}
	for _, key := range []string{"area", "dimension", "timeStart", "timeEnd", "date", "language"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	result, err := client.readPath(ctx, request, "device.weather.get", "/v1/weather/r/"+pathSegment(deviceID)+"/"+pathSegment(queryType)+"/queryWeather", http.MethodPost, body, map[string]any{"weather": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunMeshgroupDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	groupID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["meshgroupId"]), stringFromAny(request.Parameters["meshGroupId"]), stringFromAny(request.Parameters["groupId"]), stringFromAny(request.Parameters["id"])))
	if groupID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "meshgroup.detail.get", "meshgroup_context_missing"), nil
	}
	return client.readPath(ctx, request, "meshgroup.detail.get", "/v1/meshgroup/"+pathSegment(groupID)+"/r/detail", http.MethodPost, nil, map[string]any{"detail": nil})
}

func (client MetadataReadonlyClient) readPath(ctx context.Context, request MetadataReadonlyRequest, capability string, path string, method string, body map[string]any, projection map[string]any) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, method, path, body, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, capability, statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, capability, response), nil
	}
	data := map[string]any{}
	for key := range projection {
		data[key] = sanitizeCloudData(response["data"])
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   strings.TrimSpace(request.DeviceID),
		Capability: capability,
		Data:       data,
		RawShape:   responseDataType(response),
		APICalls:   1,
		Warnings:   []string{},
	}, nil
}
