package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeviceCapabilitiesClientProjectsInstanceSchema(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		gotQuery = request.URL.RawQuery
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodGet || request.URL.Path != "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{
			"success": true,
			"data": {
				"total": 1,
				"devices": [{
					"id": "device-1",
					"name": "主灯",
					"pid": 17000008,
					"pcId": 35,
					"cid": 1001,
					"category": "light",
					"roomId": "room-1",
					"nodeType": 2,
					"properties": [{"propId": "p", "desc": "开关", "access": 6, "format": "bool", "type": 1}],
					"subDevices": [{
						"cid": 2001,
						"index": 1,
						"name": "主灯组件",
						"type": 2,
						"category": "light",
						"properties": [{
							"propId": "brightness",
							"desc": "亮度",
							"access": 6,
							"format": "uint8",
							"type": 1,
							"valueRange": {"min": 1, "max": 100, "step": 1}
						}],
						"events": [{"eventId": 11, "eventTypeId": 2, "name": "状态变化"}],
						"supportActions": [{"actionName": "toggle"}]
					}],
					"events": [{"eventId": 10, "eventTypeId": 1, "name": "上线"}],
					"supportActions": [{"actionName": "set_power"}],
					"rawSecret": "must-not-leak"
				}]
			}
		}`))
	}))
	defer server.Close()

	client := NewDeviceCapabilitiesClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), DeviceCapabilitiesRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Credentials: DeviceCapabilitiesCredentials{
			Authorization: "secret-token",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if !strings.Contains(gotQuery, "crop=false") {
		t.Fatalf("query = %q", gotQuery)
	}
	if result.SchemaStatus != "connected" || result.CapabilitySource != "device_schema_endpoint" {
		t.Fatalf("result = %#v", result)
	}
	if result.Device.ID != "device-1" || result.Device.ProductID != "17000008" || result.Device.Category != "light" {
		t.Fatalf("device = %#v", result.Device)
	}
	if len(result.Device.Components) != 1 || result.Device.Components[0].ID != "2001" {
		t.Fatalf("components = %#v", result.Device.Components)
	}
	if len(result.Device.Properties) != 1 || result.Device.Properties[0].ID != "power" {
		t.Fatalf("device properties must expose standard property names: %#v", result.Device.Properties)
	}
	if len(result.Device.Components[0].Properties) != 1 || result.Device.Components[0].Properties[0].ID != "brightness" {
		t.Fatalf("component properties = %#v", result.Device.Components[0].Properties)
	}
	if result.Device.Components[0].Properties[0].Range == nil || result.Device.Components[0].Properties[0].Range.Max != 100 {
		t.Fatalf("property range = %#v", result.Device.Components[0].Properties[0].Range)
	}
	if len(result.Device.Actions) != 1 || result.Device.Actions[0].ID != "set_power" {
		t.Fatalf("actions = %#v", result.Device.Actions)
	}
	if strings.Contains(result.Device.RawDebugString(), "must-not-leak") {
		t.Fatalf("raw schema leaked: %#v", result.Device)
	}
}

func TestDeviceCapabilitiesClientReportsMissingDeviceWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[]}}`))
	}))
	defer server.Close()

	client := NewDeviceCapabilitiesClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), DeviceCapabilitiesRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: DeviceCapabilitiesCredentials{Authorization: "secret-token"},
	})
	if err == nil {
		t.Fatal("expected missing device error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}
