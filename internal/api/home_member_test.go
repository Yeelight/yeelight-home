package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHomeMemberClientConfiguresRoleWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	memberReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/memberlistV2":
			memberReads++
			if memberReads < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":0}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":2}]}}`))
		case "/apis/iot/v1/house/w/updateUserRole":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeMemberClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeMemberRequest{
		Kind:           HomeMemberConfigure,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":  float64(200171),
			"uid":      float64(9000),
			"memberId": float64(1001),
			"userRole": float64(2),
		},
		Credentials: HomeMemberCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["uid"] != float64(9000) || writeBody["memberId"] != float64(1001) || writeBody["userRole"] != float64(2) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "home.member.list" || result.Capability != "home.member.configure" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeMemberClientRemovesMemberWithReadAfterWrite(t *testing.T) {
	memberReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/memberlistV2":
			memberReads++
			if memberReads < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":0}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[]}}`))
		case "/apis/iot/v1/house/w/remove":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeMemberClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeMemberRequest{
		Kind:           HomeMemberRemove,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":  float64(200171),
			"memberId": float64(1001),
		},
		Credentials: HomeMemberCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Verified || result.Capability != "home.member.remove" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeMemberClientAcceptsShareWithCurrentUserAndVerifiesHomeList(t *testing.T) {
	var writeBody map[string]any
	homeVisible := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/share/w/acceptbarcodeshare":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			homeVisible = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":7001,"resId":200171,"toUid":9000,"status":1}}`))
		case "/apis/iot/v1/house/r/list":
			if !homeVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200171","houseName":"分享家庭"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeMemberClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeMemberRequest{
		Kind:           HomeMemberAccept,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":    float64(200171),
			"shareId":    float64(7001),
			"createTime": float64(1710000000),
			"toUid":      float64(9000),
		},
		Credentials: HomeMemberCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["shareId"] != float64(7001) || writeBody["createTime"] != float64(1710000000) || writeBody["toUid"] != float64(9000) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "home.summary" || result.Capability != "home.member.accept_share" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeMemberClientRejectsMasterRemoval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/memberlistV2" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"业主","userRole":1}]}}`))
	}))
	defer server.Close()

	_, err := NewHomeMemberClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeMemberRequest{
		Kind:    HomeMemberRemove,
		HouseID: "200171",
		Payload: map[string]any{
			"memberId": float64(1001),
		},
		Credentials: HomeMemberCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil {
		t.Fatalf("expected master removal rejection")
	}
}
