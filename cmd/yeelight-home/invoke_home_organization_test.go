package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestInvokeDeviceDetailGetReturnsRedactedProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/50018330/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"50018330","name":"主灯","mac":"AA:BB:CC:DD","localToken":"not-allowed","roomId":"401391","shadow":{"propertyMap":{"o":true,"on":true,"l":42,"ct":3000,"sp":true,"nt":2}}}}`))
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
	detail := response["result"].(map[string]any)["data"].(map[string]any)["detail"].(map[string]any)
	properties := detail["properties"].(map[string]any)
	if properties["online"] != true || properties["power"] != true || properties["brightness"] != float64(42) || properties["colorTemperature"] != float64(3000) || properties["switchPower"] != true {
		t.Fatalf("properties = %#v", properties)
	}
	if _, ok := properties["o"]; ok {
		t.Fatalf("raw property leaked: %#v", properties)
	}
	if _, ok := properties["nt"]; ok {
		t.Fatalf("unmapped property leaked: %#v", properties)
	}
}

func TestInvokeMetadataDetailResolvesNaturalNames(t *testing.T) {
	tests := []struct {
		name       string
		intent     string
		parameters string
		detailPath string
		traceID    string
	}{
		{
			name:       "device detail",
			intent:     "device.detail.get",
			parameters: `{"houseId":"200171","deviceName":"主灯"}`,
			detailPath: "/apis/iot/v1/device/50018330/r/detail",
			traceID:    "device-detail-get-readonly",
		},
		{
			name:       "group detail",
			intent:     "group.detail.get",
			parameters: `{"houseId":"200171","groupName":"已有灯组"}`,
			detailPath: "/apis/iot/v2/thing/manage/house/200171/group/600001/r/info",
			traceID:    "group-detail-get-readonly",
		},
		{
			name:       "scene detail",
			intent:     "scene.detail.get",
			parameters: `{"houseId":"200171","sceneName":"已有情景"}`,
			detailPath: "/apis/iot/v1/scene/700001/r/detail",
			traceID:    "scene-detail-get-readonly",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var detailCalled bool
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				if request.URL.Path == test.detailPath {
					detailCalled = true
					_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"resolved","name":"详情","mac":"AA:BB:CC:DD","localToken":"not-allowed"}}`))
					return
				}
				writeSeededHouseScopedListForConfigureTest(writer, request)
			}))
			defer server.Close()
			t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
			app := newInvokeTestApp(t, "Bearer token-detail-name-secret", "client-detail-name-1", "200171")

			input := `{"contractVersion":"1.0","requestId":"req-detail-by-name","locale":"zh-CN","utterance":"查看详情","intent":"` + test.intent + `","parameters":` + test.parameters + `}`
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			if !detailCalled {
				t.Fatalf("detail endpoint was not called")
			}
			for _, forbidden := range []string{"token-detail-name-secret", "not-allowed", "AA:BB:CC:DD"} {
				if strings.Contains(stdout.String(), forbidden) {
					t.Fatalf("output leaked %q: %s", forbidden, stdout.String())
				}
			}
			response := decodeInvokeResponse(t, stdout.Bytes())
			if response["status"] != "success" || response["traceId"] != test.traceID {
				t.Fatalf("response = %#v", response)
			}
		})
	}
}

func TestInvokeUpgradeFileListKeepsHouseScopeForNaturalDeviceName(t *testing.T) {
	var gotUpgradeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/apis/iot/v1/upgrade/r/listfile" {
			if err := json.NewDecoder(request.Body).Decode(&gotUpgradeBody); err != nil {
				t.Fatalf("decode upgrade body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"file-1","version":"1.0.1","secret":"nope"}]}}`))
			return
		}
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-upgrade-file-secret", "client-upgrade-file-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-upgrade-file-by-name","locale":"zh-CN","utterance":"查一下主灯有没有升级文件","intent":"upgrade.file.list","parameters":{"houseId":"200171","deviceName":"主灯"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, forbidden := range []string{"token-upgrade-file-secret", "secret", "nope"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, stdout.String())
		}
	}
	if gotUpgradeBody[semantic.FieldDeviceID] != "50018330" {
		t.Fatalf("gotUpgradeBody = %#v", gotUpgradeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "upgrade-file-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result[semantic.FieldHouseID] != "200171" {
		t.Fatalf("result should keep house scope: %#v", result)
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

	input := `{"contractVersion":"1.0","requestId":"req-fav-add","locale":"zh-CN","utterance":"把主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","targetType":"device","targetId":"50018330","rank":1}}`
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

func TestInvokeDeviceEnergySummaryResolvesNaturalDeviceName(t *testing.T) {
	var energyCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/apis/iot/v1/energy/devices/50018330/r/summary" {
			energyCalled = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"today":1.2,"month":3.4}}`))
			return
		}
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-energy-name-secret", "client-energy-name-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-energy-by-name","locale":"zh-CN","utterance":"看看客厅主灯能耗","intent":"device.energy.summary","parameters":{"houseId":"200171","roomName":"客厅","deviceName":"主灯"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if !energyCalled || response["status"] != "success" || response["traceId"] != "device-energy-summary-readonly" {
		t.Fatalf("response = %#v energyCalled=%v", response, energyCalled)
	}
}

func TestInvokeNodeSortedDeviceListResolvesNaturalRoomName(t *testing.T) {
	var nodeCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/apis/iot/v1/node/r/1/401398/device" {
			nodeCalled = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"50018330","name":"主灯","rank":1}]}}`))
			return
		}
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-node-name-secret", "client-node-name-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-node-sort-by-room-name","locale":"zh-CN","utterance":"列出客厅下面按排序保存的设备","intent":"node.sorted_device.list","parameters":{"houseId":"200171","entityType":"room","roomName":"客厅"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if !nodeCalled || response["status"] != "success" || response["traceId"] != "node-sorted_device-list-readonly" {
		t.Fatalf("response = %#v nodeCalled=%v", response, nodeCalled)
	}
}

