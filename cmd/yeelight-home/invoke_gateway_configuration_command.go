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

func (app *app) prepareGatewayConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", gatewayConfigurationAcceptedFields()), nil
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
	payload, err := buildGatewayConfigurationPayload(request, houseID, entities)
	if err != nil {
		return gatewayConfigurationClarificationResponse(request, err.Error()), nil
	}
	gatewayID := valueIDString(payload[semantic.FieldGatewayID])
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
		semantic.FieldCurrent: map[string]any{semantic.FieldID: detail.ID, semantic.FieldName: detail.Name},
		semantic.FieldPlanned: executionPayloadPreview(operation.Prepared{
			HouseID: houseID,
			Payload: payload,
		}),
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, calls), nil
}

func buildGatewayConfigurationPayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, error) {
	gatewayID, err := resolveGatewayConfigurationID(request, entities)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		semantic.FieldHouseID:   requestNumberOrString(houseID),
		semantic.FieldGatewayID: gatewayID,
		semantic.FieldID:        gatewayID,
	}
	if !copyOptionalSpaceFields(payload, request.Parameters, []string{semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldMAC, semantic.FieldRoomIDs}) || gatewayID == "" {
		return nil, fmt.Errorf("invalid_gateway_configure_payload")
	}
	if err := addGatewayConfigurationRoomIDs(payload, request, entities); err != nil {
		return nil, err
	}
	return payload, nil
}

func resolveGatewayConfigurationID(request contract.Request, entities api.EntityListResult) (string, error) {
	gatewayID := firstValueIDString(request.Parameters, semantic.FieldGatewayID, semantic.FieldDeviceID, semantic.FieldEntityID)
	target := entityGetTargetFromRequest(request)
	if gatewayID == "" && target.entityType == "device" {
		gatewayID = target.id
	}
	if gatewayID != "" {
		return gatewayID, nil
	}
	gatewayName := firstRequestString(request.Parameters, semantic.FieldGatewayName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldCurrentName)
	if gatewayName == "" && target.entityType == "device" {
		gatewayName = target.name
	}
	if gatewayName == "" {
		return "", fmt.Errorf("invalid_gateway_configure_payload")
	}
	match, candidates, _ := findEntity(entityGetTarget{name: gatewayName, entityType: "device"}, entities.Entities)
	if match.ID != "" && len(candidates) == 1 {
		return match.ID, nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("ambiguous_gateway_reference")
	}
	return "", fmt.Errorf("invalid_gateway_reference")
}

func addGatewayConfigurationRoomIDs(payload map[string]any, request contract.Request, entities api.EntityListResult) error {
	if len(valueIDList(payload[semantic.FieldRoomIDs])) > 0 {
		return nil
	}
	roomNames := requestStringList(request.Parameters[semantic.FieldRoomNames], request.Parameters[semantic.FieldRoomName], request.Parameters[semantic.FieldTargetRoomName])
	if len(roomNames) == 0 {
		return nil
	}
	roomIDs := make([]any, 0, len(roomNames))
	for _, roomName := range roomNames {
		match, candidates, _ := findEntity(entityGetTarget{name: roomName, entityType: "room"}, entities.Entities)
		if match.ID != "" && len(candidates) == 1 {
			roomIDs = append(roomIDs, match.ID)
			continue
		}
		if len(candidates) > 1 {
			return fmt.Errorf("ambiguous_gateway_room_reference")
		}
		return fmt.Errorf("invalid_gateway_room_reference")
	}
	payload[semantic.FieldRoomIDs] = roomIDs
	return nil
}

func validateGatewayConfigurationPayload(payload map[string]any, entities api.EntityListResult) string {
	for _, roomID := range valueIDList(payload[semantic.FieldRoomIDs]) {
		if !entityExists(entities, "room", roomID) {
			return "invalid_gateway_room_reference"
		}
	}
	return ""
}

func gatewayConfigurationAcceptedFields() []string {
	return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldGatewayID, semantic.FieldDeviceID, semantic.FieldGatewayName, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName, semantic.FieldDescription, semantic.FieldIcon, semantic.FieldMAC, semantic.FieldRoomIDs, semantic.FieldRoomNames, semantic.FieldRoomName, semantic.FieldTargetRoomName)
}

func gatewayConfigurationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, gatewayConfigurationAcceptedFields(), gatewayConfigurePayloadGuide())
}

func (app *app) executeGatewayConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewGatewayConfigurationClient(endpoint, nil).Run(ctx, api.GatewayConfigurationRequest{
		HouseID:        record.HouseID,
		GatewayID:      valueIDString(record.Payload[semantic.FieldGatewayID]),
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
