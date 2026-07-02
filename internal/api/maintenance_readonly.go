package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunUpgradeFileList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters,
		semantic.FieldCapabilityProductID, semantic.FieldDeviceID, semantic.FieldFirmwareType, semantic.FieldFirmwareVersion,
		semantic.FieldCurrentVersion, semantic.FieldVersion, semantic.FieldLanguageCode, semantic.FieldPageNo, semantic.FieldPageSize, semantic.FieldSort, semantic.FieldOrder, semantic.FieldOrderBy,
	)
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID])))
	if deviceID != "" {
		body[semantic.FieldDeviceID] = deviceID
	}
	if productID, ok := body[semantic.FieldCapabilityProductID]; ok && productID != nil {
		body[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)] = productID
	}
	delete(body, semantic.FieldCapabilityProductID)
	if !hasAnyKey(body, semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID), semantic.FieldDeviceID, semantic.FieldFirmwareType, semantic.FieldFirmwareVersion, semantic.FieldCurrentVersion, semantic.FieldVersion) {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.file.list", "upgrade_file_query_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "upgrade.file.list", "/v1/upgrade/r/listfile", http.MethodPost, body, map[string]any{semantic.FieldFiles: nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunUpgradeProgressGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.progress.get", "device_context_missing"), nil
	}
	body := readonlyBodyFromParameters(request.Parameters, semantic.FieldSort, semantic.FieldOrder, semantic.FieldOrderBy, semantic.FieldPageNo, semantic.FieldPageSize)
	body[semantic.FieldDeviceID] = deviceID
	result, err := client.readPath(ctx, request, "upgrade.progress.get", "/v1/upgrade/r/progress", http.MethodPost, body, map[string]any{semantic.FieldProgress: nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunUpgradeFileBatchList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters,
		semantic.FieldDeviceIDs, semantic.FieldDevices, semantic.FieldQueryList, semantic.FieldCapabilityProductIDs, semantic.FieldLanguageCode, semantic.FieldFirmwareType, semantic.FieldFirmwareVersion, semantic.FieldCurrentVersion, semantic.FieldVersion,
	)
	deviceID := strings.TrimSpace(firstNonEmpty(request.DeviceID, stringFromAny(request.Parameters[semantic.FieldDeviceID])))
	if deviceID != "" && body[semantic.FieldDeviceIDs] == nil && body[semantic.FieldDevices] == nil && body[semantic.FieldQueryList] == nil {
		body[semantic.FieldDeviceIDs] = []string{deviceID}
	}
	if productIDs := stringFromAny(body[semantic.FieldCapabilityProductIDs]); productIDs != "" {
		body[semantic.InternalProductIDsField()] = productIDs
		delete(body, semantic.FieldCapabilityProductIDs)
	}
	if !hasAnyKey(body, semantic.FieldDeviceIDs, semantic.FieldDevices, semantic.FieldQueryList, semantic.InternalProductIDsField()) {
		return metadataReadonlyMissingContext(client.endpoint.Region, "upgrade.file.batch_list", "upgrade_batch_query_context_missing"), nil
	}
	result, err := client.readPath(ctx, request, "upgrade.file.batch_list", "/v1/upgrade/r/batchlistfile", http.MethodPost, body, map[string]any{semantic.FieldFiles: nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunProgressGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	key := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldKey]), stringFromAny(request.Parameters[semantic.FieldProgressKey]), stringFromAny(request.Parameters[semantic.FieldID])))
	if key == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "progress.get", "progress_key_missing"), nil
	}
	return client.readPath(ctx, request, "progress.get", "/v1/progress/r/"+pathSegment(key), http.MethodPost, nil, map[string]any{semantic.FieldProgress: nil})
}

