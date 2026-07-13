package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type StateQueryCredentials struct {
	Authorization string
	ClientID      string
}

type StateQueryRequest struct {
	HouseID      string
	NodeType     string
	NodeID       string
	DeviceID     string
	PropertyName string
	PropertySet  []string
	Credentials  StateQueryCredentials
}

type StateQueryResult struct {
	Region       string         `json:"region"`
	HouseID      string         `json:"houseId,omitempty"`
	NodeType     string         `json:"nodeType,omitempty"`
	NodeTypeID   string         `json:"nodeTypeId,omitempty"`
	NodeID       string         `json:"nodeId,omitempty"`
	DeviceID     string         `json:"deviceId"`
	PropertyName string         `json:"propertyName,omitempty"`
	QueryScope   string         `json:"queryScope"`
	Source       string         `json:"source"`
	RawShape     string         `json:"rawShape"`
	Properties   map[string]any `json:"properties,omitempty"`
	Value        any            `json:"value,omitempty"`
	Skipped      []string       `json:"skipped,omitempty"`
	APICalls     int            `json:"apiCalls"`
}

type StateQueryClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewStateQueryClient(endpoint Endpoint, client *http.Client) StateQueryClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return StateQueryClient{endpoint: endpoint, client: client}
}

func (client StateQueryClient) Run(ctx context.Context, request StateQueryRequest) (StateQueryResult, error) {
	nodeType := NormalizeNodeType(request.NodeType)
	nodeID := strings.TrimSpace(request.NodeID)
	if nodeType != "" && nodeType != "device" {
		return client.runNode(ctx, request, nodeType, nodeID)
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" && nodeType == "device" {
		deviceID = nodeID
	}
	propertyName := strings.TrimSpace(request.PropertyName)
	if deviceID == "" {
		return StateQueryResult{}, fmt.Errorf("device id is required")
	}
	if propertyName != "" && isSensitiveCloudField(propertyName) {
		return StateQueryResult{}, fmt.Errorf("device state query refused sensitive property: %s", propertyName)
	}
	if propertyName == "" && len(request.PropertySet) > 0 {
		return client.runPropertySet(ctx, request, deviceID)
	}
	path := "/v1/controll/device/" + url.PathEscape(deviceID) + "/r/properties"
	queryScope := "all_properties"
	if propertyName != "" {
		path += "/" + url.PathEscape(propertyName)
		queryScope = "single_property"
	}
	body := map[string]any{}
	if propertyName == "" && len(request.PropertySet) > 0 {
		body["propertySet"] = compactStringSet(request.PropertySet)
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return StateQueryResult{}, err
	}
	if !isBusinessOK(response) {
		return StateQueryResult{}, fmt.Errorf("device state query returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	data := response["data"]
	result := StateQueryResult{
		Region:       client.endpoint.Region,
		DeviceID:     deviceID,
		PropertyName: propertyName,
		QueryScope:   queryScope,
		Source:       "device_properties_endpoint",
		RawShape:     stateDataShape(data),
		APICalls:     1,
	}
	if propertyName == "" {
		result.Properties = projectStateProperties(data)
		return result, nil
	}
	result.Value = data
	return result, nil
}

func (client StateQueryClient) runNode(ctx context.Context, request StateQueryRequest, nodeType string, nodeID string) (StateQueryResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	nodeTypeID, ok := NodeTypeID(nodeType)
	propertyName := strings.TrimSpace(request.PropertyName)
	if houseID == "" {
		return StateQueryResult{}, fmt.Errorf("house id is required")
	}
	if !ok {
		return StateQueryResult{}, fmt.Errorf("unsupported node type %q", nodeType)
	}
	if nodeID == "" {
		return StateQueryResult{}, fmt.Errorf("node id is required")
	}
	if propertyName != "" && isSensitiveCloudField(propertyName) {
		return StateQueryResult{}, fmt.Errorf("node state query refused sensitive property: %s", propertyName)
	}
	body := map[string]any{}
	path := "/v1/open/control/house/" + url.PathEscape(houseID) + "/control/" + url.PathEscape(nodeTypeID) + "/" + url.PathEscape(nodeID) + "/r/properties"
	queryScope := "all_properties"
	if propertyName != "" {
		path += "/" + url.PathEscape(propertyName)
		queryScope = "single_property"
	} else if propertySet := compactStringSet(request.PropertySet); len(propertySet) > 0 {
		body["propNames"] = propertySet
		queryScope = "selected_properties"
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return StateQueryResult{}, err
	}
	if !isBusinessOK(response) {
		return StateQueryResult{}, fmt.Errorf("node state query returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	data := response["data"]
	result := StateQueryResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		NodeType:     nodeType,
		NodeTypeID:   nodeTypeID,
		NodeID:       nodeID,
		PropertyName: propertyName,
		QueryScope:   queryScope,
		Source:       "open_control_node_properties_endpoint",
		RawShape:     stateDataShape(data),
		APICalls:     1,
	}
	if propertyName == "" {
		result.Properties = projectStateProperties(data)
		return result, nil
	}
	result.Value = data
	return result, nil
}

func (client StateQueryClient) runPropertySet(ctx context.Context, request StateQueryRequest, deviceID string) (StateQueryResult, error) {
	properties := map[string]any{}
	skipped := []string{}
	apiCalls := 0
	for _, property := range compactStringSet(request.PropertySet) {
		if isSensitiveCloudField(property) {
			skipped = append(skipped, property+":sensitive_property_not_readable")
			continue
		}
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+url.PathEscape(deviceID)+"/r/properties/"+url.PathEscape(property), map[string]any{}, requestCredentials{
			Authorization: request.Credentials.Authorization,
			ClientID:      request.Credentials.ClientID,
		})
		apiCalls++
		if err != nil {
			return StateQueryResult{}, err
		}
		if !isBusinessOK(response) {
			skipped = append(skipped, property+":"+responseScalar(response, "code")+":"+responseScalar(response, "message", "msg"))
			continue
		}
		properties[property] = response["data"]
	}
	if len(properties) == 0 {
		return StateQueryResult{}, fmt.Errorf("device state query returned no readable properties: skipped=%d", len(skipped))
	}
	return StateQueryResult{
		Region:     client.endpoint.Region,
		DeviceID:   deviceID,
		QueryScope: "all_properties",
		Source:     "device_properties_endpoint",
		RawShape:   fmt.Sprintf("object:%d", len(properties)),
		Properties: properties,
		Skipped:    skipped,
		APICalls:   apiCalls,
	}, nil
}

func compactStringSet(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func projectStateProperties(data any) map[string]any {
	switch typed := data.(type) {
	case map[string]any:
		if nested, ok := typed["properties"].(map[string]any); ok {
			return filterStateProperties(nested)
		}
		return filterStateProperties(typed)
	case []any:
		return filterStatePropertyRows(typed)
	default:
		return map[string]any{}
	}
}

func filterStatePropertyRows(rows []any) map[string]any {
	result := map[string]any{}
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		property := firstAnyString(item, "propertyName", "propName", "property", "name", "code", "id")
		if property == "" || isSensitiveCloudField(property) {
			continue
		}
		value, ok := firstStatePropertyRowValue(item)
		if !ok {
			continue
		}
		result[property] = value
	}
	return result
}

func firstStatePropertyRowValue(item map[string]any) (any, bool) {
	for _, key := range []string{"value", "val", "propertyValue", "data"} {
		if value, ok := item[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func filterStateProperties(properties map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range properties {
		if isSensitiveCloudField(key) {
			continue
		}
		result[key] = value
	}
	return result
}

func stateDataShape(data any) string {
	switch typed := data.(type) {
	case nil:
		return "<nil>"
	case []any:
		return "array"
	case map[string]any:
		return fmt.Sprintf("object:%d", len(typed))
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "number"
	default:
		return fmt.Sprintf("%T", typed)
	}
}
