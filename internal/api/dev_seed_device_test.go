package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestDevSeedClientCreatesAndVerifiesVirtualBoundDevice(t *testing.T) {
	var calls []string
	var createBody map[string]any
	deviceListCalls := 0
	server := newSeedDeviceServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-created","name":"Codex Dev Test Device"}]}}`))
		case "/apis/iot/v1/device/w/insert":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"device-created"}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureDevice(context.Background(), DevSeedDeviceRequest{
		HouseID:             "house-1",
		Name:                "Codex Dev Test Device",
		CapabilityProductID: 1,
		DeviceType:          1,
		ConnectType:         0,
		Bound:               true,
		AllowWriteDev:       true,
		VerifyAttempts:      1,
		Credentials:         DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureDevice error: %v", err)
	}
	if createBody["name"] != "Codex Dev Test Device" || createBody["houseId"] != "house-1" || createBody["isBind"] != float64(1) || createBody["isVirtual"] != float64(1) {
		t.Fatalf("createBody = %#v", createBody)
	}
	if !result.Created || !result.Verified || !result.Bound || result.DeviceID != "device-created" || result.VerifiedBy != "entity_list" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "POST /apis/iot/v1/device/w/insert") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientReusesExistingDeviceByName(t *testing.T) {
	var calls []string
	server := newSeedDeviceServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-existing","name":"Codex Dev Test Device"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureDevice(context.Background(), DevSeedDeviceRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Device",
		Bound:         true,
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureDevice error: %v", err)
	}
	if result.Created || !result.Verified || result.DeviceID != "device-existing" {
		t.Fatalf("result = %#v", result)
	}
	if slices.Contains(calls, "POST /apis/iot/v1/device/w/insert") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientRequiresDeviceHouseIDAndWriteGate(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.EnsureDevice(context.Background(), DevSeedDeviceRequest{
		Name:        "Codex Dev Test Device",
		Credentials: DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "--allow-write-dev") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureDevice(context.Background(), DevSeedDeviceRequest{
		Name:          "Codex Dev Test Device",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "house id is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestDevSeedClientRejectsDeviceSeedForNonDevEndpoint(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil)
	_, err := client.EnsureDevice(context.Background(), DevSeedDeviceRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Device",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "only allowed for dev") {
		t.Fatalf("err = %v", err)
	}
}

func newSeedDeviceServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		handler(writer, request)
	}))
}
