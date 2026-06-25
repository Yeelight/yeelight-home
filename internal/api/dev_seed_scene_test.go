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

func TestDevSeedClientReusesExistingSceneByName(t *testing.T) {
	var calls []string
	server := newSeedSceneServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-existing","name":"Codex Dev Test Scene"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureScene(context.Background(), DevSeedSceneRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Scene",
		DeviceID:      "50018330",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureScene error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Created || !result.Verified || result.SceneID != "scene-existing" || result.VerifiedBy != "scene_list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientCreatesAndVerifiesMissingScene(t *testing.T) {
	var calls []string
	var createBody map[string]any
	sceneListCalls := 0
	server := newSeedSceneServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			sceneListCalls++
			if sceneListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":7001,"name":"Codex Dev Test Scene"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":7001}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureScene(context.Background(), DevSeedSceneRequest{
		HouseID:        "200171",
		Name:           "Codex Dev Test Scene",
		Description:    "Runtime dev verification scene",
		DeviceID:       "50018330",
		DeviceName:     "light-dali开关灯-17000002-01",
		PropertyName:   "p",
		PropertyValue:  false,
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureScene error: %v", err)
	}
	if createBody["houseId"] != float64(200171) || createBody["name"] != "Codex Dev Test Scene" || createBody["desc"] != "Runtime dev verification scene" {
		t.Fatalf("createBody = %#v", createBody)
	}
	details, ok := createBody["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("details = %#v", createBody["details"])
	}
	detail, ok := details[0].(map[string]any)
	if !ok {
		t.Fatalf("detail = %#v", details[0])
	}
	if detail["typeId"] != float64(2) || detail["resId"] != float64(50018330) || detail["resName"] != "light-dali开关灯-17000002-01" || detail["action"] != float64(0) || detail["rank"] != float64(0) {
		t.Fatalf("detail = %#v", detail)
	}
	if detail["params"] != `{"set":{"p":false}}` {
		t.Fatalf("params = %#v", detail["params"])
	}
	if !result.Created || !result.Verified || result.SceneID != "7001" || result.VerifiedBy != "scene_list" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "PUT /apis/iot/v2/thing/manage/house/200171/scene/w/create") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientRequiresSceneInputsAndWriteGate(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.EnsureScene(context.Background(), DevSeedSceneRequest{
		Name:        "Codex Dev Test Scene",
		Credentials: DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "--allow-write-dev") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureScene(context.Background(), DevSeedSceneRequest{
		Name:          "Codex Dev Test Scene",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "house id is required") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureScene(context.Background(), DevSeedSceneRequest{
		HouseID:       "200171",
		Name:          "Codex Dev Test Scene",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "device id is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestDevSeedClientRejectsSceneSeedForNonDevEndpoint(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil)
	_, err := client.EnsureScene(context.Background(), DevSeedSceneRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Scene",
		DeviceID:      "50018330",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "only allowed for dev") {
		t.Fatalf("err = %v", err)
	}
}

func newSeedSceneServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		handler(writer, request)
	}))
}