func TestInvokeFavoriteAddAcceptsEntityIdentityAlias(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-alias-secret", "client-fav-alias-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-entity-alias","locale":"zh-CN","utterance":"把这个设备加到收藏","intent":"favorite.add","parameters":{"houseId":"200171","entityType":"device","entityId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v, calls=%#v", response, gotCalls)
	}
	result := response["result"].(map[string]any)
	payloadPreview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview["rank"] != float64(1) || result["dryRun"] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
}

func TestInvokeFavoriteAddResolvesTargetName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-name-secret", "client-fav-name-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-name","locale":"zh-CN","utterance":"把主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","targetType":"device","targetName":"主灯","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v, calls=%#v", response, gotCalls)
	}
	if strings.Contains(stdout.String(), `"typeId"`) || strings.Contains(stdout.String(), `"resId"`) {
		t.Fatalf("preview leaked internal favorite fields: %s", stdout.String())
	}
	result := response["result"].(map[string]any)
	if result["dryRun"] != true {
		t.Fatalf("result=%#v", result)
	}
	payloadPreview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview["targetType"] != "device" || payloadPreview["targetId"] != "50018330" || payloadPreview["targetName"] != "主灯" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/favourite/w/insert") {
			t.Fatalf("favorite.add dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeFavoriteAddUsesRoomQualifierForDuplicateDeviceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"},{"id":"401399","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401398"},{"id":"50018331","name":"主灯","roomId":"401399"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-fav-room-secret", "client-fav-room-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-fav-add-room-name","locale":"zh-CN","utterance":"把客厅主灯加入收藏","intent":"favorite.add","parameters":{"houseId":"200171","targetType":"device","targetName":"主灯","roomName":"客厅","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	payloadPreview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview["targetType"] != "device" || payloadPreview["targetId"] != "50018330" || payloadPreview["roomName"] != "客厅" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
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

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete","locale":"zh-CN","utterance":"删除主灯首页收藏","intent":"favorite.delete","parameters":{"houseId":"200171","favoriteId":"fav-1","targetType":"device","targetId":"50018330","rank":1,"confirmed":true}}`
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

func TestInvokeFavoriteDeleteRequiresExplicitConfirmationBeforeWriting(t *testing.T) {
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
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

	input := `{"contractVersion":"1.0","requestId":"req-fav-delete-unconfirmed","locale":"zh-CN","utterance":"删除主灯首页收藏","intent":"favorite.delete","parameters":{"houseId":"200171","favoriteId":"fav-1","targetType":"device","targetId":"50018330","rank":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if deleteCalls != 0 {
		t.Fatalf("deleteCalls = %d", deleteCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" || response["traceId"] != "r3-confirmation-required" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeHomeSortConfigureDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401398/device":
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

	input := `{"contractVersion":"1.0","requestId":"req-sort-preview","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomName":"客厅","items":[{"targetType":"device","targetName":"主灯","rank":1}]}}`
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
	if strings.Contains(stdout.String(), `"typeId"`) || strings.Contains(stdout.String(), `"resId"`) {
		t.Fatalf("preview leaked internal sort fields: %s", stdout.String())
	}
	result := response["result"].(map[string]any)
	payloadPreview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview["sortType"] != "device_room" || payloadPreview["roomId"] != "401398" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
	if _, ok := payloadPreview["type"]; ok {
		t.Fatalf("sort preview leaked backend type: %#v", payloadPreview)
	}
	if _, ok := payloadPreview["target"]; ok {
		t.Fatalf("sort preview leaked backend target: %#v", payloadPreview)
	}
	items := payloadPreview["items"].([]any)
	item := items[0].(map[string]any)
	if item["targetType"] != "device" || item["targetId"] != "50018330" || item["rank"] != float64(1) {
		t.Fatalf("item preview = %#v", item)
	}
}

func TestInvokeHomeSortConfigureExecutesDirectly(t *testing.T) {
	var writeBody []any
	sortReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401398/device":
			sortReadCalls++
			if sortReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"50018330","rank":1}]}`))
		case "/apis/iot/v1/sort/200171/w/1/401398/add":
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

	input := `{"contractVersion":"1.0","requestId":"req-sort-execute","locale":"zh-CN","utterance":"把客厅主灯排到第一位","intent":"home.sort.configure","parameters":{"houseId":"200171","sortType":"device_room","roomId":"401398","items":[{"entityType":"device","id":"50018330","rank":1}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 1 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	row := writeBody[0].(map[string]any)
	if row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)] != float64(2) ||
		row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] != float64(50018330) ||
		row[semantic.FieldRank] != float64(1) {
		t.Fatalf("writeBody row = %#v", row)
	}
	for _, leaked := range []string{semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldType, semantic.FieldTarget, semantic.FieldTargetType, semantic.FieldTargetID, semantic.FieldTargetName} {
		if _, ok := row[leaked]; ok {
			t.Fatalf("sort write body leaked %s: %#v", leaked, row)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}
