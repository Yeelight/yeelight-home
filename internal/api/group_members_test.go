package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGroupMembersClientUpdatesMembersWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	updated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/1001/group/9001/r/info":
			if updated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"id":9001,"name":"客厅格栅灯组","cid":5,"devices":[{"deviceId":5001,"name":"左灯"},{"deviceId":5003,"name":"右灯"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":9001,"name":"客厅格栅灯组","cid":5,"devices":[{"deviceId":5001,"name":"左灯"},{"deviceId":5002,"name":"旧灯"}]}}`))
		case "/apis/iot/v2/thing/schema/house/1001/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"5003","name":"右灯","subDevices":[{"cid":5,"name":"color light","category":"light","properties":[{"propId":"p"},{"propId":"l"}]}]}]}}`))
		case "/apis/iot/v2/thing/manage/house/1001/group/9001/w/devices":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			updated = true
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewGroupMembersClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), GroupMembersRequest{
		HouseID:         "1001",
		GroupID:         "9001",
		AddDeviceIDs:    []string{"5003"},
		RemoveDeviceIDs: []string{"5002"},
		VerifyAttempts:  1,
		VerifyInterval:  time.Millisecond,
		Credentials:     SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Verified != true || result.GroupID != "9001" || result.APICalls != 4 {
		t.Fatalf("result = %#v", result)
	}
	addList := writeBody["addDeviceList"].([]any)
	removeList := writeBody["removeDeviceList"].([]any)
	if addList[0] != float64(5003) || removeList[0] != float64(5002) {
		t.Fatalf("write body = %#v", writeBody)
	}
}

func TestGroupMembersClientRejectsIncompatibleComponent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/1001/group/9001/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":9001,"name":"客厅格栅灯组","cid":5,"devices":[{"deviceId":5001,"name":"左灯"}]}}`))
		case "/apis/iot/v2/thing/schema/house/1001/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"5004","name":"不兼容灯","subDevices":[{"cid":9,"name":"switch","category":"switch","properties":[{"propId":"p"}]}]}]}}`))
		case "/apis/iot/v2/thing/manage/house/1001/group/9001/w/devices":
			t.Fatal("incompatible device should not be written")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewGroupMembersClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), GroupMembersRequest{
		HouseID:      "1001",
		GroupID:      "9001",
		AddDeviceIDs: []string{"5004"},
		Credentials:  SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil {
		t.Fatal("expected incompatible component error")
	}
}
