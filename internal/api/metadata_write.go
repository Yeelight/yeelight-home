package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type MetadataWriteCredentials struct {
	Authorization string
	ClientID      string
}

type HomePropertySetRequest struct {
	HouseID     string
	Properties  map[string]any
	Credentials MetadataWriteCredentials
}

type PanelClickRequest struct {
	ResID       string
	Payload     map[string]any
	Credentials MetadataWriteCredentials
}

type SensorEventWriteRequest struct {
	Operation   string
	EventID     string
	Payload     map[string]any
	Credentials MetadataWriteCredentials
}

type MetadataWriteResult struct {
	Region     string         `json:"region"`
	HouseID    string         `json:"houseId,omitempty"`
	ID         string         `json:"id,omitempty"`
	Capability string         `json:"capability"`
	Operation  string         `json:"operation,omitempty"`
	Source     string         `json:"source"`
	RawShape   string         `json:"rawShape"`
	Result     any            `json:"result,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	APICalls   int            `json:"apiCalls"`
}

type MetadataWriteClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewMetadataWriteClient(endpoint Endpoint, client *http.Client) MetadataWriteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return MetadataWriteClient{endpoint: endpoint, client: client}
}

func (client MetadataWriteClient) RunHomePropertySet(ctx context.Context, request HomePropertySetRequest) (MetadataWriteResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return MetadataWriteResult{}, fmt.Errorf("house id is required")
	}
	properties := sanitizeWritePayload(request.Properties)
	if len(properties) == 0 {
		return MetadataWriteResult{}, fmt.Errorf("at least one house property is required")
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/house/w/"+pathSegment(houseID)+"/properties", properties, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	})
	if err != nil {
		return MetadataWriteResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataWriteResult{}, fmt.Errorf("home property set returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return MetadataWriteResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.property.set",
		Operation:  "set",
		Source:     "home_properties_endpoint",
		RawShape:   responseDataType(response),
		Result:     sanitizeCloudData(response["data"]),
		Payload:    properties,
		APICalls:   1,
	}, nil
}

func (client MetadataWriteClient) RunPanelClick(ctx context.Context, request PanelClickRequest) (MetadataWriteResult, error) {
	resID := strings.TrimSpace(request.ResID)
	if resID == "" {
		return MetadataWriteResult{}, fmt.Errorf("panel resource id is required")
	}
	payload := sanitizeWritePayload(request.Payload)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/control/panel/"+pathSegment(resID), payload, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return MetadataWriteResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataWriteResult{}, fmt.Errorf("panel click returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return MetadataWriteResult{
		Region:     client.endpoint.Region,
		ID:         resID,
		Capability: "panel.click",
		Operation:  "click",
		Source:     "panel_click_endpoint",
		RawShape:   responseDataType(response),
		Result:     sanitizeCloudData(response["data"]),
		Payload:    payload,
		APICalls:   1,
	}, nil
}

func (client MetadataWriteClient) RunSensorEventWrite(ctx context.Context, request SensorEventWriteRequest) (MetadataWriteResult, error) {
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
	if operation == "" {
		operation = "create"
	}
	path, method, body, eventID, err := sensorEventWriteCall(operation, request.EventID, request.Payload)
	if err != nil {
		return MetadataWriteResult{}, err
	}
	response, err := callJSON(ctx, client.client, method, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return MetadataWriteResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataWriteResult{}, fmt.Errorf("sensor event %s returned non-success business response: code=%s message=%s dataType=%s", operation, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return MetadataWriteResult{
		Region:     client.endpoint.Region,
		ID:         eventID,
		Capability: "sensor.event.write",
		Operation:  operation,
		Source:     "sensor_event_write_endpoint",
		RawShape:   responseDataType(response),
		Result:     sanitizeCloudData(response["data"]),
		Payload:    body,
		APICalls:   1,
	}, nil
}

func sensorEventWriteCall(operation string, eventID string, payload map[string]any) (string, string, map[string]any, string, error) {
	eventID = strings.TrimSpace(eventID)
	body := sanitizeWritePayload(payload)
	switch operation {
	case "create", "insert":
		return "/v1/sensor/w/insert", http.MethodPost, body, eventID, nil
	case "test":
		return "/v1/sensor/w/test", http.MethodPost, body, eventID, nil
	case "update":
		if eventID == "" {
			return "", "", nil, "", fmt.Errorf("sensor event id is required")
		}
		if body[semantic.FieldID] == nil {
			body[semantic.FieldID] = requestNumberOrStringForAPI(eventID)
		}
		return "/v1/sensor/" + pathSegment(eventID) + "/w/update", http.MethodPost, body, eventID, nil
	case "delete", "remove":
		if eventID == "" {
			return "", "", nil, "", fmt.Errorf("sensor event id is required")
		}
		return "/v1/sensor/" + pathSegment(eventID) + "/w/delete", http.MethodPost, nil, eventID, nil
	default:
		return "", "", nil, "", fmt.Errorf("unsupported sensor event operation %q", operation)
	}
}

func sanitizeWritePayload(payload map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range payload {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" || isSensitiveCloudField(trimmed) {
			continue
		}
		result[trimmed] = value
	}
	return result
}
