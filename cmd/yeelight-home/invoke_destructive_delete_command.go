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

func (app *app) prepareDestructiveDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	targetType, idKey, kind, ok := destructiveDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_destructive_delete_intent", []string{semantic.ParameterPath(semantic.FieldHouseID)}), nil
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", destructiveDeleteAcceptedFields(request.Intent)), nil
	}
	targetID := firstRequestString(request.Parameters, idKey, semantic.FieldID, semantic.FieldEntityID)
	if targetID == "" {
		targetID = firstValueIDString(request.Parameters, idKey, semantic.FieldID, semantic.FieldEntityID)
	}
	if targetID == "" && kind == api.DestructiveDeleteHome {
		targetID = houseID
	}
	targetNameKeys := []string{semantic.FieldName, semantic.FieldEntityName}
	switch targetType {
	case "device":
		targetNameKeys = append(targetNameKeys, semantic.FieldDeviceName)
	case "gateway":
		targetNameKeys = append(targetNameKeys, semantic.FieldGatewayName)
	case "home":
		targetNameKeys = append(targetNameKeys, semantic.FieldHomeName, semantic.FieldHouseName)
	}
	targetName := firstRequestString(request.Parameters, targetNameKeys...)
	target, entities, apiCalls, reason, err := resolveDestructiveDeleteTarget(ctx, request.Intent, targetType, targetID, targetName, houseID, endpoint, authorization, clientID)
	if err != nil {
		return contract.Response{}, err
	}
	if reason != "" {
		return configureClarificationResponse(request, reason, destructiveDeleteAcceptedFields(request.Intent)), nil
	}
	payload := map[string]any{
		semantic.FieldHouseID:    requestNumberOrString(houseID),
		semantic.FieldCapability: string(kind),
		semantic.FieldEntityType: targetType,
		semantic.FieldEntityID:   target.ID,
		semantic.FieldName:       target.Name,
	}
	payload[idKey] = target.ID
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("删除%s %s", metadataDeleteLabel(targetType), firstNonEmptyString(target.Name, target.ID)), operation.RiskR3, payload, []string{
		"这是 R3 高影响删除操作；调用方应在调用 Runtime 前完成自己的用户确认",
		"执行前 Runtime 会重新读取目标并验证仍属于当前家庭",
		"执行后 Runtime 会通过只读列表或详情验证目标已经消失",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		semantic.FieldDeleteTarget: map[string]any{
			semantic.FieldType: target.Type,
			semantic.FieldID:   target.ID,
			semantic.FieldName: target.Name,
		},
		semantic.FieldImpact: map[string]any{
			semantic.FieldMode:                       "r3_destructive_delete",
			semantic.FieldCallerShouldConfirm:        true,
			semantic.FieldRuntimeApprovalStateStored: false,
		},
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, apiCalls-entityListAPICalls(entities)), nil
}

func destructiveDeleteIntentSpec(intent string) (string, string, api.DestructiveDeleteKind, bool) {
	switch intent {
	case "device.remove":
		return "device", semantic.FieldDeviceID, api.DestructiveDeleteDevice, true
	case "gateway.delete":
		return "gateway", semantic.FieldGatewayID, api.DestructiveDeleteGateway, true
	case "home.delete":
		return "home", semantic.FieldHouseID, api.DestructiveDeleteHome, true
	default:
		return "", "", "", false
	}
}

func destructiveDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "device.remove":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldDeviceID), semantic.ParameterPath(semantic.FieldName), semantic.ParameterPath(semantic.FieldConfirmed)}
	case "gateway.delete":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldGatewayID), semantic.ParameterPath(semantic.FieldDeviceID), semantic.ParameterPath(semantic.FieldName), semantic.ParameterPath(semantic.FieldConfirmed)}
	case "home.delete":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldConfirmed)}
	default:
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldID), semantic.ParameterPath(semantic.FieldName), semantic.ParameterPath(semantic.FieldConfirmed)}
	}
}

