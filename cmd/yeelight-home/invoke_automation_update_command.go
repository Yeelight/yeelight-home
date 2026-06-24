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

func (app *app) invokeAutomationUpdatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
		return configureClarificationResponse(request, reason, automationUpdateAcceptedFields()), nil
	}
	payload, err := buildAutomationUpdatePayload(request, houseID, automation.ID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), automationUpdateAcceptedFields()), nil
	}
	if reason := validateAutomationUpdatePayload(payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, automationUpdateAcceptedFields()), nil
	}
	summaryName := planPayloadString(payload, "name")
	if summaryName == "" {
		summaryName = automation.Name
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("更新自动化 %s", summaryName), payload, []string{
		"提交前重新读取家庭自动化列表",
		"目标自动化必须属于当前家庭",
		"更新必须携带完整条件和动作 payload",
		"提交后通过 automation.list 验证目标自动化摘要",
	}, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponse(request, record, entities), nil
}

func buildAutomationUpdatePayload(request contract.Request, houseID string, automationID string) (map[string]any, error) {
	if _, ok := request.Parameters["status"]; ok {
		return nil, fmt.Errorf("automation_status_update_requires_enable_disable_intent")
	}
	repeatType, ok := requestInt(request.Parameters["repeatType"])
	if !ok {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	version, _ := requestInt(request.Parameters["version"])
	actions, ok := requestMapList(request.Parameters["actions"])
	if !ok {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	payload, err := api.BuildAutomationCreatePayload(
		houseID,
		configureName(request),
		firstRequestString(request.Parameters, "startTime", "start_time"),
		firstRequestString(request.Parameters, "endTime", "end_time"),
		repeatType,
		firstRequestString(request.Parameters, "repeatValue", "repeat_value"),
		request.Parameters["params"],
		actions,
		version,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid_automation_update_payload")
	}
	payload["id"] = requestNumberOrString(automationID)
	payload["automationId"] = automationID
	return payload, nil
}

func validateAutomationUpdatePayload(payload map[string]any, entities api.EntityListResult) string {
	automationID := valueIDString(payload["automationId"])
	if !entityExists(entities, "automation", automationID) {
		return "invalid_automation_reference"
	}
	delete(payload, "automationId")
	reason := validateAutomationCreatePayload(payload, entities)
	payload["automationId"] = automationID
	if reason == "house_automation_limit_exceeded" {
		return ""
	}
	if reason == "invalid_automation_create_payload" {
		return "invalid_automation_update_payload"
	}
	return reason
}

func automationUpdateAcceptedFields() []string {
	return []string{
		"parameters.houseId",
		"parameters.automationId",
		"parameters.name",
		"parameters.startTime",
		"parameters.endTime",
		"parameters.repeatType",
		"parameters.repeatValue",
		"parameters.params",
		"parameters.actions",
		"parameters.version",
	}
}

func (app *app) commitAutomationUpdatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewAutomationUpdateClient(endpoint, nil).Run(ctx, api.AutomationUpdateRequest{
		HouseID:        record.HouseID,
		AutomationID:   planPayloadString(record.Payload, "automationId"),
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return automationUpdateCommitResponse(request, record, result), nil
}
