package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunDeviceDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.detail.get", "device_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/device/"+deviceID+"/r/detail", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, "device.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, "device.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: "device.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectDeviceDetail(response["data"], deviceID),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunDeviceAttrList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.attr.list", "device_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/device/r/attrs", map[string]any{semantic.FieldIDs: deviceID}, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, "device.attr.list", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, "device.attr.list", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: "device.attr.list",
		Data: map[string]any{
			semantic.FieldAttributes: projectDeviceAttributes(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunDeviceList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/device/r/all", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
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
			semantic.FieldDevices: projectDeviceRows(response["data"]),
			semantic.FieldGroups:  projectMeshgroupRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunRoomDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	roomID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldRoomID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if roomID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "room.detail.get", "room_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/room/"+roomID+"/r/detail", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, request.HouseID, request.DeviceID, "room.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, "room.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   strings.TrimSpace(request.DeviceID),
		Capability: "room.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectRoomDetail(response["data"], roomID),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunRoomList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "room.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/room/r/all", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
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
			semantic.FieldRooms: projectRoomRows(response["data"]),
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
		stringFromAny(request.Parameters[semantic.FieldFuzzyName]),
		stringFromAny(request.Parameters[semantic.FieldRoomName]),
		stringFromAny(request.Parameters[semantic.FieldTargetName]),
		stringFromAny(request.Parameters[semantic.FieldEntityName]),
		stringFromAny(request.Parameters[semantic.FieldName]),
		stringFromAny(request.Parameters[semantic.FieldKeyword]),
		stringFromAny(request.Parameters[semantic.FieldQuery]),
	))
	if fuzzyName == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "room.search", "room_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	body := map[string]any{
		semantic.FieldFuzzyName: fuzzyName,
		semantic.FieldPageNo:    positiveInt(request.Parameters[semantic.FieldPageNo], 1),
		semantic.FieldPageSize:  positiveInt(firstNonNil(request.Parameters[semantic.FieldPageSize], request.Parameters[semantic.FieldLimit]), 20),
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/room/"+pathSegment(houseID)+"/r/fuzzy", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("room search", response)
	}
	rooms := projectRoomRows(response["data"])
	warnings := []string{}
	apiCalls := 1
	if len(rooms) == 0 {
		if fallbackRows, ok := client.runRoomSearchFallback(ctx, houseID, fuzzyName, request.Credentials); ok {
			rooms = fallbackRows
			apiCalls++
			warnings = append(warnings, "room_search_local_fuzzy_fallback")
		}
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "room.search",
		Data: map[string]any{
			semantic.FieldRooms: rooms,
		},
		RawShape: responseDataType(response),
		APICalls: apiCalls,
		Warnings: warnings,
	}, nil
}

func (client MetadataReadonlyClient) RunAreaDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "area.detail.get", "house_context_missing"), nil
	}
	areaID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldAreaID]), stringFromAny(request.Parameters[semantic.FieldID]), stringFromAny(request.Parameters[semantic.FieldEntityID])))
	if areaID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "area.detail.get", "area_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v2/thing/manage/house/"+pathSegment(houseID)+"/area/"+pathSegment(areaID)+"/r/info", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, houseID, request.DeviceID, "area.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, request.DeviceID, "area.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "area.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectAreaDetail(response["data"], areaID),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunHomeDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.detail.get", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v1/house/"+pathSegment(houseID)+"/r/info", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("home detail", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectHomeDetail(response["data"], houseID),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectHomeDetail(data any, fallbackHouseID string) map[string]any {
	item, _ := data.(map[string]any)
	if detail, ok := item[semantic.FieldDetail].(map[string]any); ok {
		item = detail
	}
	home := map[string]any{}
	copyHomeDetailString(home, semantic.FieldID, item, semantic.FieldID, semantic.FieldHouseID)
	if home[semantic.FieldID] == nil && strings.TrimSpace(fallbackHouseID) != "" {
		home[semantic.FieldID] = strings.TrimSpace(fallbackHouseID)
	}
	copyHomeDetailString(home, semantic.FieldHouseID, item, semantic.FieldHouseID, semantic.FieldID)
	if home[semantic.FieldHouseID] == nil && strings.TrimSpace(fallbackHouseID) != "" {
		home[semantic.FieldHouseID] = strings.TrimSpace(fallbackHouseID)
	}
	copyHomeDetailString(home, semantic.FieldName, item, semantic.FieldName, semantic.FieldHouseName)
	copyHomeDetailString(home, semantic.FieldDescription, item, semantic.FieldDescription, "desc")
	copyHomeDetailString(home, semantic.FieldAreaCode, item, semantic.FieldAreaCode)
	copyHomeDetailString(home, semantic.FieldAreaName, item, semantic.FieldAreaName)
	copyHomeDetailString(home, semantic.FieldBuildingName, item, semantic.FieldBuildingName)
	copyHomeDetailString(home, semantic.FieldBuildingAddress, item, semantic.FieldBuildingAddress)
	copyHomeDetailString(home, semantic.FieldFloorName, item, semantic.FieldFloorName)
	copyHomeDetailString(home, semantic.FieldIcon, item, semantic.FieldIcon)
	copyHomeDetailString(home, semantic.FieldImage, item, semantic.FieldImage, "img")
	if value, ok := item[semantic.FieldValid]; ok {
		home[semantic.FieldValid] = value
	}
	return home
}

