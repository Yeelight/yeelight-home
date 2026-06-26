package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DeviceCapabilitiesCredentials struct {
	Authorization string
	ClientID      string
}

type DeviceCapabilitiesRequest struct {
	HouseID     string
	DeviceID    string
	Credentials DeviceCapabilitiesCredentials
}

type DeviceCapabilitiesResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	SchemaStatus     string           `json:"schemaStatus"`
	CapabilitySource string           `json:"capabilitySource"`
	Device           DeviceCapability `json:"device"`
}

type DeviceCapability struct {
	ID         string                `json:"id"`
	Name       string                `json:"name,omitempty"`
	PID        string                `json:"pid,omitempty"`
	PCID       string                `json:"pcId,omitempty"`
	CID        string                `json:"cid,omitempty"`
	Category   string                `json:"category,omitempty"`
	RoomID     string                `json:"roomId,omitempty"`
	NodeType   string                `json:"nodeType,omitempty"`
	Properties []PropertyCapability  `json:"properties,omitempty"`
	Components []ComponentCapability `json:"components,omitempty"`
	Events     []EventCapability     `json:"events,omitempty"`
	Actions    []ActionCapability    `json:"actions,omitempty"`
}

type ComponentCapability struct {
	ID         string               `json:"id,omitempty"`
	Index      string               `json:"index,omitempty"`
	Name       string               `json:"name,omitempty"`
	Type       string               `json:"type,omitempty"`
	Category   string               `json:"category,omitempty"`
	Properties []PropertyCapability `json:"properties,omitempty"`
	Events     []EventCapability    `json:"events,omitempty"`
	Actions    []ActionCapability   `json:"actions,omitempty"`
}

type PropertyCapability struct {
	ID          string          `json:"id"`
	Description string          `json:"description,omitempty"`
	Access      string          `json:"access,omitempty"`
	Format      string          `json:"format,omitempty"`
	Unit        string          `json:"unit,omitempty"`
	Type        string          `json:"type,omitempty"`
	Range       *PropertyRange  `json:"range,omitempty"`
	ValueList   []PropertyValue `json:"valueList,omitempty"`
	Operators   []string        `json:"operators,omitempty"`
}

type PropertyRange struct {
	Min  int `json:"min"`
	Max  int `json:"max"`
	Step int `json:"step"`
}

type PropertyValue struct {
	Code string `json:"code"`
	Desc string `json:"desc,omitempty"`
}

type EventCapability struct {
	ID     string               `json:"id,omitempty"`
	TypeID string               `json:"typeId,omitempty"`
	Name   string               `json:"name,omitempty"`
	Params []PropertyCapability `json:"params,omitempty"`
}

type ActionCapability struct {
	ID     string               `json:"id"`
	Params []PropertyCapability `json:"params,omitempty"`
}

type DeviceCapabilitiesClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewDeviceCapabilitiesClient(endpoint Endpoint, client *http.Client) DeviceCapabilitiesClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DeviceCapabilitiesClient{endpoint: endpoint, client: client}
}

func (client DeviceCapabilitiesClient) Run(ctx context.Context, request DeviceCapabilitiesRequest) (DeviceCapabilitiesResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	deviceID := strings.TrimSpace(request.DeviceID)
	if houseID == "" {
		return DeviceCapabilitiesResult{}, fmt.Errorf("house id is required")
	}
	if deviceID == "" {
		return DeviceCapabilitiesResult{}, fmt.Errorf("device id is required")
	}
	query := url.Values{}
	query.Set("crop", "false")
	path := fmt.Sprintf("/v2/thing/schema/house/%s/device/r/info/1/%d?%s", houseID, entityListPageSize, query.Encode())
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+path, nil, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return DeviceCapabilitiesResult{}, err
	}
	if !isBusinessOK(response) {
		return DeviceCapabilitiesResult{}, fmt.Errorf("device schema returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	device, ok := findDeviceCapability(deviceID, extractDeviceSchemaRows(response))
	if !ok {
		return DeviceCapabilitiesResult{}, fmt.Errorf("device schema did not include requested device")
	}
	return DeviceCapabilitiesResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		SchemaStatus:     "connected",
		CapabilitySource: "device_schema_endpoint",
		Device:           device,
	}, nil
}

