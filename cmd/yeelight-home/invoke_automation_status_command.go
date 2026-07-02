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

func (app *app) prepareAutomationStatus(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID), "local profile houseId"}), nil
	}
	target := entityGetTargetFromRequest(request)
	if target.entityType == "" {
		target.entityType = "automation"
	}
	if target.id == "" && target.name == "" {
		return configureClarificationResponse(request, "missing_automation_target", automationStatusAcceptedFields()), nil
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, err
	}
	entities := resolved.Entities
	automation, reason := automationStatusResolvedTarget(resolved)
	if reason != "" {
		return configureClarificationResponse(request, reason, automationStatusAcceptedFields()), nil
	}
	summary := automationStatusSummary(request.Intent, automation)
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, map[string]any{
		semantic.FieldHouseID:      requestNumberOrString(houseID),
		semantic.FieldAutomationID: automation.ID,
	}, []string{
		"提交前重新读取家庭自动化列表",
		"目标自动化必须属于当前家庭",
		"提交后通过 automation.list 校验目标状态",
	}, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func automationStatusResolvedTarget(resolved entityResolveResult) (api.EntitySummary, string) {
	if resolved.Match.ID != "" {
		return resolved.Match, ""
	}
	if len(resolved.Candidates) > 1 {
		return api.EntitySummary{}, "ambiguous_automation_target"
	}
	return api.EntitySummary{}, "invalid_automation_reference"
}

func automationStatusTarget(request contract.Request, entities api.EntityListResult) (api.EntitySummary, string) {
	target := entityGetTargetFromRequest(request)
	if target.entityType == "" {
		target.entityType = "automation"
	}
	if target.id == "" && target.name == "" {
		return api.EntitySummary{}, "missing_automation_target"
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID != "" {
		return match, ""
	}
	if len(candidates) > 1 && target.id == "" {
		return api.EntitySummary{}, "ambiguous_automation_target"
	}
	return api.EntitySummary{}, "invalid_automation_reference"
}

func automationStatusAcceptedFields() []string {
	return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldAutomationID, semantic.FieldID, semantic.FieldAutomationName, semantic.FieldName)
}

func automationStatusSummary(intent string, automation api.EntitySummary) string {
	action := "切换"
	switch intent {
	case "automation.enable":
		action = "启用"
	case "automation.disable":
		action = "停用"
	}
	if automation.Name != "" {
		return fmt.Sprintf("%s自动化 %s", action, automation.Name)
	}
	return fmt.Sprintf("%s自动化 %s", action, automation.ID)
}

func (app *app) executeAutomationStatus(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.AutomationStatusKind) (contract.Response, error) {
	result, err := api.NewAutomationStatusClient(endpoint, nil).Run(ctx, api.AutomationStatusRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		AutomationID:   executionPayloadString(record.Payload, semantic.FieldAutomationID),
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.AutomationStatusCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return automationStatusExecuteResponse(request, record, result), nil
}