func copyHomeDetailString(output map[string]any, outputKey string, input map[string]any, inputKeys ...string) {
	for _, key := range inputKeys {
		if value := stringFromAny(input[key]); value != "" {
			output[outputKey] = value
			return
		}
	}
}

func (client MetadataReadonlyClient) RunHomeStatGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.stat.get", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "home.stat.get", "/v1/house/"+pathSegment(houseID)+"/r/stat", http.MethodPost, nil, map[string]any{semantic.FieldStats: nil})
}

func (client MetadataReadonlyClient) RunGroupStructureList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.structure.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/group/r/all", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("group structure list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "group.structure.list",
		Data: map[string]any{
			semantic.FieldGroups: projectGroupRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGroupList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "group.list", "house_context_missing"), nil
	}
	pageNo := positiveInt(request.Parameters[semantic.FieldPageNo], 1)
	pageSize := positiveInt(firstNonNil(request.Parameters[semantic.FieldPageSize], request.Parameters[semantic.FieldLimit]), 100)
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
			semantic.FieldGroups: projectGroupRows(response["data"]),
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
		stringFromAny(request.Parameters[semantic.FieldFuzzyName]),
		stringFromAny(request.Parameters[semantic.FieldGroupName]),
		stringFromAny(request.Parameters[semantic.FieldTargetName]),
		stringFromAny(request.Parameters[semantic.FieldEntityName]),
		stringFromAny(request.Parameters[semantic.FieldName]),
		stringFromAny(request.Parameters[semantic.FieldKeyword]),
		stringFromAny(request.Parameters[semantic.FieldQuery]),
	))
	if fuzzyName == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "group.search", "group_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	pageNo := positiveInt(request.Parameters[semantic.FieldPageNo], 1)
	pageSize := positiveInt(firstNonNil(request.Parameters[semantic.FieldPageSize], request.Parameters[semantic.FieldLimit]), 100)
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
			semantic.FieldGroups: filterProjectedRowsByName(projectGroupRows(response["data"]), fuzzyName),
			semantic.FieldQuery:  map[string]any{semantic.FieldName: fuzzyName, semantic.FieldPageNo: pageNo, semantic.FieldPageSize: pageSize},
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
	groupID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldGroupID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if groupID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "group.detail.get", "group_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v2/thing/manage/house/"+pathSegment(houseID)+"/group/"+pathSegment(groupID)+"/r/info", nil, request.Credentials)
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			return metadataReadonlyAuthBoundaryResult(client.endpoint.Region, houseID, request.DeviceID, "group.detail.get", statusErr.StatusCode), nil
		}
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, request.DeviceID, "group.detail.get", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "group.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectGroupDetail(response["data"], groupID),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunSceneDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	sceneID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldSceneID]), stringFromAny(request.Parameters[semantic.FieldID])))
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
		stringFromAny(request.Parameters[semantic.FieldSceneName]),
		stringFromAny(request.Parameters[semantic.FieldCurrentName]),
		stringFromAny(request.Parameters[semantic.FieldTargetName]),
		stringFromAny(request.Parameters[semantic.FieldEntityName]),
		stringFromAny(request.Parameters[semantic.FieldName]),
		stringFromAny(request.Parameters[semantic.FieldKeyword]),
		stringFromAny(request.Parameters[semantic.FieldQuery]),
		stringFromAny(request.Parameters[semantic.FieldFuzzyName]),
	))
	if keyword == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "scene.search", "scene_search_keyword_missing")
		result.HouseID = houseID
		return result, nil
	}
	body := map[string]any{}
	for _, key := range []string{semantic.FieldName, semantic.FieldFuzzyName, semantic.FieldKeyword, semantic.FieldQuery, semantic.FieldPageNo, semantic.FieldPageSize, semantic.FieldSort, "order", "orderBy"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	if body[semantic.FieldName] == nil {
		body[semantic.FieldName] = keyword
	}
	if body[semantic.FieldPageNo] == nil {
		body[semantic.FieldPageNo] = 1
	}
	if body[semantic.FieldPageSize] == nil {
		body[semantic.FieldPageSize] = 20
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/"+houseID+"/r/fuzzy", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, request.DeviceID, "scene.search", response), nil
	}
	scenes := filterProjectedRowsByName(projectSceneRows(response["data"]), keyword)
	warnings := []string{}
	apiCalls := 1
	if len(scenes) == 0 {
		if fallbackRows, ok := client.runSceneSearchFallback(ctx, houseID, keyword, request.Credentials); ok {
			scenes = fallbackRows
			apiCalls++
			warnings = append(warnings, "scene_search_local_fuzzy_fallback")
		}
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "scene.search",
		Data: map[string]any{
			semantic.FieldScenes: scenes,
			semantic.FieldQuery:  map[string]any{semantic.FieldName: keyword, semantic.FieldPageNo: body[semantic.FieldPageNo], semantic.FieldPageSize: body[semantic.FieldPageSize]},
		},
		RawShape: responseDataType(response),
		APICalls: apiCalls,
		Warnings: warnings,
	}, nil
}

