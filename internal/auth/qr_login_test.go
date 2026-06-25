package auth

import (
	"context"
	"testing"
	"time"
)

func TestRunQRLoginFlowNoWaitReturnsPayloadWithoutCredentials(t *testing.T) {
	client := &fakeQRClient{
		created: QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}

	result, err := RunQRLoginFlow(context.Background(), QRLoginOptions{
		Client:  client,
		Device:  "f82441000001",
		HouseID: "200084",
		NoWait:  true,
	})
	if err != nil {
		t.Fatalf("RunQRLoginFlow error: %v", err)
	}
	if result.Payload != "cli&F8:24:41:00:00:01&qr-nowait-1&200084" {
		t.Fatalf("payload = %q", result.Payload)
	}
	if result.Credentials != nil {
		t.Fatalf("credentials = %#v", result.Credentials)
	}
	if client.checkCalls != 0 {
		t.Fatalf("checkCalls = %d", client.checkCalls)
	}
}

func TestRunQRLoginFlowReturnsCredentialsAfterLogin(t *testing.T) {
	client := &fakeQRClient{
		created: QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []QRInfo{
			{QRCodeID: "qr-login-1", Status: "SCANNED"},
			{
				QRCodeID: "qr-login-1",
				Status:   "LOGIN",
				Token:    QRToken{AccessToken: "token-qr-123456", ClientID: "client-qr-123456"},
				Source:   `dali:{"houseId":"house-qr-123456"}`,
			},
		},
	}

	result, err := RunQRLoginFlow(context.Background(), QRLoginOptions{
		Client:       client,
		Device:       "F8:24:41:00:00:01",
		PollInterval: time.Millisecond,
		Timeout:      time.Second,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatalf("RunQRLoginFlow error: %v", err)
	}
	if result.Credentials == nil {
		t.Fatal("expected credentials")
	}
	if result.Credentials.Authorization != "Bearer token-qr-123456" {
		t.Fatalf("authorization = %q", result.Credentials.Authorization)
	}
	if result.Credentials.ClientID != "client-qr-123456" {
		t.Fatalf("clientId = %q", result.Credentials.ClientID)
	}
	if result.Credentials.HouseID != "house-qr-123456" {
		t.Fatalf("houseId = %q", result.Credentials.HouseID)
	}
}

func TestRunQRLoginFlowCallsOnCreatedBeforePolling(t *testing.T) {
	client := &fakeQRClient{
		created: QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []QRInfo{{
			QRCodeID: "qr-login-1",
			Status:   "LOGIN",
			Token:    QRToken{AccessToken: "token-qr-123456"},
		}},
	}
	var createdResult QRLoginResult

	_, err := RunQRLoginFlow(context.Background(), QRLoginOptions{
		Client:       client,
		Device:       "F8:24:41:00:00:01",
		PollInterval: time.Millisecond,
		Timeout:      time.Second,
		Sleep:        func(context.Context, time.Duration) error { return nil },
		OnCreated: func(result QRLoginResult) {
			createdResult = result
		},
	})
	if err != nil {
		t.Fatalf("RunQRLoginFlow error: %v", err)
	}
	if createdResult.Payload != "cli&F8:24:41:00:00:01&qr-login-1" {
		t.Fatalf("created payload = %q", createdResult.Payload)
	}
	if createdResult.Credentials != nil {
		t.Fatalf("created credentials = %#v", createdResult.Credentials)
	}
	if client.checkCalls == 0 {
		t.Fatal("expected polling after OnCreated")
	}
}

func TestRunQRLoginFlowRejectsLoginWithoutToken(t *testing.T) {
	client := &fakeQRClient{
		created: QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []QRInfo{{QRCodeID: "qr-login-1", Status: "LOGIN"}},
	}

	_, err := RunQRLoginFlow(context.Background(), QRLoginOptions{
		Client:       client,
		Device:       "F8:24:41:00:00:01",
		PollInterval: time.Millisecond,
		Timeout:      time.Second,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestRunQRLoginFlowRequiresResolvedDevice(t *testing.T) {
	client := &fakeQRClient{
		created: QRInfo{QRCodeID: "qr-login-1", Status: "CREATED"},
	}

	_, err := RunQRLoginFlow(context.Background(), QRLoginOptions{Client: client})
	if err == nil {
		t.Fatal("expected missing device error")
	}
}

type fakeQRClient struct {
	created    QRInfo
	checked    []QRInfo
	checkCalls int
}

func (client *fakeQRClient) Create(context.Context, string) (QRInfo, error) {
	return client.created, nil
}

func (client *fakeQRClient) Check(context.Context, string) (QRInfo, error) {
	index := client.checkCalls
	client.checkCalls++
	if index >= len(client.checked) {
		return client.checked[len(client.checked)-1], nil
	}
	return client.checked[index], nil
}
