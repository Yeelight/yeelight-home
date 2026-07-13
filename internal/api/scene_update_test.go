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

func TestSceneUpdateClientWritesAndVerifiesByDetail(t *testing.T) {
	var calls []string
	var updateBody map[string]any
	detailCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"回家灯光","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify":
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&updateBody); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			detailCalls++
			name := "回家灯光"
			if detailCalls > 1 {
				name = "回家灯光更新"
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","name":"` + name + `","details":[{"typeId":2,"resId":50018330,"rank":0,"action":0,"params":"{\"set\":{\"p\":true}}"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	payload := map[string]any{
		"sceneId": "scene-1",
		"id":      "scene-1",
		"houseId": float64(200171),
		"name":    "回家灯光更新",
		"details": []any{
			map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 0, "action": 0, "params": `{"set":{"p":true}}`},
		},
	}
	result, err := NewSceneUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SceneUpdateRequest{
		HouseID:     "200171",
		SceneID:     "scene-1",
		Payload:     payload,
		Credentials: SceneUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if updateBody["id"] != "scene-1" || updateBody["houseId"] != float64(200171) || updateBody["name"] != "回家灯光更新" {
		t.Fatalf("updateBody = %#v", updateBody)
	}
	if !result.Verified || result.SceneID != "scene-1" || result.Name != "回家灯光更新" || result.VerifiedBy != "scene.detail.get" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "POST /apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestSceneUpdateClientVerifiesPublicDetailRows(t *testing.T) {
	detailCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"4771","name":"测试RGBW组0702","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"1006157","name":"测试RGBW开灯0702","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/1006157/w/modify":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/scene/1006157/r/detail":
			detailCalls++
			if detailCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"1006157","name":"测试RGBW开灯0702","details":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"1006157","name":"测试RGBW开灯0702","details":[{"targetType":"meshGroup","targetId":"4771","targetName":"测试RGBW组0702","rank":0,"action":0,"set":{"power":true,"brightness":55,"colorTemperature":3200}}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSceneUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SceneUpdateRequest{
		HouseID: "200171",
		SceneID: "1006157",
		Payload: map[string]any{
			"sceneId": "1006157",
			"id":      "1006157",
			"houseId": float64(200171),
			"name":    "测试RGBW开灯0702",
			"details": []any{
				map[string]any{"typeId": 4, "resId": float64(4771), "rank": 0, "action": 0, "params": `{"set":{"p":true,"l":55,"ct":3200}}`},
			},
		},
		Credentials: SceneUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Verified || result.SceneID != "1006157" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSceneUpdateClientRequiresDetailVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"回家灯光","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","name":"未更新","details":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewSceneUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SceneUpdateRequest{
		HouseID:        "200171",
		SceneID:        "scene-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"name": "目标名称",
			"details": []any{
				map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 0, "params": `{"set":{"p":true}}`},
			},
		},
		Credentials: SceneUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err == nil {
		t.Fatal("expected verification mismatch")
	}
}

func TestSceneUpdateClientRejectsNoopWhenDescriptionDidNotPersist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"回家灯光","houseId":"200171"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","name":"回家灯光","desc":"旧说明","details":[{"typeId":2,"resId":50018330,"rank":0,"action":0,"params":"{\"set\":{\"p\":true}}"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewSceneUpdateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SceneUpdateRequest{
		HouseID:        "200171",
		SceneID:        "scene-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"name": "回家灯光",
			"desc": "新说明",
			"details": []any{
				map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 0, "action": 0, "params": `{"set":{"p":true}}`},
			},
		},
		Credentials: SceneUpdateCredentials{Authorization: "Bearer secret", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "verification mismatch") {
		t.Fatalf("expected description verification mismatch, got %v", err)
	}
}
