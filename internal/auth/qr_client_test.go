package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQRLoginClientCreatesAndChecksQRCode(t *testing.T) {
	var calls []string
	var acceptLanguage string
	var bizType string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		acceptLanguage = request.Header.Get("Accept-Language")
		bizType = request.Header.Get("bizType")
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/scan-login/query/qrcode/F8:24:41:00:00:01":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"qrCodeId": "qr-login-1",
					"status":   "CREATED",
					"expireAt": int64(123456),
				},
			})
		case "/apis/account/user/scan-login/check/qrcode/qr-login-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"qrCodeId": "qr-login-1",
					"status":   "LOGIN",
					"token": map[string]any{
						"accessToken": "token-qr-123456",
						"clientId":    "client-qr-123456",
					},
					"source": `dali:{"houseId":"house-qr-123456"}`,
				},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewQRLoginClient(server.URL, server.Client())
	created, err := client.Create(context.Background(), "F8:24:41:00:00:01")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	checked, err := client.Check(context.Background(), created.QRCodeID)
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("calls = %#v", calls)
	}
	if calls[0] != "POST /apis/account/user/scan-login/query/qrcode/F8:24:41:00:00:01" {
		t.Fatalf("create call = %q", calls[0])
	}
	if calls[1] != "POST /apis/account/user/scan-login/check/qrcode/qr-login-1" {
		t.Fatalf("check call = %q", calls[1])
	}
	if acceptLanguage != "zh-CN" {
		t.Fatalf("Accept-Language = %q", acceptLanguage)
	}
	if bizType != "1" {
		t.Fatalf("bizType = %q", bizType)
	}
	if created.Status != "CREATED" || checked.Status != "LOGIN" {
		t.Fatalf("created=%#v checked=%#v", created, checked)
	}
	credentials := ExtractQRLoginCredentials(checked)
	if credentials.Authorization != "Bearer token-qr-123456" || credentials.ClientID != "client-qr-123456" || credentials.HouseID != "house-qr-123456" {
		t.Fatalf("credentials = %#v", credentials)
	}
}

func TestQRLoginClientRejectsBusinessFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"success": false,
			"message": "扫码登录接口返回失败",
		})
	}))
	defer server.Close()

	client := NewQRLoginClient(server.URL, server.Client())
	_, err := client.Create(context.Background(), "F8:24:41:00:00:01")
	if err == nil {
		t.Fatal("expected business error")
	}
}

func TestQRLoginClientUsesSelectedBizType(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got = request.Header.Get("bizType")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"qrCodeId":"qr-1","status":"CREATED"}}`))
	}))
	defer server.Close()
	client := NewQRLoginClientWithBizType(server.URL, server.Client(), "0")
	if _, err := client.Create(context.Background(), "F8:24:41:00:00:01"); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if got != "0" {
		t.Fatalf("bizType = %q", got)
	}
}
