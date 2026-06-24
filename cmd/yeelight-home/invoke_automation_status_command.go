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

func (app *app) invokeAutomationStatusPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
	automation, reason := automationStatusTarget(request, entities)
	if reason != "" {
		return configureClarificationResponse(request, reason, automationStatusAcceptedFields()), nil
	}
	summary := automationStatusSummary(request.Intent, automation)
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, summary, map[string]any{
		"houseId":      requestNumberOrString(houseID),
		"automationId": automation.ID,
	}, []string{
		"提交前重新读取家庭自动化列表",
		"目标自动化必须属于当前家庭",
		"提交后通过 automation.list 校验目标状态",
	}, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponse(request, record, entities), nil
}

func automationStatusTarget(request contract.Request, entities api.EntityListResult) (api.EntitySummary, string) {
	targetID := firstRequestString(request.Parameters, "automationId", "id", "entityId")
	targetName := firstRequestString(request.Parameters, "name", "automationName")
	if targetID == "" && targetName == "" {
		return api.EntitySummary{}, "missing_automation_target"
	}
	matches := []api.EntitySummary{}
	for _, entity := range entities.Entities {
		if entity.Type != "automation" {
			continue
		}
		if targetID != "" && entity.ID == targetID {
			return entity, ""
		}
		if targetID == "" && targetName != "" && entity.Name == targetName {
			matches = append(matches, entity)
		}
	}
	if len(matches) == 1 {
		return matches[0], ""
	}
	if len(matches) > 1 {
		return api.EntitySummary{}, "ambiguous_automation_target"
	}
	return api.EntitySummary{}, "invalid_automation_reference"
}

func automationStatusAcceptedFields() []string {
	return []string{"parameters.houseId", "parameters.automationId", "parameters.id", "parameters.name"}
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

func (app *app) commitAutomationStatusPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.AutomationStatusKind) (contract.Response, error) {
	result, err := api.NewAutomationStatusClient(endpoint, nil).Run(ctx, api.AutomationStatusRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		AutomationID:   planPayloadString(record.Payload, "automationId"),
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return automationStatusCommitResponse(request, record, result), nil
}
