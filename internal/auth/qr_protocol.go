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
	AccessToken   string `json:"accessToken"`
	Token         string `json:"token"`
	Authorization string `json:"authorization"`
	ClientID      string `json:"clientId"`
}

type QRInfo struct {
	QRCodeID string  `json:"qrCodeId"`
	Status   string  `json:"status"`
	ExpireAt int64   `json:"expireAt"`
	Token    QRToken `json:"token"`
	Source   string  `json:"source"`
	HouseID  string  `json:"houseId"`
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
		Authorization: NormalizeAuthorization(firstNonEmpty(info.Token.Authorization, info.Token.AccessToken, info.Token.Token)),
		ClientID:      strings.TrimSpace(info.Token.ClientID),
		HouseID:       firstNonEmpty(info.HouseID, ExtractHouseID(info.Source)),
	}
}

func (token *QRToken) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var tokenText string
	if err := json.Unmarshal(data, &tokenText); err == nil {
		token.Token = strings.TrimSpace(tokenText)
		return nil
	}
	var parsed struct {
		AccessToken      string `json:"accessToken"`
		AccessTokenSnake string `json:"access_token"`
		Token            string `json:"token"`
		Authorization    string `json:"authorization"`
		ClientID         string `json:"clientId"`
		ClientIDSnake    string `json:"client_id"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	token.AccessToken = firstNonEmpty(parsed.AccessToken, parsed.AccessTokenSnake)
	token.Token = strings.TrimSpace(parsed.Token)
	token.Authorization = strings.TrimSpace(parsed.Authorization)
	token.ClientID = firstNonEmpty(parsed.ClientID, parsed.ClientIDSnake)
	return nil
}

func (info *QRInfo) UnmarshalJSON(data []byte) error {
	var parsed struct {
		QRCodeID         string  `json:"qrCodeId"`
		QRCodeIDSnake    string  `json:"qr_code_id"`
		Status           string  `json:"status"`
		ExpireAt         int64   `json:"expireAt"`
		ExpireAtSnake    int64   `json:"expire_at"`
		Token            QRToken `json:"token"`
		Source           string  `json:"source"`
		Authorization    string  `json:"authorization"`
		AccessToken      string  `json:"accessToken"`
		AccessTokenSnake string  `json:"access_token"`
		ClientID         string  `json:"clientId"`
		ClientIDSnake    string  `json:"client_id"`
		HouseID          any     `json:"houseId"`
		HouseIDSnake     any     `json:"house_id"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	info.QRCodeID = firstNonEmpty(parsed.QRCodeID, parsed.QRCodeIDSnake)
	info.Status = strings.TrimSpace(parsed.Status)
	info.ExpireAt = parsed.ExpireAt
	if info.ExpireAt == 0 {
		info.ExpireAt = parsed.ExpireAtSnake
	}
	info.Token = parsed.Token
	info.Token.Authorization = firstNonEmpty(info.Token.Authorization, parsed.Authorization)
	info.Token.AccessToken = firstNonEmpty(info.Token.AccessToken, parsed.AccessToken, parsed.AccessTokenSnake)
	info.Token.ClientID = firstNonEmpty(info.Token.ClientID, parsed.ClientID, parsed.ClientIDSnake)
	info.Source = strings.TrimSpace(parsed.Source)
	info.HouseID = firstNonEmpty(houseIDFromAny(parsed.HouseID), houseIDFromAny(parsed.HouseIDSnake))
	return nil
}

func ExtractHouseID(source string) string {
	normalized := strings.TrimSpace(source)
	if normalized == "" {
		return ""
	}
	normalized = strings.TrimPrefix(normalized, "dali:")
	var parsed struct {
		HouseID      any `json:"houseId"`
		HouseIDSnake any `json:"house_id"`
	}
	if err := json.Unmarshal([]byte(normalized), &parsed); err == nil {
		return firstNonEmpty(houseIDFromAny(parsed.HouseID), houseIDFromAny(parsed.HouseIDSnake))
	}
	if regexp.MustCompile(`^\d+$`).MatchString(normalized) {
		return normalized
	}
	return ""
}

func houseIDFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimRight(strings.TrimRight(jsonNumber(typed), "0"), ".")
	default:
		return ""
	}
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
