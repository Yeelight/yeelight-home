package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const nodeTypeDevice = "2"

type NodePropertySetCredentials struct {
	Authorization string
	ClientID      string
}

type NodePropertySetRequest struct {
	HouseID      string
	NodeType     string
	NodeID       string
	PropertyName string
	Value        any
	Command      string
	Duration     any
	Delay        any
	Index        any
	Category     any
	Credentials  NodePropertySetCredentials
}

type NodePropertySetResult struct {
	Region       string `json:"region"`
	HouseID      string `json:"houseId"`
	NodeType     string `json:"nodeType"`
	NodeTypeID   string `json:"nodeTypeId"`
	NodeID       string `json:"nodeId"`
	PropertyName string `json:"propertyName"`
	Command      string `json:"command"`
	Source       string `json:"source"`
	RawShape     string `json:"rawShape"`
	APICalls     int    `json:"apiCalls"`
}

type NodePropertySetClient struct {
	endpoint Endpoint
	client   *http.Client
}

type DevicePropertySetCredentials struct {
	Authorization string
	ClientID      string
}

type DevicePropertySetRequest struct {
	HouseID      string
	DeviceID     string
	PropertyName string
	Value        any
	Command      string
	Credentials  DevicePropertySetCredentials
}

type DevicePropertySetResult struct {
	Region       string `json:"region"`
	HouseID      string `json:"houseId"`
	DeviceID     string `json:"deviceId"`
	PropertyName string `json:"propertyName"`
	Command      string `json:"command"`
	Source       string `json:"source"`
	RawShape     string `json:"rawShape"`
	APICalls     int    `json:"apiCalls"`
}

type DevicePropertySetClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewDevicePropertySetClient(endpoint Endpoint, client *http.Client) DevicePropertySetClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DevicePropertySetClient{endpoint: endpoint, client: client}
}

func NewNodePropertySetClient(endpoint Endpoint, client *http.Client) NodePropertySetClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return NodePropertySetClient{endpoint: endpoint, client: client}
}

func (client NodePropertySetClient) Run(ctx context.Context, request NodePropertySetRequest) (NodePropertySetResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	nodeType := strings.TrimSpace(request.NodeType)
	nodeTypeID, ok := NodeTypeID(nodeType)
	nodeID := strings.TrimSpace(request.NodeID)
	propertyName := strings.TrimSpace(request.PropertyName)
	command := strings.TrimSpace(request.Command)
	if command == "" {
		command = "set"
	}
	if houseID == "" {
		return NodePropertySetResult{}, fmt.Errorf("house id is required")
	}
	if !ok {
		return NodePropertySetResult{}, fmt.Errorf("unsupported node type %q", nodeType)
	}
	if nodeID == "" {
		return NodePropertySetResult{}, fmt.Errorf("node id is required")
	}
	if propertyName == "" {
		return NodePropertySetResult{}, fmt.Errorf("property name is required")
	}
	body := map[string]any{
		"value": request.Value,
	}
	if command != "set" {
		body["command"] = command
	}
	copyOptionalControlField(body, "duration", request.Duration)
	copyOptionalControlField(body, "delay", request.Delay)
	copyOptionalControlField(body, "index", request.Index)
	copyOptionalControlField(body, "category", request.Category)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/open/control/house/"+url.PathEscape(houseID)+"/control/"+url.PathEscape(nodeTypeID)+"/"+url.PathEscape(nodeID)+"/w/properties/"+url.PathEscape(propertyName), body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return NodePropertySetResult{}, err
	}
	if !isBusinessOK(response) {
		return NodePropertySetResult{}, fmt.Errorf("node property set returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return NodePropertySetResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		NodeType:     NormalizeNodeType(nodeType),
		NodeTypeID:   nodeTypeID,
		NodeID:       nodeID,
		PropertyName: propertyName,
		Command:      command,
		Source:       "open_control_node_property_set_endpoint",
		RawShape:     responseDataType(response),
		APICalls:     1,
	}, nil
}

func (client DevicePropertySetClient) Run(ctx context.Context, request DevicePropertySetRequest) (DevicePropertySetResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	deviceID := strings.TrimSpace(request.DeviceID)
	propertyName := strings.TrimSpace(request.PropertyName)
	command := strings.TrimSpace(request.Command)
	if command == "" {
		command = "set"
	}
	if houseID == "" {
		return DevicePropertySetResult{}, fmt.Errorf("house id is required")
	}
	if deviceID == "" {
		return DevicePropertySetResult{}, fmt.Errorf("device id is required")
	}
	if propertyName == "" {
		return DevicePropertySetResult{}, fmt.Errorf("property name is required")
	}
	body := map[string]any{
		"value": request.Value,
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+nodeTypeDevice+"/"+url.PathEscape(deviceID)+"/w/properties/"+url.PathEscape(propertyName), body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return DevicePropertySetResult{}, err
	}
	if !isBusinessOK(response) {
		return DevicePropertySetResult{}, fmt.Errorf("device property set returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return DevicePropertySetResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		DeviceID:     deviceID,
		PropertyName: propertyName,
		Command:      command,
		Source:       "device_property_set_endpoint",
		RawShape:     responseDataType(response),
		APICalls:     1,
	}, nil
}

func copyOptionalControlField(body map[string]any, key string, value any) {
	if value != nil {
		body[key] = value
	}
}

func NormalizeNodeType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "5", "home", "house", "wholehome", "whole_home", "全屋", "家庭":
		return "home"
	case "3", "area", "customgroup", "custom_group", "区域":
		return "area"
	case "1", "room", "房间":
		return "room"
	case "4", "group", "meshgroup", "mesh_group", "devicegroup", "device_group", "灯组", "设备组":
		return "group"
	case "2", "device", "gateway", "panel", "knob", "screen", "设备", "网关", "面板", "旋钮", "屏":
		return "device"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func NodeTypeID(value string) (string, bool) {
	switch NormalizeNodeType(value) {
	case "room":
		return "1", true
	case "device":
		return nodeTypeDevice, true
	case "area":
		return "3", true
	case "group":
		return "4", true
	case "home":
		return "5", true
	default:
		return "", false
	}
}
