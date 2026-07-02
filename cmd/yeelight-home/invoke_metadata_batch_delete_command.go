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

const metadataBatchDeleteLimit = 20

func (app *app) prepareMetadataBatchDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
	}
	targetType, idKey, kind, ok := metadataBatchDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_metadata_batch_delete_intent", semanticParameterPaths(semantic.FieldHouseID)), nil
	}
	requestedItems, ok := metadataBatchDeleteRequestedItems(request, idKey, targetType)
	if !ok || len(requestedItems) == 0 || len(requestedItems) > metadataBatchDeleteLimit {
		return metadataBatchDeleteClarificationResponse(request, "invalid_metadata_batch_delete_payload"), nil
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
		return metadataBatchDeleteClarificationResponse(request, reason), nil
	}
	payload := map[string]any{
		semantic.FieldHouseID:    requestNumberOrString(houseID),
		semantic.FieldCapability: string(kind),
		semantic.FieldEntityType: targetType,
		semantic.FieldItems:      items,
	}
	preview := map[string]any{
		semantic.FieldDeleteTargets: previewItems,
		semantic.FieldImpact: map[string]any{
			semantic.FieldMode:      "multi_target_delete",
			semantic.FieldItemCount: len(items),
			semantic.FieldFanOut:    "single_target_delete_calls",
		},
	}
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("批量删除%d个%s", len(items), metadataDeleteLabel(targetType)), operation.RiskR3, payload, []string{
		"提交前重新读取家庭实体列表并确认全部目标仍存在",
		"单次计划最多删除 20 个目标",
		"执行时逐个调用已验证的单目标删除能力",
		"Runtime 根据当前请求构建受控删除 payload",
		"提交后通过 entity.list 逐项验证目标对象已不存在",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func metadataBatchDeleteIntentSpec(intent string) (string, string, api.MetadataBatchDeleteKind, bool) {
	switch intent {
	case "room.batch_delete":
		return "room", semantic.FieldRoomID, api.MetadataBatchDeleteRoom, true
	case "area.batch_delete":
		return "area", semantic.FieldAreaID, api.MetadataBatchDeleteArea, true
	case "group.batch_delete":
		return "group", semantic.FieldGroupID, api.MetadataBatchDeleteGroup, true
	case "scene.batch_delete":
		return "scene", semantic.FieldSceneID, api.MetadataBatchDeleteScene, true
	case "automation.batch_delete":
		return "automation", semantic.FieldAutomationID, api.MetadataBatchDeleteAutomation, true
	default:
		return "", "", "", false
	}
}

func metadataBatchDeleteRequestedItems(request contract.Request, idKey string, targetType string) ([]map[string]any, bool) {
	if rawItems, ok := requestMapList(request.Parameters[semantic.FieldItems]); ok {
		return rawItems, true
	}
	ids := requestStringList(request.Parameters[idKey], request.Parameters[semantic.FieldIDs], request.Parameters[semantic.FieldEntityIDs])
	names := requestStringList(request.Parameters[semantic.FieldNames], request.Parameters[semantic.FieldEntityNames])
	if len(ids) == 0 && len(names) == 0 {
		return nil, false
	}
	items := make([]map[string]any, 0, len(ids)+len(names))
	if len(ids) > 0 && len(ids) == len(names) {
		for index, id := range ids {
			items = append(items, map[string]any{idKey: id, semantic.FieldName: names[index]})
		}
		return items, true
	}
	for _, id := range ids {
		items = append(items, map[string]any{idKey: id})
	}
	for _, name := range names {
		items = append(items, map[string]any{semantic.FieldName: name})
	}
	return items, true
}

func resolveMetadataBatchDeleteItems(targetType string, idKey string, requested []map[string]any, entities api.EntityListResult) ([]any, []any, string) {
	items := make([]any, 0, len(requested))
	preview := make([]any, 0, len(requested))
	seen := map[string]bool{}
	for _, raw := range requested {
		targetID := firstRequestString(raw, idKey, semantic.FieldID, semantic.FieldEntityID)
		if targetID == "" {
			targetID = firstValueIDString(raw, idKey, semantic.FieldID, semantic.FieldEntityID)
		}
		targetName := firstRequestString(raw, metadataDeleteNameFields(targetType)...)
		match, candidates, matchedBy := findEntity(entityGetTarget{id: targetID, name: targetName, entityType: targetType}, entities.Entities)
		if len(candidates) > 1 && targetID == "" {
			return nil, nil, "ambiguous_target"
		}
		if match.ID == "" {
			return nil, nil, "entity_not_found"
		}
		if seen[match.ID] {
			return nil, nil, "duplicate_delete_target"
		}
		seen[match.ID] = true
		item := map[string]any{
			semantic.FieldEntityID: match.ID,
			idKey:                  match.ID,
			semantic.FieldName:     match.Name,
		}
		items = append(items, item)
		preview = append(preview, map[string]any{
			semantic.FieldType:      match.Type,
			semantic.FieldID:        match.ID,
			semantic.FieldName:      match.Name,
			semantic.FieldRoomID:    match.RoomID,
			semantic.FieldMatchedBy: matchedBy,
		})
	}
	return items, preview, ""
}

func metadataBatchDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "room.batch_delete":
		return metadataBatchDeleteAcceptedFieldsForID(semantic.FieldRoomID, "room")
	case "area.batch_delete":
		return metadataBatchDeleteAcceptedFieldsForID(semantic.FieldAreaID, "area")
	case "group.batch_delete":
		return metadataBatchDeleteAcceptedFieldsForID(semantic.FieldGroupID, "group")
	case "scene.batch_delete":
		return metadataBatchDeleteAcceptedFieldsForID(semantic.FieldSceneID, "scene")
	case "automation.batch_delete":
		return metadataBatchDeleteAcceptedFieldsForID(semantic.FieldAutomationID, "automation")
	default:
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldItems, semantic.FieldIDs, semantic.FieldNames)
	}
}

