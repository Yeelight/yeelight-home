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

func TestDevSeedClientReusesExistingAutomationByName(t *testing.T) {
	var calls []string
	server := newSeedAutomationServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"automation-existing","name":"Codex Dev Test Automation"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		HouseID:       "200171",
		Name:          "Codex Dev Test Automation",
		DeviceID:      "50018330",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureAutomation error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"POST /apis/iot/v1/automations/r/list",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Created || !result.Verified || result.AutomationID != "automation-existing" || result.VerifiedBy != "automation_list" || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientCreatesAndVerifiesMissingAutomation(t *testing.T) {
	var calls []string
	var createBody map[string]any
	automationListCalls := 0
	server := newSeedAutomationServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			automationListCalls++
			if automationListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":9001,"name":"Codex Dev Test Automation"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/automation/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":9001}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		HouseID:        "200171",
		Name:           "Codex Dev Test Automation",
		DeviceID:       "50018330",
		DeviceName:     "light-dali开关灯-17000002-01",
		PropertyName:   "p",
		PropertyValue:  false,
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureAutomation error: %v", err)
	}
	if createBody["houseId"] != float64(200171) || createBody["name"] != "Codex Dev Test Automation" || createBody["status"] != float64(0) {
		t.Fatalf("createBody = %#v", createBody)
	}
	if createBody["startTime"] != "00:00:00" || createBody["endTime"] != "23:59:59" || createBody["repeatType"] != float64(2) || createBody["repeatValue"] != "0x7f" {
		t.Fatalf("createBody = %#v", createBody)
	}
	if _, ok := createBody["params"].(string); !ok {
		t.Fatalf("params = %#v", createBody["params"])
	}
	actions, ok := createBody["actions"].([]any)
	if !ok || len(actions) != 1 {
		t.Fatalf("actions = %#v", createBody["actions"])
	}
	action, ok := actions[0].(map[string]any)
	if !ok || action["typeId"] != float64(2) || action["resId"] != float64(50018330) || action["params"] != `{"set":{"p":false}}` {
		t.Fatalf("action = %#v", actions[0])
	}
	if !result.Created || !result.Verified || result.AutomationID != "9001" || result.VerifiedBy != "automation_list" || result.Status != 0 {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "PUT /apis/iot/v2/thing/manage/house/200171/automation/w/create") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientRequiresAutomationInputsAndWriteGate(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		Name:        "Codex Dev Test Automation",
		Credentials: DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "--allow-write-dev") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		Name:          "Codex Dev Test Automation",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "house id is required") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		HouseID:       "200171",
		Name:          "Codex Dev Test Automation",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "device id is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestDevSeedClientRejectsAutomationSeedForNonDevEndpoint(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil)
	_, err := client.EnsureAutomation(context.Background(), DevSeedAutomationRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Automation",
		DeviceID:      "50018330",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "only allowed for dev") {
		t.Fatalf("err = %v", err)
	}
}

func newSeedAutomationServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		handler(writer, request)
	}))
}
