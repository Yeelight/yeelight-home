package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareHomeSpaceConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
	}
	payload, preconditions, summary, err := buildHomeSpaceConfigurationPayload(request, houseID)
	if err != nil {
		return homeSpaceConfigurationClarificationResponse(request, err.Error()), nil
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
	if reason := resolveHomeSpaceConfigurationReferences(request.Intent, payload, entities); reason != "" {
		return homeSpaceConfigurationClarificationResponse(request, reason), nil
	}
	if reason := validateHomeSpaceConfigurationPayload(request.Intent, payload, entities); reason != "" {
		return homeSpaceConfigurationClarificationResponse(request, reason), nil
	}
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, homeSpaceConfigurationPreview(request.Intent, payload, entities), 0), nil
}

func buildHomeSpaceConfigurationPayload(request contract.Request, houseID string) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "home.update":
		payload := map[string]any{semantic.FieldHouseID: requestNumberOrString(houseID)}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{
			semantic.FieldName,
			semantic.FieldDescription,
			semantic.FieldIcon,
			semantic.FieldAreaID,
			semantic.FieldAreaCode,
			semantic.FieldAreaName,
			semantic.FieldBuildingName,
			semantic.FieldBuildingAddress,
			semantic.FieldFloorName,
		}) {
			return nil, nil, "", fmt.Errorf("invalid_home_update_payload")
		}
		return payload, []string{
			"提交前读取当前家庭详情",
			"只更新计划中显式提交的家庭资料字段",
			"Runtime 根据当前请求构建受控更新 payload",
			"提交后通过家庭详情回读验证；名称字段可做精确验证，其余字段按云端写入确认加详情可读性验证",
		}, "更新家庭资料", nil
	case "room.batch_create":
		rooms, ok := normalizeHomeSpaceRoomItems(firstNonNil(request.Parameters[semantic.FieldRooms], request.Parameters[semantic.FieldItems]), false)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_room_batch_create_payload")
		}
		return map[string]any{
				semantic.FieldHouseID: requestNumberOrString(houseID),
				semantic.FieldRooms:   rooms,
			}, []string{
				"提交前重新读取家庭实体列表",
				"所有新房间名称必须非空且当前家庭内不存在",
				"单次计划最多创建 20 个房间",
				"提交后通过 entity.list 按房间名称逐项验证",
			}, fmt.Sprintf("批量创建%d个房间", len(rooms)), nil
	case "room.batch_update":
		rooms, ok := normalizeHomeSpaceRoomItems(firstNonNil(request.Parameters[semantic.FieldRooms], request.Parameters[semantic.FieldItems]), true)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_room_batch_update_payload")
		}
		return map[string]any{
				semantic.FieldHouseID: requestNumberOrString(houseID),
				semantic.FieldRooms:   rooms,
			}, []string{
				"提交前重新读取家庭实体列表",
				"每个目标房间必须属于当前家庭",
				"单次计划最多更新 20 个房间",
				"Runtime 根据当前请求构建受控更新 payload",
				"提交后通过 entity.list 按房间 id/name 逐项验证",
			}, fmt.Sprintf("批量更新%d个房间", len(rooms)), nil
	case "room.area.configure":
		roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldID, semantic.FieldEntityID)
		roomName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldEntityName, semantic.FieldTargetName)
		addAreaIDs := requestStringList(request.Parameters[semantic.FieldAddAreaIDs])
		removeAreaIDs := requestStringList(request.Parameters[semantic.FieldRemoveAreaIDs])
		addAreaNames := requestStringList(request.Parameters[semantic.FieldAddAreaNames])
		removeAreaNames := requestStringList(request.Parameters[semantic.FieldRemoveAreaNames])
		if (roomID == "" && roomName == "") || (len(addAreaIDs) == 0 && len(removeAreaIDs) == 0 && len(addAreaNames) == 0 && len(removeAreaNames) == 0) {
			return nil, nil, "", fmt.Errorf("invalid_room_area_configure_payload")
		}
		payload := map[string]any{
			semantic.FieldHouseID: requestNumberOrString(houseID),
			semantic.FieldRoomID:  roomID,
			semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs):    stringListAsRequestIDs(addAreaIDs),
			semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs): stringListAsRequestIDs(removeAreaIDs),
		}
		if roomName != "" {
			payload[semantic.FieldCurrentName] = roomName
		}
		if len(addAreaNames) > 0 {
			payload[semantic.FieldAddAreaNames] = stringListAsRequestIDs(addAreaNames)
		}
		if len(removeAreaNames) > 0 {
			payload[semantic.FieldRemoveAreaNames] = stringListAsRequestIDs(removeAreaNames)
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标房间和所有区域必须属于当前家庭",
			"Runtime 根据当前请求构建受控区域 payload",
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
		for _, key := range []string{
			semantic.FieldRoomID,
			semantic.FieldID,
			semantic.FieldName,
			semantic.FieldDescription,
			semantic.FieldIcon,
			semantic.FieldImage,
			semantic.FieldGatewayDeviceID,
			semantic.FieldGatewayIDs,
			semantic.FieldDefaultGatewayIDs,
			semantic.FieldSequence,
			semantic.FieldRank,
			semantic.FieldCapability,
		} {
			if value, ok := item[key]; ok {
				room[semantic.InternalField(semantic.DomainCommon, key)] = value
			}
		}
		if requireID {
			roomID := firstRequestString(room, semantic.FieldRoomID, semantic.FieldID)
			if roomID == "" {
				roomID = firstValueIDString(room, semantic.FieldRoomID, semantic.FieldID)
			}
			if roomID == "" {
				return nil, false
			}
			room[semantic.FieldID] = roomID
			room[semantic.FieldRoomID] = roomID
		}
		if !requireID && strings.TrimSpace(requestString(room[semantic.FieldName])) == "" {
			return nil, false
		}
		result = append(result, room)
	}
	return result, true
}

