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

func (app *app) prepareAutomationUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
	automation, reason := automationUpdateTarget(request, entities)
	if reason != "" {
		return automationUpdateClarificationResponse(request, reason), nil
	}
	payload, err := buildAutomationUpdatePayload(request, houseID, automation)
	if err != nil {
		return automationUpdateClarificationResponse(request, err.Error()), nil
	}
	if reason := validateAutomationUpdatePayload(payload, entities); reason != "" {
		return automationUpdateClarificationResponse(request, reason), nil
	}
	summaryName := executionPayloadString(payload, semantic.FieldName)
	if summaryName == "" {
		summaryName = automation.Name
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("更新自动化 %s", summaryName), payload, []string{
		"提交前重新读取家庭自动化列表",
		"目标自动化必须属于当前家庭",
		"更新必须携带完整条件和动作 payload",
		"提交后通过 automation.list 验证目标自动化摘要",
	}, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func automationUpdateClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, automationUpdateAcceptedFields(), automationPayloadGuide("automation.update"))
}

func buildAutomationUpdatePayload(request contract.Request, houseID string, automation api.EntitySummary) (map[string]any, error) {
	if _, ok := request.Parameters[semantic.FieldStatus]; ok {
		return nil, fmt.Errorf("automation_status_update_requires_enable_disable_intent")
	}
	schedule, ok := semantic.AutomationScheduleFromRequest(request.Parameters)
	if !ok {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	version, _ := requestInt(request.Parameters[semantic.FieldVersion])
	actions, ok := normalizeAutomationActionRows(request.Parameters[semantic.FieldActions])
	if !ok {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	name := firstRequestString(request.Parameters, semantic.FieldNewName, semantic.FieldName)
	if name == "" {
		name = automation.Name
	}
	payload, err := api.BuildAutomationCreatePayload(
		houseID,
		name,
		schedule.StartTime,
		schedule.EndTime,
		schedule.RepeatType,
		schedule.RepeatValue,
		normalizeAutomationParamsFromRequest(request.Parameters),
		actions,
		version,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	payload[semantic.FieldID] = requestNumberOrString(automation.ID)
	payload[semantic.FieldAutomationID] = automation.ID
	return payload, nil
}

func automationUpdateTarget(request contract.Request, entities api.EntityListResult) (api.EntitySummary, string) {
	automationID := firstRequestString(request.Parameters, semantic.FieldAutomationID, semantic.FieldID, semantic.FieldEntityID)
	if automationID != "" {
		match, _, _ := findEntity(entityGetTarget{id: automationID, entityType: "automation"}, entities.Entities)
		if match.ID != "" {
			return match, ""
		}
		return api.EntitySummary{}, "invalid_automation_reference"
	}
	nameKeys := []string{
		semantic.FieldAutomationName,
		semantic.FieldCurrentName,
		semantic.FieldEntityName,
		semantic.FieldTargetName,
	}
	if firstRequestString(request.Parameters, semantic.FieldNewName) == "" {
		nameKeys = append(nameKeys, semantic.FieldName)
	}
	automationName := firstRequestString(request.Parameters, nameKeys...)
	if automationName == "" {
		return api.EntitySummary{}, "missing_automation_target"
	}
	match, candidates, _ := findEntity(entityGetTarget{name: automationName, entityType: "automation"}, entities.Entities)
	if match.ID != "" {
		return match, ""
	}
	if len(candidates) > 0 {
		return api.EntitySummary{}, "ambiguous_automation_target"
	}
	return api.EntitySummary{}, "invalid_automation_reference"
}

func validateAutomationUpdatePayload(payload map[string]any, entities api.EntityListResult) string {
	automationID := valueIDString(payload[semantic.FieldAutomationID])
	if !entityExists(entities, "automation", automationID) {
		return "invalid_automation_reference"
	}
	delete(payload, semantic.FieldAutomationID)
	reason := validateAutomationCreatePayload(payload, entities)
	payload[semantic.FieldAutomationID] = automationID
	if reason == "house_automation_limit_exceeded" {
		return ""
	}
	if reason == "invalid_automation_create_payload" {
		return "invalid_automation_update_payload"
	}
	return reason
}

func automationUpdateAcceptedFields() []string {
	return append(semanticParameterPaths(
		semantic.FieldHouseID,
		semantic.FieldAutomationID,
		semantic.FieldAutomationName,
		semantic.FieldCurrentName,
		semantic.FieldEntityName,
		semantic.FieldTargetName,
		semantic.FieldName,
		semantic.FieldNewName,
		semantic.FieldActiveWindow,
		semantic.FieldRepeat,
		semantic.FieldTrigger,
		semantic.FieldConditions,
		semantic.FieldActions,
		semantic.FieldVersion,
	),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName),
		semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet),
	)
}

func (app *app) executeAutomationUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewAutomationUpdateClient(endpoint, nil).Run(ctx, api.AutomationUpdateRequest{
		HouseID:        record.HouseID,
		AutomationID:   executionPayloadString(record.Payload, semantic.FieldAutomationID),
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.AutomationUpdateCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return automationUpdateExecuteResponse(request, record, result), nil
}
