package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStateQueryClientReadsAllProperties(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost {
			http.NotFound(writer, request)
			return
		}
		switch request.URL.Path {
		case "/apis/iot/v1/controll/device/device-1/r/properties/power":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/brightness":
			_, _ = writer.Write([]byte(`{"success":true,"data":72}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/pf":
			_, _ = writer.Write([]byte(`{"success":false,"code":"601","message":"invalid property","data":null}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewStateQueryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), StateQueryRequest{
		DeviceID: "device-1",
		PropertySet: []string{
			"power",
			"brightness",
			"localToken",
			"pf",
			"power",
			"",
		},
		Credentials: StateQueryCredentials{
			Authorization: "state-secret-token",
			ClientID:      "client-state-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"POST /apis/iot/v1/controll/device/device-1/r/properties/power",
		"POST /apis/iot/v1/controll/device/device-1/r/properties/brightness",
		"POST /apis/iot/v1/controll/device/device-1/r/properties/pf",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("calls = %#v", gotCalls)
	}
	if gotAuthorization != "Bearer state-secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-state-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if result.QueryScope != "all_properties" || result.RawShape != "object:2" || result.Source != "device_properties_endpoint" {
		t.Fatalf("result = %#v", result)
	}
	if result.Properties["power"] != true || result.Properties["brightness"] != float64(72) {
		t.Fatalf("properties = %#v", result.Properties)
	}
	if result.APICalls != 3 || len(result.Skipped) != 2 || !strings.Contains(strings.Join(result.Skipped, "\n"), "localToken:sensitive_property_not_readable") || !strings.Contains(strings.Join(result.Skipped, "\n"), "pf:601") {
		t.Fatalf("result = %#v", result)
	}
}

func TestStateQueryClientFiltersSensitiveAllProperties(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/controll/device/device-1/r/properties" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"properties":{"p":true,"localToken":"not-allowed","deviceKey":"secret-key","l":72}}}`))
	}))
	defer server.Close()

	client := NewStateQueryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), StateQueryRequest{
		DeviceID:    "device-1",
		Credentials: StateQueryCredentials{Authorization: "state-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Properties["p"] != true || result.Properties["l"] != float64(72) {
		t.Fatalf("properties = %#v", result.Properties)
	}
	for _, forbidden := range []string{"localToken", "deviceKey"} {
		if _, ok := result.Properties[forbidden]; ok {
			t.Fatalf("sensitive property leaked: %#v", result.Properties)
		}
	}
}

func TestStateQueryClientReadsSingleProperty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/controll/device/device-1/r/properties/power" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
	}))
	defer server.Close()

	client := NewStateQueryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), StateQueryRequest{
		DeviceID:     "device-1",
		PropertyName: "power",
		Credentials:  StateQueryCredentials{Authorization: "state-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.QueryScope != "single_property" || result.PropertyName != "power" || result.Value != true || result.RawShape != "bool" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Properties) != 0 {
		t.Fatalf("properties = %#v", result.Properties)
	}
}

func TestStateQueryClientRejectsSensitiveSingleProperty(t *testing.T) {
	client := NewStateQueryClient(Endpoint{Region: "dev", BaseURL: "http://127.0.0.1"}, nil)
	_, err := client.Run(context.Background(), StateQueryRequest{
		DeviceID:     "device-1",
		PropertyName: "localToken",
		Credentials:  StateQueryCredentials{Authorization: "state-secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "refused sensitive property") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "state-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func TestStateQueryClientReportsBusinessFailureWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":"40301","message":"no permission","data":null}`))
	}))
	defer server.Close()

	client := NewStateQueryClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), StateQueryRequest{
		DeviceID:    "device-1",
		Credentials: StateQueryCredentials{Authorization: "state-secret-token"},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "code=40301") || !strings.Contains(err.Error(), "message=no permission") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "state-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}
