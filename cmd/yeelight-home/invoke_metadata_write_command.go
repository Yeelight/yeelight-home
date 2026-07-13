package main

import (
	"context"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) invokeHomePropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	properties := metadataWriteProperties(request)
	if houseID == "" || len(properties) == 0 {
		return metadataWriteClarificationResponse(request, "missing_home_property_payload", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.ParameterPath(semantic.FieldProperties),
			semantic.ParameterPath(semantic.FieldPayload),
		}), nil
	}
	result, err := api.NewMetadataWriteClient(endpoint, nil).RunHomePropertySet(ctx, api.HomePropertySetRequest{
		HouseID:    houseID,
		Properties: properties,
		Credentials: api.MetadataWriteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return metadataWriteResponse(request, result, "已更新家庭属性。", "home-property-set-command"), nil
}

func (app *app) invokePanelClick(ctx context.Context, request contract.Request, endpoint api.Endpoint, authorization string, clientID string) (contract.Response, error) {
	resID := firstRequestString(request.Parameters, semantic.FieldPanelID, semantic.FieldDeviceID, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldID)
	if resID == "" {
		return metadataWriteClarificationResponse(request, "missing_panel_resource", []string{
			semantic.ParameterPath(semantic.FieldPanelID),
			semantic.ParameterPath(semantic.FieldDeviceID),
			semantic.ParameterPath(semantic.FieldTargetID),
			semantic.ParameterPath(semantic.FieldPayload),
		}), nil
	}
	result, err := api.NewMetadataWriteClient(endpoint, nil).RunPanelClick(ctx, api.PanelClickRequest{
		ResID:   resID,
		Payload: metadataWritePayload(request),
		Credentials: api.MetadataWriteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return metadataWriteResponse(request, result, "已发送面板点击。", "panel-click-command"), nil
}

func (app *app) invokeSensorEventWrite(ctx context.Context, request contract.Request, endpoint api.Endpoint, authorization string, clientID string) (contract.Response, error) {
	operation := firstNonEmptyString(firstRequestString(request.Parameters, semantic.FieldOperation), "create")
	eventID := firstRequestString(request.Parameters, semantic.FieldEventID, semantic.FieldID)
	payload := metadataWritePayload(request)
	if len(payload) == 0 && operation != "delete" && operation != "remove" {
		return metadataWriteClarificationResponse(request, "missing_sensor_event_payload", []string{
			semantic.ParameterPath(semantic.FieldOperation),
			semantic.ParameterPath(semantic.FieldEventID),
			semantic.ParameterPath(semantic.FieldPayload),
		}), nil
	}
	result, err := api.NewMetadataWriteClient(endpoint, nil).RunSensorEventWrite(ctx, api.SensorEventWriteRequest{
		Operation: operation,
		EventID:   eventID,
		Payload:   payload,
		Credentials: api.MetadataWriteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return metadataWriteResponse(request, result, "已更新传感器事件。", "sensor-event-write-command"), nil
}

func metadataWriteProperties(request contract.Request) map[string]any {
	if properties := requestMap(request.Parameters[semantic.FieldProperties]); properties != nil {
		return properties
	}
	return metadataWritePayload(request)
}

func metadataWritePayload(request contract.Request) map[string]any {
	if payload := requestMap(request.Parameters[semantic.FieldPayload]); payload != nil {
		return payload
	}
	if payload := requestMap(request.Parameters[semantic.FieldParameters]); payload != nil {
		return payload
	}
	return map[string]any{}
}

func metadataWriteResponse(request contract.Request, result api.MetadataWriteResult, message string, traceID string) contract.Response {
	payload := map[string]any{
		semantic.FieldRegion:     result.Region,
		semantic.FieldHouseID:    result.HouseID,
		semantic.FieldID:         result.ID,
		semantic.FieldCapability: result.Capability,
		semantic.FieldOperation:  result.Operation,
		semantic.FieldSource:     result.Source,
		semantic.FieldRawShape:   result.RawShape,
		semantic.FieldResult:     result.Result,
	}
	if len(result.Payload) > 0 {
		payload[semantic.FieldPayload] = result.Payload
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     message,
		Result:          payload,
		Warnings:        []string{},
		TraceID:         traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func metadataWriteClarificationResponse(request contract.Request, reason string, acceptedFields []string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充要写入的内容。",
		Clarification: map[string]any{
			semantic.FieldReason:         reason,
			semantic.FieldAcceptedFields: acceptedFields,
			semantic.FieldPayloadGuide:   payloadGuideForIntent(request.Intent),
		},
		Warnings: []string{},
		TraceID:  "metadata-write-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
}
