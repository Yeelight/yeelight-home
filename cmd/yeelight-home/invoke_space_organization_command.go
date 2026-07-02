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
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareSpaceOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
			"local profile houseId",
		}), nil
	}
	payload, preconditions, summary, err := buildSpaceOrganizationPayload(request, houseID)
	if err != nil {
		return spaceOrganizationClarificationResponse(request, err.Error()), nil
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
	if reason := resolveSpaceOrganizationReferences(request.Intent, payload, entities); reason != "" {
		return spaceOrganizationClarificationResponse(request, reason), nil
	}
	if reason := validateSpaceOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return spaceOrganizationClarificationResponse(request, reason), nil
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
		roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldEntityName, semantic.FieldTargetName)
		name := firstRequestString(request.Parameters, semantic.FieldNewName, semantic.FieldName)
		if name == "" || (roomID == "" && currentName == "") {
			return nil, nil, "", fmt.Errorf("invalid_room_rename_payload")
		}
		payload := map[string]any{
			semantic.FieldHouseID: requestNumberOrString(houseID),
			semantic.FieldRoomID:  roomID,
			semantic.FieldID:      roomID,
			semantic.FieldName:    name,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标房间必须属于当前家庭",
			"提交后通过 entity.list 验证房间名称",
		}, fmt.Sprintf("重命名房间为 %s", name), nil
	case "room.update":
		roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldEntityName, semantic.FieldTargetName)
		payload := map[string]any{
			semantic.FieldHouseID: requestNumberOrString(houseID),
			semantic.FieldRoomID:  roomID,
			semantic.FieldID:      roomID,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldImage, semantic.FieldGatewayDeviceID, semantic.FieldGatewayIDs, semantic.FieldDefaultGatewayIDs, semantic.FieldSequence, semantic.FieldCapability}) || (roomID == "" && currentName == "") {
			return nil, nil, "", fmt.Errorf("invalid_room_update_payload")
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标房间和引用网关设备必须属于当前家庭",
			"提交后通过 entity.list 验证房间名称；图片、网关和能力字段按云端写入响应确认",
		}, "更新房间信息", nil
	case "area.update":
		areaID := firstRequestString(request.Parameters, semantic.FieldAreaID, semantic.FieldID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldAreaName, semantic.FieldEntityName, semantic.FieldTargetName)
		payload := map[string]any{
			semantic.FieldHouseID: requestNumberOrString(houseID),
			semantic.FieldAreaID:  areaID,
			semantic.FieldID:      areaID,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		if !copyOptionalSpaceFields(payload, request.Parameters, []string{semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldParentID, semantic.FieldRoomIDs}) || (areaID == "" && currentName == "") {
			return nil, nil, "", fmt.Errorf("invalid_area_update_payload")
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标区域、父区域和关联房间必须属于当前家庭",
			"提交后通过 entity.list 验证区域名称",
		}, "更新区域信息", nil
	case "device.rename":
		deviceID := firstRequestString(request.Parameters, semantic.FieldDeviceID, semantic.FieldID, semantic.FieldEntityID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName)
		name := firstRequestString(request.Parameters, semantic.FieldNewName, semantic.FieldName, semantic.FieldAlias)
		if name == "" || (deviceID == "" && currentName == "") {
			return nil, nil, "", fmt.Errorf("invalid_device_rename_payload")
		}
		payload := map[string]any{
			semantic.FieldHouseID:  requestNumberOrString(houseID),
			semantic.FieldDeviceID: deviceID,
			semantic.FieldID:       deviceID,
			semantic.FieldName:     name,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标设备必须属于当前家庭",
			"提交后通过 entity.list 验证设备名称",
		}, fmt.Sprintf("重命名设备为 %s", name), nil
	case "device.move":
		deviceID := firstRequestString(request.Parameters, semantic.FieldDeviceID, semantic.FieldID, semantic.FieldEntityID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName)
		roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldTargetRoomID, semantic.FieldTargetID)
		targetRoomName := firstRequestString(request.Parameters, semantic.FieldTargetRoomName, semantic.FieldRoomName)
		if (deviceID == "" && currentName == "") || (roomID == "" && targetRoomName == "") {
			return nil, nil, "", fmt.Errorf("invalid_device_move_payload")
		}
		payload := map[string]any{
			semantic.FieldHouseID:  requestNumberOrString(houseID),
			semantic.FieldDeviceID: deviceID,
			semantic.FieldID:       deviceID,
			semantic.FieldRoomID:   roomID,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		if targetRoomName != "" {
			payload[semantic.FieldTargetRoomName] = targetRoomName
		}
		return payload, []string{
			"提交前重新读取家庭实体列表",
			"目标设备和目标房间必须属于当前家庭",
			"提交后通过 entity.list 验证设备 roomId",
		}, "移动设备到指定房间", nil
	case "group.update":
		groupID := firstRequestString(request.Parameters, semantic.FieldGroupID, semantic.FieldID)
		currentName := firstRequestString(request.Parameters, semantic.FieldCurrentName, semantic.FieldGroupName, semantic.FieldEntityName, semantic.FieldTargetName)
		targetRoomName := firstRequestString(request.Parameters, semantic.FieldTargetRoomName, semantic.FieldRoomName)
		payload := map[string]any{
			semantic.FieldHouseID: requestNumberOrString(houseID),
			semantic.FieldGroupID: groupID,
			semantic.FieldID:      groupID,
		}
		if currentName != "" {
			payload[semantic.FieldCurrentName] = currentName
		}
		copied := copyOptionalSpaceFields(payload, request.Parameters, []string{semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldRoomID})
		if targetRoomName != "" {
			payload[semantic.FieldTargetRoomName] = targetRoomName
			copied = true
		}
		if !copied || (groupID == "" && currentName == "") {
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
				payload[semantic.InternalField(semantic.DomainCommon, key)] = strings.TrimSpace(typed)
				copied = true
			}
		case float64, int, int64, bool:
			payload[semantic.InternalField(semantic.DomainCommon, key)] = typed
			copied = true
		case []any:
			if len(typed) > 0 {
				payload[semantic.InternalField(semantic.DomainCommon, key)] = typed
				copied = true
			}
		case map[string]any:
			if len(typed) > 0 {
				payload[semantic.InternalField(semantic.DomainCommon, key)] = typed
				copied = true
			}
		}
	}
	return copied
}

