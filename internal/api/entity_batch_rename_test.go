package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEntityBatchRenameClientRenamesDeviceAndSceneWithReadAfterWrite(t *testing.T) {
	var writeBody []any
	deviceListCalls := 0
	sceneListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"阅读主灯"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			sceneListCalls++
			if sceneListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"700001","name":"晚安"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"700001","name":"睡前晚安"}]}}`))
		case "/apis/iot/v1/ai/200171/name/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode rename body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewEntityBatchRenameClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), EntityBatchRenameRequest{
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"items": []any{
				map[string]any{"id": "50018330", "typeId": 2, "name": "阅读主灯"},
				map[string]any{"id": "700001", "typeId": 6, "name": "睡前晚安"},
			},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 2 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	first := writeBody[0].(map[string]any)
	if first["id"] != float64(50018330) || first["typeId"] != float64(2) || first["name"] != "阅读主灯" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.ItemCount != 2 || result.VerifiedBy != "entity.list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestEntityBatchRenameClientRejectsUnsupportedType(t *testing.T) {
	_, err := NewEntityBatchRenameClient(Endpoint{Region: "dev", BaseURL: "http://example.invalid"}, nil).Run(context.Background(), EntityBatchRenameRequest{
		HouseID: "200171",
		Payload: map[string]any{
			"items": []any{map[string]any{"id": "401398", "typeId": 1, "name": "客厅新名"}},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid entity rename resource type") {
		t.Fatalf("err = %v", err)
	}
}

func TestEntityBatchRenameClientRejectsNameCollision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯"},{"id":"50018430","name":"筒灯"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewEntityBatchRenameClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), EntityBatchRenameRequest{
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"items": []any{map[string]any{"id": "50018330", "typeId": 2, "name": "筒灯"}},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "device name already exists") {
		t.Fatalf("err = %v", err)
	}
}
