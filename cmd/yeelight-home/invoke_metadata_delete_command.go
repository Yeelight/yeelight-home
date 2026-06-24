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

func (app *app) invokeMetadataDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	targetType, targetIDKey, kind, ok := metadataDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_metadata_delete_intent", []string{"parameters.houseId"}), nil
	}
	targetID := firstRequestString(request.Parameters, targetIDKey, "id", "entityId")
	if targetID == "" {
		targetID = firstValueIDString(request.Parameters, targetIDKey, "id", "entityId")
	}
	targetName := firstRequestString(request.Parameters, "name", "entityName", targetType+"Name")
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
	match, candidates, matchedBy := findEntity(entityGetTarget{id: targetID, name: targetName, entityType: targetType}, entities.Entities)
	if match.ID == "" {
		return configureClarificationResponse(request, "entity_not_found", metadataDeleteAcceptedFields(request.Intent)), nil
	}
	if len(candidates) > 1 && targetID == "" {
		return configureClarificationResponse(request, "ambiguous_target", metadataDeleteAcceptedFields(request.Intent)), nil
	}
	payload := map[string]any{
		"houseId":    requestNumberOrString(houseID),
		"capability": string(kind),
		"entityType": targetType,
		"entityId":   match.ID,
		"name":       match.Name,
	}
	payload[targetIDKey] = match.ID
	preview := map[string]any{
		"deleteTarget": map[string]any{
			"type":      match.Type,
			"id":        match.ID,
			"name":      match.Name,
			"roomId":    match.RoomID,
			"matchedBy": matchedBy,
		},
		"impact": metadataDeleteImpact(targetType, match, entities),
	}
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("删除%s %s", metadataDeleteLabel(targetType), match.Name), payload, []string{
		"提交前重新读取家庭实体列表并确认目标仍存在",
		"只删除本计划中已解析的单个目标对象",
		"plan.commit 只接受 planId，忽略提交时附带的删除字段",
		"提交后通过 entity.list 验证目标对象已不存在",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, entities, preview, 0), nil
}

func firstValueIDString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := valueIDString(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func metadataDeleteIntentSpec(intent string) (string, string, api.MetadataDeleteKind, bool) {
	switch intent {
	case "room.delete":
		return "room", "roomId", api.MetadataDeleteRoom, true
	case "area.delete":
		return "area", "areaId", api.MetadataDeleteArea, true
	case "group.delete":
		return "group", "groupId", api.MetadataDeleteGroup, true
	case "scene.delete":
		return "scene", "sceneId", api.MetadataDeleteScene, true
	case "automation.delete":
		return "automation", "automationId", api.MetadataDeleteAutomation, true
	default:
		return "", "", "", false
	}
}

func metadataDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "room.delete":
		return []string{"parameters.houseId", "parameters.roomId", "parameters.name"}
	case "area.delete":
		return []string{"parameters.houseId", "parameters.areaId", "parameters.name"}
	case "group.delete":
		return []string{"parameters.houseId", "parameters.groupId", "parameters.name"}
	case "scene.delete":
		return []string{"parameters.houseId", "parameters.sceneId", "parameters.name"}
	case "automation.delete":
		return []string{"parameters.houseId", "parameters.automationId", "parameters.name"}
	default:
		return []string{"parameters.houseId", "parameters.id", "parameters.name"}
	}
}

func metadataDeleteLabel(entityType string) string {
	switch entityType {
	case "room":
		return "房间"
	case "area":
		return "区域"
	case "group":
		return "设备组"
	case "scene":
		return "情景"
	case "automation":
		return "自动化"
	default:
		return "对象"
	}
}

func metadataDeleteImpact(entityType string, target api.EntitySummary, entities api.EntityListResult) map[string]any {
	impact := map[string]any{
		"mode": "single_target_delete",
	}
	switch entityType {
	case "room":
		impact["deviceCountInRoom"] = countEntities(entities.Entities, func(entity api.EntitySummary) bool {
			return entity.Type == "device" && entity.RoomID == target.ID
		})
		impact["groupCountInRoom"] = countEntities(entities.Entities, func(entity api.EntitySummary) bool {
			return entity.Type == "group" && entity.RoomID == target.ID
		})
	case "area":
		impact["roomCountTotal"] = entities.Counts["room"]
	case "group":
		impact["roomId"] = target.RoomID
	case "scene", "automation":
		impact["status"] = target.Status
	}
	return impact
}

func countEntities(entities []api.EntitySummary, match func(api.EntitySummary) bool) int {
	count := 0
	for _, entity := range entities {
		if match(entity) {
			count++
		}
	}
	return count
}

func (app *app) commitMetadataDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.MetadataDeleteKind) (contract.Response, error) {
	result, err := api.NewMetadataDeleteClient(endpoint, nil).Run(ctx, api.MetadataDeleteRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		EntityID:       valueIDString(record.Payload["entityId"]),
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.MetadataDeleteCredentials{
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
	return metadataDeleteCommitResponse(request, record, result), nil
}
