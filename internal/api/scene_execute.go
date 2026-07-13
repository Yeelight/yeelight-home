package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type SceneExecuteCredentials struct {
	Authorization string
	ClientID      string
}

type SceneExecuteRequest struct {
	HouseID     string
	SceneID     string
	Credentials SceneExecuteCredentials
}

type SceneExecuteResult struct {
	Region   string `json:"region"`
	HouseID  string `json:"houseId"`
	SceneID  string `json:"sceneId"`
	Source   string `json:"source"`
	RawShape string `json:"rawShape"`
	APICalls int    `json:"apiCalls"`
}

type SceneExecuteClient struct {
	endpoint Endpoint
	client   *http.Client
}

type sceneExecuteEndpoint struct {
	source       string
	path         string
	includeHouse bool
}

func NewSceneExecuteClient(endpoint Endpoint, client *http.Client) SceneExecuteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SceneExecuteClient{endpoint: endpoint, client: client}
}

func (client SceneExecuteClient) Run(ctx context.Context, request SceneExecuteRequest) (SceneExecuteResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	sceneID := strings.TrimSpace(request.SceneID)
	if houseID == "" {
		return SceneExecuteResult{}, fmt.Errorf("house id is required")
	}
	if sceneID == "" {
		return SceneExecuteResult{}, fmt.Errorf("scene id is required")
	}
	endpoints := []sceneExecuteEndpoint{
		{
			source: "open_control_scene_endpoint",
			path:   "/v1/open/control/house/" + url.PathEscape(houseID) + "/control/w/scenes/" + url.PathEscape(sceneID),
		},
		{
			source:       "control_device_scene_endpoint",
			path:         "/v1/controll/device/w/scene/" + url.PathEscape(sceneID),
			includeHouse: true,
		},
		{
			source:       "thing_device_scene_endpoint",
			path:         "/v1/thing/device/w/scene/" + url.PathEscape(sceneID),
			includeHouse: true,
		},
	}
	var firstNoGatewayErr error
	for index, endpoint := range endpoints {
		response, err := client.callSceneExecuteEndpoint(ctx, request, houseID, endpoint)
		if err != nil {
			if index == 0 && sceneExecuteNoValidGatewayError(err) {
				firstNoGatewayErr = err
				continue
			}
			if firstNoGatewayErr != nil {
				continue
			}
			return SceneExecuteResult{}, err
		}
		if !isBusinessOK(response) {
			err := sceneExecuteBusinessError(response)
			if index == 0 && sceneExecuteNoValidGatewayResponse(response) {
				firstNoGatewayErr = err
				continue
			}
			if firstNoGatewayErr != nil {
				continue
			}
			return SceneExecuteResult{}, err
		}
		return SceneExecuteResult{
			Region:   client.endpoint.Region,
			HouseID:  houseID,
			SceneID:  sceneID,
			Source:   endpoint.source,
			RawShape: responseDataType(response),
			APICalls: index + 1,
		}, nil
	}
	if firstNoGatewayErr != nil {
		if result, ok := client.runBatchSceneFallback(ctx, request, houseID, sceneID, len(endpoints)); ok {
			return result, nil
		}
		if result, ok := client.runSceneActionPropertyFallback(ctx, request, houseID, sceneID, len(endpoints)); ok {
			return result, nil
		}
	}
	return SceneExecuteResult{}, firstNoGatewayErr
}

func (client SceneExecuteClient) callSceneExecuteEndpoint(ctx context.Context, request SceneExecuteRequest, houseID string, endpoint sceneExecuteEndpoint) (map[string]any, error) {
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if endpoint.includeHouse {
		credentials.HouseID = houseID
	}
	return callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+endpoint.path, map[string]any{}, credentials)
}

