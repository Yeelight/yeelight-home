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

func (app *app) invokeSceneUpdatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
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
	payload, err := buildSceneUpdatePayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), sceneUpdateAcceptedFields()), nil
	}
	if reason := validateSceneUpdatePayload(payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, sceneUpdateAcceptedFields()), nil
	}
	name := planPayloadString(payload, "name")
	if name == "" {
		name = valueIDString(payload["sceneId"])
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("更新情景 %s", name), payload, []string{
		"提交前重新读取家庭实体列表",
		"目标情景必须属于当前家庭",
		"情景动作资源必须属于当前家庭",
		"提交后通过 scene.detail.get 验证情景摘要",
	}, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponse(request, record, entities), nil
}

func buildSceneUpdatePayload(request contract.Request, houseID string) (map[string]any, error) {
	sceneID := firstRequestString(request.Parameters, "sceneId", "id", "entityId")
	if sceneID == "" {
		return nil, fmt.Errorf("invalid_scene_update_payload")
	}
	details, ok := requestMapList(request.Parameters["details"])
	if !ok {
		detail, ok := sceneSingleDetail(request)
		if !ok {
			return nil, fmt.Errorf("invalid_scene_update_payload")
		}
		details = []map[string]any{detail}
	}
	payload, err := api.BuildSceneCreatePayload(
		houseID,
		configureName(request),
		firstRequestString(request.Parameters, "description", "desc"),
		firstRequestString(request.Parameters, "icon"),
		details,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid_scene_update_payload")
	}
	payload["id"] = requestNumberOrString(sceneID)
	payload["sceneId"] = sceneID
	if roomID := firstRequestString(request.Parameters, "roomId", "room_id"); roomID != "" {
		payload["roomId"] = requestNumberOrString(roomID)
	}
	if gatewayDeviceID := firstRequestString(request.Parameters, "gatewayDeviceId", "gateway_device_id"); gatewayDeviceID != "" {
		payload["gatewayDeviceId"] = requestNumberOrString(gatewayDeviceID)
	}
	return payload, nil
}

func validateSceneUpdatePayload(payload map[string]any, entities api.EntityListResult) string {
	sceneID := valueIDString(payload["sceneId"])
	if !entityExists(entities, "scene", sceneID) {
		return "invalid_scene_reference"
	}
	delete(payload, "sceneId")
	reason := validateSceneCreatePayload(payload, entities)
	payload["sceneId"] = sceneID
	if reason == "house_scene_limit_exceeded" {
		return ""
	}
	if reason == "invalid_scene_create_payload" {
		return "invalid_scene_update_payload"
	}
	return reason
}

func sceneUpdateAcceptedFields() []string {
	return []string{
		"parameters.houseId",
		"parameters.sceneId",
		"parameters.name",
		"parameters.description",
		"parameters.icon",
		"parameters.details",
		"parameters.deviceId",
		"parameters.params",
		"parameters.roomId",
		"parameters.gatewayDeviceId",
	}
}

func (app *app) commitSceneUpdatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewSceneUpdateClient(endpoint, nil).Run(ctx, api.SceneUpdateRequest{
		HouseID:        record.HouseID,
		SceneID:        planPayloadString(record.Payload, "sceneId"),
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SceneUpdateCredentials{
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
	return sceneUpdateCommitResponse(request, record, result), nil
}