func (client MetadataReadonlyClient) runRoomSearchFallback(ctx context.Context, houseID string, keyword string, credentials MetadataReadonlyCredentials) ([]any, bool) {
	response, err := client.call(ctx, http.MethodPost, "/v1/room/r/all", map[string]any{semantic.FieldHouseID: houseID}, credentials)
	if err != nil || !isBusinessOK(response) {
		return nil, false
	}
	rows := filterProjectedRowsByName(projectRoomRows(response["data"]), keyword)
	return rows, len(rows) > 0
}

func (client MetadataReadonlyClient) runSceneSearchFallback(ctx context.Context, houseID string, keyword string, credentials MetadataReadonlyCredentials) ([]any, bool) {
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/r/all", map[string]any{semantic.FieldHouseID: houseID}, credentials)
	if err != nil || !isBusinessOK(response) {
		return nil, false
	}
	rows := filterProjectedRowsByName(projectSceneRows(response["data"]), keyword)
	return rows, len(rows) > 0
}

func (client MetadataReadonlyClient) RunAutomationSupportedList(ctx context.Context, request MetadataReadonlyRequest, v2 bool) (MetadataReadonlyResult, error) {
	capability := "automation.supported.list"
	path := "/v1/automations/r/supported"
	key := semantic.FieldSupported
	if v2 {
		capability = "automation.supported.v2.list"
		path = "/v1/automations/r/supported/v2"
		key = semantic.FieldSupportedV2
	}
	response, err := client.call(ctx, http.MethodPost, path, map[string]any{}, request.Credentials)
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
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   strings.TrimSpace(request.DeviceID),
		Capability: capability,
		Data: map[string]any{
			key: projectAutomationSupportedRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectAutomationSupportedRows(value any) []any {
	rows := rowsFromData(value)
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		row := map[string]any{}
		if capabilityPID := firstCloudAny(item, semantic.FieldCapabilityPID, "pid"); capabilityPID != nil {
			row[semantic.FieldCapabilityPID] = sanitizeCloudData(capabilityPID)
		}
		if conditions := projectAutomationSupportedConditions(firstCloudAny(item, semantic.FieldActions, semantic.FieldConditions)); len(conditions) > 0 {
			row[semantic.FieldConditions] = conditions
		}
		if len(row) > 0 {
			result = append(result, row)
		}
	}
	return result
}

func projectAutomationSupportedConditions(value any) []any {
	rows := rowsFromData(value)
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		row := map[string]any{}
		if id := firstCloudAny(item, semantic.FieldID); id != nil {
			row[semantic.FieldID] = sanitizeCloudData(id)
		}
		if kind := strings.TrimSpace(stringFromAny(firstCloudAny(item, semantic.FieldType))); kind != "" {
			row[semantic.FieldConditionKind] = kind
		}
		name, description := automationSupportedNames(firstCloudAny(item, semantic.DescriptionFields()...))
		if name == "" {
			name = strings.TrimSpace(stringFromAny(firstCloudAny(item, semantic.FieldName)))
		}
		if name != "" {
			row[semantic.FieldName] = name
		}
		if description != "" && description != name {
			row[semantic.FieldDescription] = description
		}
		if inputs := projectAutomationSupportedInputs(firstCloudAny(item, "argsDesc", semantic.FieldInputs)); len(inputs) > 0 {
			row[semantic.FieldInputs] = inputs
		}
		if versions := supportedVersionList(firstCloudAny(item, "supportVersion", semantic.FieldSupportedVersions)); len(versions) > 0 {
			row[semantic.FieldSupportedVersions] = versions
		}
		if len(row) > 0 {
			result = append(result, row)
		}
	}
	return result
}

