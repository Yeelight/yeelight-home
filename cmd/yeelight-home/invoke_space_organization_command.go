package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) prepareSpaceOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, preconditions, summary, err := buildSpaceOrganizationPayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), spaceOrganizationAcceptedFields(request.Intent)), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	if reason := validateSpaceOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, spaceOrganizationAcceptedFields(request.Intent)), nil
	}
	preview := spaceOrganizationPreview(request.Intent, payload, entities)
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func buildSpaceOrganizationPayload(request contract.Request, houseID string) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "room.rename":
		roomID := firstRequestString(request.Parameters, "roomId", "id")
		name := firstRequestString(request.Parameters, "name", "newName", "roomName")
		if roomID == "" || name == "" {
			return nil, nil, "", fmt.Errorf("invalid_room_rename_payload")
		}
		return map[string]any{
				"houseId": requestNumberOrString(houseID),
				"roomId":  roomID,
				"id":      roomID,
				"name":    name,
			}, []string{
				"提交前重新读取家庭实体列表",
				"目标房间必须属于当前家庭",
				"提交后通过 entity.list 验证房间名称",
			}, fmt.Sprintf("重命名房间为 %s", name), nil
	case "room.update":
		roomID := firstRequestString(request.Parameters, "roomId", "id")
		payload := map[string]any{
			"houseId": requestNumberOrString(houseID),
			"roomId":  roomID,
			"id":      roomID,
		}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{"name", "img", "gatewayDeviceId", "gatewayIds", "defaultGatewayIds", "seq", "capability"}) || roomID == "" {
			return nil, nil, "", fmt.Errorf("invalid_room_update_payload")
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标房间和引用网关设备必须属于当前家庭",
			"提交后通过 entity.list 验证房间名称；图片、网关和能力字段按云端写入响应确认",
		}, "更新房间信息", nil
	case "area.update":
		areaID := firstRequestString(request.Parameters, "areaId", "id")
		payload := map[string]any{
			"houseId": requestNumberOrString(houseID),
			"areaId":  areaID,
			"id":      areaID,
		}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{"name", "desc", "icon", "parentId", "roomIds"}) || areaID == "" {
			return nil, nil, "", fmt.Errorf("invalid_area_update_payload")
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标区域、父区域和关联房间必须属于当前家庭",
			"提交后通过 entity.list 验证区域名称",
		}, "更新区域信息", nil
	case "device.rename":
		deviceID := firstRequestString(request.Parameters, "deviceId", "id", "entityId")
		name := firstRequestString(request.Parameters, "name", "newName", "deviceName", "alias")
		if deviceID == "" || name == "" {
			return nil, nil, "", fmt.Errorf("invalid_device_rename_payload")
		}
		return map[string]any{
				"houseId":  requestNumberOrString(houseID),
				"deviceId": deviceID,
				"id":       deviceID,
				"name":     name,
			}, []string{
				"提交前重新读取家庭实体列表",
				"目标设备必须属于当前家庭",
				"提交后通过 entity.list 验证设备名称",
			}, fmt.Sprintf("重命名设备为 %s", name), nil
	case "device.move":
		deviceID := firstRequestString(request.Parameters, "deviceId", "id", "entityId")
		roomID := firstRequestString(request.Parameters, "roomId", "targetRoomId", "targetId")
		if deviceID == "" || roomID == "" {
			return nil, nil, "", fmt.Errorf("invalid_device_move_payload")
		}
		return map[string]any{
				"houseId":  requestNumberOrString(houseID),
				"deviceId": deviceID,
				"id":       deviceID,
				"roomId":   roomID,
			}, []string{
				"提交前重新读取家庭实体列表",
				"目标设备和目标房间必须属于当前家庭",
				"提交后通过 entity.list 验证设备 roomId",
			}, "移动设备到指定房间", nil
	case "group.update":
		groupID := firstRequestString(request.Parameters, "groupId", "id")
		payload := map[string]any{
			"houseId": requestNumberOrString(houseID),
			"groupId": groupID,
			"id":      groupID,
		}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{"name", "desc", "icon", "roomId"}) || groupID == "" {
			return nil, nil, "", fmt.Errorf("invalid_group_update_payload")
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标设备组和目标房间必须属于当前家庭",
			"提交后通过 entity.list 验证设备组名称和 roomId",
			"设备组成员变更 details 仍保持阻断",
		}, "更新设备组信息", nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported_space_organization_intent")
	}
}

