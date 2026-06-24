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

func (app *app) invokeHomeCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, authorization string, clientID string) (contract.Response, error) {
	payload, err := buildHomeCreatePayload(request)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeCreateAcceptedFields()), nil
	}
	credentials := api.HomeCreateCredentials{Authorization: authorization, ClientID: clientID}
	homeName := requestString(payload["name"])
	client := api.NewHomeCreateClient(endpoint, nil)
	existing, apiCalls, err := client.FindHouseByNameForPlan(ctx, homeName, credentials)
	if err != nil {
		return contract.Response{}, err
	}
	if existing.ID != "" {
		response := homeCreateAlreadyExistsResponse(request, existing, apiCalls)
		response.Result["region"] = endpoint.Region
		return response, nil
	}
	record, err := plan.NewAccountRecord(profile, region, request.Intent, request.RequestID, fmt.Sprintf("创建家庭 %s", homeName), payload, []string{
		"提交前重新读取当前账号家庭列表",
		"家庭名称不存在时才创建",
		"plan.commit 只接受 planId，忽略提交时附带的创建字段",
		"提交后优先通过家庭列表按名称验证；如果列表延迟但创建返回 houseId，则通过该家庭的实体列表验证可访问性",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, api.EntityListResult{Region: endpoint.Region, APICalls: apiCalls}, map[string]any{
		"scope": "account",
		"planned": map[string]any{
			"name":     homeName,
			"areaCode": requestString(payload["areaCode"]),
			"areaName": requestString(payload["areaName"]),
		},
	}, 0), nil
}

func buildHomeCreatePayload(request contract.Request) (map[string]any, error) {
	name := firstRequestString(request.Parameters, "name", "homeName", "houseName")
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("invalid_home_create_payload")
	}
	payload := map[string]any{"name": strings.TrimSpace(name)}
	if value := strings.TrimSpace(firstRequestString(request.Parameters, "description", "desc")); value != "" {
		payload["desc"] = value
	}
	for _, key := range []string{"icon", "areaCode", "areaName"} {
		if value := strings.TrimSpace(requestString(request.Parameters[key])); value != "" {
			payload[key] = value
		}
	}
	return payload, nil
}

func homeCreateAcceptedFields() []string {
	return []string{"parameters.name", "parameters.description", "parameters.desc", "parameters.icon", "parameters.areaCode", "parameters.areaName"}
}

func (app *app) commitHomeCreatePlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewHomeCreateClient(endpoint, nil).Run(ctx, api.HomeCreateRequest{
		Name:           planPayloadString(record.Payload, "name"),
		Description:    planPayloadString(record.Payload, "desc"),
		Icon:           planPayloadString(record.Payload, "icon"),
		AreaCode:       planPayloadString(record.Payload, "areaCode"),
		AreaName:       planPayloadString(record.Payload, "areaName"),
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.HomeCreateCredentials{
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
	return homeCreateCommitResponse(request, record, result), nil
}
