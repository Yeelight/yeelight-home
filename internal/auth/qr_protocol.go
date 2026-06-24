package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultQRLoginBaseURL        = "https://api.yeelight.com"
	DefaultQRLoginPollIntervalMS = 3000
	DefaultQRLoginTimeoutMS      = 180000
)

type QRToken struct {
	AccessToken string `json:"accessToken"`
	Token       string `json:"token"`
	ClientID    string `json:"clientId"`
}

type QRInfo struct {
	QRCodeID string  `json:"qrCodeId"`
	Status   string  `json:"status"`
	ExpireAt int64   `json:"expireAt"`
	Token    QRToken `json:"token"`
	Source   string  `json:"source"`
}

type LoginCredentials struct {
	Authorization string `json:"authorization"`
	ClientID      string `json:"clientId"`
	HouseID       string `json:"houseId"`
}

var (
	plainMACPattern = regexp.MustCompile(`^[0-9a-fA-F]{12}$`)
	colonMACPattern = regexp.MustCompile(`^[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}$`)
)

func NormalizeQRLoginBaseURL(value string) string {
	text := strings.TrimRight(strings.TrimSpace(value), "/")
	if text == "" {
		return DefaultQRLoginBaseURL
	}
	return text
}

func NormalizeDeviceMAC(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if plainMACPattern.MatchString(raw) {
		parts := make([]string, 0, 6)
		for index := 0; index < len(raw); index += 2 {
			parts = append(parts, raw[index:index+2])
		}
		return strings.ToUpper(strings.Join(parts, ":"))
	}
	if colonMACPattern.MatchString(raw) {
		return strings.ToUpper(raw)
	}
	return raw
}

func BuildQRPayload(qrCodeID string, device string, houseID string) string {
	payload := "cli&" + NormalizeDeviceMAC(device) + "&" + qrCodeID
	if strings.TrimSpace(houseID) != "" {
		payload += "&" + strings.TrimSpace(houseID)
	}
	return payload
}

func GenerateQRLoginDevice() string {
	var suffix [3]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "F8:24:41:AA:BB:CC"
	}
	return fmt.Sprintf("F8:24:41:%02X:%02X:%02X", suffix[0], suffix[1], suffix[2])
}

func ExtractQRLoginCredentials(info QRInfo) LoginCredentials {
	return LoginCredentials{
		Authorization: NormalizeAuthorization(firstNonEmpty(info.Token.AccessToken, info.Token.Token)),
		ClientID:      strings.TrimSpace(info.Token.ClientID),
		HouseID:       ExtractHouseID(info.Source),
	}
}

func ExtractHouseID(source string) string {
	normalized := strings.TrimSpace(source)
	if normalized == "" {
		return ""
	}
	normalized = strings.TrimPrefix(normalized, "dali:")
	var parsed struct {
		HouseID any `json:"houseId"`
	}
	if err := json.Unmarshal([]byte(normalized), &parsed); err == nil && parsed.HouseID != nil {
		switch value := parsed.HouseID.(type) {
		case string:
			return value
		case float64:
			return strings.TrimRight(strings.TrimRight(jsonNumber(value), "0"), ".")
		}
	}
	if regexp.MustCompile(`^\d+$`).MatchString(normalized) {
		return normalized
	}
	return ""
}

func NormalizeAuthorization(value string) string {
	token := strings.TrimSpace(value)
	for strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

func IsQRLoginStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "LOGIN")
}

func IsQRExpiredStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "EXPIRED")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func jsonNumber(value float64) string {
	data, _ := json.Marshal(value)
	return string(data)
}
