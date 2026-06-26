package auth

import (
	"encoding/json"
	"testing"
)

func TestNormalizeDeviceMACAcceptsPlainHexAndColonFormat(t *testing.T) {
	tests := map[string]string{
		"f82441000001":      "F8:24:41:00:00:01",
		"F8:24:41:00:00:01": "F8:24:41:00:00:01",
		"invalid-device":    "invalid-device",
		"":                  "",
	}

	for input, want := range tests {
		if got := NormalizeDeviceMAC(input); got != want {
			t.Fatalf("NormalizeDeviceMAC(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildQRPayloadUsesDALIFormat(t *testing.T) {
	if got := BuildQRPayload("qr-1", "f82441000001", ""); got != "cli&F8:24:41:00:00:01&qr-1" {
		t.Fatalf("payload = %q", got)
	}
	if got := BuildQRPayload("qr-1", "F8:24:41:00:00:01", "200084"); got != "cli&F8:24:41:00:00:01&qr-1&200084" {
		t.Fatalf("payload with house = %q", got)
	}
}

func TestGenerateQRLoginDeviceReturnsNonFixedNormalizedMAC(t *testing.T) {
	device := GenerateQRLoginDevice()
	if device == "F8:24:41:00:00:01" {
		t.Fatal("generated QR login device must not use the old fixed default")
	}
	if NormalizeDeviceMAC(device) != device {
		t.Fatalf("device is not normalized: %q", device)
	}
}

func TestExtractQRLoginCredentialsReadsTokenClientAndHouse(t *testing.T) {
	info := QRInfo{
		Status: "LOGIN",
		Token: QRToken{
			AccessToken: "token-qr-123456",
			ClientID:    "client-qr-123456",
		},
		Source: `dali:{"houseId":"house-qr-123456"}`,
	}

	credentials := ExtractQRLoginCredentials(info)
	if credentials.Authorization != "Bearer token-qr-123456" {
		t.Fatalf("authorization = %q", credentials.Authorization)
	}
	if credentials.ClientID != "client-qr-123456" {
		t.Fatalf("clientId = %q", credentials.ClientID)
	}
	if credentials.HouseID != "house-qr-123456" {
		t.Fatalf("houseId = %q", credentials.HouseID)
	}
}

func TestQRInfoUnmarshalAcceptsCompatibilityCredentialShapes(t *testing.T) {
	tests := []struct {
		name              string
		payload           string
		wantAuthorization string
		wantClientID      string
		wantHouseID       string
	}{
		{
			name: "snake case nested token",
			payload: `{
				"qr_code_id": "qr-snake",
				"expire_at": 123456,
				"status": "LOGIN",
				"token": {
					"access_token": "snake-token",
					"client_id": "snake-client"
				},
				"house_id": "snake-house"
			}`,
			wantAuthorization: "Bearer snake-token",
			wantClientID:      "snake-client",
			wantHouseID:       "snake-house",
		},
		{
			name: "top level authorization and client",
			payload: `{
				"qrCodeId": "qr-top",
				"status": "LOGIN",
				"authorization": "Bearer top-token",
				"clientId": "top-client",
				"houseId": 123456
			}`,
			wantAuthorization: "Bearer top-token",
			wantClientID:      "top-client",
			wantHouseID:       "123456",
		},
		{
			name: "token as string and snake case source",
			payload: `{
				"qrCodeId": "qr-string",
				"status": "LOGIN",
				"token": "string-token",
				"source": "dali:{\"house_id\":\"source-house\"}"
			}`,
			wantAuthorization: "Bearer string-token",
			wantHouseID:       "source-house",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var info QRInfo
			if err := json.Unmarshal([]byte(test.payload), &info); err != nil {
				t.Fatalf("unmarshal QRInfo: %v", err)
			}
			credentials := ExtractQRLoginCredentials(info)
			if credentials.Authorization != test.wantAuthorization {
				t.Fatalf("authorization = %q, want %q", credentials.Authorization, test.wantAuthorization)
			}
			if credentials.ClientID != test.wantClientID {
				t.Fatalf("clientId = %q, want %q", credentials.ClientID, test.wantClientID)
			}
			if credentials.HouseID != test.wantHouseID {
				t.Fatalf("houseId = %q, want %q", credentials.HouseID, test.wantHouseID)
			}
		})
	}
}

func TestExtractHouseIDAcceptsSnakeCaseSource(t *testing.T) {
	if got := ExtractHouseID(`dali:{"house_id":"house-snake"}`); got != "house-snake" {
		t.Fatalf("houseId = %q", got)
	}
}

func TestNormalizeAuthorizationRemovesRepeatedBearerPrefix(t *testing.T) {
	tests := map[string]string{
		"token":               "Bearer token",
		"Bearer token":        "Bearer token",
		"bearer Bearer token": "Bearer token",
		"  bearer   token  ":  "Bearer token",
		"Bearer bearer token": "Bearer token",
		"Bearer BEARER token": "Bearer token",
		"":                    "",
	}

	for input, want := range tests {
		if got := NormalizeAuthorization(input); got != want {
			t.Fatalf("NormalizeAuthorization(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestQRStatusHelpersAreCaseInsensitive(t *testing.T) {
	if !IsQRLoginStatus("login") {
		t.Fatal("expected login status")
	}
	if !IsQRExpiredStatus("EXPIRED") {
		t.Fatal("expected expired status")
	}
	if IsQRLoginStatus("created") {
		t.Fatal("created is not login")
	}
}
