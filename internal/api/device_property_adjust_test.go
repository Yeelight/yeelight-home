package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDevicePropertyAdjustClientAdjustsSingleDeviceProperty(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/controll/device/2/device-1/w/properties/l/adjust" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewDevicePropertyAdjustClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), DevicePropertyAdjustRequest{
		DeviceID:     "device-1",
		PropertyName: "l",
		Value:        -1,
		Credentials: DevicePropertyAdjustCredentials{
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
	if gotBody["value"] != float64(-1) {
		t.Fatalf("body = %#v", gotBody)
	}
	if result.Region != "dev" || result.DeviceID != "device-1" || result.PropertyName != "l" || result.Command != "adjust" || result.Source != "device_property_adjust_endpoint" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevicePropertyAdjustClientReportsBusinessFailureWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":"40020","message":"invalid adjust","data":null}`))
	}))
	defer server.Close()

	client := NewDevicePropertyAdjustClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), DevicePropertyAdjustRequest{
		DeviceID:     "device-1",
		PropertyName: "l",
		Value:        -1,
		Credentials:  DevicePropertyAdjustCredentials{Authorization: "control-secret-token"},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "code=40020") || !strings.Contains(err.Error(), "message=invalid adjust") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "control-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}
