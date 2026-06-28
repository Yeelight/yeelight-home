package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/plan"
)

func TestInvokeDeviceSlotCreateCreatesPendingPlanWithoutWriting(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/room/r/info/"),
			strings.Contains(request.URL.Path, "/device/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("device.slot.create should not write before plan.commit")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-slot-plan","locale":"zh-CN","utterance":"先给客厅预留两个黑色格栅灯槽位","intent":"device.slot.create","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯","quantity":2,"category":"格栅灯","color":"黑色"}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response=%#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	if confirmation["intent"] != "device.slot.create" {
		t.Fatalf("confirmation=%#v", confirmation)
	}
	if len(calls) != 6 {
		t.Fatalf("calls=%#v", calls)
	}
	records, err := app.planStore.List()
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(records) != 1 || records[0].Intent != "device.slot.create" {
		t.Fatalf("records=%#v", records)
	}
	devices := records[0].Payload["devices"].([]any)
	if len(devices) != 2 {
		t.Fatalf("stored devices=%#v", devices)
	}
}

func TestInvokeLightingDesignImportPreservesSelectedProductInPendingPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/room/r/info/"),
			strings.Contains(request.URL.Path, "/device/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("lighting.design.import should not write before plan.commit")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-design-product-plan","locale":"zh-CN","utterance":"主卧预留四个36度射灯槽位","intent":"lighting.design.import","parameters":{"houseId":"200191","rooms":[{"name":"主卧","items":[{"name":"36°射灯","quantity":4,"materialCode":"1-000004714","notes":"AI按主卧重点照明选定S系列75开孔36度15w候选"}]}],"autoGroup":true}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response=%#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	preview := confirmation["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	productResolution := preview["productResolution"].(map[string]any)
	if productResolution["matchedDeviceSlots"] != float64(4) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
	records, err := app.planStore.List()
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	devices := records[0].Payload["devices"].([]any)
	for _, raw := range devices {
		attrs := raw.(map[string]any)["attrs"].(map[string]any)
		if attrs["materialCode"] != "1-000004714" || attrs["productMatchConfidence"] != "explicit" {
			t.Fatalf("device attrs=%#v", attrs)
		}
	}
}

func TestInvokePlanCommitImportsLightingDesignFromStoredPlan(t *testing.T) {
	var syncBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/design/syncMetadata":
			if err := json.NewDecoder(request.Body).Decode(&syncBody); err != nil {
				t.Fatalf("decode sync body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceLocalIdToCloudSlotIds":{"1002":5001}}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")
	planID := createLightingDesignImportPlanForTest(t, app, "200191", map[string]any{
		"houseId":  float64(200191),
		"clearAll": false,
		"gateways": []any{map[string]any{"localId": float64(1), "localName": "AI照明设计网关槽位", "pid": float64(17000001), "pcId": float64(2)}},
		"rooms":    []any{map[string]any{"localId": float64(1001), "localName": "客厅", "gatewayIds": []any{float64(1)}}},
		"devices": []any{
			map[string]any{"localId": float64(1002), "localName": "黑色格栅灯1", "gatewayDeviceId": float64(1), "roomId": float64(1001), "addr": float64(1002), "pid": float64(-1), "connectType": float64(-1)},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-slot-commit","locale":"zh-CN","utterance":"确认创建槽位","intent":"plan.commit","parameters":{"planId":"` + planID + `","rooms":[{"name":"恶意覆盖"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response=%#v", response)
	}
	if syncBody["rooms"].([]any)[0].(map[string]any)["localName"] != "客厅" {
		t.Fatalf("commit used caller payload instead of stored plan: %#v", syncBody)
	}
}

func createLightingDesignImportPlanForTest(t *testing.T, app *app, houseID string, payload map[string]any) string {
	t.Helper()
	record, err := plan.NewRecord("default", "dev", houseID, "device.slot.create", "req-slot-seed", "创建设备预留槽位", payload, []string{"test precondition"}, time.Now(), pendingPlanTTL)
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan: %v", err)
	}
	return record.ID
}
