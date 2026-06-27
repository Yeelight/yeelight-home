package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSpaceOrganizationClientRenamesRoomWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"影音室"}]}}`))
		case "/apis/iot/v1/room/401391/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationRoomRename,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"roomId":  "401391",
			"id":      "401391",
			"name":    "影音室",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["id"] != "401391" || writeBody["roomId"] != nil || writeBody["name"] != "影音室" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.EntityType != "room" || result.EntityID != "401391" || result.Name != "影音室" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceOrganizationClientUpdatesRoomWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gw-1","name":"网关"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"会客厅"}]}}`))
		case "/apis/iot/v1/room/401391/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationRoomUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":         float64(200171),
			"roomId":          "401391",
			"id":              "401391",
			"name":            "会客厅",
			"img":             "room-living",
			"gatewayDeviceId": "gw-1",
			"seq":             float64(2),
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["roomId"] != nil || writeBody["id"] != "401391" || writeBody["name"] != "会客厅" || writeBody["img"] != "room-living" || writeBody["gatewayDeviceId"] != "gw-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.EntityType != "room" || result.EntityID != "401391" || result.Name != "会客厅" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceOrganizationClientMovesDeviceWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	deviceListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401392"}]}}`))
		case "/apis/iot/v1/device/50018330/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode device update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationDeviceMove,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":  float64(200171),
			"deviceId": "50018330",
			"id":       "50018330",
			"roomId":   "401392",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["deviceId"] != nil || writeBody["id"] != "50018330" || writeBody["roomId"] != "401392" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.EntityType != "device" || result.RoomID != "401392" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceOrganizationClientUpdatesAreaAndGroupWithReadAfterWrite(t *testing.T) {
	var areaWriteBody map[string]any
	var groupWriteBody map[string]any
	areaListCalls := 0
	groupListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			areaListCalls++
			if areaListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"area-1","name":"一楼"},{"id":"area-parent","name":"全屋"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"area-1","name":"公共区"},{"id":"area-parent","name":"全屋"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			groupListCalls++
			if groupListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"主灯组","roomId":"401392"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/area-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&areaWriteBody); err != nil {
				t.Fatalf("decode area update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&groupWriteBody); err != nil {
				t.Fatalf("decode group update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	areaResult, err := client.Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationAreaUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":  float64(200171),
			"areaId":   "area-1",
			"id":       "area-1",
			"name":     "公共区",
			"parentId": "area-parent",
			"roomIds":  []any{"401391", "401392"},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("area update error: %v", err)
	}
	groupResult, err := client.Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationGroupUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"groupId": "group-1",
			"id":      "group-1",
			"name":    "主灯组",
			"roomId":  "401392",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("group update error: %v", err)
	}
	if areaWriteBody["areaId"] != nil || areaWriteBody["id"] != "area-1" || areaWriteBody["name"] != "公共区" {
		t.Fatalf("areaWriteBody = %#v", areaWriteBody)
	}
	if groupWriteBody["groupId"] != nil || groupWriteBody["id"] != "group-1" || groupWriteBody["roomId"] != "401392" {
		t.Fatalf("groupWriteBody = %#v", groupWriteBody)
	}
	if !areaResult.Verified || areaResult.EntityType != "area" || areaResult.Name != "公共区" {
		t.Fatalf("areaResult = %#v", areaResult)
	}
	if !groupResult.Verified || groupResult.EntityType != "group" || groupResult.Name != "主灯组" || groupResult.RoomID != "401392" {
		t.Fatalf("groupResult = %#v", groupResult)
	}
}

func TestSpaceOrganizationClientUpdatesGroupNameWithoutRoomTarget(t *testing.T) {
	var writeBody map[string]any
	groupListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			groupListCalls++
			if groupListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"主灯组","roomId":"401391"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode group update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationGroupUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"groupId": "group-1",
			"id":      "group-1",
			"name":    "主灯组",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("group update error: %v", err)
	}
	if writeBody["groupId"] != nil || writeBody["id"] != "group-1" || writeBody["name"] != "主灯组" || writeBody["roomId"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.EntityType != "group" || result.Name != "主灯组" || result.RoomID != "401391" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceOrganizationClientVerifiesGroupUpdateWithDetailWhenListOmitsRoomID(t *testing.T) {
	var writeBody map[string]any
	groupListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			groupListCalls++
			if groupListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"主灯组"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"group-1","name":"主灯组","roomId":"401392"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode group update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationGroupUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"groupId": "group-1",
			"id":      "group-1",
			"name":    "主灯组",
			"roomId":  "401392",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("group update error: %v", err)
	}
	if writeBody["groupId"] != nil || writeBody["id"] != "group-1" || writeBody["roomId"] != "401392" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.EntityType != "group" || result.Name != "主灯组" || result.RoomID != "401392" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceOrganizationClientReportsVerificationMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
		case "/apis/iot/v1/room/401391/w/update":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewSpaceOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceOrganizationRequest{
		Kind:           SpaceOrganizationRoomRename,
		HouseID:        "200171",
		VerifyAttempts: 1,
		VerifyInterval: time.Millisecond,
		Payload: map[string]any{
			"houseId": float64(200171),
			"roomId":  "401391",
			"name":    "影音室",
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "write verification mismatch") {
		t.Fatalf("err = %v", err)
	}
}
