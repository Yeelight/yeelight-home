package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) prepareGatewayConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", gatewayConfigurationAcceptedFields()), nil
	}
	payload, err := buildGatewayConfigurationPayload(request, houseID)
	if err != nil {
		return gatewayConfigurationClarificationResponse(request, err.Error()), nil
	}
	gatewayID := valueIDString(payload["gatewayId"])
	detail, calls, err := api.NewDestructiveDeleteClient(endpoint, nil).ProbeGateway(ctx, houseID, gatewayID, api.DestructiveDeleteCredentials{
		Authorization: authorization,
		ClientID:      clientID,
	})
	if err != nil {
		return contract.Response{}, err
	}
	if detail.ID == "" {
		return gatewayConfigurationClarificationResponse(request, "invalid_gateway_reference"), nil
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
	if reason := validateGatewayConfigurationPayload(payload, entities); reason != "" {
		return gatewayConfigurationClarificationResponse(request, reason), nil
	}
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("更新网关 %s", firstNonEmptyString(detail.Name, gatewayID)), payload, []string{
		"提交前重新读取网关详情",
		"关联房间必须属于当前家庭",
		"Runtime 根据当前请求构建受控网关 payload",
		"提交后通过 gateway.detail.get 验证名称字段；其他字段按云端写入确认加详情可读性验证",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		"current": map[string]any{"id": detail.ID, "name": detail.Name},
		"planned": executionPayloadPreview(operation.Prepared{
			HouseID: houseID,
			Payload: payload,
		}),
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, calls), nil
}

func buildGatewayConfigurationPayload(request contract.Request, houseID string) (map[string]any, error) {
	gatewayID := firstValueIDString(request.Parameters, "gatewayId", "deviceId", "id", "entityId")
	payload := map[string]any{
		"houseId":   requestNumberOrString(houseID),
		"gatewayId": gatewayID,
		"id":        gatewayID,
	}
	if !copyOptionalSpaceFields(payload, request.Parameters, []string{"name", "desc", "icon", "mac", "roomIds"}) || gatewayID == "" {
		return nil, fmt.Errorf("invalid_gateway_configure_payload")
	}
	return payload, nil
}

func validateGatewayConfigurationPayload(payload map[string]any, entities api.EntityListResult) string {
	for _, roomID := range valueIDList(payload["roomIds"]) {
		if !entityExists(entities, "room", roomID) {
			return "invalid_gateway_room_reference"
		}
	}
	return ""
}

func gatewayConfigurationAcceptedFields() []string {
	return []string{"parameters.houseId", "parameters.gatewayId", "parameters.name", "parameters.desc", "parameters.icon", "parameters.mac", "parameters.roomIds"}
}

func gatewayConfigurationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, gatewayConfigurationAcceptedFields(), gatewayConfigurePayloadGuide())
}

func (app *app) executeGatewayConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewGatewayConfigurationClient(endpoint, nil).Run(ctx, api.GatewayConfigurationRequest{
		HouseID:        record.HouseID,
		GatewayID:      valueIDString(record.Payload["gatewayId"]),
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.GatewayConfigurationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return gatewayConfigurationExecuteResponse(request, record, result), nil
}