func copyOptionalSpaceFields(payload map[string]any, parameters map[string]any, keys []string) bool {
	copied := false
	for _, key := range keys {
		value, ok := parameters[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				payload[key] = strings.TrimSpace(typed)
				copied = true
			}
		case float64, int, int64, bool:
			payload[key] = typed
			copied = true
		case []any:
			if len(typed) > 0 {
				payload[key] = typed
				copied = true
			}
		case map[string]any:
			if len(typed) > 0 {
				payload[key] = typed
				copied = true
			}
		}
	}
	return copied
}

func validateSpaceOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "room.rename", "room.update":
		roomID := valueIDString(payload["roomId"])
		current, ok := findEntitySummary(entities, "room", roomID)
		if !ok {
			return "invalid_room_reference"
		}
		name := strings.TrimSpace(requestString(payload["name"]))
		if intent == "room.update" && name == "" {
			if current.Name == "" {
				return "invalid_room_update_payload"
			}
			payload["name"] = current.Name
			name = current.Name
		}
		for _, entity := range entities.Entities {
			if entity.Type == "room" && entity.ID != roomID && entity.Name == name {
				return "room_name_already_exists"
			}
		}
		if intent == "room.update" {
			if reason := validateRoomUpdateGatewayReferences(payload, entities); reason != "" {
				return reason
			}
		}
	case "area.update":
		areaID := valueIDString(payload["areaId"])
		if !entityExists(entities, "area", areaID) {
			return "invalid_area_reference"
		}
		if parentID := valueIDString(payload["parentId"]); parentID != "" && parentID != areaID && !entityExists(entities, "area", parentID) {
			return "invalid_parent_area_reference"
		}
		if parentID := valueIDString(payload["parentId"]); parentID == areaID {
			return "invalid_parent_area_reference"
		}
		if roomIDs, ok := payload["roomIds"].([]any); ok {
			if len(roomIDs) > 50 {
				return "area_room_limit_exceeded"
			}
			for _, roomID := range roomIDs {
				if !entityExists(entities, "room", valueIDString(roomID)) {
					return "invalid_area_room_reference"
				}
			}
		}
		if name := strings.TrimSpace(requestString(payload["name"])); name != "" {
			for _, entity := range entities.Entities {
				if entity.Type == "area" && entity.ID != areaID && entity.Name == name {
					return "area_name_already_exists"
				}
			}
		}
	case "device.rename":
		if !entityExists(entities, "device", valueIDString(payload["deviceId"])) {
			return "invalid_device_reference"
		}
	case "device.move":
		if !entityExists(entities, "device", valueIDString(payload["deviceId"])) {
			return "invalid_device_reference"
		}
		if !entityExists(entities, "room", valueIDString(payload["roomId"])) {
			return "invalid_target_room_reference"
		}
	case "group.update":
		groupID := valueIDString(payload["groupId"])
		if !entityExists(entities, "group", groupID) {
			return "invalid_group_reference"
		}
		if roomID := valueIDString(payload["roomId"]); roomID != "" && !entityExists(entities, "room", roomID) {
			return "invalid_target_room_reference"
		}
		if name := strings.TrimSpace(requestString(payload["name"])); name != "" {
			for _, entity := range entities.Entities {
				if entity.Type == "group" && entity.ID != groupID && entity.Name == name {
					return "group_name_already_exists"
				}
			}
		}
	default:
		return "unsupported_space_organization_intent"
	}
	return ""
}

