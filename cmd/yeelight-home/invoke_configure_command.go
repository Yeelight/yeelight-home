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

const preparedExecutionTTL = 10 * time.Minute

func (app *app) prepareRoomCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	roomName := roomCreateName(request)
	if strings.TrimSpace(roomName) == "" {
		return configureClarificationResponse(request, "missing_room_name", semanticParameterPaths(semantic.FieldName, semantic.FieldRoomName)), nil
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
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
	for _, entity := range entities.Entities {
		if entity.Type == "room" && entity.Name == roomName {
			return roomCreateAlreadyExistsResponse(request, entities, entity), nil
		}
	}
	if reason := validateConfigureCreatePayload("room", nil, entities); reason != "" {
		return configureClarificationResponse(request, reason, semanticParameterPaths(semantic.FieldHouseID, semantic.FieldName)), nil
	}
	payload, err := api.BuildRoomCreatePayload(houseID, roomName, firstRequestString(request.Parameters, semantic.FieldDescription), firstRequestString(request.Parameters, semantic.FieldIcon))
	if err != nil {
		return configureClarificationResponse(request, "invalid_room_create_payload", semanticParameterPaths(semantic.FieldHouseID, semantic.FieldName)), nil
	}
	roomCreatePayload := map[string]any{
		semantic.FieldHouseID:     payload[semantic.FieldHouseID],
		semantic.FieldName:        payload[semantic.FieldName],
		semantic.FieldDescription: firstRequestString(request.Parameters, semantic.FieldDescription),
		semantic.FieldIcon:        payload[semantic.FieldIcon],
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, "room.create", request.RequestID, fmt.Sprintf("创建房间 %s", roomName), roomCreatePayload, []string{
		"提交前重新读取家庭实体列表",
		"房间名不存在时才创建",
		"创建后通过房间列表按名称验证",
	}, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func (app *app) prepareMetadataCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec configureCreateSpec) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
	}
	payload, err := spec.buildPayload(request, houseID)
	if err != nil {
		return configureClarificationResponseWithGuide(request, spec.invalidReason, spec.acceptedFields, payloadGuideForIntent(request.Intent)), nil
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
	if reason := validateConfigureCreatePayload(spec.entityType, payload, entities); reason != "" {
		if spec.entityType == "group" {
			candidates := groupCreateReferenceCandidateMaps(payload, entities, reason)
			return configureClarificationResponseWithCandidates(request, reason, spec.acceptedFields, payloadGuideForIntent(request.Intent), candidates), nil
		}
		return configureClarificationResponseWithGuide(request, reason, spec.acceptedFields, payloadGuideForIntent(request.Intent)), nil
	}
	name := executionPayloadString(payload, semantic.FieldName)
	for _, entity := range entities.Entities {
		if entity.Type == spec.entityType && entity.Name == name {
			return metadataCreateAlreadyExistsResponse(request, entities, entity, spec.entityLabel), nil
		}
	}
	extraAPICalls := 0
	warnings := []string{}
	if spec.entityType == "group" {
		groupCapability := firstRequestString(request.Parameters, semantic.FieldGroupCapability, semantic.FieldGroupCategory)
		calls, enrichWarnings, reason := enrichGroupCreatePayload(ctx, endpoint, houseID, authorization, clientID, groupCapability, payload)
		extraAPICalls += calls
		warnings = append(warnings, enrichWarnings...)
		if reason != "" {
			response := configureClarificationResponseWithGuide(request, reason, spec.acceptedFields, payloadGuideForIntent(request.Intent))
			response.Warnings = append(response.Warnings, warnings...)
			return response, nil
		}
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("创建%s %s", spec.entityLabel, name), payload, spec.preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	response := executionPreviewResponseWithDetails(request, record, entities, nil, extraAPICalls)
	response.Warnings = append(response.Warnings, warnings...)
	return response, nil
}

func executionVerifyCode(err error) (string, string) {
	if err == nil {
		return "operation_not_executable", "resend_same_intent"
	}
	if strings.Contains(err.Error(), "token-like") {
		return "operation_payload_rejected", "regenerate_without_sensitive_fields"
	}
	return "operation_not_executable", "resend_same_intent"
}

func appendWarning(warnings []string, warning string) []string {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return warnings
	}
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}

