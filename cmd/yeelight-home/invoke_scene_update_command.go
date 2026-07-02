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

func (app *app) prepareSceneUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
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
	payload, err := buildSceneUpdatePayload(request, houseID, entities)
	if err != nil {
		return sceneUpdateClarificationResponse(request, err.Error()), nil
	}
	if reason := validateSceneUpdatePayload(payload, entities); reason != "" {
		return sceneUpdateClarificationResponse(request, reason), nil
	}
	name := executionPayloadString(payload, semantic.FieldName)
	if name == "" {
		name = valueIDString(payload[semantic.FieldSceneID])
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("更新情景 %s", name), payload, []string{
		"提交前重新读取家庭实体列表",
		"目标情景必须属于当前家庭",
		"情景动作资源必须属于当前家庭",
		"提交后通过 scene.detail.get 验证情景摘要",
	}, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func sceneUpdateClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, sceneUpdateAcceptedFields(), scenePayloadGuide("scene.update"))
}

func buildSceneUpdatePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, error) {
	scene, err := resolveSceneUpdateTarget(request, entities)
	if err != nil {
		return nil, err
	}
	details, ok := normalizeSceneActionRows(request.Parameters[semantic.FieldActions])
	if !ok {
		return nil, fmt.Errorf("invalid_scene_update_payload")
	}
	name := firstRequestString(request.Parameters, semantic.FieldNewName, semantic.FieldName)
	if name == "" {
		name = scene.Name
	}
	payload, err := api.BuildSceneCreatePayload(
		houseID,
		name,
		firstRequestString(request.Parameters, semantic.FieldDescription),
		firstRequestString(request.Parameters, semantic.FieldIcon),
		details,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid_scene_update_payload")
	}
	payload[semantic.FieldID] = requestNumberOrString(scene.ID)
	payload[semantic.FieldSceneID] = scene.ID
	if roomID := firstRequestString(request.Parameters, semantic.FieldRoomID); roomID != "" {
		payload[semantic.FieldRoomID] = requestNumberOrString(roomID)
	}
	if gatewayDeviceID := firstRequestString(request.Parameters, semantic.FieldGatewayDeviceID); gatewayDeviceID != "" {
		payload[semantic.FieldGatewayDeviceID] = requestNumberOrString(gatewayDeviceID)
	}
	return payload, nil
}

type sceneUpdateTarget struct {
	ID   string
	Name string
}

func resolveSceneUpdateTarget(request contract.Request, entities api.EntityListResult) (sceneUpdateTarget, error) {
	sceneID := firstRequestString(request.Parameters, semantic.FieldSceneID, semantic.FieldID, semantic.FieldEntityID)
	if sceneID != "" {
		match, _, _ := findEntity(entityGetTarget{id: sceneID, entityType: "scene"}, entities.Entities)
		if match.ID == "" {
			return sceneUpdateTarget{}, fmt.Errorf("invalid_scene_reference")
		}
		return sceneUpdateTarget{ID: match.ID, Name: match.Name}, nil
	}
	sceneName := firstRequestString(
		request.Parameters,
		semantic.FieldSceneName,
		semantic.FieldCurrentName,
		semantic.FieldEntityName,
		semantic.FieldTargetName,
	)
	if sceneName == "" {
		return sceneUpdateTarget{}, fmt.Errorf("invalid_scene_update_payload")
	}
	match, candidates, _ := findEntity(entityGetTarget{name: sceneName, entityType: "scene"}, entities.Entities)
	if match.ID != "" {
		return sceneUpdateTarget{ID: match.ID, Name: match.Name}, nil
	}
	if len(candidates) > 0 {
		return sceneUpdateTarget{}, fmt.Errorf("ambiguous_scene_reference")
	}
	return sceneUpdateTarget{}, fmt.Errorf("invalid_scene_reference")
}

func validateSceneUpdatePayload(payload map[string]any, entities api.EntityListResult) string {
	sceneID := valueIDString(payload[semantic.FieldSceneID])
	if !entityExists(entities, "scene", sceneID) {
		return "invalid_scene_reference"
	}
	delete(payload, semantic.FieldSceneID)
	reason := validateSceneCreatePayload(payload, entities)
	payload[semantic.FieldSceneID] = sceneID
	if reason == "house_scene_limit_exceeded" {
		return ""
	}
	if reason == "invalid_scene_create_payload" {
		return "invalid_scene_update_payload"
	}
	return reason
}

func sceneUpdateAcceptedFields() []string {
	return append(semanticParameterPaths(
		semantic.FieldHouseID,
		semantic.FieldSceneID,
		semantic.FieldSceneName,
		semantic.FieldCurrentName,
		semantic.FieldEntityName,
		semantic.FieldTargetName,
		semantic.FieldName,
		semantic.FieldNewName,
		semantic.FieldDescription,
		semantic.FieldIcon,
		semantic.FieldActions,
		semantic.FieldRoomID,
		semantic.FieldGatewayDeviceID,
	),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet),
	)
}

func (app *app) executeSceneUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewSceneUpdateClient(endpoint, nil).Run(ctx, api.SceneUpdateRequest{
		HouseID:        record.HouseID,
		SceneID:        executionPayloadString(record.Payload, semantic.FieldSceneID),
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
	return sceneUpdateExecuteResponse(request, record, result), nil
}