func automationSupportedNames(value any) (string, string) {
	rows := rowsFromData(value)
	first := ""
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		text := strings.TrimSpace(stringFromAny(firstCloudAny(item, semantic.FieldValue, semantic.FieldName)))
		if text == "" {
			continue
		}
		if first == "" {
			first = text
		}
		if languageID := strings.TrimSpace(stringFromAny(firstCloudAny(item, "languageId", semantic.FieldLanguage))); languageID == "2" {
			return text, first
		}
	}
	return first, ""
}

func projectAutomationSupportedInputs(value any) []any {
	rows := rowsFromData(value)
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		input := map[string]any{}
		if key := firstCloudAny(item, semantic.FieldType, semantic.FieldKey); key != nil {
			input[semantic.FieldKey] = sanitizeCloudData(key)
		}
		if inputType := firstCloudAny(item, "dataType", semantic.FieldInputType); inputType != nil {
			input[semantic.FieldInputType] = sanitizeCloudData(inputType)
		}
		if unit := firstCloudAny(item, semantic.FieldUnit); unit != nil {
			input[semantic.FieldUnit] = sanitizeCloudData(unit)
		}
		if valueRange := firstCloudAny(item, "valueRange", semantic.FieldValueRange); valueRange != nil {
			input[semantic.FieldValueRange] = sanitizeCloudData(valueRange)
		}
		if len(input) > 0 {
			result = append(result, input)
		}
	}
	return result
}

func supportedVersionList(value any) []any {
	text := strings.TrimSpace(stringFromAny(value))
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ",")
	result := make([]any, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (client MetadataReadonlyClient) RunAutomationRuleList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.rule.list", "house_context_missing"), nil
	}
	body := map[string]any{semantic.FieldHouseID: houseID}
	for _, key := range []string{semantic.FieldGatewayDeviceID, semantic.FieldName, semantic.FieldStatus, semantic.FieldValid} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	return client.readPath(ctx, request, "automation.rule.list", "/v1/rule/r/list", http.MethodPost, body, map[string]any{semantic.FieldRules: nil})
}

