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

const pendingPlanTTL = 10 * time.Minute

func (app *app) invokeRoomCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	roomName := roomCreateName(request)
	if strings.TrimSpace(roomName) == "" {
		return configureClarificationResponse(request, "missing_room_name", []string{"parameters.name", "parameters.roomName"}), nil
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
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
		return configureClarificationResponse(request, reason, []string{"parameters.houseId", "parameters.name"}), nil
	}
	payload, err := api.BuildRoomCreatePayload(houseID, roomName, firstRequestString(request.Parameters, "description", "desc"), firstRequestString(request.Parameters, "icon"))
	if err != nil {
		return configureClarificationResponse(request, "invalid_room_create_payload", []string{"parameters.houseId", "parameters.name"}), nil
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, "room.create", request.RequestID, fmt.Sprintf("创建房间 %s", roomName), payload, []string{
		"提交前重新读取家庭实体列表",
		"房间名不存在时才创建",
		"创建后通过房间列表按名称验证",
	}, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponse(request, record, entities), nil
}

func (app *app) invokeMetadataCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec configureCreateSpec) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, err := spec.buildPayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, spec.invalidReason, spec.acceptedFields), nil
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
		return configureClarificationResponse(request, reason, spec.acceptedFields), nil
	}
	name := planPayloadString(payload, "name")
	for _, entity := range entities.Entities {
		if entity.Type == spec.entityType && entity.Name == name {
			return metadataCreateAlreadyExistsResponse(request, entities, entity, spec.entityLabel), nil
		}
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("创建%s %s", spec.entityLabel, name), payload, spec.preconditions, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponse(request, record, entities), nil
}

func (app *app) invokePlanCommit(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, authorization string, clientID string) (contract.Response, error) {
	planID := firstRequestString(request.Parameters, "planId", "planID", "id")
	if planID == "" {
		return configureClarificationResponse(request, "missing_plan_id", []string{"parameters.planId"}), nil
	}
	record, ok, err := app.planStore.Load(planID)
	if err != nil {
		return contract.Response{}, err
	}
	if !ok {
		return planCommitBlockedResponse(request, planID, "plan_not_found", "未找到待提交计划。"), nil
	}
	if record.Profile != profile {
		return planCommitBlockedResponse(request, planID, "profile_mismatch", "计划不属于当前本地 profile。"), nil
	}
	if record.Region != region {
		return planCommitBlockedResponse(request, planID, "region_mismatch", "计划环境与当前 Runtime 环境不一致。"), nil
	}
	if err := record.Verify(time.Now()); err != nil {
		return planCommitBlockedResponse(request, planID, "plan_not_committable", err.Error()), nil
	}
	if record.ApprovalRequired && record.ApprovedAt <= 0 {
		return planCommitBlockedResponse(request, planID, "local_approval_required", "该计划属于 R3 高影响操作，必须先在本机终端运行确认计划返回的 approveCommand。"), nil
	}
	switch record.Intent {
	case "home.create":
		return app.commitHomeCreatePlan(ctx, request, endpoint, record, authorization, clientID)
	case "room.create":
		return app.commitRoomCreatePlan(ctx, request, endpoint, record, authorization, clientID)
	case "area.create":
		return app.commitMetadataCreatePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindArea, "区域")
	case "group.create":
		return app.commitMetadataCreatePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindGroup, "设备组")
	case "scene.create":
		return app.commitMetadataCreatePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindScene, "情景")
	case "scene.update":
		return app.commitSceneUpdatePlan(ctx, request, endpoint, record, authorization, clientID)
	case "lighting.design.apply":
		return app.commitLightingDesignApplyPlan(ctx, request, endpoint, record, authorization, clientID)
	case "memory.remember":
		return app.commitMemoryRememberPlan(ctx, request, record)
	case "home.sort.configure":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationSortConfigure)
	case "favorite.add":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteAdd)
	case "favorite.update":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteUpdate)
	case "favorite.delete":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteDelete)
	case "favorite.batch_add":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchAdd)
	case "favorite.batch_update":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchUpdate)
	case "favorite.batch_delete":
		return app.commitHomeOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeOrganizationFavoriteBatchDelete)
	case "home.member.invite":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberInvite)
	case "home.member.accept_share":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberAccept)
	case "home.member.configure":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberConfigure)
	case "home.member.remove":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberRemove)
	case "home.member.transfer":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberTransfer)
	case "home.member.quit":
		return app.commitHomeMemberPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeMemberQuit)
	case "home.lock_all":
		return app.commitHomeLockPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeLockAll)
	case "home.unlock_all":
		return app.commitHomeLockPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeUnlockAll)
	case "home.update":
		return app.commitHomeSpaceConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceHomeUpdate)
	case "room.rename":
		return app.commitSpaceOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationRoomRename)
	case "room.update":
		return app.commitSpaceOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationRoomUpdate)
	case "room.batch_create":
		return app.commitHomeSpaceConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomBatchCreate)
	case "room.batch_update":
		return app.commitHomeSpaceConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomBatchUpdate)
	case "room.area.configure":
		return app.commitHomeSpaceConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.HomeSpaceRoomAreaConfigure)
	case "area.update":
		return app.commitSpaceOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationAreaUpdate)
	case "device.rename":
		return app.commitSpaceOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationDeviceRename)
	case "device.move":
		return app.commitDeviceMovePlan(ctx, request, endpoint, record, authorization, clientID)
	case "device.move_room.batch":
		return app.commitSpaceBatchOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceBatchDeviceMoveRoom)
	case "device.remove":
		return app.commitDestructiveDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteDevice)
	case "device.unbind":
		return app.commitDeviceUnbindPlan(ctx, request, endpoint, record, authorization, clientID)
	case "entity.rename.batch":
		return app.commitEntityBatchRenamePlan(ctx, request, endpoint, record, authorization, clientID)
	case "group.update":
		return app.commitSpaceOrganizationPlan(ctx, request, endpoint, record, authorization, clientID, api.SpaceOrganizationGroupUpdate)
	case "room.delete":
		return app.commitMetadataDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteRoom)
	case "area.delete":
		return app.commitMetadataDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteArea)
	case "group.delete":
		return app.commitMetadataDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteGroup)
	case "scene.delete":
		return app.commitMetadataDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteScene)
	case "automation.delete":
		return app.commitMetadataDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataDeleteAutomation)
	case "room.batch_delete":
		return app.commitMetadataBatchDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteRoom)
	case "area.batch_delete":
		return app.commitMetadataBatchDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteArea)
	case "group.batch_delete":
		return app.commitMetadataBatchDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteGroup)
	case "scene.batch_delete":
		return app.commitMetadataBatchDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteScene)
	case "automation.batch_delete":
		return app.commitMetadataBatchDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataBatchDeleteAutomation)
	case "gateway.delete":
		return app.commitDestructiveDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteGateway)
	case "home.delete":
		return app.commitDestructiveDeletePlan(ctx, request, endpoint, record, authorization, clientID, api.DestructiveDeleteHome)
	case "gateway.configure":
		return app.commitGatewayConfigurationPlan(ctx, request, endpoint, record, authorization, clientID)
	case "panel.button.configure":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonConfigure)
	case "panel.button_event.update":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventUpdate)
	case "panel.button_event.batch_update":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventBatchUpdate)
	case "panel.button_event.reset":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.PanelButtonEventReset)
	case "knob.configure":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.KnobConfigure)
	case "knob.reset":
		return app.commitPanelConfigurationPlan(ctx, request, endpoint, record, authorization, clientID, api.KnobReset)
	case "automation.create":
		return app.commitMetadataCreatePlan(ctx, request, endpoint, record, authorization, clientID, api.MetadataKindAutomation, "自动化")
	case "automation.update":
		return app.commitAutomationUpdatePlan(ctx, request, endpoint, record, authorization, clientID)
	case "automation.enable":
		return app.commitAutomationStatusPlan(ctx, request, endpoint, record, authorization, clientID, api.AutomationStatusEnable)
	case "automation.disable":
		return app.commitAutomationStatusPlan(ctx, request, endpoint, record, authorization, clientID, api.AutomationStatusDisable)
	default:
		return planCommitBlockedResponse(request, planID, "unsupported_plan_intent", "当前 Runtime 尚未支持提交该计划类型。"), nil
	}
}