func resolveDestructiveDeleteTarget(ctx context.Context, intent string, targetType string, targetID string, targetName string, houseID string, endpoint api.Endpoint, authorization string, clientID string) (api.EntitySummary, api.EntityListResult, int, string, error) {
	credentials := api.EntityListCredentials{Authorization: authorization, ClientID: clientID}
	if intent == "home.delete" {
		target, entities, calls, reason, err := resolveHomeDeleteTarget(ctx, targetID, targetName, endpoint, credentials)
		return target, entities, calls, reason, err
	}
	if intent == "gateway.delete" {
		if strings.TrimSpace(targetID) == "" {
			return api.EntitySummary{}, api.EntityListResult{Region: endpoint.Region, HouseID: houseID}, 0, "gateway_context_missing", nil
		}
		target, calls, err := api.NewDestructiveDeleteClient(endpoint, nil).ProbeGateway(ctx, houseID, targetID, api.DestructiveDeleteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		})
		if err != nil {
			return api.EntitySummary{}, api.EntityListResult{}, calls, "", err
		}
		if target.ID == "" {
			return api.EntitySummary{}, api.EntityListResult{Region: endpoint.Region, HouseID: houseID}, calls, "entity_not_found", nil
		}
		if targetName != "" && target.Name != "" && target.Name != targetName {
			return api.EntitySummary{}, api.EntityListResult{Region: endpoint.Region, HouseID: houseID}, calls, "entity_not_found", nil
		}
		if target.Name == "" {
			target.Name = targetName
		}
		return target, api.EntityListResult{Region: endpoint.Region, HouseID: houseID}, calls, "", nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID:     ternaryHouseID(intent == "home.delete", "", houseID),
		Credentials: credentials,
	})
	if err != nil {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "", err
	}
	match, candidates, _ := findEntity(entityGetTarget{id: targetID, name: targetName, entityType: targetType}, entities.Entities)
	if len(candidates) > 1 && targetID == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "ambiguous_target", nil
	}
	if match.ID == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "entity_not_found", nil
	}
	return match, entities, entityListAPICalls(entities), "", nil
}

func resolveHomeDeleteTarget(ctx context.Context, targetID string, targetName string, endpoint api.Endpoint, credentials api.EntityListCredentials) (api.EntitySummary, api.EntityListResult, int, string, error) {
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{Credentials: credentials})
	if err != nil {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "", err
	}
	match, candidates, _ := findEntity(entityGetTarget{id: targetID, name: targetName, entityType: "home"}, entities.Entities)
	if match.ID != "" {
		if len(candidates) > 1 && targetID == "" {
			return api.EntitySummary{}, entities, entityListAPICalls(entities), "ambiguous_target", nil
		}
		return match, entities, entityListAPICalls(entities), "", nil
	}
	if len(candidates) > 1 && targetID == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "ambiguous_target", nil
	}
	if strings.TrimSpace(targetID) == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "entity_not_found", nil
	}
	if _, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{HouseID: targetID, Credentials: credentials}); err != nil {
		return api.EntitySummary{}, entities, entityListAPICalls(entities) + api.HouseScopedEntityListCallCount(), "entity_not_found", nil
	}
	return api.EntitySummary{
		Type:    "home",
		ID:      targetID,
		Name:    targetName,
		HouseID: targetID,
	}, entities, entityListAPICalls(entities) + api.HouseScopedEntityListCallCount(), "", nil
}

func ternaryHouseID(condition bool, whenTrue string, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func (app *app) executeDestructiveDelete(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.DestructiveDeleteKind) (contract.Response, error) {
	result, err := api.NewDestructiveDeleteClient(endpoint, nil).Run(ctx, api.DestructiveDeleteRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		EntityID:       valueIDString(record.Payload[semantic.FieldEntityID]),
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.DestructiveDeleteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return destructiveDeleteExecuteResponse(request, record, result), nil
}