func resolveSpaceOrganizationReferences(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "room.rename", "room.update":
		if valueIDString(payload[semantic.FieldRoomID]) != "" {
			return ""
		}
		currentName := strings.TrimSpace(requestString(payload[semantic.FieldCurrentName]))
		if currentName == "" {
			if intent == "room.update" {
				return "invalid_room_update_payload"
			}
			return "invalid_room_rename_payload"
		}
		match, ambiguous := findUniqueEntityByName(entities, "room", currentName)
		if ambiguous {
			return "ambiguous_room_reference"
		}
		if match.ID == "" {
			return "invalid_room_reference"
		}
		payload[semantic.FieldRoomID] = match.ID
		payload[semantic.FieldID] = match.ID
	case "device.rename", "device.move":
		if valueIDString(payload[semantic.FieldDeviceID]) == "" {
			currentName := strings.TrimSpace(requestString(payload[semantic.FieldCurrentName]))
			if currentName == "" {
				return "invalid_device_reference"
			}
			match, ambiguous := findUniqueEntityByName(entities, "device", currentName)
			if ambiguous {
				return "ambiguous_device_reference"
			}
			if match.ID == "" {
				return "invalid_device_reference"
			}
			payload[semantic.FieldDeviceID] = match.ID
			payload[semantic.FieldID] = match.ID
		}
		if intent == "device.move" && valueIDString(payload[semantic.FieldRoomID]) == "" {
			targetRoomName := strings.TrimSpace(requestString(payload[semantic.FieldTargetRoomName]))
			if targetRoomName == "" {
				return "invalid_target_room_reference"
			}
			match, ambiguous := findUniqueEntityByName(entities, "room", targetRoomName)
			if ambiguous {
				return "ambiguous_target_room_reference"
			}
			if match.ID == "" {
				return "invalid_target_room_reference"
			}
			payload[semantic.FieldRoomID] = match.ID
		}
	case "group.update":
		if valueIDString(payload[semantic.FieldGroupID]) == "" {
			currentName := strings.TrimSpace(requestString(payload[semantic.FieldCurrentName]))
			if currentName == "" {
				return "invalid_group_reference"
			}
			match, ambiguous := findUniqueEntityByName(entities, "group", currentName)
			if ambiguous {
				return "ambiguous_group_reference"
			}
			if match.ID == "" {
				return "invalid_group_reference"
			}
			payload[semantic.FieldGroupID] = match.ID
			payload[semantic.FieldID] = match.ID
		}
		if valueIDString(payload[semantic.FieldRoomID]) == "" {
			targetRoomName := strings.TrimSpace(requestString(payload[semantic.FieldTargetRoomName]))
			if targetRoomName != "" {
				match, ambiguous := findUniqueEntityByName(entities, "room", targetRoomName)
				if ambiguous {
					return "ambiguous_target_room_reference"
				}
				if match.ID == "" {
					return "invalid_target_room_reference"
				}
				payload[semantic.FieldRoomID] = match.ID
			}
		}
	case "area.update":
		if valueIDString(payload[semantic.FieldAreaID]) == "" {
			currentName := strings.TrimSpace(requestString(payload[semantic.FieldCurrentName]))
			if currentName == "" {
				return "invalid_area_update_payload"
			}
			match, ambiguous := findUniqueEntityByName(entities, "area", currentName)
			if ambiguous {
				return "ambiguous_area_reference"
			}
			if match.ID == "" {
				return "invalid_area_reference"
			}
			payload[semantic.FieldAreaID] = match.ID
			payload[semantic.FieldID] = match.ID
		}
	}
	return ""
}

func validateSpaceOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "room.rename", "room.update":
		roomID := valueIDString(payload[semantic.FieldRoomID])
		current, ok := findEntitySummary(entities, "room", roomID)
		if !ok {
			return "invalid_room_reference"
		}
		name := strings.TrimSpace(requestString(payload[semantic.FieldName]))
		if intent == "room.update" && name == "" {
			if current.Name == "" {
				return "invalid_room_update_payload"
			}
			payload[semantic.FieldName] = current.Name
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
		areaID := valueIDString(payload[semantic.FieldAreaID])
		if !entityExists(entities, "area", areaID) {
			return "invalid_area_reference"
		}
		if parentID := valueIDString(payload[semantic.FieldParentID]); parentID != "" && parentID != areaID && !entityExists(entities, "area", parentID) {
			return "invalid_parent_area_reference"
		}
		if parentID := valueIDString(payload[semantic.FieldParentID]); parentID == areaID {
			return "invalid_parent_area_reference"
		}
		if roomIDs, ok := payload[semantic.FieldRoomIDs].([]any); ok {
			if len(roomIDs) > 50 {
				return "area_room_limit_exceeded"
			}
			for _, roomID := range roomIDs {
				if !entityExists(entities, "room", valueIDString(roomID)) {
					return "invalid_area_room_reference"
				}
			}
		}
		if name := strings.TrimSpace(requestString(payload[semantic.FieldName])); name != "" {
			for _, entity := range entities.Entities {
				if entity.Type == "area" && entity.ID != areaID && entity.Name == name {
					return "area_name_already_exists"
				}
			}
		}
	case "device.rename":
		if !entityExists(entities, "device", valueIDString(payload[semantic.FieldDeviceID])) {
			return "invalid_device_reference"
		}
	case "device.move":
		if !entityExists(entities, "device", valueIDString(payload[semantic.FieldDeviceID])) {
			return "invalid_device_reference"
		}
		if !entityExists(entities, "room", valueIDString(payload[semantic.FieldRoomID])) {
			return "invalid_target_room_reference"
		}
	case "group.update":
		groupID := valueIDString(payload[semantic.FieldGroupID])
		if !entityExists(entities, "group", groupID) {
			return "invalid_group_reference"
		}
		if roomID := valueIDString(payload[semantic.FieldRoomID]); roomID != "" && !entityExists(entities, "room", roomID) {
			return "invalid_target_room_reference"
		}
		if name := strings.TrimSpace(requestString(payload[semantic.FieldName])); name != "" {
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
		entityType, entityID = "room", valueIDString(payload[semantic.FieldRoomID])
	case "area.update":
		entityType, entityID = "area", valueIDString(payload[semantic.FieldAreaID])
	case "device.rename", "device.move":
		entityType, entityID = "device", valueIDString(payload[semantic.FieldDeviceID])
	case "group.update":
		entityType, entityID = "group", valueIDString(payload[semantic.FieldGroupID])
	default:
		return preview
	}
	if current, ok := findEntitySummary(entities, entityType, entityID); ok {
		preview[semantic.FieldCurrent] = map[string]any{
			semantic.FieldType:   current.Type,
			semantic.FieldID:     current.ID,
			semantic.FieldName:   current.Name,
			semantic.FieldRoomID: current.RoomID,
		}
	}
	planned := map[string]any{}
	for _, key := range []string{
		semantic.FieldName,
		semantic.FieldDescription,
		semantic.FieldIcon,
		semantic.FieldParentID,
		semantic.FieldRoomIDs,
		semantic.FieldRoomID,
		semantic.FieldImage,
		semantic.FieldGatewayDeviceID,
		semantic.FieldGatewayIDs,
		semantic.FieldDefaultGatewayIDs,
		semantic.FieldSequence,
		semantic.FieldCapability,
	} {
		if value, ok := payload[semantic.InternalField(semantic.DomainCommon, key)]; ok {
			planned[key] = value
		}
	}
	if len(planned) > 0 {
		preview[semantic.FieldPlanned] = planned
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

func findUniqueEntityByName(entities api.EntityListResult, entityType string, name string) (api.EntitySummary, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return api.EntitySummary{}, false
	}
	match, candidates, _ := findEntity(entityGetTarget{name: name, entityType: entityType}, entities.Entities)
	if match.ID != "" {
		return match, false
	}
	return api.EntitySummary{}, len(candidates) > 0
}

func spaceOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "room.rename":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldNewName, semantic.FieldName)
	case "room.update":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldCurrentName, semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldImage, semantic.FieldGatewayDeviceID, semantic.FieldGatewayIDs, semantic.FieldDefaultGatewayIDs, semantic.FieldSequence, semantic.FieldCapability)
	case "area.update":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldAreaID, semantic.FieldCurrentName, semantic.FieldAreaName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldParentID, semantic.FieldRoomIDs)
	case "device.rename":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldCurrentName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldNewName, semantic.FieldName, semantic.FieldAlias)
	case "device.move":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldCurrentName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldRoomID, semantic.FieldTargetRoomID, semantic.FieldTargetRoomName, semantic.FieldRoomName)
	case "group.update":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldGroupID, semantic.FieldCurrentName, semantic.FieldGroupName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldRoomID, semantic.FieldTargetRoomName, semantic.FieldRoomName)
	default:
		return semanticParameterPaths(semantic.FieldHouseID)
	}
}

func spaceOrganizationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, spaceOrganizationAcceptedFields(request.Intent), payloadGuideForIntent(request.Intent))
}

func validateRoomUpdateGatewayReferences(payload map[string]any, entities api.EntityListResult) string {
	if gatewayID := valueIDString(payload[semantic.FieldGatewayDeviceID]); gatewayID != "" && !entityExists(entities, "device", gatewayID) {
		return "invalid_gateway_device_reference"
	}
	for _, key := range []string{semantic.FieldGatewayIDs, semantic.FieldDefaultGatewayIDs} {
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
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
	}
	payload, preconditions, summary, err := buildSpaceBatchOrganizationPayload(request, houseID)
	if err != nil {
		return spaceBatchOrganizationClarificationResponse(request, err.Error()), nil
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
	if reason := resolveSpaceBatchOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return spaceBatchOrganizationClarificationResponse(request, reason), nil
	}
	if reason := validateSpaceBatchOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return spaceBatchOrganizationClarificationResponse(request, reason), nil
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
		items, ok := normalizeDeviceRoomBatchItems(request.Parameters[semantic.FieldItems])
		if !ok {
			items, ok = normalizeDeviceRoomBatchNaturalItems(request.Parameters)
			if !ok {
				return nil, nil, "", fmt.Errorf("invalid_device_move_room_batch_payload")
			}
		}
		return map[string]any{
				semantic.FieldHouseID: requestNumberOrString(houseID),
				semantic.FieldItems:   items,
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
			deviceID := firstRequestString(item, semantic.FieldDeviceID, semantic.FieldID, semantic.FieldEntityID, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldName)
			roomID := firstRequestString(item, semantic.FieldRoomID, semantic.FieldTargetRoomID, semantic.FieldTargetID, semantic.FieldTargetRoomName)
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

func normalizeDeviceRoomBatchNaturalItems(parameters map[string]any) (map[string]any, bool) {
	deviceNames := requestStringList(parameters[semantic.FieldDeviceNames])
	if len(deviceNames) == 0 || len(deviceNames) > 20 {
		return nil, false
	}
	targetRoom := firstRequestString(parameters, semantic.FieldRoomID, semantic.FieldTargetRoomID, semantic.FieldTargetRoomName)
	if targetRoom == "" {
		return nil, false
	}
	result := map[string]any{}
	for _, deviceName := range deviceNames {
		deviceName = strings.TrimSpace(deviceName)
		if deviceName == "" {
			return nil, false
		}
		if _, exists := result[deviceName]; exists {
			return nil, false
		}
		result[deviceName] = targetRoom
	}
	return result, true
}

func resolveSpaceBatchOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	if intent != "device.move_room.batch" {
		return ""
	}
	items, ok := payload[semantic.FieldItems].(map[string]any)
	if !ok || len(items) == 0 {
		return "invalid_device_move_room_batch_payload"
	}
	resolved := map[string]any{}
	for rawDevice, rawRoom := range items {
		deviceID := strings.TrimSpace(rawDevice)
		if !entityExists(entities, "device", deviceID) {
			match, candidates, _ := findEntity(entityGetTarget{name: deviceID, entityType: "device"}, entities.Entities)
			if len(candidates) > 1 {
				return "ambiguous_device_reference"
			}
			if match.ID == "" {
				return "invalid_device_reference"
			}
			deviceID = match.ID
		}
		roomID := valueIDString(rawRoom)
		if !entityExists(entities, "room", roomID) {
			match, candidates, _ := findEntity(entityGetTarget{name: roomID, entityType: "room"}, entities.Entities)
			if len(candidates) > 1 {
				return "ambiguous_target_room_reference"
			}
			if match.ID == "" {
				return "invalid_target_room_reference"
			}
			roomID = match.ID
		}
		if _, exists := resolved[deviceID]; exists {
			return "duplicate_device_reference"
		}
		resolved[deviceID] = roomID
	}
	payload[semantic.FieldItems] = resolved
	return ""
}

func validateSpaceBatchOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "device.move_room.batch":
		items, ok := payload[semantic.FieldItems].(map[string]any)
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
	items, _ := payload[semantic.FieldItems].(map[string]any)
	deviceIDs := make([]string, 0, len(items))
	for deviceID := range items {
		deviceIDs = append(deviceIDs, deviceID)
	}
	sort.Strings(deviceIDs)
	previewItems := make([]any, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		item := map[string]any{
			semantic.FieldDeviceID:     deviceID,
			semantic.FieldTargetRoomID: valueIDString(items[deviceID]),
		}
		if current, ok := findEntitySummary(entities, "device", deviceID); ok {
			item[semantic.FieldCurrentRoomID] = current.RoomID
			item[semantic.FieldDeviceName] = current.Name
		}
		previewItems = append(previewItems, item)
	}
	return map[string]any{
		semantic.FieldItemCount: len(previewItems),
		semantic.FieldItems:     previewItems,
	}
}

func spaceBatchOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "device.move_room.batch":
		return []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.ParameterPath(semantic.ArrayField(semantic.FieldItems), semantic.FieldDeviceID),
			semantic.ParameterPath(semantic.ArrayField(semantic.FieldItems), semantic.FieldDeviceName),
			semantic.ParameterPath(semantic.ArrayField(semantic.FieldItems), semantic.FieldRoomID),
			semantic.ParameterPath(semantic.ArrayField(semantic.FieldItems), semantic.FieldTargetRoomName),
			semantic.ParameterPath(semantic.FieldDeviceNames),
			semantic.ParameterPath(semantic.FieldTargetRoomName),
			semantic.ParameterPath(semantic.FieldRoomID),
			semantic.ParameterPath(semantic.FieldItems) + " as {deviceId: roomId}",
			semantic.ParameterPath(semantic.FieldItems) + " as {deviceName: targetRoomName}",
		}
	default:
		return []string{semantic.ParameterPath(semantic.FieldHouseID)}
	}
}

func spaceBatchOrganizationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, spaceBatchOrganizationAcceptedFields(request.Intent), payloadGuideForIntent(request.Intent))
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
	deviceID := valueIDString(firstNonNil(record.Payload[semantic.FieldDeviceID], record.Payload[semantic.FieldID]))
	roomID := valueIDString(record.Payload[semantic.FieldRoomID])
	if deviceID == "" || roomID == "" {
		return contract.Response{}, fmt.Errorf("invalid_device_move_payload")
	}
	batchRecord := record
	batchRecord.Intent = "device.move_room.batch"
	batchRecord.Payload = map[string]any{
		semantic.FieldHouseID: record.HouseID,
		semantic.FieldItems: map[string]any{
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
	response.Result[semantic.FieldCapability] = "device.move"
	return response, nil
}
