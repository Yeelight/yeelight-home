package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceSlotCreateDryRunPreviewsWithoutWriting(t *testing.T) {
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
			t.Fatalf("device.slot.create dry-run should not write")
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
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	if preview["intent"] != "device.slot.create" {
		t.Fatalf("preview=%#v", preview)
	}
	if len(calls) != 6 {
		t.Fatalf("calls=%#v", calls)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
	payloadPreview := preview["payloadPreview"].(map[string]any)
	semanticPreview := payloadPreview["semanticPreview"].(map[string]any)
	counts := semanticPreview["counts"].(map[string]any)
	if counts["devices"] != float64(2) {
		t.Fatalf("counts=%#v", counts)
	}
}

func TestInvokeDeviceSlotCreateExecutesDirectly(t *testing.T) {
	var syncBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/design/syncMetadata":
			if err := json.NewDecoder(request.Body).Decode(&syncBody); err != nil {
				t.Fatalf("decode sync body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceLocalIdToCloudSlotIds":{"1002":5001,"1003":5002}}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001},{"id":5002,"name":"黑色格栅灯2","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-slot-execute","locale":"zh-CN","utterance":"给客厅预留两个黑色格栅灯槽位","intent":"device.slot.create","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯","quantity":2,"category":"格栅灯","color":"黑色"}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	devices := syncBody["devices"].([]any)
	if len(devices) != 2 {
		t.Fatalf("sync devices=%#v", devices)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "lighting-design-import-execute" {
		t.Fatalf("response=%#v", response)
	}
}

func TestInvokeLightingDesignImportDryRunPreservesSelectedProduct(t *testing.T) {
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
			t.Fatalf("lighting.design.import dry-run should not write")
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
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	semanticPreview := preview["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	productResolution := semanticPreview["productResolution"].(map[string]any)
	if productResolution["matchedDeviceSlots"] != float64(4) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func TestInvokeLightingDesignImportDryRunResolvesNamedHomeOverProfileDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200171","houseName":"默认家庭"},{"houseId":"200191","houseName":"粒粒的美丽家庭"}]}}`))
		case strings.Contains(request.URL.Path, "/thing/manage/house/200191/area/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/room/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/device/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/group/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("lighting.design.import dry-run should not write")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-home-name","locale":"zh-CN","utterance":"帮我设计一下粒粒的美丽家庭这个家庭，客厅一个吸顶灯","intent":"lighting.design.import","homeRef":{"name":"粒粒的美丽家庭"},"parameters":{"rooms":[{"name":"客厅","items":[{"name":"吸顶灯","quantity":1}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	payloadPreview := preview["payloadPreview"].(map[string]any)
	if payloadPreview["houseId"] != "200191" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
}

func TestInvokeLightingDesignImportExecutesDirectly(t *testing.T) {
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

	input := `{"contractVersion":"1.0","requestId":"req-slot-execute","locale":"zh-CN","utterance":"创建客厅黑色格栅灯槽位","intent":"lighting.design.import","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯1","quantity":1}]}]}}`
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
		t.Fatalf("direct import used wrong payload: %#v", syncBody)
	}
}
