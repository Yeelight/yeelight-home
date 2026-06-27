package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func (client MetadataReadonlyClient) RunUpgradeFileList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters,
		"pid", "productId", "deviceId", "did", "firmwareType", "firmwareVersion",
		"currentVersion", "version", "languageCode", "pageNo", "pageSize", "sort", "order", "orderBy",
	)
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["did"])))
	if deviceID != "" {
		body["deviceId"] = deviceID
	}
	if body["pid"] == nil {
		if productID := stringFromAny(body["productId"]); productID != "" {
			body["pid"] = productID
		}
	}
	delete(body, "productId")
	delete(body, "did")
	if !hasAnyKey(body, "pid", "productId", "deviceId", "did", "firmwareType", "firmwareVersion", "currentVersion", "version") {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.file.list", "upgrade_file_query_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "upgrade.file.list", "/v1/upgrade/r/listfile", http.MethodPost, body, map[string]any{"files": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunUpgradeProgressGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["did"]), stringFromAny(request.Parameters["id"])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.progress.get", "device_context_missing"), nil
	}
	body := readonlyBodyFromParameters(request.Parameters, "sort", "order", "orderBy", "pageNo", "pageSize")
	body["deviceId"] = deviceID
	result, err := client.readPath(ctx, request, "upgrade.progress.get", "/v1/upgrade/r/progress", http.MethodPost, body, map[string]any{"progress": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunUpgradeFileBatchList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters,
		"deviceIds", "devices", "queryList", "pids", "pid", "productIds", "languageCode", "firmwareType", "firmwareVersion", "currentVersion", "version",
	)
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters["deviceId"]), stringFromAny(request.Parameters["did"])))
	if deviceID != "" && body["deviceIds"] == nil && body["devices"] == nil && body["queryList"] == nil {
		body["deviceIds"] = []string{deviceID}
	}
	if !hasAnyKey(body, "deviceIds", "devices", "queryList", "pids", "pid", "productIds") {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.file.batch_list", "upgrade_batch_query_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "upgrade.file.batch_list", "/v1/upgrade/r/batchlistfile", http.MethodPost, body, map[string]any{"files": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunProgressGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	key := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["key"]), stringFromAny(request.Parameters["progressKey"]), stringFromAny(request.Parameters["id"])))
	if key == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "progress.get", "progress_key_missing"), nil
	}
	return client.readPath(ctx, request, "progress.get", "/v1/progress/r/"+pathSegment(key), http.MethodPost, nil, map[string]any{"progress": nil})
}

func (client MetadataReadonlyClient) RunAppUpgradeLatestGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, "type", "appType", "osType", "languageCode")
	if body["type"] == nil {
		if appType := stringFromAny(body["appType"]); appType != "" {
			body["type"] = appType
		}
	}
	delete(body, "appType")
	if !hasAnyKey(body, "type") || !hasAnyKey(body, "osType") {
		return metadataReadonlyMissingContext(client.endpoint.Region, "app_upgrade.latest.get", "app_upgrade_query_context_missing"), nil
	}
	return client.readPath(ctx, request, "app_upgrade.latest.get", "/v1/appupgrade/r/latestfile", http.MethodPost, body, map[string]any{"latestFile": nil})
}

func (client MetadataReadonlyClient) RunOTAVersionFileBatchList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, "queryList", "firmwareType", "firmwareVersion", "version", "languageCode")
	if body["queryList"] == nil && hasAnyKey(body, "firmwareType", "firmwareVersion", "version") {
		query := map[string]any{}
		for _, key := range []string{"firmwareType", "firmwareVersion", "version", "languageCode"} {
			if value, ok := body[key]; ok {
				query[key] = value
				delete(body, key)
			}
		}
		body["queryList"] = []any{query}
	}
	if !hasAnyKey(body, "queryList") {
		return metadataReadonlyMissingContext(client.endpoint.Region, "ota.version_file.batch_list", "ota_version_file_query_context_missing"), nil
	}
	path := "/v1/ota/upgrade/r/batchListFilesByVersion"
	if query := readonlyQueryFromParameters(request.Parameters, "language", "script", "region"); query != "" {
		path += "?" + query
	}
	return client.readPath(ctx, request, "ota.version_file.batch_list", path, http.MethodPost, body, map[string]any{"files": nil})
}

func (client MetadataReadonlyClient) RunNodePropertyConfigGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	nodeID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["nodeId"]),
		stringFromAny(request.Parameters["nodeID"]),
		request.DeviceID,
		stringFromAny(request.Parameters["deviceId"]),
		stringFromAny(request.Parameters["did"]),
		stringFromAny(request.Parameters["id"]),
	))
	nodeType := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["nodeType"]),
		stringFromAny(request.Parameters["type"]),
		stringFromAny(request.Parameters["entityType"]),
	))
	if nodeID == "" || nodeType == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "node.property_config.get", "node_property_config_context_missing")
		result.DeviceID = nodeID
		result.HouseID = strings.TrimSpace(request.HouseID)
		return result, nil
	}
	path := "/v1/nodeConfig/r/node_property?" + readonlyQueryValues(map[string]any{
		"nodeId":   nodeID,
		"nodeType": nodeType,
	})
	result, err := client.readPath(ctx, request, "node.property_config.get", path, http.MethodPost, nil, map[string]any{"properties": nil})
	if err != nil {
		var statusErr HTTPStatusError
		if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusBadRequest {
			result := metadataReadonlyMissingContext(client.endpoint.Region, "node.property_config.get", "cloud_read_endpoint_unavailable")
			result.HouseID = strings.TrimSpace(request.HouseID)
			result.DeviceID = nodeID
			result.APICalls = 1
			result.RawShape = "<http_400>"
			return result, nil
		}
	}
	result.DeviceID = nodeID
	return result, err
}

func readonlyBodyFromParameters(parameters map[string]any, keys ...string) map[string]any {
	body := map[string]any{}
	for _, key := range keys {
		if value, ok := parameters[key]; ok && value != nil {
			body[key] = value
		}
	}
	return body
}

func readonlyQueryFromParameters(parameters map[string]any, keys ...string) string {
	values := map[string]any{}
	for _, key := range keys {
		if value, ok := parameters[key]; ok && value != nil {
			values[key] = value
		}
	}
	return readonlyQueryValues(values)
}

func readonlyQueryValues(parameters map[string]any) string {
	query := url.Values{}
	for key, value := range parameters {
		if text := stringFromAny(value); text != "" {
			query.Set(key, text)
		}
	}
	return query.Encode()
}

func hasAnyKey(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch typed := value.(type) {
			case []any:
				if len(typed) > 0 {
					return true
				}
			case []string:
				if len(typed) > 0 {
					return true
				}
			case map[string]any:
				if len(typed) > 0 {
					return true
				}
			default:
				if stringFromAny(value) != "" {
					return true
				}
			}
		}
	}
	return false
}
