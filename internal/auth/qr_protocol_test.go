package auth

import "testing"

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