func spaceOrganizationPreview(intent string, payload map[string]any, entities api.EntityListResult) map[string]any {
	preview := map[string]any{}
	var entityType string
	var entityID string
	switch intent {
	case "room.rename", "room.update":
		entityType, entityID = "room", valueIDString(payload["roomId"])
	case "area.update":
		entityType, entityID = "area", valueIDString(payload["areaId"])
	case "device.rename", "device.move":
		entityType, entityID = "device", valueIDString(payload["deviceId"])
	case "group.update":
		entityType, entityID = "group", valueIDString(payload["groupId"])
	default:
		return preview
	}
	if current, ok := findEntitySummary(entities, entityType, entityID); ok {
		preview["current"] = map[string]any{
			"type":   current.Type,
			"id":     current.ID,
			"name":   current.Name,
			"roomId": current.RoomID,
		}
	}
	planned := map[string]any{}
	for _, key := range []string{"name", "desc", "icon", "parentId", "roomIds", "roomId", "img", "gatewayDeviceId", "gatewayIds", "defaultGatewayIds", "seq", "capability"} {
		if value, ok := payload[key]; ok {
			planned[key] = value
		}
	}
	if len(planned) > 0 {
		preview["planned"] = planned
	}
	return preview
}

func findEntitySummary(entities api.EntityListResult, entityType string, entityID string) (api.EntitySummary, bool) {
	for _, entity := range entities.Entities {
		if entity.Type == entityType && entity.ID == entityID {
			return entity, true
		}
	}
	return api.EntitySummary{}, false
}

func spaceOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "room.rename":
		return []string{"parameters.houseId", "parameters.roomId", "parameters.name"}
	case "room.update":
		return []string{"parameters.houseId", "parameters.roomId", "parameters.name", "parameters.img", "parameters.gatewayDeviceId", "parameters.gatewayIds", "parameters.defaultGatewayIds", "parameters.seq", "parameters.capability"}
	case "area.update":
		return []string{"parameters.houseId", "parameters.areaId", "parameters.name", "parameters.desc", "parameters.icon", "parameters.parentId", "parameters.roomIds"}
	case "device.rename":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.name"}
	case "device.move":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.roomId"}
	case "group.update":
		return []string{"parameters.houseId", "parameters.groupId", "parameters.name", "parameters.desc", "parameters.icon", "parameters.roomId"}
	default:
		return []string{"parameters.houseId"}
	}
}

func validateRoomUpdateGatewayReferences(payload map[string]any, entities api.EntityListResult) string {
	if gatewayID := valueIDString(payload["gatewayDeviceId"]); gatewayID != "" && !entityExists(entities, "device", gatewayID) {
		return "invalid_gateway_device_reference"
	}
	for _, key := range []string{"gatewayIds", "defaultGatewayIds"} {
		for _, gatewayID := range valueIDList(payload[key]) {
			if !entityExists(entities, "device", gatewayID) {
				return "invalid_gateway_device_reference"
			}
		}
	}
	return ""
}

func (app *app) executeSpaceOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.SpaceOrganizationKind) (contract.Response, error) {
	result, err := api.NewSpaceOrganizationClient(endpoint, nil).Run(ctx, api.SpaceOrganizationRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SpaceOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return spaceOrganizationExecuteResponse(request, record, result), nil
}

func (app *app) prepareSpaceBatchOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, preconditions, summary, err := buildSpaceBatchOrganizationPayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), spaceBatchOrganizationAcceptedFields(request.Intent)), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	if reason := validateSpaceBatchOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, spaceBatchOrganizationAcceptedFields(request.Intent)), nil
	}
	preview := spaceBatchOrganizationPreview(request.Intent, payload, entities)
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func buildSpaceBatchOrganizationPayload(request contract.Request, houseID string) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "device.move_room.batch":
		items, ok := normalizeDeviceRoomBatchItems(request.Parameters["items"])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_device_move_room_batch_payload")
		}
		return map[string]any{
				"houseId": requestNumberOrString(houseID),
				"items":   items,
			}, []string{
				"提交前重新读取家庭实体列表",
				"每个目标设备和目标房间必须属于当前家庭",
				"单次计划最多移动 20 个设备",
				"提交后通过 entity.list 逐项验证设备 roomId",
			}, "批量移动设备到指定房间", nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported_space_batch_organization_intent")
	}
}