func (app *app) invokePlanCancel(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	planID := firstRequestString(request.Parameters, "planId", "planID", "id")
	if planID == "" {
		return configureClarificationResponse(request, "missing_plan_id", []string{"parameters.planId"}), nil
	}
	record, ok, err := app.planStore.Load(planID)
	if err != nil {
		return contract.Response{}, err
	}
	if !ok {
		return planCancelBlockedResponse(request, planID, "plan_not_found", "未找到要取消的本地计划。"), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if record.Profile != profile || record.Region != region || (!plan.IsAccountScope(record.HouseID) && record.HouseID != houseID) {
		return planCancelBlockedResponse(request, planID, "plan_scope_mismatch", "该计划不属于当前本地配置范围，未取消。"), nil
	}
	if err := record.Verify(time.Now()); err != nil {
		return planCancelBlockedResponse(request, planID, "plan_not_cancelable", "该计划已过期、已提交、已取消或校验失败，未重复取消。"), nil
	}
	canceled, err := app.planStore.MarkCanceled(planID)
	if err != nil {
		return contract.Response{}, err
	}
	return planCancelResponse(request, canceled), nil
}

func (app *app) invokeExecutionUndo(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	planID := firstRequestString(request.Parameters, "planId", "planID", "id")
	if planID == "" {
		return planCancelBlockedResponse(request, "", "undo_requires_plan_id", "当前 Runtime 只支持撤销未提交的本地待确认计划；请提供要取消的 planId。"), nil
	}
	response, err := app.invokePlanCancel(request, profile, region, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	if response.Status == "success" {
		response.UserMessage = "已撤销未提交的本地待确认计划。"
		response.TraceID = "execution-undo-plan-cancel"
		if response.Result != nil {
			response.Result["undoType"] = "pending_plan_cancel"
			response.Result["persistentWrites"] = false
		}
	}
	return response, nil
}

func (app *app) commitRoomCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string) (contract.Response, error) {
	roomName := planPayloadString(record.Payload, "name")
	result, err := api.NewRoomCreateClient(endpoint, nil).Run(ctx, api.RoomCreateRequest{
		HouseID:        record.HouseID,
		Name:           roomName,
		Description:    planPayloadString(record.Payload, "desc"),
		Icon:           planPayloadString(record.Payload, "icon"),
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return roomCreateCommitResponse(request, record, result), nil
}

func (app *app) commitMetadataCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.MetadataKind, label string) (contract.Response, error) {
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return metadataCreateCommitResponse(request, record, result, label), nil
}

func (app *app) commitHomeOrganizationPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.HomeOrganizationKind) (contract.Response, error) {
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return homeOrganizationCommitResponse(request, record, result), nil
}

func roomCreateName(request contract.Request) string {
	return firstRequestString(request.Parameters, "name", "roomName", "room_name")
}

func planPayloadString(payload map[string]any, key string) string {
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
