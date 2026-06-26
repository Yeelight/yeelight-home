package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
			"detail": projectGatewaySummary(response["data"]),
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
			"gateways": projectGatewayRows(response["data"]),
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
	result, err := client.readPath(ctx, request, "gateway.thread.get", "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/thread-info", http.MethodGet, nil, map[string]any{"threadInfo": nil})
	result.DeviceID = gatewayID
	return result, err
}

func (client MetadataReadonlyClient) RunGatewayStatsList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.stats.list", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "gateway.stats.list", "/v1/device/r/gatewayswithstats", http.MethodPost, map[string]any{"houseId": houseID}, map[string]any{"gateways": nil})
}

func (client MetadataReadonlyClient) RunGatewaySceneRelationList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	gatewayID := gatewayIDFromReadonlyRequest(request)
	if gatewayID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.scene_relation.list", "gateway_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "gateway.scene_relation.list", "/v1/scene/r/"+pathSegment(gatewayID)+"/related/sceneId", http.MethodPost, nil, map[string]any{"sceneIds": nil})
	result.DeviceID = gatewayID
	return result, err
}

func gatewayIDFromReadonlyRequest(request MetadataReadonlyRequest) string {
	return strings.TrimSpace(firstNonEmpty(
		request.DeviceID,
		stringFromAny(request.Parameters["gatewayId"]),
		stringFromAny(request.Parameters["gatewayID"]),
		stringFromAny(request.Parameters["id"]),
		stringFromAny(request.Parameters["deviceId"]),
		stringFromAny(request.Parameters["deviceID"]),
	))
}

func readonlyPage(parameters map[string]any, defaultPageNo int, defaultPageSize int) (string, string) {
	pageNo := positiveIntString(firstNonNil(parameters["pageNo"], parameters["page"]), defaultPageNo)
	pageSize := positiveIntString(firstNonNil(parameters["pageSize"], parameters["size"]), defaultPageSize)
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
	rows := nestedRowsFromData(data, "gateways", "rows", "list")
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
	for outputKey, inputKeys := range map[string][]string{
		"id":              {"id", "gatewayId", "deviceId"},
		"did":             {"did"},
		"gatewayDeviceId": {"gatewayDeviceId"},
		"pid":             {"pid"},
		"pcId":            {"pcId", "pcid"},
		"type":            {"type"},
		"name":            {"name", "gatewayName", "deviceName", "alias", "remark"},
		"img":             {"img"},
		"houseId":         {"houseId"},
		"roomId":          {"roomId"},
		"capability":      {"capability"},
		"connectType":     {"connectType"},
		"typeName":        {"typeName"},
		"model":           {"model"},
		"firmwareVersion": {"firmwareVersion", "fwVersion", "version"},
	} {
		for _, inputKey := range inputKeys {
			if raw, ok := item[inputKey]; ok {
				if value := stringFromAny(raw); value != "" {
					gateway[outputKey] = value
					break
				}
			}
		}
	}
	for outputKey, inputKeys := range map[string][]string{
		"online":  {"online", "isOnline"},
		"bind":    {"bind", "isBind"},
		"enabled": {"enabled", "enable"},
	} {
		for _, inputKey := range inputKeys {
			if value, ok := boolFromAny(item[inputKey]); ok {
				gateway[outputKey] = value
				break
			}
		}
	}
	if text := stringFromAny(item["mac"]); text != "" {
		gateway["macMasked"] = maskTail(text, 4)
	}
	if values := stringListFromAny(item["supportedBridgeType"]); len(values) > 0 {
		gateway["supportedBridgeType"] = values
	}
	if values := stringListFromAny(item["roomIds"]); len(values) > 0 {
		gateway["roomIds"] = values
	}
	if values := stringListFromAny(item["deviceIds"]); len(values) > 0 {
		gateway["childDeviceCount"] = len(values)
	}
	if rows := nestedRowsFromData(item["configs"], "rows", "list"); len(rows) > 0 {
		gateway["configCount"] = len(rows)
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
