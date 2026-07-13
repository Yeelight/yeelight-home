package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDevicePropertySetClientSetsSingleDeviceProperty(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/controll/device/2/device-1/w/properties/p" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewDevicePropertySetClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), DevicePropertySetRequest{
		HouseID:      "house-1",
		DeviceID:     "device-1",
		PropertyName: "p",
		Value:        false,
		Command:      "set",
		Credentials: DevicePropertySetCredentials{
			Authorization: "control-secret-token",
			ClientID:      "client-control-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer control-secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-control-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if _, ok := gotBody["command"]; ok || gotBody["value"] != false {
		t.Fatalf("body = %#v", gotBody)
	}
	if result.Region != "dev" || result.HouseID != "house-1" || result.DeviceID != "device-1" || result.PropertyName != "p" || result.Source != "device_property_set_endpoint" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestNodePropertySetClientSetsRoomPropertyThroughOpenControl(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/open/control/house/house-1/control/1/room-1/w/properties/l" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewNodePropertySetClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), NodePropertySetRequest{
		HouseID:      "house-1",
		NodeType:     "room",
		NodeID:       "room-1",
		PropertyName: "l",
		Value:        70,
		Credentials:  NodePropertySetCredentials{Authorization: "control-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if _, ok := gotBody["command"]; ok || gotBody["value"] != float64(70) {
		t.Fatalf("body = %#v", gotBody)
	}
	if result.Region != "dev" || result.HouseID != "house-1" || result.NodeType != "room" || result.NodeTypeID != "1" || result.NodeID != "room-1" || result.PropertyName != "l" || result.Source != "open_control_node_property_set_endpoint" {
		t.Fatalf("result = %#v", result)
	}
}

func TestNodePropertySetClientMapsHomeToHouseNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/open/control/house/house-1/control/5/house-1/w/properties/p" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewNodePropertySetClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), NodePropertySetRequest{
		HouseID:      "house-1",
		NodeType:     "home",
		NodeID:       "house-1",
		PropertyName: "p",
		Value:        true,
		Credentials:  NodePropertySetCredentials{Authorization: "control-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.NodeType != "home" || result.NodeTypeID != "5" || result.NodeID != "house-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevicePropertySetClientReportsBusinessFailureWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":"40301","message":"no permission","data":null}`))
	}))
	defer server.Close()

	client := NewDevicePropertySetClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), DevicePropertySetRequest{
		HouseID:      "house-1",
		DeviceID:     "device-1",
		PropertyName: "p",
		Value:        true,
		Credentials:  DevicePropertySetCredentials{Authorization: "control-secret-token"},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "code=40301") || !strings.Contains(err.Error(), "message=no permission") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "control-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}
