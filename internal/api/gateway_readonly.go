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
	result, err := client.readPath(ctx, request, "gateway.detail.get", "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/info", http.MethodGet, nil, map[string]any{"detail": nil})
	result.DeviceID = gatewayID
	return result, err
}

func (client MetadataReadonlyClient) RunGatewayList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "gateway.list", "house_context_missing"), nil
	}
	pageNo, pageSize := readonlyPage(request.Parameters, 1, 100)
	return client.readPath(ctx, request, "gateway.list", "/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/r/info/"+pageNo+"/"+pageSize, http.MethodGet, nil, map[string]any{"gateways": nil})
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