func sceneExecuteBusinessError(response map[string]any) error {
	return fmt.Errorf("scene execute returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
}

func sceneExecuteNoValidGatewayResponse(response map[string]any) bool {
	return strings.Contains(responseScalar(response, "message", "msg"), "当前情景无有效网关")
}

func sceneExecuteNoValidGatewayError(err error) bool {
	return err != nil && strings.Contains(strings.TrimSpace(err.Error()), "当前情景无有效网关")
}

func (client SceneExecuteClient) runBatchSceneFallback(ctx context.Context, request SceneExecuteRequest, houseID string, sceneID string, previousCalls int) (SceneExecuteResult, bool) {
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	gatewayIDs, calls := client.sceneExecuteGatewayIDs(ctx, houseID, sceneID, credentials)
	if len(gatewayIDs) == 0 {
		return SceneExecuteResult{}, false
	}
	mapping := map[string]any{}
	numericMapping := map[string]any{}
	for _, gatewayID := range gatewayIDs {
		mapping[gatewayID] = sceneID
		numericMapping[gatewayID] = sceneIDValue(sceneID)
	}
	for _, body := range []any{mapping, numericMapping, map[string]any{"deviceIdSceneMap": mapping}, map[string]any{"deviceIdSceneMap": numericMapping}} {
		calls++
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/thing/device/w/scenes", body, credentials)
		if err == nil && isBusinessOK(response) {
			return SceneExecuteResult{
				Region:   client.endpoint.Region,
				HouseID:  houseID,
				SceneID:  sceneID,
				Source:   "thing_device_scenes_batch_endpoint",
				RawShape: responseDataType(response),
				APICalls: previousCalls + calls,
			}, true
		}
	}
	return SceneExecuteResult{}, false
}

func (client SceneExecuteClient) runSceneActionPropertyFallback(ctx context.Context, request SceneExecuteRequest, houseID string, sceneID string, previousCalls int) (SceneExecuteResult, bool) {
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	detail, calls, ok := client.sceneExecuteDetail(ctx, sceneID, credentials)
	if !ok {
		return SceneExecuteResult{}, false
	}
	rows := sceneExecuteDetailRows(detail, sceneID)
	actions := scenePropertyActionsFromRows(rows)
	if len(actions) == 0 {
		return SceneExecuteResult{}, false
	}
	propertyClient := NewNodePropertySetClient(client.endpoint, client.client)
	for _, action := range actions {
		for propertyName, value := range action.Properties {
			calls++
			_, err := propertyClient.Run(ctx, NodePropertySetRequest{
				HouseID:      houseID,
				NodeType:     action.TargetType,
				NodeID:       action.TargetID,
				PropertyName: propertyName,
				Value:        value,
				Credentials: NodePropertySetCredentials{
					Authorization: request.Credentials.Authorization,
					ClientID:      request.Credentials.ClientID,
				},
			})
			if err != nil {
				return SceneExecuteResult{}, false
			}
		}
	}
	return SceneExecuteResult{
		Region:   client.endpoint.Region,
		HouseID:  houseID,
		SceneID:  sceneID,
		Source:   "scene_actions_node_property_fallback",
		RawShape: "actions",
		APICalls: previousCalls + calls,
	}, true
}

func (client SceneExecuteClient) sceneExecuteDetail(ctx context.Context, sceneID string, credentials requestCredentials) (map[string]any, int, bool) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/scene/"+url.PathEscape(sceneID)+"/r/detail", nil, credentials)
	if err != nil || !isBusinessOK(response) {
		return nil, 1, false
	}
	detail, ok := response["data"].(map[string]any)
	return detail, 1, ok
}

func sceneIDValue(sceneID string) any {
	normalized := strings.TrimSpace(sceneID)
	if normalized == "" {
		return sceneID
	}
	if value, err := strconv.ParseInt(normalized, 10, 64); err == nil {
		return value
	}
	return sceneID
}

func (client SceneExecuteClient) sceneExecuteGatewayIDs(ctx context.Context, houseID string, sceneID string, credentials requestCredentials) ([]string, int) {
	detail, calls, ok := client.sceneExecuteDetail(ctx, sceneID, credentials)
	if !ok {
		return nil, calls
	}
	rows := sceneExecuteDetailRows(detail, sceneID)
	gatewayIDs := uniqueNonEmpty(sceneExecuteGatewayIDsFromRows(rows))
	if len(gatewayIDs) > 0 {
		return gatewayIDs, calls
	}
	for _, targetID := range uniqueNonEmpty(sceneExecuteTargetDeviceIDs(rows)) {
		calls++
		gatewayID, detailCalls := client.sceneExecuteGatewayIDFromDeviceDetail(ctx, houseID, targetID, credentials)
		calls += detailCalls - 1
		if gatewayID == "" {
			continue
		}
		gatewayIDs = append(gatewayIDs, gatewayID)
	}
	return uniqueNonEmpty(gatewayIDs), calls
}

func (client SceneExecuteClient) sceneExecuteGatewayIDFromDeviceDetail(ctx context.Context, houseID string, targetID string, credentials requestCredentials) (string, int) {
	paths := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/v2/thing/manage/house/" + url.PathEscape(houseID) + "/device/" + url.PathEscape(targetID) + "/r/info",
		},
		{
			method: http.MethodPost,
			path:   "/v1/device/" + url.PathEscape(targetID) + "/r/detail",
		},
	}
	for index, candidate := range paths {
		detail, err := callJSON(ctx, client.client, candidate.method, strings.TrimRight(client.endpoint.BaseURL, "/")+candidate.path, nil, credentials)
		if err != nil || !isBusinessOK(detail) {
			continue
		}
		if data, ok := detail["data"].(map[string]any); ok {
			if gatewayID := strings.TrimSpace(firstAnyString(data, semantic.FieldGatewayDeviceID, semantic.FieldGatewayID)); gatewayID != "" {
				return gatewayID, index + 1
			}
			if nested, ok := data[semantic.FieldDetail].(map[string]any); ok {
				if gatewayID := strings.TrimSpace(firstAnyString(nested, semantic.FieldGatewayDeviceID, semantic.FieldGatewayID)); gatewayID != "" {
					return gatewayID, index + 1
				}
			}
		}
	}
	return "", len(paths)
}

