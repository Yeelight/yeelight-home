package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeviceDiagnosticsReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/device/r/sensors":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"sensor-1","name":"门磁","mac":"AA:BB:CC:DD","localToken":"not-allowed"}]}`))
		case "/apis/iot/v1/sensor/r/events":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"event-1","name":"开门事件","accessToken":"not-allowed"}]}`))
		case "/apis/iot/v1/energy/devices/device-1/r/summary":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"daySum":1.2,"monthSum":8.5}}`))
		case "/apis/iot/v1/weather/r/device-1/default/queryWeather":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"temperature":26,"humidity":60}]}`))
		case "/apis/iot/v1/meshgroup/group-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"group-1","name":"灯组","devices":[{"id":"device-1"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	credentials := MetadataReadonlyCredentials{Authorization: "Bearer token-diagnostics-secret", ClientID: "client-1"}
	baseRequest := MetadataReadonlyRequest{HouseID: "house-1", DeviceID: "device-1", Parameters: map[string]any{"groupId": "group-1"}, Credentials: credentials}

	results := []MetadataReadonlyResult{}
	for _, run := range []func() (MetadataReadonlyResult, error){
		func() (MetadataReadonlyResult, error) { return client.RunSensorList(context.Background(), baseRequest) },
		func() (MetadataReadonlyResult, error) {
			return client.RunSensorEventList(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunDeviceEnergySummary(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunDeviceWeatherGet(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunMeshgroupDetailGet(context.Background(), baseRequest)
		},
	} {
		result, err := run()
		if err != nil {
			t.Fatalf("run err = %v", err)
		}
		results = append(results, result)
	}

	if len(gotCalls) != 5 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range results {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		for _, forbidden := range []string{"not-allowed", "AA:BB:CC:DD", "token-diagnostics-secret"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
}

func TestDeviceEnergyRequiresDeviceContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunDeviceEnergySummary(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-diagnostics-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("energy err = %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "device_context_missing" {
		t.Fatalf("result = %#v", result)
	}
}
