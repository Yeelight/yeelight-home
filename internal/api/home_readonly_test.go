package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/memberinfoV2":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","userRole":"owner","accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/house/house-1/r/stat":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"roomCount":2,"deviceCount":8,"localToken":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID:    "house-1",
		Parameters: map[string]any{"uid": "1234567890"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-home-secret",
			ClientID:      "client-1",
		},
	}

	member, err := client.RunHomeMemberCurrentGet(context.Background(), request)
	if err != nil {
		t.Fatalf("member err = %v", err)
	}
	stat, err := client.RunHomeStatGet(context.Background(), request)
	if err != nil {
		t.Fatalf("stat err = %v", err)
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/house/r/memberinfoV2\nPOST /apis/iot/v1/house/house-1/r/stat" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range []MetadataReadonlyResult{member, stat} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		for _, forbidden := range []string{"token-home-secret", "not-allowed", "1234567890", "13800138000", "owner@example.com"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
}

func TestHomeReadonlyMissingContextDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	member, err := client.RunHomeMemberCurrentGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("member err = %v", err)
	}
	if !member.Partial || member.APICalls != 0 || len(member.Warnings) != 1 || member.Warnings[0] != "member_context_missing" {
		t.Fatalf("member = %#v", member)
	}
	stat, err := client.RunHomeStatGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("stat err = %v", err)
	}
	if !stat.Partial || stat.APICalls != 0 || len(stat.Warnings) != 1 || stat.Warnings[0] != "house_context_missing" {
		t.Fatalf("stat = %#v", stat)
	}
}

func TestHomeMemberCurrentCanUseCurrentAccountUID(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/house/r/memberinfoV2":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","userRole":"owner","accessToken":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	member, err := client.RunHomeMemberCurrentGet(context.Background(), MetadataReadonlyRequest{
		HouseID:    "house-1",
		Parameters: map[string]any{},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-home-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("member err = %v", err)
	}
	if member.Partial || member.APICalls != 2 {
		t.Fatalf("member = %#v", member)
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/account/user/info\nPOST /apis/iot/v1/house/r/memberinfoV2" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	data, err := json.Marshal(member.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	for _, forbidden := range []string{"token-home-secret", "not-allowed", "1234567890", "13800138000", "owner@example.com"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, string(data))
		}
	}
}
