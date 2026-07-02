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

func (app *app) prepareMetadataDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
	targetType, targetIDKey, kind, ok := metadataDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_metadata_delete_intent", []string{semantic.ParameterPath(semantic.FieldHouseID)}), nil
	}
	targetID := firstRequestString(request.Parameters, targetIDKey, semantic.FieldID, semantic.FieldEntityID)
	if targetID == "" {
		targetID = firstValueIDString(request.Parameters, targetIDKey, semantic.FieldID, semantic.FieldEntityID)
	}
	targetName := firstRequestString(request.Parameters, metadataDeleteNameFields(targetType)...)
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
	if len(candidates) > 1 && targetID == "" {
		return configureClarificationResponse(request, "ambiguous_target", metadataDeleteAcceptedFields(request.Intent)), nil
	}
	if match.ID == "" {
		return configureClarificationResponse(request, "entity_not_found", metadataDeleteAcceptedFields(request.Intent)), nil
	}
	payload := map[string]any{
		semantic.FieldHouseID:    requestNumberOrString(houseID),
		semantic.FieldCapability: string(kind),
		semantic.FieldEntityType: targetType,
		semantic.FieldEntityID:   match.ID,
		semantic.FieldName:       match.Name,
	}
	payload[targetIDKey] = match.ID
	preview := map[string]any{
		semantic.FieldDeleteTarget: map[string]any{
			semantic.FieldType:      match.Type,
			semantic.FieldID:        match.ID,
			semantic.FieldName:      match.Name,
			semantic.FieldRoomID:    match.RoomID,
			semantic.FieldMatchedBy: matchedBy,
		},
		semantic.FieldImpact: metadataDeleteImpact(targetType, match, entities),
	}
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("删除%s %s", metadataDeleteLabel(targetType), match.Name), operation.RiskR3, payload, []string{
		"提交前重新读取家庭实体列表并确认目标仍存在",
		"只删除本计划中已解析的单个目标对象",
		"Runtime 根据当前请求构建受控删除 payload",
		"提交后通过 entity.list 验证目标对象已不存在",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
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
		return "room", semantic.FieldRoomID, api.MetadataDeleteRoom, true
	case "area.delete":
		return "area", semantic.FieldAreaID, api.MetadataDeleteArea, true
	case "group.delete":
		return "group", semantic.FieldGroupID, api.MetadataDeleteGroup, true
	case "scene.delete":
		return "scene", semantic.FieldSceneID, api.MetadataDeleteScene, true
	case "automation.delete":
		return "automation", semantic.FieldAutomationID, api.MetadataDeleteAutomation, true
	default:
		return "", "", "", false
	}
}

func metadataDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "room.delete":
		return metadataDeleteAcceptedFieldsFor(semantic.FieldRoomID, "room")
	case "area.delete":
		return metadataDeleteAcceptedFieldsFor(semantic.FieldAreaID, "area")
	case "group.delete":
		return metadataDeleteAcceptedFieldsFor(semantic.FieldGroupID, "group")
	case "scene.delete":
		return metadataDeleteAcceptedFieldsFor(semantic.FieldSceneID, "scene")
	case "automation.delete":
		return metadataDeleteAcceptedFieldsFor(semantic.FieldAutomationID, "automation")
	default:
		return metadataDeleteAcceptedFieldsFor(semantic.FieldID, "")
	}
}

func metadataDeleteAcceptedFieldsFor(idField string, targetType string) []string {
	fields := []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semantic.ParameterPath(idField),
	}
	for _, nameField := range metadataDeleteNameFields(targetType) {
		fields = append(fields, semantic.ParameterPath(nameField))
	}
	fields = append(fields, semantic.ParameterPath(semantic.FieldConfirmed))
	return fields
}

func metadataDeleteNameFields(targetType string) []string {
	fields := []string{
		semantic.FieldName,
		semantic.FieldEntityName,
		semantic.FieldCurrentName,
		semantic.FieldTargetName,
	}
	switch targetType {
	case "room":
		fields = append(fields, semantic.FieldRoomName)
	case "area":
		fields = append(fields, semantic.FieldAreaName)
	case "group":
		fields = append(fields, semantic.FieldGroupName)
	case "scene":
		fields = append(fields, semantic.FieldSceneName)
	case "automation":
		fields = append(fields, semantic.FieldAutomationName)
	}
	return fields
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
		semantic.FieldMode: "single_target_delete",
	}
	switch entityType {
	case "room":
		impact[semantic.FieldDeviceCountInRoom] = countEntities(entities.Entities, func(entity api.EntitySummary) bool {
			return entity.Type == "device" && entity.RoomID == target.ID
		})
		impact[semantic.FieldGroupCountInRoom] = countEntities(entities.Entities, func(entity api.EntitySummary) bool {
			return entity.Type == "group" && entity.RoomID == target.ID
		})
	case "area":
		impact[semantic.FieldRoomCountTotal] = entities.Counts["room"]
	case "group":
		impact[semantic.FieldRoomID] = target.RoomID
	case "scene", "automation":
		impact[semantic.FieldStatus] = target.Status
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

func (app *app) executeMetadataDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.MetadataDeleteKind) (contract.Response, error) {
	result, err := api.NewMetadataDeleteClient(endpoint, nil).Run(ctx, api.MetadataDeleteRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		EntityID:       valueIDString(record.Payload[semantic.FieldEntityID]),
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
	return metadataDeleteExecuteResponse(request, record, result), nil
}
