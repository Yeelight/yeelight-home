package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DevicePropertyAdjustCredentials struct {
	Authorization string
	ClientID      string
}

type DevicePropertyAdjustRequest struct {
	DeviceID     string
	PropertyName string
	Value        any
	Credentials  DevicePropertyAdjustCredentials
}

type DevicePropertyAdjustResult struct {
	Region       string `json:"region"`
	DeviceID     string `json:"deviceId"`
	PropertyName string `json:"propertyName"`
	Command      string `json:"command"`
	Source       string `json:"source"`
	RawShape     string `json:"rawShape"`
	APICalls     int    `json:"apiCalls"`
}

type DevicePropertyAdjustClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewDevicePropertyAdjustClient(endpoint Endpoint, client *http.Client) DevicePropertyAdjustClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DevicePropertyAdjustClient{endpoint: endpoint, client: client}
}

func (client DevicePropertyAdjustClient) Run(ctx context.Context, request DevicePropertyAdjustRequest) (DevicePropertyAdjustResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	propertyName := strings.TrimSpace(request.PropertyName)
	if deviceID == "" {
		return DevicePropertyAdjustResult{}, fmt.Errorf("device id is required")
	}
	if propertyName == "" {
		return DevicePropertyAdjustResult{}, fmt.Errorf("property name is required")
	}
	body := map[string]any{
		"value": request.Value,
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/controll/device/"+nodeTypeDevice+"/"+url.PathEscape(deviceID)+"/w/properties/"+url.PathEscape(propertyName)+"/adjust", body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return DevicePropertyAdjustResult{}, err
	}
	if !isBusinessOK(response) {
		return DevicePropertyAdjustResult{}, fmt.Errorf("device property adjust returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return DevicePropertyAdjustResult{
		Region:       client.endpoint.Region,
		DeviceID:     deviceID,
		PropertyName: propertyName,
		Command:      "adjust",
		Source:       "device_property_adjust_endpoint",
		RawShape:     responseDataType(response),
		APICalls:     1,
	}, nil
}