func metadataBatchDeleteAcceptedFieldsForID(idField string, targetType string) []string {
	fields := []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semanticParameterArrayPath(semantic.FieldItems, idField),
		semantic.ParameterPath(semantic.FieldIDs),
		semantic.ParameterPath(semantic.FieldNames),
		semantic.ParameterPath(semantic.FieldEntityNames),
	}
	for _, nameField := range metadataDeleteNameFields(targetType) {
		fields = append(fields, semanticParameterArrayPath(semantic.FieldItems, nameField))
	}
	fields = append(fields,
		semantic.ParameterPath(semantic.FieldConfirmed),
	)
	return fields
}

func metadataBatchDeleteClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, metadataBatchDeleteAcceptedFields(request.Intent), metadataBatchDeletePayloadGuide(request.Intent))
}

func (app *app) executeMetadataBatchDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.MetadataBatchDeleteKind) (contract.Response, error) {
	items, err := metadataBatchDeleteItemsFromPreparedPayload(record.Payload)
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
	return metadataBatchDeleteExecuteResponse(request, record, result), nil
}

func metadataBatchDeleteItemsFromPreparedPayload(payload map[string]any) ([]api.MetadataBatchDeleteItem, error) {
	rawItems, ok := payload[semantic.FieldItems].([]any)
	if !ok || len(rawItems) == 0 {
		return nil, fmt.Errorf("batch delete items are required")
	}
	items := make([]api.MetadataBatchDeleteItem, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("batch delete item must be an object")
		}
		entityID := valueIDString(firstNonNil(item[semantic.FieldEntityID], item[semantic.FieldRoomID], item[semantic.FieldAreaID], item[semantic.FieldGroupID], item[semantic.FieldSceneID], item[semantic.FieldAutomationID], item[semantic.FieldID]))
		if entityID == "" {
			return nil, fmt.Errorf("batch delete item id is required")
		}
		items = append(items, api.MetadataBatchDeleteItem{EntityID: entityID, Name: requestString(item[semantic.FieldName])})
	}
	return items, nil
}
