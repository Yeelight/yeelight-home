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

func (app *app) invokeDestructiveDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	targetType, idKey, kind, ok := destructiveDeleteIntentSpec(request.Intent)
	if !ok {
		return configureClarificationResponse(request, "unsupported_destructive_delete_intent", []string{"parameters.houseId"}), nil
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", destructiveDeleteAcceptedFields(request.Intent)), nil
	}
	targetID := firstRequestString(request.Parameters, idKey, "id", "entityId")
	if targetID == "" {
		targetID = firstValueIDString(request.Parameters, idKey, "id", "entityId")
	}
	if targetID == "" && kind == api.DestructiveDeleteHome {
		targetID = houseID
	}
	targetName := firstRequestString(request.Parameters, "name", "entityName", targetType+"Name")
	target, entities, apiCalls, reason, err := resolveDestructiveDeleteTarget(ctx, request.Intent, targetType, targetID, targetName, houseID, endpoint, authorization, clientID)
	if err != nil {
		return contract.Response{}, err
	}
	if reason != "" {
		return configureClarificationResponse(request, reason, destructiveDeleteAcceptedFields(request.Intent)), nil
	}
	payload := map[string]any{
		"houseId":    requestNumberOrString(houseID),
		"capability": string(kind),
		"entityType": targetType,
		"entityId":   target.ID,
		"name":       target.Name,
	}
	payload[idKey] = target.ID
	challenge := destructiveDeleteChallenge(request.Intent, target.ID, target.Name)
	record, err := plan.NewRecordWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("删除%s %s", metadataDeleteLabel(targetType), firstNonEmptyString(target.Name, target.ID)), plan.RiskR3, challenge, payload, []string{
		"这是 R3 高影响删除计划，普通 plan.commit 会被阻断",
		"必须先在本机终端运行 approveCommand 完成一次性审批",
		"plan.commit 只接受 planId，忽略提交时附带的删除字段",
		"提交前 Runtime 会重新读取目标并验证仍属于当前家庭",
		"提交后 Runtime 会通过只读列表或详情验证目标已经消失",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	preview := map[string]any{
		"deleteTarget": map[string]any{
			"type": target.Type,
			"id":   target.ID,
			"name": target.Name,
		},
		"impact": map[string]any{
			"mode":                 "r3_destructive_delete",
			"requiresLocalApprove": true,
			"approvalChallenge":    challenge,
		},
	}
	return pendingPlanResponseWithPreview(request, record, entities, preview, apiCalls-entityListAPICalls(entities)), nil
}

func destructiveDeleteIntentSpec(intent string) (string, string, api.DestructiveDeleteKind, bool) {
	switch intent {
	case "device.remove":
		return "device", "deviceId", api.DestructiveDeleteDevice, true
	case "gateway.delete":
		return "gateway", "gatewayId", api.DestructiveDeleteGateway, true
	case "home.delete":
		return "home", "houseId", api.DestructiveDeleteHome, true
	default:
		return "", "", "", false
	}
}

func destructiveDeleteAcceptedFields(intent string) []string {
	switch intent {
	case "device.remove":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.name"}
	case "gateway.delete":
		return []string{"parameters.houseId", "parameters.gatewayId", "parameters.deviceId", "parameters.name"}
	case "home.delete":
		return []string{"parameters.houseId"}
	default:
		return []string{"parameters.houseId", "parameters.id", "parameters.name"}
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
	if match.ID == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "entity_not_found", nil
	}
	if len(candidates) > 1 && targetID == "" {
		return api.EntitySummary{}, entities, entityListAPICalls(entities), "ambiguous_target", nil
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

func destructiveDeleteChallenge(intent string, entityID string, name string) string {
	if strings.TrimSpace(name) == "" {
		return "DELETE " + intent + " " + strings.TrimSpace(entityID)
	}
	return "DELETE " + intent + " " + strings.TrimSpace(entityID) + " " + strings.TrimSpace(name)
}

func (app *app) commitDestructiveDeletePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.DestructiveDeleteKind) (contract.Response, error) {
	result, err := api.NewDestructiveDeleteClient(endpoint, nil).Run(ctx, api.DestructiveDeleteRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		EntityID:       valueIDString(record.Payload["entityId"]),
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return destructiveDeleteCommitResponse(request, record, result), nil
}
