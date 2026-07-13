package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type NodeControlCredentials struct {
	Authorization string
	ClientID      string
}

type NodeControlRequest struct {
	HouseID      string
	NodeType     string
	NodeID       string
	PropertyName string
	ActionName   string
	Payload      map[string]any
	Flow         any
	Duration     any
	Delay        any
	Credentials  NodeControlCredentials
}

type NodeControlResult struct {
	Region       string   `json:"region"`
	HouseID      string   `json:"houseId,omitempty"`
	NodeType     string   `json:"nodeType"`
	NodeTypeID   string   `json:"nodeTypeId"`
	NodeID       string   `json:"nodeId"`
	PropertyName string   `json:"propertyName,omitempty"`
	ActionName   string   `json:"actionName,omitempty"`
	Command      string   `json:"command"`
	Source       string   `json:"source"`
	RawShape     string   `json:"rawShape"`
	PropertySet  []string `json:"propertySet,omitempty"`
	TargetCount  int      `json:"targetCount,omitempty"`
	APICalls     int      `json:"apiCalls"`
}

type NodePropertiesSetRequest struct {
	HouseID     string
	NodeType    string
	NodeID      string
	Properties  map[string]any
	Credentials NodeControlCredentials
}

type NodePropertyBatchSetRequest struct {
	HouseID      string
	NodeType     string
	NodeIDs      []string
	PropertyName string
	Value        any
	Credentials  NodeControlCredentials
}

type NodeControlClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewNodeControlClient(endpoint Endpoint, client *http.Client) NodeControlClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return NodeControlClient{endpoint: endpoint, client: client}
}

func (client NodeControlClient) RunToggle(ctx context.Context, request NodeControlRequest) (NodeControlResult, error) {
	houseID, nodeType, nodeTypeID, nodeID, err := normalizeNodeControlTarget(request.HouseID, request.NodeType, request.NodeID)
	if err != nil {
		return NodeControlResult{}, err
	}
	propertyName := strings.TrimSpace(request.PropertyName)
	if propertyName == "" {
		return NodeControlResult{}, fmt.Errorf("property name is required")
	}
	if isSensitiveCloudField(propertyName) {
		return NodeControlResult{}, fmt.Errorf("node property toggle refused sensitive property: %s", propertyName)
	}
	body := safeControlPayload(request.Payload)
	copyOptionalControlField(body, "duration", request.Duration)
	copyOptionalControlField(body, "delay", request.Delay)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+url.PathEscape(nodeTypeID)+"/"+url.PathEscape(nodeID)+"/w/properties/"+url.PathEscape(propertyName)+"/toggle", body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return NodeControlResult{}, err
	}
	if !isBusinessOK(response) {
		return NodeControlResult{}, fmt.Errorf("node property toggle returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return NodeControlResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		NodeType:     nodeType,
		NodeTypeID:   nodeTypeID,
		NodeID:       nodeID,
		PropertyName: propertyName,
		Command:      "toggle",
		Source:       "node_property_toggle_endpoint",
		RawShape:     responseDataType(response),
		APICalls:     1,
	}, nil
}

func (client NodeControlClient) RunAction(ctx context.Context, request NodeControlRequest) (NodeControlResult, error) {
	houseID, nodeType, nodeTypeID, nodeID, err := normalizeNodeControlTarget(request.HouseID, request.NodeType, request.NodeID)
	if err != nil {
		return NodeControlResult{}, err
	}
	actionName := strings.TrimSpace(request.ActionName)
	if actionName == "" {
		return NodeControlResult{}, fmt.Errorf("action name is required")
	}
	body := safeControlPayload(request.Payload)
	copyOptionalControlField(body, "duration", request.Duration)
	copyOptionalControlField(body, "delay", request.Delay)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+url.PathEscape(nodeTypeID)+"/"+url.PathEscape(nodeID)+"/w/actions/"+url.PathEscape(actionName), body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return NodeControlResult{}, err
	}
	if !isBusinessOK(response) {
		return NodeControlResult{}, fmt.Errorf("node action execute returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return NodeControlResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		NodeType:   nodeType,
		NodeTypeID: nodeTypeID,
		NodeID:     nodeID,
		ActionName: actionName,
		Command:    "action",
		Source:     "node_action_execute_endpoint",
		RawShape:   responseDataType(response),
		APICalls:   1,
	}, nil
}

func (client NodeControlClient) RunFlow(ctx context.Context, request NodeControlRequest) (NodeControlResult, error) {
	houseID, nodeType, nodeTypeID, nodeID, err := normalizeNodeControlTarget(request.HouseID, request.NodeType, request.NodeID)
	if err != nil {
		return NodeControlResult{}, err
	}
	body := flowPayload(request.Flow, request.Payload)
	if len(body) == 0 {
		return NodeControlResult{}, fmt.Errorf("flow payload is required")
	}
	copyOptionalControlField(body, "duration", request.Duration)
	copyOptionalControlField(body, "delay", request.Delay)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+url.PathEscape(nodeTypeID)+"/"+url.PathEscape(nodeID)+"/w/flow", body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return NodeControlResult{}, err
	}
	if !isBusinessOK(response) {
		return NodeControlResult{}, fmt.Errorf("lighting flow execute returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return NodeControlResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		NodeType:   nodeType,
		NodeTypeID: nodeTypeID,
		NodeID:     nodeID,
		Command:    "flow",
		Source:     "lighting_flow_execute_endpoint",
		RawShape:   responseDataType(response),
		APICalls:   1,
	}, nil
}