func resolveHomeSpaceConfigurationReferences(intent string, payload map[string]any, entities api.EntityListResult) string {
	if intent != "room.area.configure" {
		return ""
	}
	if valueIDString(payload[semantic.FieldRoomID]) == "" {
		roomName := strings.TrimSpace(requestString(payload[semantic.FieldCurrentName]))
		if roomName == "" {
			return "invalid_room_reference"
		}
		match, ambiguous := findUniqueEntityByName(entities, "room", roomName)
		if ambiguous {
			return "ambiguous_room_reference"
		}
		if match.ID == "" {
			return "invalid_room_reference"
		}
		payload[semantic.FieldRoomID] = match.ID
	}
	if reason := resolveAreaNameList(payload, semantic.FieldAddAreaNames, semantic.FieldAddAreaIDs, entities); reason != "" {
		return reason
	}
	if reason := resolveAreaNameList(payload, semantic.FieldRemoveAreaNames, semantic.FieldRemoveAreaIDs, entities); reason != "" {
		return reason
	}
	return ""
}

func resolveAreaNameList(payload map[string]any, namesKey string, idsKey string, entities api.EntityListResult) string {
	areaNames := requestStringList(payload[namesKey])
	if len(areaNames) == 0 {
		return ""
	}
	internalIDsKey := semantic.InternalField(semantic.DomainCommon, idsKey)
	areaIDs := valueIDList(payload[internalIDsKey])
	for _, areaName := range areaNames {
		match, ambiguous := findUniqueEntityByName(entities, "area", areaName)
		if ambiguous {
			return "ambiguous_area_reference"
		}
		if match.ID == "" {
			return "invalid_area_reference"
		}
		areaIDs = append(areaIDs, match.ID)
	}
	payload[internalIDsKey] = stringListAsRequestIDs(areaIDs)
	delete(payload, namesKey)
	return ""
}

func validateHomeSpaceConfigurationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "home.update":
		return ""
	case "room.batch_create":
		rooms, ok := payload[semantic.FieldRooms].([]any)
		if !ok || len(rooms) == 0 || len(rooms) > 20 {
			return "invalid_room_batch_create_payload"
		}
		seenNames := map[string]bool{}
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				return "invalid_room_batch_create_payload"
			}
			name := strings.TrimSpace(requestString(room[semantic.FieldName]))
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
		rooms, ok := payload[semantic.FieldRooms].([]any)
		if !ok || len(rooms) == 0 || len(rooms) > 20 {
			return "invalid_room_batch_update_payload"
		}
		seenIDs := map[string]bool{}
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				return "invalid_room_batch_update_payload"
			}
			roomID := valueIDString(firstNonNil(room[semantic.FieldRoomID], room[semantic.FieldID]))
			current, exists := findEntitySummary(entities, "room", roomID)
			if !exists {
				return "invalid_room_reference"
			}
			if seenIDs[roomID] {
				return "duplicate_room_target"
			}
			seenIDs[roomID] = true
			if strings.TrimSpace(requestString(room[semantic.FieldName])) == "" {
				room[semantic.FieldName] = current.Name
			}
			if name := strings.TrimSpace(requestString(room[semantic.FieldName])); name != "" {
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
		roomID := valueIDString(payload[semantic.FieldRoomID])
		if !entityExists(entities, "room", roomID) {
			return "invalid_room_reference"
		}
		areaIDs := append(
			valueIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs)]),
			valueIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs)])...,
		)
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
	preview := map[string]any{semantic.FieldPlanned: executionPayloadPreview(operation.Prepared{HouseID: requestString(payload[semantic.FieldHouseID]), Payload: payload})}
	switch intent {
	case "room.batch_update":
		rooms, _ := payload[semantic.FieldRooms].([]any)
		current := make([]any, 0, len(rooms))
		for _, raw := range rooms {
			room, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if entity, ok := findEntitySummary(entities, "room", valueIDString(firstNonNil(room[semantic.FieldRoomID], room[semantic.FieldID]))); ok {
				current = append(current, map[string]any{semantic.FieldID: entity.ID, semantic.FieldName: entity.Name})
			}
		}
		preview[semantic.FieldCurrent] = current
	case "room.area.configure":
		if entity, ok := findEntitySummary(entities, "room", valueIDString(payload[semantic.FieldRoomID])); ok {
			preview[semantic.FieldCurrent] = map[string]any{semantic.FieldID: entity.ID, semantic.FieldName: entity.Name}
		}
		planned := map[string]any{}
		if addAreaIDs := valueIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs)]); len(addAreaIDs) > 0 {
			planned[semantic.FieldAddAreaIDs] = addAreaIDs
		}
		if removeAreaIDs := valueIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs)]); len(removeAreaIDs) > 0 {
			planned[semantic.FieldRemoveAreaIDs] = removeAreaIDs
		}
		if len(planned) > 0 {
			preview[semantic.FieldPlanned] = planned
		}
	}
	return preview
}

func homeSpaceConfigurationAcceptedFields(intent string) []string {
	switch intent {
	case "home.update":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldAreaID, semantic.FieldAreaCode, semantic.FieldAreaName, semantic.FieldBuildingName, semantic.FieldBuildingAddress, semantic.FieldFloorName)
	case "room.batch_create":
		return []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldName),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldDescription),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldIcon),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldGatewayDeviceID),
		}
	case "room.batch_update":
		return []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldRoomID),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldName),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldImage),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldGatewayDeviceID),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldSequence),
			semanticParameterArrayPath(semantic.FieldRooms, semantic.FieldCapability),
		}
	case "room.area.configure":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldAddAreaIDs, semantic.FieldRemoveAreaIDs, semantic.FieldAddAreaNames, semantic.FieldRemoveAreaNames)
	default:
		return semanticParameterPaths(semantic.FieldHouseID)
	}
}

func homeSpaceConfigurationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, homeSpaceConfigurationAcceptedFields(request.Intent), payloadGuideForIntent(request.Intent))
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

func (app *app) executeHomeSpaceConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.HomeSpaceConfigurationKind) (contract.Response, error) {
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
	return homeSpaceConfigurationExecuteResponse(request, record, result), nil
}