func (app *app) executePreparedExecution(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	switch record.Intent {
	case "home.create":
		return app.executeHomeCreate(ctx, request, endpoint, record, authorization, clientID)
	case "room.create":
		return app.executeRoomCreate(ctx, request, endpoint, record, authorization, clientID)
	case "area.create":
		return app.executeMetadataCreate(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindArea, "区域")
	case "group.create":
		return app.executeMetadataCreate(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindGroup, "设备组")
	case "scene.create":
		return app.executeMetadataCreate(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindScene, "情景")
	case "scene.update":
		return app.executeSceneUpdate(ctx, request, endpoint, record, authorization, clientID)
	case "lighting.design.apply":
		return app.executeLightingDesignApply(ctx, request, endpoint, record, authorization, clientID)
	case "lighting.design.import", "device.slot.create":
		return app.executeLightingDesignImport(ctx, request, endpoint, record, authorization, clientID)
	case "home.sort.configure":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationSortConfigure)
	case "favorite.add":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteAdd)
	case "favorite.update":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteUpdate)
	case "favorite.delete":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteDelete)
	case "favorite.batch_add":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchAdd)
	case "favorite.batch_update":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchUpdate)
	case "favorite.batch_delete":
		return app.executeHomeOrganization(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchDelete)
	case "home.member.invite":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberInvite)
	case "home.member.accept_share":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberAccept)
	case "home.member.configure":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberConfigure)
	case "home.member.remove":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberRemove)
	case "home.member.transfer":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberTransfer)
	case "home.member.quit":
		return app.executeHomeMember(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberQuit)
	case "home.lock_all":
		return app.executeHomeLock(ctx, request, endpoint, record, authorization, clientID, api.HomeLockAll)
	case "home.unlock_all":
		return app.executeHomeLock(ctx, request, endpoint, record, authorization, clientID, api.HomeUnlockAll)
	case "home.update":
		return app.executeHomeSpaceConfiguration(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceHomeUpdate)
	case "room.rename":
		return app.executeSpaceOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationRoomRename)
	case "room.update":
		return app.executeSpaceOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationRoomUpdate)
	case "room.batch_create":
		return app.executeHomeSpaceConfiguration(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomBatchCreate)
	case "room.batch_update":
		return app.executeHomeSpaceConfiguration(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomBatchUpdate)
	case "room.area.configure":
		return app.executeHomeSpaceConfiguration(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomAreaConfigure)
	case "area.update":
		return app.executeSpaceOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationAreaUpdate)
	case "device.rename":
		return app.executeSpaceOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationDeviceRename)
	case "device.move":
		return app.executeDeviceMove(ctx, request, endpoint, record, authorization, clientID)
	case "device.move_room.batch":
		return app.executeSpaceBatchOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceBatchDeviceMoveRoom)
	case "device.remove":
		return app.executeDestructiveDelete(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteDevice)
	case "device.unbind":
		return app.executeDeviceUnbind(ctx, request, endpoint, record, authorization, clientID)
	case "entity.rename.batch":
		return app.executeEntityBatchRename(ctx, request, endpoint, record, authorization, clientID)
	case "group.update":
		return app.executeSpaceOrganization(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationGroupUpdate)
	case "room.delete":
		return app.executeMetadataDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteRoom)
	case "area.delete":
		return app.executeMetadataDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteArea)
	case "group.delete":
		return app.executeMetadataDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteGroup)
	case "scene.delete":
		return app.executeMetadataDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteScene)
	case "automation.delete":
		return app.executeMetadataDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteAutomation)
	case "room.batch_delete":
		return app.executeMetadataBatchDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteRoom)
	case "area.batch_delete":
		return app.executeMetadataBatchDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteArea)
	case "group.batch_delete":
		return app.executeMetadataBatchDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteGroup)
	case "scene.batch_delete":
		return app.executeMetadataBatchDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteScene)
	case "automation.batch_delete":
		return app.executeMetadataBatchDelete(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteAutomation)
	case "gateway.delete":
		return app.executeDestructiveDelete(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteGateway)
	case "home.delete":
		return app.executeDestructiveDelete(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteHome)
	case "gateway.configure":
		return app.executeGatewayConfiguration(ctx, request, endpoint, record, authorization, clientID)
	case "panel.button.configure":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonConfigure)
	case "panel.button_event.update":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventUpdate)
	case "panel.button_event.batch_update":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventBatchUpdate)
	case "panel.button_event.reset":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventReset)
	case "knob.configure":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.KnobConfigure)
	case "knob.reset":
		return app.executePanelConfiguration(ctx, request, endpoint, record, authorization, clientID, api.KnobReset)
	case "automation.create":
		return app.executeMetadataCreate(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindAutomation, "自动化")
	case "automation.update":
		return app.executeAutomationUpdate(ctx, request, endpoint, record, authorization, clientID)
	case "automation.enable":
		return app.executeAutomationStatus(ctx, request, endpoint, record, authorization, clientID, api.AutomationStatusEnable)
	case "automation.disable":
		return app.executeAutomationStatus(ctx, request, endpoint, record, authorization, clientID, api.AutomationStatusDisable)
	case "operation.batch.configure":
		return app.executeOperationBatchConfigure(ctx, request, endpoint, record, authorization, clientID)
	default:
		return executionBlockedResponse(request, "unsupported_operation_intent", "当前 Runtime 尚未支持执行该操作类型。"), nil
	}
}

func (app *app) executeRoomCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	roomName := executionPayloadString(record.Payload, semantic.FieldName)
	result, err := api.NewRoomCreateClient(endpoint, nil).Run(ctx, api.RoomCreateRequest{
		HouseID:        record.HouseID,
		Name:           roomName,
		Description:    executionPayloadString(record.Payload, semantic.FieldDescription),
		Icon:           executionPayloadString(record.Payload, semantic.FieldIcon),
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.RoomCreateCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return roomCreateExecuteResponse(request, record, result), nil
}

func (app *app) executeMetadataCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.MetadataKind, label string) (contract.Response, error) {
	result, err := api.NewMetadataCreateClient(endpoint, nil).Run(ctx, api.MetadataCreateRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.MetadataCreateCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return metadataCreateExecuteResponse(request, record, result, label), nil
}

func (app *app) executeHomeOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.HomeOrganizationKind) (contract.Response, error) {
	result, err := api.NewHomeOrganizationClient(endpoint, nil).Run(ctx, api.HomeOrganizationRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.HomeOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return homeOrganizationExecuteResponse(request, record, result), nil
}

func roomCreateName(request contract.Request) string {
	return firstRequestString(request.Parameters, semantic.FieldName, semantic.FieldRoomName)
}

func executionPayloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}