func sceneExecuteDetailRows(value any, sceneID string) []map[string]any {
	detail, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	result := sceneExecuteRowsFromPayload(sceneExecuteActionsPayload(detail))
	if len(result) > 0 {
		return result
	}
	projected := sceneDetailData(detail, sceneID)
	return sceneExecuteRowsFromPayload(sceneExecuteActionsPayload(projected))
}

func sceneExecuteRowsFromPayload(value any) []map[string]any {
	rows := rowsFromData(value)
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if item, ok := row.(map[string]any); ok {
			result = append(result, item)
		}
	}
	return result
}

func sceneExecuteActionsPayload(detail map[string]any) any {
	for _, candidate := range []any{
		detail[semantic.FieldDetails],
		detail[semantic.FieldActions],
		nestedSceneExecuteActions(detail, "detail"),
		nestedSceneExecuteActions(detail, "editablePayload"),
		nestedSceneExecuteActions(detail, "scene"),
	} {
		if rows := rowsFromData(candidate); len(rows) > 0 {
			return candidate
		}
	}
	return nil
}

func nestedSceneExecuteActions(detail map[string]any, key string) any {
	nested, ok := detail[key].(map[string]any)
	if !ok {
		return nil
	}
	return firstNonNil(nested[semantic.FieldDetails], nested[semantic.FieldActions])
}

func sceneExecuteGatewayIDsFromRows(rows []map[string]any) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, firstAnyString(row, semantic.FieldGatewayDeviceID, semantic.FieldGatewayID))
	}
	return ids
}

func sceneExecuteTargetDeviceIDs(rows []map[string]any) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		targetType := strings.TrimSpace(firstAnyString(row, semantic.FieldTargetType, semantic.FieldType, "resType"))
		typeID := strings.TrimSpace(firstAnyString(row, "typeId"))
		if targetType != "" && targetType != "device" && targetType != "2" {
			continue
		}
		if typeID != "" && typeID != "2" {
			continue
		}
		ids = append(ids, firstAnyString(row, semantic.FieldTargetID, "resId", semantic.FieldDeviceID, semantic.FieldID))
	}
	return ids
}

type scenePropertyAction struct {
	TargetType string
	TargetID   string
	Properties map[string]any
}

func scenePropertyActionsFromRows(rows []map[string]any) []scenePropertyAction {
	actions := make([]scenePropertyAction, 0, len(rows))
	for _, row := range rows {
		targetType := sceneActionTargetType(row)
		targetID := strings.TrimSpace(firstAnyString(row, semantic.FieldTargetID, "resId", semantic.FieldDeviceID, semantic.FieldGroupID, semantic.FieldRoomID, semantic.FieldAreaID, semantic.FieldHouseID, semantic.FieldID))
		if targetType == "" || targetID == "" {
			continue
		}
		properties := sceneActionSetProperties(row)
		if len(properties) == 0 {
			continue
		}
		actions = append(actions, scenePropertyAction{
			TargetType: targetType,
			TargetID:   targetID,
			Properties: properties,
		})
	}
	return actions
}

func sceneActionTargetType(row map[string]any) string {
	targetType := strings.TrimSpace(firstAnyString(row, semantic.FieldTargetType, semantic.FieldType, "resType", "typeId", semantic.FieldNodeType))
	if targetType == "" {
		return ""
	}
	normalized := NormalizeNodeType(targetType)
	if _, ok := NodeTypeID(normalized); ok {
		return normalized
	}
	return ""
}

func sceneActionSetProperties(row map[string]any) map[string]any {
	raw := editableJSONValue(firstNonNil(row[semantic.FieldSet], row[semantic.FieldParameters], row["params"], row["param"]))
	rawSet, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if nested, ok := editableJSONValue(rawSet[semantic.FieldSet]).(map[string]any); ok {
		rawSet = nested
	}
	set := semantic.ToPublicLightSet(rawSet)
	properties := make(map[string]any, len(set))
	for key, value := range set {
		if propertyName := sceneActionPropertyID(key); propertyName != "" {
			properties[propertyName] = value
		}
	}
	return properties
}

func sceneActionPropertyID(property string) string {
	property = strings.TrimSpace(property)
	if property == "" || semantic.PropertySensitive(property) {
		return ""
	}
	if id, ok := semantic.PropertyID(property); ok {
		return id
	}
	return property
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}
