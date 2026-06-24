package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func (app *app) invokeHomeSpaceConfigurationPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, preconditions, summary, err := buildHomeSpaceConfigurationPayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeSpaceConfigurationAcceptedFields(request.Intent)), nil
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
	if reason := validateHomeSpaceConfigurationPayload(request.Intent, payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, homeSpaceConfigurationAcceptedFields(request.Intent)), nil
	}
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, entities, homeSpaceConfigurationPreview(request.Intent, payload, entities), 0), nil
}

func buildHomeSpaceConfigurationPayload(request contract.Request, houseID string) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "home.update":
		payload := map[string]any{"houseId": requestNumberOrString(houseID)}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{"name", "desc", "icon", "areaId", "areaCode", "areaName", "buildingName", "buildingAddr", "floorName"}) {
			return nil, nil, "", fmt.Errorf("invalid_home_update_payload")
		}
		return payload, []string{
			"提交前读取当前家庭详情",
			"只更新计划中显式提交的家庭资料字段",
			"plan.commit 只接受 planId，忽略提交时附带的更新字段",
			"提交后通过家庭详情回读验证；名称字段可做精确验证，其余字段按云端写入确认加详情可读性验证",
		}, "更新家庭资料", nil
	case "room.batch_create":
		rooms, ok := normalizeHomeSpaceRoomItems(firstNonNil(request.Parameters["rooms"], request.Parameters["items"]), false)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_room_batch_create_payload")
		}
		return map[string]any{
				"houseId": requestNumberOrString(houseID),
				"rooms":   rooms,
			}, []string{
				"提交前重新读取家庭实体列表",
				"所有新房间名称必须非空且当前家庭内不存在",
				"单次计划最多创建 20 个房间",
				"提交后通过 entity.list 按房间名称逐项验证",
			}, fmt.Sprintf("批量创建%d个房间", len(rooms)), nil
	case "room.batch_update":
		rooms, ok := normalizeHomeSpaceRoomItems(firstNonNil(request.Parameters["rooms"], request.Parameters["items"]), true)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_room_batch_update_payload")
		}
		return map[string]any{
				"houseId": requestNumberOrString(houseID),
				"rooms":   rooms,
			}, []string{
				"提交前重新读取家庭实体列表",
				"每个目标房间必须属于当前家庭",
				"单次计划最多更新 20 个房间",
				"plan.commit 只接受 planId，忽略提交时附带的更新字段",
				"提交后通过 entity.list 按房间 id/name 逐项验证",
			}, fmt.Sprintf("批量更新%d个房间", len(rooms)), nil
	case "room.area.configure":
		roomID := firstRequestString(request.Parameters, "roomId", "id", "entityId")
		addAreaList := requestStringList(request.Parameters["addAreaList"], request.Parameters["addAreaIds"], request.Parameters["addAreaId"])
		removeAreaList := requestStringList(request.Parameters["removeAreaList"], request.Parameters["removeAreaIds"], request.Parameters["removeAreaId"])
		if roomID == "" || (len(addAreaList) == 0 && len(removeAreaList) == 0) {
			return nil, nil, "", fmt.Errorf("invalid_room_area_configure_payload")
		}
		return map[string]any{
				"houseId":        requestNumberOrString(houseID),
				"roomId":         roomID,
				"addAreaList":    stringListAsRequestIDs(addAreaList),
				"removeAreaList": stringListAsRequestIDs(removeAreaList),
			}, []string{
				"提交前重新读取家庭实体列表",
				"目标房间和所有区域必须属于当前家庭",
				"plan.commit 只接受 planId，忽略提交时附带的区域字段",
				"提交后通过 entity.list 确认房间和区域仍可访问；区域归属字段以云端写入确认作为证据",
			}, "调整房间区域归属", nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported_home_space_configuration_intent")
	}
}

func normalizeHomeSpaceRoomItems(value any, requireID bool) ([]any, bool) {
	items, ok := requestMapList(value)
	if !ok || len(items) == 0 || len(items) > 20 {
		return nil, false
	}
	result := make([]any, 0, len(items))
	for _, item := range items {
		room := map[string]any{}
		for _, key := range []string{"roomId", "id", "name", "desc", "icon", "img", "gatewayDeviceId", "gatewayIds", "gatewayDeviceIds", "defaultGatewayIds", "seq", "rank", "capability"} {
			if value, ok := item[key]; ok {
				room[key] = value
			}
		}
		if requireID {
			roomID := firstRequestString(room, "roomId", "id")
			if roomID == "" {
				roomID = firstValueIDString(room, "roomId", "id")
			}
			if roomID == "" {
				return nil, false
			}
			room["id"] = roomID
			room["roomId"] = roomID
		}
		if !requireID && strings.TrimSpace(requestString(room["name"])) == "" {
			return nil, false
		}
		result = append(result, room)
	}
	return result, true
}

func validateHomeSpaceConfigurationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "home.update":
		return ""
	case "room.batch_create":
		rooms, ok := payload["rooms"].([]any)
		if !ok || len(rooms) == 0 || len(rooms) > 20 {
			return "invalid_room_batch_create_payload"
		}
		seenNames := map[string]bool{}
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				return "invalid_room_batch_create_payload"
			}
			name := strings.TrimSpace(requestString(room["name"]))
			if name == "" {
				return "invalid_room_batch_create_payload"
			}
			if seenNames[name] {
				return "duplicate_room_name"
			}
			seenNames[name] = true
			for _, entity := range entities.Entities {
				if entity.Type == "room" && entity.Name == name {
					return "room_name_already_exists"
				}
			}
			if reason := validateRoomUpdateGatewayReferences(room, entities); reason != "" {
				return reason
			}
		}
	case "room.batch_update":
		rooms, ok := payload["rooms"].([]any)
		if !ok || len(rooms) == 0 || len(rooms) > 20 {
			return "invalid_room_batch_update_payload"
		}
		seenIDs := map[string]bool{}
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				return "invalid_room_batch_update_payload"
			}
			roomID := valueIDString(firstNonNil(room["roomId"], room["id"]))
			current, exists := findEntitySummary(entities, "room", roomID)
			if !exists {
				return "invalid_room_reference"
			}
			if seenIDs[roomID] {
				return "duplicate_room_target"
			}
			seenIDs[roomID] = true
			if strings.TrimSpace(requestString(room["name"])) == "" {
				room["name"] = current.Name
			}
			if name := strings.TrimSpace(requestString(room["name"])); name != "" {
				for _, entity := range entities.Entities {
					if entity.Type == "room" && entity.ID != roomID && entity.Name == name {
						return "room_name_already_exists"
					}
				}
			}
			if reason := validateRoomUpdateGatewayReferences(room, entities); reason != "" {
				return reason
			}
		}
	case "room.area.configure":
		roomID := valueIDString(payload["roomId"])
		if !entityExists(entities, "room", roomID) {
			return "invalid_room_reference"
		}
		areaIDs := append(valueIDList(payload["addAreaList"]), valueIDList(payload["removeAreaList"])...)
		if len(areaIDs) == 0 {
			return "invalid_room_area_configure_payload"
		}
		seen := map[string]bool{}
		for _, areaID := range areaIDs {
			if seen[areaID] {
				return "duplicate_area_target"
			}
			seen[areaID] = true
			if !entityExists(entities, "area", areaID) {
				return "invalid_area_reference"
			}
		}
	default:
		return "unsupported_home_space_configuration_intent"
	}
	return ""
}

func homeSpaceConfigurationPreview(intent string, payload map[string]any, entities api.EntityListResult) map[string]any {
	preview := map[string]any{"planned": pendingPlanPayloadPreview(plan.Record{HouseID: requestString(payload["houseId"]), Payload: payload})}
	switch intent {
	case "room.batch_update":
		rooms, _ := payload["rooms"].([]any)
		current := make([]any, 0, len(rooms))
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if entity, ok := findEntitySummary(entities, "room", valueIDString(firstNonNil(room["roomId"], room["id"]))); ok {
				current = append(current, map[string]any{"id": entity.ID, "name": entity.Name})
			}
		}
		preview["current"] = current
	case "room.area.configure":
		if entity, ok := findEntitySummary(entities, "room", valueIDString(payload["roomId"])); ok {
			preview["current"] = map[string]any{"id": entity.ID, "name": entity.Name}
		}
	}
	return preview
}

func homeSpaceConfigurationAcceptedFields(intent string) []string {
	switch intent {
	case "home.update":
		return []string{"parameters.houseId", "parameters.name", "parameters.desc", "parameters.icon", "parameters.areaId", "parameters.areaCode", "parameters.areaName", "parameters.buildingName", "parameters.buildingAddr", "parameters.floorName"}
	case "room.batch_create":
		return []string{"parameters.houseId", "parameters.rooms[].name", "parameters.rooms[].desc", "parameters.rooms[].icon", "parameters.rooms[].gatewayDeviceId"}
	case "room.batch_update":
		return []string{"parameters.houseId", "parameters.rooms[].roomId", "parameters.rooms[].name", "parameters.rooms[].img", "parameters.rooms[].gatewayDeviceId", "parameters.rooms[].seq", "parameters.rooms[].capability"}
	case "room.area.configure":
		return []string{"parameters.houseId", "parameters.roomId", "parameters.addAreaList", "parameters.removeAreaList"}
	default:
		return []string{"parameters.houseId"}
	}
}

func stringListAsRequestIDs(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, requestNumberOrString(value))
		}
	}
	return result
}

func (app *app) commitHomeSpaceConfigurationPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.HomeSpaceConfigurationKind) (contract.Response, error) {
	result, err := api.NewHomeSpaceConfigurationClient(endpoint, nil).Run(ctx, api.HomeSpaceConfigurationRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.HomeSpaceConfigurationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return homeSpaceConfigurationCommitResponse(request, record, result), nil
}
