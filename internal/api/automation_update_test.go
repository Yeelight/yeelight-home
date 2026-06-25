package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestAutomationUpdateClientWritesAndVerifiesByList(t *testing.T) {
	var calls []string
	var updateBody map[string]any
	automationListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
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
			name := "回家开灯"
			if automationListCalls > 1 {
				name = "回家开灯更新"
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"` + name + `","houseId":"200171","status":1}]}}`))
		case "/apis/iot/v1/automations/auto-1/w/update":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&updateBody); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	payload := map[string]any{
		"automationId": "auto-1",
		"name":         "回家开灯更新",
		"houseId":      float64(200171),
		"startTime":    "00:00:00",
		"endTime":      "23:59:59",
		"repeatType":   2,
		"repeatValue":  "0x7f",
		"params":       `{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]}`,
		"actions":      []any{map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 0, "params": `{"set":{"p":true}}`}},
	}
	client := NewAutomationUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), AutomationUpdateRequest{
		HouseID:      "200171",
		AutomationID: "auto-1",
		Payload:      payload,
		Credentials:  AutomationUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if updateBody["automationId"] != nil || updateBody["id"] != "auto-1" || updateBody["name"] != "回家开灯更新" {
		t.Fatalf("updateBody = %#v", updateBody)
	}
	if !result.Verified || result.AutomationID != "auto-1" || result.Name != "回家开灯更新" || result.VerifiedBy != "automation.list" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "PUT /apis/iot/v1/automations/auto-1/w/update") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestAutomationUpdateClientRequiresListVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"未更新","houseId":"200171","status":1}]}}`))
		case "/apis/iot/v1/automations/auto-1/w/update":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewAutomationUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	_, err := client.Run(context.Background(), AutomationUpdateRequest{
		HouseID:        "200171",
		AutomationID:   "auto-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"name":       "目标名称",
			"houseId":    float64(200171),
			"startTime":  "00:00:00",
			"endTime":    "23:59:59",
			"repeatType": 2,
			"params":     `{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]}`,
			"actions":    []any{},
		},
		Credentials: AutomationUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err == nil {
		t.Fatal("expected verification mismatch")
	}
}