func (client MetadataReadonlyClient) RunAutomationListPage(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "automation.list.page", "house_context_missing"), nil
	}
	pageNo, pageSize := readonlyPage(request.Parameters, 1, 20)
	return client.readPath(ctx, request, "automation.list.page", "/v1/automations/"+pathSegment(houseID)+"/r/list/"+pageNo+"/"+pageSize, http.MethodGet, nil, map[string]any{semantic.FieldAutomations: nil})
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
		copyResponseStringMappings(room, item, semantic.RoomSummaryMappings())
		if ids := stringListFromAny(firstNonNil(item["deviceIds"], item["devices"])); len(ids) > 0 {
			room[semantic.FieldDeviceCount] = len(ids)
			room[semantic.FieldDeviceIDs] = ids
		}
		if ids := stringListFromAny(item["gatewayDeviceIds"]); len(ids) > 0 {
			room[semantic.FieldGatewayDeviceIDs] = ids
		}
		if len(room) > 0 {
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func projectRoomDetail(data any, fallbackRoomID string) map[string]any {
	item, _ := data.(map[string]any)
	if detail, ok := item[semantic.FieldDetail].(map[string]any); ok {
		item = detail
	}
	room := map[string]any{}
	if summaries := projectRoomRows(item); len(summaries) > 0 {
		if summary, ok := summaries[0].(map[string]any); ok {
			for key, value := range summary {
				room[key] = value
			}
		}
	}
	if room[semantic.FieldID] == nil && strings.TrimSpace(fallbackRoomID) != "" {
		room[semantic.FieldID] = strings.TrimSpace(fallbackRoomID)
	}
	devices := projectRoomDetailDeviceRows(item)
	scenes := projectRoomDetailSceneRows(item)
	counts := map[string]any{}
	if len(devices) > 0 {
		room[semantic.FieldDevices] = devices
		counts[semantic.FieldDevices] = len(devices)
		if room[semantic.FieldDeviceCount] == nil {
			room[semantic.FieldDeviceCount] = len(devices)
		}
	}
	if len(scenes) > 0 {
		room[semantic.FieldScenes] = scenes
		counts[semantic.FieldScenes] = len(scenes)
	}
	if len(counts) > 0 {
		room[semantic.FieldCounts] = counts
	}
	return compactMap(room)
}

func projectRoomDetailDeviceRows(item map[string]any) []any {
	rows := projectDeviceRows(map[string]any{semantic.FieldDevices: firstNonNil(item[semantic.FieldDevices], item["deviceList"])})
	for _, row := range rows {
		device, ok := row.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{
			semantic.FieldDeviceIdentifier,
			semantic.FieldTypeName,
			semantic.FieldBind,
			semantic.FieldVirtual,
			semantic.FieldConnectType,
			semantic.FieldPosition,
			semantic.FieldSequence,
			semantic.FieldRoomRank,
		} {
			delete(device, key)
		}
	}
	return rows
}

func projectRoomDetailSceneRows(item map[string]any) []any {
	return projectSceneRows(map[string]any{
		semantic.FieldScenes: firstNonNil(item[semantic.FieldScenes], item["sceneList"]),
		"userscenes":         item["userscenes"],
	})
}

func projectAreaDetail(data any, fallbackAreaID string) map[string]any {
	item, _ := data.(map[string]any)
	if detail, ok := item[semantic.FieldDetail].(map[string]any); ok {
		item = detail
	}
	area := map[string]any{}
	copyResponseStringMappings(area, item, semantic.AreaSummaryMappings())
	if area[semantic.FieldID] == nil && strings.TrimSpace(fallbackAreaID) != "" {
		area[semantic.FieldID] = strings.TrimSpace(fallbackAreaID)
	}
	rooms := projectAreaDetailRoomRows(item)
	if len(rooms) > 0 {
		area[semantic.FieldRooms] = rooms
		area[semantic.FieldRoomCount] = len(rooms)
		area[semantic.FieldCounts] = map[string]any{semantic.FieldRooms: len(rooms)}
	}
	return compactMap(area)
}

func projectAreaDetailRoomRows(item map[string]any) []any {
	rows := nestedRowsFromData(firstNonNil(item[semantic.FieldRooms], item["roomList"]), semantic.FieldRooms, "roomList")
	rooms := make([]any, 0, len(rows))
	for _, row := range rows {
		source, ok := row.(map[string]any)
		if !ok {
			continue
		}
		room := map[string]any{}
		copyResponseStringMappings(room, source, semantic.RoomSummaryMappings())
		if description := firstAnyString(source, semantic.DescriptionFields()...); description != "" {
			room[semantic.FieldDescription] = description
		}
		if count := intFromAny(firstNonNil(source[semantic.FieldDeviceCount], source["deviceNum"])); count > 0 {
			room[semantic.FieldDeviceCount] = count
		}
		if ids := stringListFromAny(firstNonNil(source[semantic.FieldGatewayIDs], source["gatewayIds"])); len(ids) > 0 {
			room[semantic.FieldGatewayIDs] = ids
		}
		if len(room) > 0 {
			rooms = append(rooms, compactMap(room))
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
		copyResponseStringMappings(group, item, semantic.GroupSummaryMappings())
		if ids := stringListFromAny(item["roomIds"]); len(ids) > 0 {
			group[semantic.FieldRoomCount] = len(ids)
			group[semantic.FieldRoomIDs] = ids
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func projectGroupDetail(data any, fallbackGroupID string) map[string]any {
	item, _ := data.(map[string]any)
	if detail, ok := item[semantic.FieldDetail].(map[string]any); ok {
		item = detail
	}
	group := map[string]any{}
	if summaries := projectGroupRows(item); len(summaries) > 0 {
		if summary, ok := summaries[0].(map[string]any); ok {
			for key, value := range summary {
				group[key] = value
			}
		}
	}
	if group[semantic.FieldID] == nil && strings.TrimSpace(fallbackGroupID) != "" {
		group[semantic.FieldID] = strings.TrimSpace(fallbackGroupID)
	}
	if configs := projectConfigRows(firstNonNil(item[semantic.FieldConfigs], item["configs"])); len(configs) > 0 {
		group[semantic.FieldConfigs] = configs
		group[semantic.FieldConfigCount] = len(configs)
	}
	if devices := projectDeviceRows(map[string]any{semantic.FieldDevices: firstNonNil(item[semantic.FieldDevices], item["deviceList"])}); len(devices) > 0 {
		group[semantic.FieldDevices] = devices
		group[semantic.FieldDeviceCount] = len(devices)
	}
	return compactMap(group)
}

func projectConfigRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.ConfigRowContainers()...)
	configs := make([]any, 0, len(rows))
	for _, row := range rows {
		source, ok := row.(map[string]any)
		if !ok {
			continue
		}
		config := map[string]any{}
		copyResponseStringMappings(config, source, semantic.ConfigSummaryMappings())
		if value := source[semantic.FieldValue]; value != nil {
			config[semantic.FieldValue] = value
		}
		if len(config) > 0 {
			configs = append(configs, compactMap(config))
		}
	}
	return configs
}

func filterProjectedRowsByName(rows []any, name string) []any {
	keyword := strings.TrimSpace(name)
	if keyword == "" {
		return rows
	}
	candidates := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		candidates = append(candidates, item)
	}
	matches := semantic.RankNameMatches(keyword, candidates, func(item map[string]any) string {
		return stringFromAny(item[semantic.FieldName])
	})
	filtered := make([]any, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, match.Value)
	}
	return filtered
}

func projectDeviceRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.DeviceRowContainers()...)
	devices := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		device := map[string]any{}
		copyResponseStringMappings(device, item, semantic.DeviceSummaryMappings())
		if ids := stringListFromAny(item["roomIds"]); len(ids) > 0 {
			device[semantic.FieldRoomIDs] = ids
		}
		if ids := stringListFromAny(item["deviceIds"]); len(ids) > 0 {
			device[semantic.FieldChildDeviceCount] = len(ids)
			device[semantic.FieldDeviceIDs] = ids
		}
		if len(device) > 0 {
			devices = append(devices, device)
		}
	}
	return devices
}