func findDeviceCapability(deviceID string, rows []any) (DeviceCapability, bool) {
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		device := projectDeviceCapability(item)
		if device.ID == deviceID {
			return device, true
		}
	}
	return DeviceCapability{}, false
}

func projectDeviceCapability(item map[string]any) DeviceCapability {
	return DeviceCapability{
		ID:         firstAnyString(item, "id", "deviceId"),
		Name:       firstAnyString(item, "name"),
		PID:        firstAnyString(item, "pid"),
		PCID:       firstAnyString(item, "pcId"),
		CID:        firstAnyString(item, "cid"),
		Category:   firstAnyString(item, "category"),
		RoomID:     firstAnyString(item, "roomId"),
		NodeType:   firstAnyString(item, "nodeType"),
		Properties: projectProperties(item["properties"]),
		Components: projectComponents(item["subDevices"]),
		Events:     projectEvents(item["events"]),
		Actions:    projectActions(item["supportActions"]),
	}
}

func projectComponents(value any) []ComponentCapability {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	components := make([]ComponentCapability, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		components = append(components, ComponentCapability{
			ID:         firstAnyString(item, "cid"),
			Index:      firstAnyString(item, "index"),
			Name:       firstAnyString(item, "name"),
			Type:       firstAnyString(item, "type"),
			Category:   firstAnyString(item, "category"),
			Properties: projectProperties(item["properties"]),
			Events:     projectEvents(item["events"]),
			Actions:    projectActions(item["supportActions"]),
		})
	}
	return components
}

func projectProperties(value any) []PropertyCapability {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	properties := make([]PropertyCapability, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		property := PropertyCapability{
			ID:          firstAnyString(item, "propId", "id"),
			Description: firstAnyString(item, "desc", "description"),
			Access:      firstAnyString(item, "access"),
			Format:      firstAnyString(item, "format"),
			Unit:        firstAnyString(item, "unit"),
			Type:        firstAnyString(item, "type"),
			Range:       projectPropertyRange(item["valueRange"]),
			ValueList:   projectPropertyValues(item["valueList"]),
			Operators:   projectStringList(item["operators"]),
		}
		if property.ID != "" {
			properties = append(properties, property)
		}
	}
	return properties
}

func projectEvents(value any) []EventCapability {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	events := make([]EventCapability, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		events = append(events, EventCapability{
			ID:     firstAnyString(item, "eventId"),
			TypeID: firstAnyString(item, "eventTypeId"),
			Name:   firstAnyString(item, "name"),
			Params: projectProperties(item["params"]),
		})
	}
	return events
}

func projectActions(value any) []ActionCapability {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	actions := make([]ActionCapability, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		action := ActionCapability{
			ID:     firstAnyString(item, "actionName", "id", "name"),
			Params: projectProperties(item["params"]),
		}
		if action.ID != "" {
			actions = append(actions, action)
		}
	}
	return actions
}

func projectPropertyRange(value any) *PropertyRange {
	item, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &PropertyRange{
		Min:  intFromAny(item["min"]),
		Max:  intFromAny(item["max"]),
		Step: intFromAny(item["step"]),
	}
}

func projectPropertyValues(value any) []PropertyValue {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]PropertyValue, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if code := firstAnyString(item, "code"); code != "" {
			values = append(values, PropertyValue{Code: code, Desc: firstAnyString(item, "desc")})
		}
	}
	return values
}

func projectStringList(value any) []string {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if value, ok := row.(string); ok && strings.TrimSpace(value) != "" {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
}

func extractDeviceSchemaRows(response map[string]any) []any {
	data, ok := response["data"]
	if !ok {
		return []any{}
	}
	switch typed := data.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range []string{"devices", "rows", "list"} {
			if rows, ok := typed[key].([]any); ok {
				return rows
			}
		}
	}
	return []any{}
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		result, _ := typed.Int64()
		return int(result)
	case string:
		result, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return result
		}
	}
	return 0
}

func (device DeviceCapability) RawDebugString() string {
	data, _ := json.Marshal(device)
	return string(data)
}
