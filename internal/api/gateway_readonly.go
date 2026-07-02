package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunGatewayDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.detail.get", "house_context_missing"), nil
	}
	gatewayID := gatewayIDFromReadonlyRequest(request)
	if gatewayID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "gateway.detail.get", "gateway_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/info", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("gateway.detail.get", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		DeviceID:   gatewayID,
		Capability: "gateway.detail.get",
		Data: map[string]any{
			semantic.FieldDetail: projectGatewaySummary(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGatewayList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.list", "house_context_missing"), nil
	}
	pageNo, pageSize := readonlyPage(request.Parameters, 1, 100)
	response, err := client.call(ctx, http.MethodGet, "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/r/info/"+pageNo+"/"+pageSize, nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("gateway.list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "gateway.list",
		Data: map[string]any{
			semantic.FieldGateways: projectGatewayRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGatewayThreadGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.thread.get", "house_context_missing"), nil
	}
	gatewayID := gatewayIDFromReadonlyRequest(request)
	if gatewayID == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "gateway.thread.get", "gateway_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	result, err := client.readPath(ctx, request, "gateway.thread.get", "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/thread-info", http.MethodGet, nil, map[string]any{semantic.FieldThreadInfo: nil})
	result.DeviceID = gatewayID
	return result, err
}

func (client MetadataReadonlyClient) RunGatewayStatsList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.stats.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/device/r/gatewayswithstats", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("gateway.stats.list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "gateway.stats.list",
		Data: map[string]any{
			semantic.FieldGateways: projectGatewayStatsRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGatewaySceneRelationList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	gatewayID := gatewayIDFromReadonlyRequest(request)
	if gatewayID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.scene_relation.list", "gateway_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "gateway.scene_relation.list", "/v1/scene/r/"+pathSegment(gatewayID)+"/related/sceneId", http.MethodPost, nil, map[string]any{semantic.FieldSceneIDs: nil})
	result.DeviceID = gatewayID
	return result, err
}

func gatewayIDFromReadonlyRequest(request MetadataReadonlyRequest) string {
	return strings.TrimSpace(firstNonEmpty(
		request.DeviceID,
		stringFromAny(request.Parameters[semantic.FieldGatewayID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
		stringFromAny(request.Parameters[semantic.FieldDeviceID]),
	))
}

func readonlyPage(parameters map[string]any, defaultPageNo int, defaultPageSize int) (string, string) {
	pageNo := positiveIntString(parameters[semantic.FieldPageNo], defaultPageNo)
	pageSize := positiveIntString(firstNonNil(parameters[semantic.FieldPageSize], parameters[semantic.FieldLimit]), defaultPageSize)
	return pageNo, pageSize
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func positiveIntString(value any, fallback int) string {
	if fallback <= 0 {
		fallback = 1
	}
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return strconv.Itoa(typed)
		}
	case float64:
		if typed >= 1 {
			return strconv.Itoa(int(typed))
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return strconv.Itoa(parsed)
		}
	}
	return strconv.Itoa(fallback)
}

func pathSegment(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func projectGatewayRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.GatewayRowContainers()...)
	gateways := make([]any, 0, len(rows))
	for _, row := range rows {
		summary := projectGatewaySummary(row)
		if len(summary) > 0 {
			gateways = append(gateways, summary)
		}
	}
	return gateways
}

func projectGatewayStatsRows(data any) []any {
	rows := nestedRowsFromData(data, semantic.FieldDevices, semantic.FieldGateways, "gateways", "devices", "rows", "list")
	gateways := make([]any, 0, len(rows))
	for _, row := range rows {
		summary := projectGatewaySummary(row)
		if len(summary) > 0 {
			gateways = append(gateways, summary)
		}
	}
	return gateways
}

func projectGatewaySummary(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	gateway := map[string]any{}
	copyResponseStringMappings(gateway, item, semantic.GatewayStringMappings())
	for _, mapping := range semantic.GatewayBoolMappings() {
		for _, inputKey := range mapping.Internal {
			if value, ok := boolFromAny(item[inputKey]); ok {
				gateway[mapping.Public] = value
				break
			}
		}
	}
	if text := stringFromAny(item[semantic.FieldMAC]); text != "" {
		gateway[semantic.FieldMacMasked] = maskTail(text, 4)
	}
	if values := stringListFromAny(item[semantic.SupportedBridgeTypeField()]); len(values) > 0 {
		gateway[semantic.FieldSupportedBridgeType] = values
	}
	if values := stringListFromAny(item[semantic.FieldRoomIDs]); len(values) > 0 {
		gateway[semantic.FieldRoomIDs] = values
	}
	if values := stringListFromAny(item[semantic.FieldDeviceIDs]); len(values) > 0 {
		gateway[semantic.FieldChildDeviceCount] = len(values)
	}
	if rows := nestedRowsFromData(item[semantic.ConfigsField()], semantic.ConfigRowContainers()...); len(rows) > 0 {
		gateway[semantic.FieldConfigCount] = len(rows)
	}
	return gateway
}

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	case float64:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	case int:
		if typed == 1 {
			return true, true
		}
		if typed == 0 {
			return false, true
		}
	}
	return false, false
}
