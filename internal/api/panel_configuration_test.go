package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPanelConfigurationClientUpdatesPanelButtonsWithReadAfterWrite(t *testing.T) {
	var writeBody []any
	buttonReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"panel-1","name":"面板"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1":
			buttonReadCalls++
			if buttonReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"btn-1","keyValue":1,"resId":"old","resType":2}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"btn-1","keyValue":1,"resId":"scene-1","resType":6,"visible":1}]}`))
		case "/apis/iot/v1/panel/w/button/update/panel-1":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewPanelConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), PanelConfigurationRequest{
		Kind:           PanelButtonConfigure,
		HouseID:        "house-1",
		DeviceID:       "panel-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId": "panel-1",
			"buttons": []any{
				map[string]any{"id": "btn-1", "keyValue": 1, "resId": "scene-1", "resType": 6, "visible": 1},
			},
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-panel-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 1 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	item := writeBody[0].(map[string]any)
	if item["resId"] != "scene-1" || item["resType"] != float64(6) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "panel.button.configure_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPanelConfigurationClientUpdatesKnobWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	knobReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/multi-knob/knob-1/detail":
			knobReadCalls++
			if knobReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"id":"detail-1","index":1,"mode":"old"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"id":"detail-1","index":1,"mode":"scene","resId":"scene-1"}]}}`))
		case "/apis/iot/v1/multi-knob/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewPanelConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), PanelConfigurationRequest{
		Kind:           KnobConfigure,
		HouseID:        "house-1",
		DeviceID:       "knob-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId": "knob-1",
			"details": []any{
				map[string]any{"id": "detail-1", "index": 1, "mode": "scene", "resId": "scene-1"},
			},
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-knob-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["id"] != "knob-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "knob.configure_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPanelConfigurationClientUpdatesPanelButtonEventsWithReadAfterWrite(t *testing.T) {
	var singleBody map[string]any
	var batchBody map[string]any
	buttonReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"panel-1","name":"面板"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1":
			buttonReadCalls++
			if buttonReadCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"buttonEventId":101,"alias":"old","details":[{"resId":"old","typeId":2}]}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"buttonEventId":101,"alias":"单击","details":[{"resId":"scene-1","typeId":6}]},{"buttonEventId":102,"alias":"双击","details":[{"resId":"scene-2","typeId":6}]}]}`))
		case "/apis/iot/v1/panel/w/button/event/update":
			if err := json.NewDecoder(request.Body).Decode(&singleBody); err != nil {
				t.Fatalf("decode single body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v1/panel/w/button/event/update/batch":
			if err := json.NewDecoder(request.Body).Decode(&batchBody); err != nil {
				t.Fatalf("decode batch body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewPanelConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	singleResult, err := client.Run(context.Background(), PanelConfigurationRequest{
		Kind:           PanelButtonEventUpdate,
		HouseID:        "house-1",
		DeviceID:       "panel-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId": "panel-1",
			"buttonEvent": map[string]any{
				"buttonEventId": "101",
				"alias":         "单击",
				"details": []any{
					map[string]any{"resId": "scene-1", "typeId": 6},
				},
			},
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-panel-event-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("single Run error: %v", err)
	}
	if singleBody["buttonEventId"] != "101" || singleBody["alias"] != "单击" {
		t.Fatalf("singleBody = %#v", singleBody)
	}
	if !singleResult.Verified || singleResult.VerifiedBy != "panel.button_event.update_read_after_write" {
		t.Fatalf("singleResult = %#v", singleResult)
	}

	batchResult, err := client.Run(context.Background(), PanelConfigurationRequest{
		Kind:           PanelButtonEventBatchUpdate,
		HouseID:        "house-1",
		DeviceID:       "panel-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId": "panel-1",
			"buttonEvents": []any{
				map[string]any{"buttonEventId": "101", "alias": "单击", "details": []any{map[string]any{"resId": "scene-1", "typeId": 6}}},
				map[string]any{"buttonEventId": "102", "alias": "双击", "details": []any{map[string]any{"resId": "scene-2", "typeId": 6}}},
			},
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-panel-event-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("batch Run error: %v", err)
	}
	eventsJSON, ok := batchBody["buttonEvents"].(string)
	if !ok || !strings.Contains(eventsJSON, "scene-2") {
		t.Fatalf("batchBody = %#v", batchBody)
	}
	if !batchResult.Verified || batchResult.VerifiedBy != "panel.button_event.batch_update_read_after_write" {
		t.Fatalf("batchResult = %#v", batchResult)
	}
}

func TestPanelConfigurationClientResetsPanelButtonEventAndKnobWithReadAfterWrite(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"panel-1","name":"面板"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"buttonEventId":101,"details":[]}]}`))
		case "/apis/iot/v1/panel/w/button/event/101/reset":
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v1/multi-knob/knob-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"index":1,"mode":"scene"}]}}`))
		case "/apis/iot/v1/multi-knob/knob-1/1/reset":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewPanelConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	panelResult, err := client.Run(context.Background(), PanelConfigurationRequest{
		Kind:           PanelButtonEventReset,
		HouseID:        "house-1",
		DeviceID:       "panel-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId":      "panel-1",
			"buttonEventId": "101",
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-reset-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("panel reset err = %v", err)
	}
	knobResult, err := client.Run(context.Background(), PanelConfigurationRequest{
		Kind:           KnobReset,
		HouseID:        "house-1",
		DeviceID:       "knob-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"deviceId": "knob-1",
			"index":    1,
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-reset-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("knob reset err = %v", err)
	}
	if !panelResult.Verified || !knobResult.Verified {
		t.Fatalf("panelResult=%#v knobResult=%#v", panelResult, knobResult)
	}
	joined := strings.Join(gotCalls, "\n")
	for _, want := range []string{
		"POST /apis/iot/v1/panel/w/button/event/101/reset",
		"POST /apis/iot/v1/multi-knob/knob-1/1/reset",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in calls %#v", want, gotCalls)
		}
	}
}

func TestPanelConfigurationClientReportsVerificationMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/multi-knob/knob-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"id":"detail-1","index":1,"mode":"old"}]}}`))
		case "/apis/iot/v1/multi-knob/update":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewPanelConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), PanelConfigurationRequest{
		Kind:           KnobConfigure,
		HouseID:        "house-1",
		DeviceID:       "knob-1",
		VerifyAttempts: 1,
		VerifyInterval: time.Millisecond,
		Payload: map[string]any{
			"deviceId": "knob-1",
			"details": []any{
				map[string]any{"id": "detail-1", "index": 1, "mode": "scene"},
			},
		},
		Credentials: PanelConfigurationCredentials{Authorization: "Bearer token-knob-secret", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "write verification mismatch") {
		t.Fatalf("err = %v", err)
	}
}
