package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceDetailGetReturnsRedactedProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/50018330/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"50018330","name":"主灯","mac":"AA:BB:CC:DD","localToken":"not-allowed","roomId":"401391"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-detail-secret", "client-detail-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-detail","locale":"zh-CN","utterance":"查看主灯详情","intent":"device.detail.get","targets":[{"entityType":"device","id":"50018330"}],"parameters":{"houseId":"200171"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, forbidden := range []string{"token-detail-secret", "not-allowed", "AA:BB:CC:DD"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, stdout.String())
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "device-detail-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeFavoriteAddExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
		case "/apis/iot/v1/favourite/w/insert":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"fav-1"}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-write-secret", "client-fav-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add","locale":"zh-CN","utterance":"把主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","entityType":"device","resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["resId"] != float64(50018330) || writeBody["typeId"] != float64(2) || writeBody["houseId"] != float64(200171) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeFavoriteDeleteExecutesDirectly(t *testing.T) {
	favoriteListCalls := 0
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/favourite/fav-1/w/delete":
			deleteCalls++
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete","locale":"zh-CN","utterance":"删除主灯首页收藏","intent":"favorite.delete","parameters":{"houseId":"200171","favoriteId":"fav-1","typeId":2,"resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d", deleteCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeHomeSortConfigureDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401391/device":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"50018330","rank":2}]}`))
		case "/apis/iot/v1/sort/r/getSort":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"typeId":2,"resId":50018330,"rank":2}]}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-preview-secret", "client-sort-preview-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-sort-preview","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomId":"401391","items":[{"entityType":"device","resId":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/") {
			t.Fatalf("dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeHomeSortConfigureExecutesDirectly(t *testing.T) {
	var writeBody []any
	sortReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401391/device":
			sortReadCalls++
			if sortReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"50018330","rank":1}]}`))
		case "/apis/iot/v1/sort/200171/w/1/401391/add":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode sort body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-write-secret", "client-sort-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-sort-execute","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomId":"401391","items":[{"entityType":"device","resId":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 1 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}