func (client NodeControlClient) RunPropertiesSet(ctx context.Context, request NodePropertiesSetRequest) (NodeControlResult, error) {
	houseID, nodeType, nodeTypeID, nodeID, err := normalizeNodeControlTarget(request.HouseID, request.NodeType, request.NodeID)
	if err != nil {
		return NodeControlResult{}, err
	}
	properties := sanitizeControlProperties(request.Properties)
	if len(properties) == 0 {
		return NodeControlResult{}, fmt.Errorf("at least one writable property is required")
	}
	apiCalls := 0
	propertyNames := make([]string, 0, len(properties))
	setClient := NewNodePropertySetClient(client.endpoint, client.client)
	for propertyName, value := range properties {
		_, err := setClient.Run(ctx, NodePropertySetRequest{
			HouseID:      houseID,
			NodeType:     nodeType,
			NodeID:       nodeID,
			PropertyName: propertyName,
			Value:        value,
			Credentials: NodePropertySetCredentials{
				Authorization: request.Credentials.Authorization,
				ClientID:      request.Credentials.ClientID,
			},
		})
		apiCalls++
		if err != nil {
			return NodeControlResult{}, err
		}
		propertyNames = append(propertyNames, propertyName)
	}
	return NodeControlResult{
		Region:      client.endpoint.Region,
		HouseID:     houseID,
		NodeType:    nodeType,
		NodeTypeID:  nodeTypeID,
		NodeID:      nodeID,
		Command:     "set",
		Source:      "node_properties_set_fanout",
		RawShape:    fmt.Sprintf("fanout:%d", apiCalls),
		PropertySet: propertyNames,
		APICalls:    apiCalls,
	}, nil
}

func (client NodeControlClient) RunPropertyBatchSet(ctx context.Context, request NodePropertyBatchSetRequest) (NodeControlResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	nodeType := NormalizeNodeType(request.NodeType)
	nodeTypeID, ok := NodeTypeID(nodeType)
	if !ok {
		return NodeControlResult{}, fmt.Errorf("unsupported node type %q", request.NodeType)
	}
	if houseID == "" {
		return NodeControlResult{}, fmt.Errorf("house id is required")
	}
	propertyName := strings.TrimSpace(request.PropertyName)
	if propertyName == "" {
		return NodeControlResult{}, fmt.Errorf("property name is required")
	}
	if isSensitiveCloudField(propertyName) {
		return NodeControlResult{}, fmt.Errorf("node property batch set refused sensitive property: %s", propertyName)
	}
	nodeIDs := compactStringSet(request.NodeIDs)
	if len(nodeIDs) == 0 {
		return NodeControlResult{}, fmt.Errorf("at least one node id is required")
	}
	setClient := NewNodePropertySetClient(client.endpoint, client.client)
	for _, nodeID := range nodeIDs {
		if _, err := setClient.Run(ctx, NodePropertySetRequest{
			HouseID:      houseID,
			NodeType:     nodeType,
			NodeID:       nodeID,
			PropertyName: propertyName,
			Value:        request.Value,
			Credentials: NodePropertySetCredentials{
				Authorization: request.Credentials.Authorization,
				ClientID:      request.Credentials.ClientID,
			},
		}); err != nil {
			return NodeControlResult{}, err
		}
	}
	return NodeControlResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		NodeType:     nodeType,
		NodeTypeID:   nodeTypeID,
		NodeID:       strings.Join(nodeIDs, ","),
		PropertyName: propertyName,
		Command:      "set",
		Source:       "node_property_batch_set_fanout",
		RawShape:     fmt.Sprintf("fanout:%d", len(nodeIDs)),
		TargetCount:  len(nodeIDs),
		APICalls:     len(nodeIDs),
	}, nil
}

func normalizeNodeControlTarget(houseID string, nodeTypeValue string, nodeIDValue string) (string, string, string, string, error) {
	houseID = strings.TrimSpace(houseID)
	nodeType := NormalizeNodeType(nodeTypeValue)
	nodeTypeID, ok := NodeTypeID(nodeType)
	nodeID := strings.TrimSpace(nodeIDValue)
	if houseID == "" {
		return "", "", "", "", fmt.Errorf("house id is required")
	}
	if !ok {
		return "", "", "", "", fmt.Errorf("unsupported node type %q", nodeTypeValue)
	}
	if nodeID == "" {
		return "", "", "", "", fmt.Errorf("node id is required")
	}
	return houseID, nodeType, nodeTypeID, nodeID, nil
}

func safeControlPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	result := map[string]any{}
	for key, value := range payload {
		if isSensitiveCloudField(key) {
			continue
		}
		result[key] = value
	}
	return result
}

func sanitizeControlProperties(properties map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range properties {
		propertyName := strings.TrimSpace(key)
		if propertyName == "" || isSensitiveCloudField(propertyName) {
			continue
		}
		result[propertyName] = value
	}
	return result
}

func flowPayload(flow any, payload map[string]any) map[string]any {
	if flowMap, ok := flow.(map[string]any); ok {
		return safeControlPayload(flowMap)
	}
	body := safeControlPayload(payload)
	if flow != nil {
		body["flow"] = flow
	}
	return body
}
