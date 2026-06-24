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

func (app *app) invokeHomeLockPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
	payload, preconditions, summary, err := buildHomeLockPayload(request, houseID, entities)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeLockAcceptedFields(request.Intent)), nil
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, entities, homeLockPreview(request.Intent, entities), 0), nil
}

func buildHomeLockPayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "home.lock_all":
		return map[string]any{
				"houseId":     requestNumberOrString(houseID),
				"deviceCount": entities.Counts["device"],
			}, []string{
				"提交前重新读取家庭实体列表",
				"影响范围是当前家庭下所有设备的重置锁定能力",
				"提交后通过写入确认和 entity.list 可访问性验证",
			}, fmt.Sprintf("锁定当前家庭全部设备的重置能力，当前可见设备数 %d", entities.Counts["device"]), nil
	case "home.unlock_all":
		return map[string]any{
				"houseId":     requestNumberOrString(houseID),
				"deviceCount": entities.Counts["device"],
			}, []string{
				"提交前重新读取家庭实体列表",
				"影响范围是当前家庭下所有设备的重置锁定能力",
				"提交后通过写入确认和 entity.list 可访问性验证",
			}, fmt.Sprintf("解锁当前家庭全部设备的重置能力，当前可见设备数 %d", entities.Counts["device"]), nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported_home_lock_intent")
	}
}

func homeLockPreview(intent string, entities api.EntityListResult) map[string]any {
	action := "lock"
	if intent == "home.unlock_all" {
		action = "unlock"
	}
	return map[string]any{
		"affectedScope": "whole_house",
		"deviceCount":   entities.Counts["device"],
		"action":        action,
		"verification":  "write_ack_plus_entity_list_accessible",
	}
}

func homeLockAcceptedFields(intent string) []string {
	switch intent {
	case "home.lock_all", "home.unlock_all":
		return []string{"parameters.houseId", "homeRef.id"}
	default:
		return []string{"parameters.houseId"}
	}
}

func (app *app) commitHomeLockPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.HomeLockKind) (contract.Response, error) {
	result, err := api.NewHomeLockClient(endpoint, nil).Run(ctx, api.HomeLockRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SpaceOrganizationCredentials{
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
	return homeLockCommitResponse(request, record, result), nil
}