func normalizeDeviceRoomBatchItems(value any) (map[string]any, bool) {
	result := map[string]any{}
	switch typed := value.(type) {
	case map[string]any:
		for rawDeviceID, rawRoomID := range typed {
			deviceID := strings.TrimSpace(rawDeviceID)
			roomID := strings.TrimSpace(requestString(rawRoomID))
			if deviceID == "" || roomID == "" {
				return nil, false
			}
			result[deviceID] = roomID
		}
	case []any:
		for _, raw := range typed {
			item, ok := raw.(map[string]any)
			if !ok {
				return nil, false
			}
			deviceID := firstRequestString(item, "deviceId", "id", "entityId")
			roomID := firstRequestString(item, "roomId", "targetRoomId", "targetId")
			if deviceID == "" || roomID == "" {
				return nil, false
			}
			if _, exists := result[deviceID]; exists {
				return nil, false
			}
			result[deviceID] = roomID
		}
	default:
		return nil, false
	}
	if len(result) == 0 || len(result) > 20 {
		return nil, false
	}
	return result, true
}

func validateSpaceBatchOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "device.move_room.batch":
		items, ok := payload["items"].(map[string]any)
		if !ok || len(items) == 0 || len(items) > 20 {
			return "invalid_device_move_room_batch_payload"
		}
		for deviceID, roomValue := range items {
			if !entityExists(entities, "device", deviceID) {
				return "invalid_device_reference"
			}
			if !entityExists(entities, "room", valueIDString(roomValue)) {
				return "invalid_target_room_reference"
			}
		}
	default:
		return "unsupported_space_batch_organization_intent"
	}
	return ""
}

func spaceBatchOrganizationPreview(intent string, payload map[string]any, entities api.EntityListResult) map[string]any {
	if intent != "device.move_room.batch" {
		return map[string]any{}
	}
	items, _ := payload["items"].(map[string]any)
	deviceIDs := make([]string, 0, len(items))
	for deviceID := range items {
		deviceIDs = append(deviceIDs, deviceID)
	}
	sort.Strings(deviceIDs)
	previewItems := make([]any, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		item := map[string]any{
			"deviceId":     deviceID,
			"targetRoomId": valueIDString(items[deviceID]),
		}
		if current, ok := findEntitySummary(entities, "device", deviceID); ok {
			item["currentRoomId"] = current.RoomID
			item["deviceName"] = current.Name
		}
		previewItems = append(previewItems, item)
	}
	return map[string]any{
		"itemCount": len(previewItems),
		"items":     previewItems,
	}
}

func spaceBatchOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "device.move_room.batch":
		return []string{"parameters.houseId", "parameters.items[].deviceId", "parameters.items[].roomId", "parameters.items as {deviceId: roomId}"}
	default:
		return []string{"parameters.houseId"}
	}
}

func (app *app) executeSpaceBatchOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.SpaceBatchOrganizationKind) (contract.Response, error) {
	result, err := api.NewSpaceBatchOrganizationClient(endpoint, nil).Run(ctx, api.SpaceBatchOrganizationRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SpaceOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return spaceBatchOrganizationExecuteResponse(request, record, result), nil
}

func (app *app) executeDeviceMove(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	deviceID := valueIDString(firstNonNil(record.Payload["deviceId"], record.Payload["id"]))
	roomID := valueIDString(record.Payload["roomId"])
	if deviceID == "" || roomID == "" {
		return contract.Response{}, fmt.Errorf("invalid_device_move_payload")
	}
	batchRecord := record
	batchRecord.Intent = "device.move_room.batch"
	batchRecord.Payload = map[string]any{
		"houseId": record.HouseID,
		"items": map[string]any{
			deviceID: roomID,
		},
	}
	result, err := api.NewSpaceBatchOrganizationClient(endpoint, nil).Run(ctx, api.SpaceBatchOrganizationRequest{
		Kind:           api.SpaceBatchDeviceMoveRoom,
		HouseID:        record.HouseID,
		Payload:        batchRecord.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SpaceOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	response := spaceBatchOrganizationExecuteResponse(request, batchRecord, result)
	response.Result["capability"] = "device.move"
	return response, nil
}
