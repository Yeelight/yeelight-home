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

const metadataBatchDeleteLimit = 20

func (app *app) invokeMetadataBatchDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	targetType, idKey, kind, ok := metadataBatchDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_metadata_batch_delete_intent", []string{"parameters.houseId"}), nil
	}
	requestedItems, ok := metadataBatchDeleteRequestedItems(request, idKey, targetType)
	if !ok || len(requestedItems) == 0 || len(requestedItems) > metadataBatchDeleteLimit {
		return configureClarificationResponse(request, "invalid_metadata_batch_delete_payload", metadataBatchDeleteAcceptedFields(request.Intent)), nil
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
	items, previewItems, reason := resolveMetadataBatchDeleteItems(targetType, idKey, requestedItems, entities)
	if reason != "" {
		return configureClarificationResponse(request, reason, metadataBatchDeleteAcceptedFields(request.Intent)), nil
	}
	payload := map[string]any{
		"houseId":    requestNumberOrString(houseID),
		"capability": string(kind),
		"entityType": targetType,
		"items":      items,
	}
	preview := map[string]any{
		"deleteTargets": previewItems,
		"impact": map[string]any{
			"mode":      "multi_target_delete",
			"itemCount": len(items),
			"fanOut":    "single_target_delete_adapter",
		},
	}
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("批量删除%d个%s", len(items), metadataDeleteLabel(targetType)), payload, []string{
		"提交前重新读取家庭实体列表并确认全部目标仍存在",
		"单次计划最多删除 20 个目标",
		"执行时逐个调用已验证的单目标删除 adapter",
		"plan.commit 只接受 planId，忽略提交时附带的删除字段",
		"提交后通过 entity.list 逐项验证目标对象已不存在",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, entities, preview, 0), nil
}

func metadataBatchDeleteIntentSpec(intent string) (string, string, api.MetadataBatchDeleteKind, bool) {
	switch intent {
	case "room.batch_delete":
		return "room", "roomId", api.MetadataBatchDeleteRoom, true
	case "area.batch_delete":
		return "area", "areaId", api.MetadataBatchDeleteArea, true
	case "group.batch_delete":
		return "group", "groupId", api.MetadataBatchDeleteGroup, true
	case "scene.batch_delete":
		return "scene", "sceneId", api.MetadataBatchDeleteScene, true
	case "automation.batch_delete":
		return "automation", "automationId", api.MetadataBatchDeleteAutomation, true
	default:
		return "", "", "", false
	}
}

func metadataBatchDeleteRequestedItems(request contract.Request, idKey string, targetType string) ([]map[string]any, bool) {
	if rawItems, ok := requestMapList(request.Parameters["items"]); ok {
		return rawItems, true
	}
	ids := requestStringList(request.Parameters[idKey], request.Parameters["ids"], request.Parameters["entityIds"])
	names := requestStringList(request.Parameters["names"], request.Parameters["entityNames"], request.Parameters[targetType+"Names"])
	if len(ids) == 0 && len(names) == 0 {
		return nil, false
	}
	items := make([]map[string]any, 0, len(ids)+len(names))
	for _, id := range ids {
		items = append(items, map[string]any{idKey: id})
	}
	for _, name := range names {
		items = append(items, map[string]any{"name": name})
	}
	return items, true
}

func resolveMetadataBatchDeleteItems(targetType string, idKey string, requested []map[string]any, entities api.EntityListResult) ([]any, []any, string) {
	items := make([]any, 0, len(requested))
	preview := make([]any, 0, len(requested))
	seen := map[string]bool{}
	for _, raw := range requested {
		targetID := firstRequestString(raw, idKey, "id", "entityId")
		if targetID == "" {
			targetID = firstValueIDString(raw, idKey, "id", "entityId")
		}
		targetName := firstRequestString(raw, "name", "entityName", targetType+"Name")
		match, candidates, matchedBy := findEntity(entityGetTarget{id: targetID, name: targetName, entityType: targetType}, entities.Entities)
		if match.ID == "" {
			return nil, nil, "entity_not_found"
		}
		if len(candidates) > 1 && targetID == "" {
			return nil, nil, "ambiguous_target"
		}
		if seen[match.ID] {
			return nil, nil, "duplicate_delete_target"
		}
		seen[match.ID] = true
		item := map[string]any{
			"entityId": match.ID,
			idKey:      match.ID,
			"name":     match.Name,
		}
		items = append(items, item)
		preview = append(preview, map[string]any{
			"type":      match.Type,
			"id":        match.ID,
			"name":      match.Name,
			"roomId":    match.RoomID,
			"matchedBy": matchedBy,
		})
	}
	return items, preview, ""
}

func metadataBatchDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "room.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].roomId", "parameters.items[].name", "parameters.ids", "parameters.names"}
	case "area.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].areaId", "parameters.items[].name", "parameters.ids", "parameters.names"}
	case "group.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].groupId", "parameters.items[].name", "parameters.ids", "parameters.names"}
	case "scene.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].sceneId", "parameters.items[].name", "parameters.ids", "parameters.names"}
	case "automation.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].automationId", "parameters.items[].name", "parameters.ids", "parameters.names"}
	default:
		return []string{"parameters.houseId", "parameters.items", "parameters.ids", "parameters.names"}
	}
}

func (app *app) commitMetadataBatchDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.MetadataBatchDeleteKind) (contract.Response, error) {
	items, err := metadataBatchDeleteItemsFromPlan(record.Payload)
	if err != nil {
		return contract.Response{}, err
	}
	result, err := api.NewMetadataBatchDeleteClient(endpoint, nil).Run(ctx, api.MetadataBatchDeleteRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Items:          items,
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
	return metadataBatchDeleteCommitResponse(request, record, result), nil
}

func metadataBatchDeleteItemsFromPlan(payload map[string]any) ([]api.MetadataBatchDeleteItem, error) {
	rawItems, ok := payload["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return nil, fmt.Errorf("batch delete items are required")
	}
	items := make([]api.MetadataBatchDeleteItem, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("batch delete item must be an object")
		}
		entityID := valueIDString(firstNonNil(item["entityId"], item["roomId"], item["areaId"], item["groupId"], item["sceneId"], item["automationId"], item["id"]))
		if entityID == "" {
			return nil, fmt.Errorf("batch delete item id is required")
		}
		items = append(items, api.MetadataBatchDeleteItem{EntityID: entityID, Name: requestString(item["name"])})
	}
	return items, nil
}
