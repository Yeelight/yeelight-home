package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunHomeSortListUsesNodeSortedDeviceReadback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401391/device":
			if request.Header.Get("houseId") != "200171" || request.Header.Get("house-id") != "200171" {
				t.Fatalf("home sort list missing house headers: houseId=%q house-id=%q", request.Header.Get("houseId"), request.Header.Get("house-id"))
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"name":"主灯","roomId":401391,"rank":1}]}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":50018330,"name":"主灯","roomId":401391}]}}`))
		case "/apis/iot/v1/sort/r/getSort":
			t.Fatalf("device-room sort should not call getSort when node readback succeeds")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunHomeSortList(context.Background(), MetadataReadonlyRequest{
		HouseID: "200171",
		Parameters: map[string]any{
			"type":   1,
			"target": "401391",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-sort-secret"},
	})
	if err != nil {
		t.Fatalf("RunHomeSortList error: %v", err)
	}
	if result.Partial || result.APICalls != 1 || result.Capability != "home.sort.list" {
		t.Fatalf("result = %#v", result)
	}
	data := result.Data.(map[string]any)
	if data["readback"] != "node.sorted_device.list" {
		t.Fatalf("data = %#v", data)
	}
	sortRows := data["sort"].([]any)
	row := sortRows[0].(map[string]any)
	if row["id"] != "50018330" || row["targetId"] != "50018330" || row["targetType"] != "device" {
		t.Fatalf("sort rows not enriched = %#v", sortRows)
	}
	if _, ok := row["resId"]; ok {
		t.Fatalf("sort row leaked resId = %#v", row)
	}
	if _, ok := row["typeId"]; ok {
		t.Fatalf("sort row leaked typeId = %#v", row)
	}
}

func TestRunHomeSortListUsesRoomSceneReadback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/room/scene":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"roomId":401391,"sceneOrder":{"1005999":3}}]}`))
		case "/apis/iot/v1/sort/r/getSort":
			t.Fatalf("scene-room sort should not call getSort when room-scene readback succeeds")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunHomeSortList(context.Background(), MetadataReadonlyRequest{
		HouseID: "200171",
		Parameters: map[string]any{
			"type":   2,
			"target": "401391",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-sort-secret"},
	})
	if err != nil {
		t.Fatalf("RunHomeSortList error: %v", err)
	}
	if result.Partial || result.APICalls != 1 || result.Capability != "home.sort.list" {
		t.Fatalf("result = %#v", result)
	}
	data := result.Data.(map[string]any)
	if data["readback"] != "room.scene.sort" {
		t.Fatalf("data = %#v", data)
	}
}

func TestRunHomeSortListBusinessFailureReturnsPartialEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/room-1/device":
			http.NotFound(writer, request)
			return
		case "/apis/iot/v1/sort/r/getSort":
			_, _ = writer.Write([]byte(`{"success":false,"code":400,"message":"bad request"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunHomeSortList(context.Background(), MetadataReadonlyRequest{
		HouseID: "house-1",
		Parameters: map[string]any{
			"type":   1,
			"target": "room-1",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-sort-secret"},
	})
	if err != nil {
		t.Fatalf("RunHomeSortList error: %v", err)
	}
	if !result.Partial || result.APICalls != 1 || result.Capability != "home.sort.list" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "home_sort_cloud_read_failed" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("data = %#v", result.Data)
	}
	evidence, ok := data["backendEvidence"].(map[string]any)
	if !ok || evidence["status"] != "failed" || evidence["code"] != "400" || evidence["controller"] != nil || evidence["adapter"] != nil {
		t.Fatalf("evidence = %#v", data["backendEvidence"])
	}
}
