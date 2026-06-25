package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestMetadataCreateClientCreatesAreaAndVerifiesByList(t *testing.T) {
	var createBody map[string]any
	areaListCalls := 0
	server := metadataCreateServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			areaListCalls++
			if areaListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return true
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"area-created","name":"一楼"}]}}`))
			return true
		case "/apis/iot/v2/thing/manage/house/200171/area/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"area-created"}`))
			return true
		default:
			return false
		}
	})
	defer server.Close()
	payload, err := BuildAreaCreatePayload("200171", "一楼", "", "", "", []string{"401391"})
	if err != nil {
		t.Fatalf("BuildAreaCreatePayload error: %v", err)
	}

	result, err := NewMetadataCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), MetadataCreateRequest{
		Kind:    MetadataKindArea,
		HouseID: "200171",
		Payload: payload,
		Credentials: MetadataCreateCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if createBody["name"] != "一楼" || result.EntityID != "area-created" || !result.Verified || result.VerifiedBy != "area_list" {
		t.Fatalf("createBody=%#v result=%#v", createBody, result)
	}
}

func TestMetadataCreateClientDedupeGroupByName(t *testing.T) {
	server := metadataCreateServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100" {
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-existing","name":"客厅灯组"}]}}`))
			return true
		}
		if request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/group/w/create" {
			t.Fatal("group create should not write when name exists")
		}
		return false
	})
	defer server.Close()
	payload, err := BuildGroupCreatePayload("200171", "客厅灯组", "401391", "7", []string{}, "", "")
	if err != nil {
		t.Fatalf("BuildGroupCreatePayload error: %v", err)
	}

	result, err := NewMetadataCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), MetadataCreateRequest{
		Kind:    MetadataKindGroup,
		HouseID: "200171",
		Payload: payload,
		Credentials: MetadataCreateCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.EntityID != "group-existing" || result.Created {
		t.Fatalf("result = %#v", result)
	}
}

func TestMetadataCreateClientDedupeGroupByNameOnLaterPage(t *testing.T) {
	var calls []string
	server := metadataCreateServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[` + testEntityRows("group", "灯组", 1, 100) + `]}}`))
			return true
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/2/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-existing","name":"客厅灯组"}]}}`))
			return true
		case "/apis/iot/v2/thing/manage/house/200171/group/w/create":
			t.Fatal("group create should not write when name exists on later page")
			return true
		default:
			return false
		}
	})
	defer server.Close()
	payload, err := BuildGroupCreatePayload("200171", "客厅灯组", "401391", "7", []string{}, "", "")
	if err != nil {
		t.Fatalf("BuildGroupCreatePayload error: %v", err)
	}

	result, err := NewMetadataCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), MetadataCreateRequest{
		Kind:    MetadataKindGroup,
		HouseID: "200171",
		Payload: payload,
		Credentials: MetadataCreateCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/2/100",
		"POST /apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/200171/group/r/info/2/100",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.EntityID != "group-existing" || result.Created || result.APICalls != 9 {
		t.Fatalf("result = %#v", result)
	}
}

func TestMetadataCreateClientCreatesScene(t *testing.T) {
	var createBody map[string]any
	sceneListCalls := 0
	server := metadataCreateServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			sceneListCalls++
			if sceneListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return true
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-created","name":"回家灯光"}]}}`))
			return true
		case "/apis/iot/v2/thing/manage/house/200171/scene/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"scene-created"}`))
			return true
		default:
			return false
		}
	})
	defer server.Close()
	payload, err := BuildSceneCreatePayload("200171", "回家灯光", "", "", []map[string]any{{
		"typeId": 2,
		"resId":  50018330,
		"params": `{"set":{"p":true}}`,
		"rank":   0,
		"action": 0,
	}})
	if err != nil {
		t.Fatalf("BuildSceneCreatePayload error: %v", err)
	}

	result, err := NewMetadataCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), MetadataCreateRequest{
		Kind:    MetadataKindScene,
		HouseID: "200171",
		Payload: payload,
		Credentials: MetadataCreateCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if createBody["name"] != "回家灯光" || result.EntityID != "scene-created" || !result.Created {
		t.Fatalf("createBody=%#v result=%#v", createBody, result)
	}
}

func TestBuildAutomationCreatePayloadValidatesRequiredShape(t *testing.T) {
	status := 1
	payload, err := BuildAutomationCreatePayload("200171", "每天关灯", "00:00:00", "23:59:59", 2, "0x7f", map[string]any{
		"type":       "and",
		"conditions": []any{map[string]any{"type": "alarm", "clock": "22:00:00"}},
	}, []map[string]any{{
		"typeId": 2,
		"resId":  "50018330",
		"params": `{"set":{"p":false}}`,
		"rank":   0,
	}}, 2, &status)
	if err != nil {
		t.Fatalf("BuildAutomationCreatePayload error: %v", err)
	}
	if payload["repeatType"] != 2 || payload["status"] != 1 || payload["params"] == "" {
		t.Fatalf("payload = %#v", payload)
	}
}

func metadataCreateServer(t *testing.T, handler func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/2/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			if handler(writer, request) {
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			if handler(writer, request) {
				return
			}
			http.NotFound(writer, request)
		}
	}))
}