func projectDeviceDetail(data any, fallbackDeviceID string) map[string]any {
	item, _ := data.(map[string]any)
	if detail, ok := item[semantic.FieldDetail].(map[string]any); ok {
		item = detail
	}
	devices := projectDeviceRows(item)
	detail := map[string]any{}
	if len(devices) > 0 {
		if summary, ok := devices[0].(map[string]any); ok {
			for key, value := range summary {
				detail[key] = value
			}
		}
	}
	removeDeviceDetailInternalFields(detail)
	if detail[semantic.FieldID] == nil && strings.TrimSpace(fallbackDeviceID) != "" {
		detail[semantic.FieldID] = strings.TrimSpace(fallbackDeviceID)
	}
	if item == nil {
		return detail
	}
	if state := projectDeviceShadow(item["shadow"]); len(state) > 0 {
		detail[semantic.FieldProperties] = state
	}
	if attrs := projectDeviceAttributes(item[semantic.FieldAttributes]); len(attrs) > 0 {
		detail[semantic.FieldAttributes] = attrs
	} else if attrs := projectDeviceAttributes(item["attr"]); len(attrs) > 0 {
		detail[semantic.FieldAttributes] = attrs
	}
	return detail
}

func removeDeviceDetailInternalFields(detail map[string]any) {
	for _, key := range []string{
		semantic.FieldDeviceIdentifier,
		semantic.FieldCapability,
		semantic.FieldBind,
		semantic.FieldVirtual,
		semantic.FieldConnectType,
		semantic.FieldTypeName,
		semantic.FieldPosition,
		semantic.FieldSequence,
		semantic.FieldRoomRank,
	} {
		delete(detail, key)
	}
}

