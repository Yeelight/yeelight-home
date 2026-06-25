package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestAutomationStatusClientEnablesAndVerifiesByListStatus(t *testing.T) {
	var calls []string
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
			if automationListCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"回家开灯","houseId":"200171","status":0}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"回家开灯","houseId":"200171","status":1}]}}`))
		case "/apis/iot/v1/automations/w/enable/auto-1":
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s", request.Method)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewAutomationStatusClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), AutomationStatusRequest{
		Kind:         AutomationStatusEnable,
		HouseID:      "200171",
		AutomationID: "auto-1",
		Credentials:  AutomationStatusCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"POST /apis/iot/v1/automations/w/enable/auto-1",
		"GET /apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if !result.Verified || result.AutomationID != "auto-1" || result.Status != "1" || result.VerifiedBy != "automation.list.status" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAutomationStatusClientRejectsMissingStatusVerification(t *testing.T) {
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"回家开灯","houseId":"200171"}]}}`))
		case "/apis/iot/v1/automations/w/disable/auto-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewAutomationStatusClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	_, err := client.Run(context.Background(), AutomationStatusRequest{
		Kind:           AutomationStatusDisable,
		HouseID:        "200171",
		AutomationID:   "auto-1",
		VerifyAttempts: 1,
		Credentials:    AutomationStatusCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err == nil {
		t.Fatal("expected verification mismatch")
	}
}
