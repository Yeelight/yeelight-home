package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataReadonlyReadPathBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":600,"message":"参数格式错误"}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceWeatherGet(context.Background(), MetadataReadonlyRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Parameters: map[string]any{
			"queryType": "default",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceWeatherGet error = %v", err)
	}
	if !result.Partial || result.Capability != "device.weather.get" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if result.Data != nil {
		t.Fatalf("partial business result should not expose raw data: %#v", result.Data)
	}
}