func projectDeviceShadow(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	properties, ok := firstCloudAny(item, semantic.FieldProperties, "propertyMap").(map[string]any)
	if !ok {
		properties = item
	}
	return projectPublicProperties(properties)
}

func projectDeviceAttributes(data any) []any {
	rows := nestedRowsFromData(data, semantic.FieldAttributes, "attr")
	attributes := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		projected := projectPublicProperties(item)
		if id := firstCloudAny(item, semantic.FieldID, semantic.FieldDeviceID); id != nil {
			projected[semantic.FieldID] = sanitizeCloudData(id)
		}
		if mac := stringFromAny(firstCloudAny(item, semantic.FieldMAC)); mac != "" {
			projected[semantic.FieldMacMasked] = maskTail(mac, 4)
		}
		if len(projected) > 0 {
			attributes = append(attributes, projected)
		}
	}
	return attributes
}

func projectPublicProperties(item map[string]any) map[string]any {
	properties := map[string]any{}
	for key, value := range item {
		propertyID, ok := semantic.PropertyID(key)
		if !ok || semantic.PropertySensitive(propertyID) {
			continue
		}
		publicName := semantic.PropertyName(propertyID)
		if publicName == "" {
			continue
		}
		properties[publicName] = sanitizeCloudData(value)
	}
	return properties
}

func projectMeshgroupRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.MeshGroupRowContainers()...)
	groups := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		group := map[string]any{}
		copyResponseStringMappings(group, item, semantic.MeshGroupSummaryMappings())
		if ids := stringListFromAny(item["deviceIds"]); len(ids) > 0 {
			group[semantic.FieldDeviceCount] = len(ids)
			group[semantic.FieldDeviceIDs] = ids
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
		stringFromAny(request.Parameters[semantic.FieldAutomationID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
		stringFromAny(request.Parameters[semantic.FieldEntityID]),
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
	return client.readPath(ctx, request, "sensor.list", "/v1/device/r/sensors", http.MethodPost, map[string]any{semantic.FieldHouseID: houseID}, map[string]any{semantic.FieldSensors: nil})
}

func (client MetadataReadonlyClient) RunSensorEventList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "sensor.event.list", "house_context_missing"), nil
	}
	body := map[string]any{semantic.FieldHouseID: houseID}
	for _, key := range semantic.SensorEventForwardFields() {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	return client.readPath(ctx, request, "sensor.event.list", "/v1/sensor/r/events", http.MethodPost, body, map[string]any{semantic.FieldEvents: nil})
}

func (client MetadataReadonlyClient) RunDeviceEnergySummary(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.energy.summary", "device_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "device.energy.summary", "/v1/energy/devices/"+pathSegment(deviceID)+"/r/summary", http.MethodGet, nil, map[string]any{semantic.FieldSummary: nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunDeviceWeatherGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "device.weather.get", "device_context_missing"), nil
	}
	queryType := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldQueryType]), stringFromAny(request.Parameters[semantic.FieldType])))
	if queryType == "" {
		queryType = "default"
	}
	body := map[string]any{}
	for _, key := range semantic.WeatherQueryForwardFields() {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	result, err := client.readPath(ctx, request, "device.weather.get", "/v1/weather/r/"+pathSegment(deviceID)+"/"+pathSegment(queryType)+"/queryWeather", http.MethodPost, body, map[string]any{semantic.FieldWeather: nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunMeshgroupDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	groupID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldMeshGroupID]), stringFromAny(request.Parameters[semantic.FieldGroupID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if groupID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "meshgroup.detail.get", "meshgroup_context_missing"), nil
	}
	return client.readPath(ctx, request, "meshgroup.detail.get", "/v1/meshgroup/"+pathSegment(groupID)+"/r/detail", http.MethodPost, nil, map[string]any{semantic.FieldDetail: nil})
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
		data[key] = projectReadonlyPathData(capability, key, response["data"])
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

func projectReadonlyPathData(capability string, key string, data any) any {
	switch capability {
	case "automation.list.page":
		if key == semantic.FieldAutomations {
			return projectAutomationPage(data)
		}
	case "automation.rule.list":
		if key == semantic.FieldRules {
			return projectAutomationRuleRows(data)
		}
	}
	return sanitizeCloudData(data)
}
