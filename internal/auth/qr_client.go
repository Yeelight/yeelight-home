package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type QRLoginClient struct {
	baseURL string
	client  *http.Client
	bizType string
}

func NewQRLoginClient(baseURL string, client *http.Client) QRLoginClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return QRLoginClient{
		baseURL: NormalizeQRLoginBaseURL(baseURL),
		client:  client,
		bizType: "1",
	}
}

func NewQRLoginClientWithBizType(baseURL string, client *http.Client, bizType string) QRLoginClient {
	result := NewQRLoginClient(baseURL, client)
	if bizType == "0" || bizType == "1" {
		result.bizType = bizType
	}
	return result
}

func (client QRLoginClient) Create(ctx context.Context, device string) (QRInfo, error) {
	return client.post(ctx, "/apis/account/user/scan-login/query/qrcode/"+url.PathEscape(device))
}

func (client QRLoginClient) Check(ctx context.Context, qrCodeID string) (QRInfo, error) {
	return client.post(ctx, "/apis/account/user/scan-login/check/qrcode/"+url.PathEscape(qrCodeID))
}

func (client QRLoginClient) post(ctx context.Context, path string) (QRInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, nil)
	if err != nil {
		return QRInfo{}, fmt.Errorf("build QR login request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "zh-CN")
	request.Header.Set("bizType", client.bizType)

	response, err := client.client.Do(request)
	if err != nil {
		return QRInfo{}, fmt.Errorf("call QR login API: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return QRInfo{}, fmt.Errorf("read QR login response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return QRInfo{}, fmt.Errorf("QR login API returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Success *bool           `json:"success"`
		Message string          `json:"message"`
		Msg     string          `json:"msg"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return QRInfo{}, fmt.Errorf("decode QR login response: %w", err)
	}
	if envelope.Success != nil && !*envelope.Success {
		message := firstNonEmpty(envelope.Message, envelope.Msg, "QR login API returned business failure")
		return QRInfo{}, fmt.Errorf("%s", message)
	}
	payload := envelope.Data
	if len(payload) == 0 || string(payload) == "null" {
		payload = data
	}
	var info QRInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return QRInfo{}, fmt.Errorf("decode QR login data: %w", err)
	}
	return info, nil
}
