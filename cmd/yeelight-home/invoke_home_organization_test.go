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

func TestInvokeFavoriteAddCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-write-secret", "client-fav-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-plan","locale":"zh-CN","utterance":"把主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","typeId":2,"resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/favourite/w/insert") {
			t.Fatalf("favorite.add should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "favorite.add" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	if typeID, ok := requestInt(record.Payload["typeId"]); !ok || typeID != 2 {
		t.Fatalf("record payload = %#v", record.Payload)
	}
}

func TestInvokeFavoriteAddAcceptsSemanticEntityType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-semantic-secret", "client-fav-semantic-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-semantic-plan","locale":"zh-CN","utterance":"把主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","entityType":"device","resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	record, ok, err := app.planStore.Load(response["confirmation"].(map[string]any)["planId"].(string))
	typeID, typeOK := requestInt(record.Payload["typeId"])
	if err != nil || !ok || !typeOK || typeID != 2 {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeFavoriteBatchAddCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-batch-secret", "client-fav-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-batch-plan","locale":"zh-CN","utterance":"把主灯和筒灯加入收藏","intent":"favorite.batch_add","parameters":{"houseId":"200171","items":[{"typeId":2,"resId":"50018330","rank":1},{"typeId":2,"resId":"50018430","rank":2}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/favourite/w/insert") {
			t.Fatalf("favorite.batch_add should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	preview := confirmation["payloadPreview"].(map[string]any)
	items := preview["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("preview = %#v", preview)
	}
	planID := confirmation["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "favorite.batch_add" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeHomeSortAcceptsSemanticSortAndEntityType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/apis/iot/v1/sort/r/getSort" {
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sort":[]}}`))
			return
		}
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-semantic-secret", "client-sort-semantic-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-sort-semantic-plan","locale":"zh-CN","utterance":"把灯光区主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomId":"401391","items":[{"entityType":"device","resId":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	record, ok, err := app.planStore.Load(response["confirmation"].(map[string]any)["planId"].(string))
	if err != nil || !ok {
		t.Fatalf("record ok=%v err=%v", ok, err)
	}
	if sortType, ok := requestInt(record.Payload["type"]); !ok || sortType != 1 {
		t.Fatalf("payload = %#v", record.Payload)
	}
	if target := requestString(record.Payload["target"]); target != "401391" {
		t.Fatalf("payload = %#v", record.Payload)
	}
	if roomID := requestString(record.Payload["roomId"]); roomID != "401391" {
		t.Fatalf("payload = %#v", record.Payload)
	}
	items := record.Payload["items"].([]any)
	first := items[0].(map[string]any)
	typeID, typeOK := requestInt(first["typeId"])
	resID := requestString(first["resId"])
	if !typeOK || typeID != 2 || resID != "50018330" {
		t.Fatalf("payload = %#v", record.Payload)
	}
}

func TestInvokeFavoriteDeleteCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete-plan","locale":"zh-CN","utterance":"删除主灯首页收藏","intent":"favorite.delete","parameters":{"houseId":"200171","typeId":2,"resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/delete") {
			t.Fatalf("favorite.delete should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	preview := confirmation["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if preview["deleteTarget"].(map[string]any)["favoriteId"] != "fav-1" {
		t.Fatalf("preview = %#v", preview)
	}
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok || record.Intent != "favorite.delete" || record.Payload["favoriteId"] != "fav-1" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeFavoriteDeleteAcceptsNestedDeviceResponseWithoutFavoriteID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"50018330","deviceId":"50018330","houseId":"200171","rank":1}],"meshgroups":[],"userscenes":[]}}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete-nested-plan","locale":"zh-CN","utterance":"把主灯从首页收藏里移除掉","intent":"favorite.delete","parameters":{"houseId":"200171","typeId":2,"resId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	preview := confirmation["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	deleteTarget := preview["deleteTarget"].(map[string]any)
	if deleteTarget["typeId"] != float64(2) || deleteTarget["resId"] != "50018330" {
		t.Fatalf("preview = %#v", preview)
	}
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok || record.Intent != "favorite.delete" || record.Payload["favoriteId"] != nil {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeFavoriteBatchDeleteCreatesSinglePendingPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1},{"id":"fav-2","houseId":200171,"typeId":6,"resId":"700001","rank":2}]}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-batch-delete-plan","locale":"zh-CN","utterance":"删除主灯和晚安情景的首页收藏","intent":"favorite.batch_delete","parameters":{"houseId":"200171","items":[{"typeId":2,"resId":"50018330","rank":1},{"typeId":6,"resId":"700001","rank":2}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["confirmation"].(map[string]any)["payloadPreview"].(map[string]any)
	targets := preview["semanticPreview"].(map[string]any)["deleteTargets"].([]any)
	if len(targets) != 2 {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokePlanCommitAddsFavoriteFromStoredPlan(t *testing.T) {
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
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-write-secret", "client-fav-write-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "favorite.add", map[string]any{
		"houseId": float64(200171),
		"typeId":  2,
		"resId":   float64(50018330),
		"rank":    1,
	})

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-commit","locale":"zh-CN","utterance":"确认收藏","intent":"plan.commit","parameters":{"planId":"` + planID + `","resId":"ignored"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["resId"] != float64(50018330) || writeBody["houseId"] != float64(200171) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitDeletesFavoriteFromStoredPlan(t *testing.T) {
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
			if strings.Contains(request.URL.Path, "ignored") {
				t.Fatalf("commit request payload leaked into API path: %s", request.URL.Path)
			}
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "favorite.delete", map[string]any{
		"houseId":    float64(200171),
		"favoriteId": "fav-1",
		"typeId":     2,
		"resId":      float64(50018330),
		"rank":       1,
	})

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete-commit","locale":"zh-CN","utterance":"确认删除收藏","intent":"plan.commit","parameters":{"planId":"` + planID + `","favoriteId":"ignored"}}`
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
	if response["status"] != "success" || response["traceId"] != "home-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitBatchDeletesFavoritesFromStoredPlan(t *testing.T) {
	var writeBody []any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1},{"id":"fav-2","houseId":200171,"typeId":6,"resId":"700001","rank":2}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/favourite/w/batchdelete":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode batch delete body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-delete-secret", "client-fav-delete-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "favorite.batch_delete", map[string]any{
		"houseId": float64(200171),
		"items": []any{
			map[string]any{"houseId": float64(200171), "favoriteId": "fav-1", "typeId": 2, "resId": float64(50018330), "rank": 1},
			map[string]any{"houseId": float64(200171), "favoriteId": "fav-2", "typeId": 6, "resId": "700001", "rank": 2},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-fav-batch-delete-commit","locale":"zh-CN","utterance":"确认批量删除收藏","intent":"plan.commit","parameters":{"planId":"` + planID + `","items":[{"favoriteId":"ignored"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 2 || writeBody[0].(map[string]any)["favoriteId"] != "fav-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	result := response["result"].(map[string]any)
	if response["status"] != "success" || result["itemCount"] != float64(2) {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitBatchAddsFavoritesFromStoredPlan(t *testing.T) {
	var writeBody []any
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
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1},{"id":"fav-2","houseId":200171,"typeId":6,"resId":"700001","rank":2}]}`))
		case "/apis/iot/v1/favourite/w/batchinsert":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite batch insert body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-batch-secret", "client-fav-batch-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "favorite.batch_add", map[string]any{
		"houseId": float64(200171),
		"items": []any{
			map[string]any{"houseId": float64(200171), "typeId": 2, "resId": float64(50018330), "rank": 1},
			map[string]any{"houseId": float64(200171), "typeId": 6, "resId": "700001", "rank": 2},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-fav-batch-commit","locale":"zh-CN","utterance":"确认批量收藏","intent":"plan.commit","parameters":{"planId":"` + planID + `","items":[{"typeId":9,"resId":"ignored"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 2 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	result := response["result"].(map[string]any)
	if response["status"] != "success" || result["itemCount"] != float64(2) {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitUpdatesFavoriteFromStoredPlan(t *testing.T) {
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":3}]}`))
		case "/apis/iot/v1/favourite/fav-1/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-write-secret", "client-fav-write-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "favorite.update", map[string]any{
		"houseId":    float64(200171),
		"favoriteId": "fav-1",
		"typeId":     2,
		"resId":      float64(50018330),
		"rank":       3,
	})

	input := `{"contractVersion":"1.0","requestId":"req-fav-update-commit","locale":"zh-CN","utterance":"确认更新收藏","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, exists := writeBody["favoriteId"]; exists || writeBody["rank"] != float64(3) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeFavoriteUpdateAcceptsResourceIdentityWithoutFavoriteID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-update-secret", "client-fav-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-update-resource-plan","locale":"zh-CN","utterance":"把主灯首页收藏排到第二位","intent":"favorite.update","parameters":{"houseId":"200171","typeId":2,"resId":"50018330","rank":2}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	record, ok, err := app.planStore.Load(response["confirmation"].(map[string]any)["planId"].(string))
	if err != nil || !ok || record.Intent != "favorite.update" || record.Payload["favoriteId"] != nil || record.Payload["rank"] != float64(2) {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeHomeSortConfigurePlanIncludesSemanticPreview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/getSort":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"typeId":2,"resId":50018330,"rank":2}]}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-preview-secret", "client-sort-preview-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-sort-preview","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","type":"1","target":"401391","roomId":"401391","items":[{"typeId":2,"resId":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	confirmation := response["confirmation"].(map[string]any)
	payloadPreview := confirmation["payloadPreview"].(map[string]any)
	semanticPreview := payloadPreview["semanticPreview"].(map[string]any)
	if semanticPreview["currentItems"] != float64(1) || semanticPreview["plannedItems"] != float64(1) {
		t.Fatalf("semanticPreview = %#v", semanticPreview)
	}
}

func TestInvokeHomeSortConfigureCreatesPlanWhenPreviewReadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/getSort":
			_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-preview-secret", "client-sort-preview-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-sort-preview-fail","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomId":"401391","items":[{"entityType":"device","resId":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	semanticPreview := confirmation["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if semanticPreview["previewUnavailable"] != true || semanticPreview["warning"] != "home_sort_preview_unavailable" {
		t.Fatalf("semanticPreview = %#v", semanticPreview)
	}
}

func TestInvokePlanCommitConfiguresHomeSortFromStoredPlan(t *testing.T) {
	var writeBody []any
	sortReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/getSort":
			sortReadCalls++
			if sortReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"typeId":2,"resId":50018330,"rank":1}]}`))
		case "/apis/iot/v1/sort/200171/w/1/401391/add":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode sort body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-write-secret", "client-sort-write-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "home.sort.configure", map[string]any{
		"houseId": float64(200171),
		"type":    "1",
		"target":  "401391",
		"roomId":  "401391",
		"typeId":  2,
		"resId":   float64(50018330),
		"rank":    1,
		"items": []any{
			map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 1},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-sort-commit","locale":"zh-CN","utterance":"确认排序","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
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
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
}

func createHomeOrganizationPlanForTest(t *testing.T, app *app, houseID string, intent string, payload map[string]any) string {
	t.Helper()
	record, err := plan.NewRecord("default", "dev", houseID, intent, "req-plan", intent+" test", payload, []string{"test precondition"}, time.Now(), pendingPlanTTL)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}
	return record.ID
}