func (client MetadataReadonlyClient) RunAppUpgradeLatestGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, semantic.FieldType, semantic.FieldAppType, semantic.FieldOSType, semantic.FieldLanguageCode)
	if body[semantic.FieldType] == nil {
		if appType := stringFromAny(body[semantic.FieldAppType]); appType != "" {
			body[semantic.FieldType] = appType
		}
	}
	delete(body, semantic.FieldAppType)
	if appType := normalizeAppUpgradeAppType(stringFromAny(body[semantic.FieldType])); appType != "" {
		body[semantic.FieldType] = appType
	}
	if osType := normalizeAppUpgradeOSType(stringFromAny(body[semantic.FieldOSType])); osType != "" {
		body[semantic.FieldOSType] = osType
	}
	if !hasAnyKey(body, semantic.FieldType) || !hasAnyKey(body, semantic.FieldOSType) {
		return metadataReadonlyMissingContext(client.endpoint.Region, "app_upgrade.latest.get", "app_upgrade_query_context_missing"), nil
	}
	return client.readPath(ctx, request, "app_upgrade.latest.get", "/v1/appupgrade/r/latestfile", http.MethodPost, body, map[string]any{semantic.FieldLatestFile: nil})
}

func normalizeAppUpgradeAppType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "user", "yeelight", "yeelight app", "yeelight-app", "用户版", "用户端":
		return "1"
	case "2", "installer", "master", "师傅版", "师傅端":
		return "2"
	case "3", "tv", "电视版", "tv版":
		return "3"
	case "4", "commercial", "commercial-saas", "saas", "商照", "商照saas", "商照saas用户版":
		return "4"
	default:
		return ""
	}
}

func normalizeAppUpgradeOSType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "android", "安卓":
		return "1"
	case "2", "ios", "iphone", "苹果":
		return "2"
	default:
		return ""
	}
}

func (client MetadataReadonlyClient) RunOTAVersionFileBatchList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, semantic.FieldQueryList, semantic.FieldFirmwareType, semantic.FieldFirmwareVersion, semantic.FieldVersion, semantic.FieldLanguageCode)
	if body[semantic.FieldQueryList] == nil && hasAnyKey(body, semantic.FieldFirmwareType, semantic.FieldFirmwareVersion, semantic.FieldVersion) {
		query := map[string]any{}
		for _, key := range []string{semantic.FieldFirmwareType, semantic.FieldFirmwareVersion, semantic.FieldVersion, semantic.FieldLanguageCode} {
			if value, ok := body[key]; ok {
				query[key] = value
				delete(body, key)
			}
		}
		body[semantic.FieldQueryList] = []any{query}
	}
	if !hasAnyKey(body, semantic.FieldQueryList) {
		return metadataReadonlyMissingContext(client.endpoint.Region, "ota.version_file.batch_list", "ota_version_file_query_context_missing"), nil
	}
	path := "/v1/ota/upgrade/r/batchListFilesByVersion"
	queryParameters := map[string]any{}
	if languageCode := request.Parameters[semantic.FieldLanguageCode]; languageCode != nil {
		queryParameters[semantic.FieldLanguage] = languageCode
	}
	if script := request.Parameters[semantic.FieldScript]; script != nil {
		queryParameters[semantic.FieldScript] = script
	}
	if region := request.Parameters[semantic.FieldRegion]; region != nil {
		queryParameters[semantic.FieldRegion] = region
	}
	if query := readonlyQueryValues(queryParameters); query != "" {
		path += "?" + query
	}
	return client.readPath(ctx, request, "ota.version_file.batch_list", path, http.MethodPost, body, map[string]any{semantic.FieldFiles: nil})
}

func (client MetadataReadonlyClient) RunNodePropertyConfigGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	nodeID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldNodeID]),
		request.DeviceID,
		stringFromAny(request.Parameters[semantic.FieldDeviceID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
	))
	nodeType := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldNodeType]),
		stringFromAny(request.Parameters[semantic.FieldType]),
		stringFromAny(request.Parameters[semantic.FieldEntityType]),
	))
	if nodeID == "" || nodeType == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "node.property_config.get", "node_property_config_context_missing")
		result.DeviceID = nodeID
		result.HouseID = strings.TrimSpace(request.HouseID)
		return result, nil
	}
	path := "/v1/nodeConfig/r/node_property?" + readonlyQueryValues(map[string]any{
		semantic.FieldNodeID:   nodeID,
		semantic.FieldNodeType: nodeType,
	})
	result, err := client.readPath(ctx, request, "node.property_config.get", path, http.MethodPost, nil, map[string]any{semantic.FieldProperties: nil})
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
