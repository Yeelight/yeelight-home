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
