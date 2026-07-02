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

func (app *app) prepareHomeCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, authorization string, clientID string) (contract.Response, error) {
	payload, err := buildHomeCreatePayload(request)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeCreateAcceptedFields()), nil
	}
	credentials := api.HomeCreateCredentials{Authorization: authorization, ClientID: clientID}
	homeName := requestString(payload[semantic.FieldName])
	client := api.NewHomeCreateClient(endpoint, nil)
	existing, apiCalls, err := client.FindHouseByNameForPlan(ctx, homeName, credentials)
	if err != nil {
		return contract.Response{}, err
	}
	if existing.ID != "" {
		response := homeCreateAlreadyExistsResponse(request, existing, apiCalls)
		response.Result[semantic.FieldRegion] = endpoint.Region
		return response, nil
	}
	record, err := operation.NewAccountPrepared(profile, region, request.Intent, request.RequestID, fmt.Sprintf("创建家庭 %s", homeName), payload, []string{
		"提交前重新读取当前账号家庭列表",
		"家庭名称不存在时才创建",
		"Runtime 根据当前请求构建受控创建 payload",
		"提交后优先通过家庭列表按名称验证；如果列表延迟但创建返回 houseId，则通过该家庭的实体列表验证可访问性",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, api.EntityListResult{Region: endpoint.Region, APICalls: apiCalls}, map[string]any{
		semantic.FieldScope: "account",
		semantic.FieldPlanned: map[string]any{
			semantic.FieldName:     homeName,
			semantic.FieldAreaCode: requestString(payload[semantic.FieldAreaCode]),
			semantic.FieldAreaName: requestString(payload[semantic.FieldAreaName]),
		},
	}, 0), nil
}

func buildHomeCreatePayload(request contract.Request) (map[string]any, error) {
	name := firstRequestString(request.Parameters, semantic.FieldName, semantic.FieldHomeName, semantic.FieldHouseName)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("invalid_home_create_payload")
	}
	payload := map[string]any{semantic.FieldName: strings.TrimSpace(name)}
	if value := strings.TrimSpace(firstRequestString(request.Parameters, semantic.FieldDescription)); value != "" {
		payload[semantic.FieldDescription] = value
	}
	for _, key := range []string{semantic.FieldIcon, semantic.FieldAreaCode, semantic.FieldAreaName} {
		if value := strings.TrimSpace(requestString(request.Parameters[key])); value != "" {
			payload[key] = value
		}
	}
	return payload, nil
}

func homeCreateAcceptedFields() []string {
	return semanticParameterPaths(semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldAreaCode, semantic.FieldAreaName)
}

func (app *app) executeHomeCreate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewHomeCreateClient(endpoint, nil).Run(ctx, api.HomeCreateRequest{
		Name:           executionPayloadString(record.Payload, semantic.FieldName),
		Description:    executionPayloadString(record.Payload, semantic.FieldDescription),
		Icon:           executionPayloadString(record.Payload, semantic.FieldIcon),
		AreaCode:       executionPayloadString(record.Payload, semantic.FieldAreaCode),
		AreaName:       executionPayloadString(record.Payload, semantic.FieldAreaName),
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
	return homeCreateExecuteResponse(request, record, result), nil
}
