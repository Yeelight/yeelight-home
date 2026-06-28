package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPStatusError struct {
	StatusCode int
}

func (err HTTPStatusError) Error() string {
	if err.StatusCode == http.StatusUnauthorized || err.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("authorization failed with HTTP %d", err.StatusCode)
	}
	return fmt.Sprintf("endpoint returned HTTP %d", err.StatusCode)
}

type requestCredentials struct {
	Authorization string
	ClientID      string
	HouseID       string
	BizType       string
}

func callJSON(ctx context.Context, client *http.Client, method string, url string, body map[string]any, credentials requestCredentials) (map[string]any, error) {
	return callJSONBody(ctx, client, method, url, body, credentials)
}

func callJSONBody(ctx context.Context, client *http.Client, method string, url string, body any, credentials requestCredentials) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "zh-CN")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(credentials.Authorization) != "" {
		request.Header.Set("Authorization", normalizeAuthorization(credentials.Authorization))
	}
	if strings.TrimSpace(credentials.ClientID) != "" {
		request.Header.Set("Client-Id", strings.TrimSpace(credentials.ClientID))
	}
	if strings.TrimSpace(credentials.HouseID) != "" {
		houseID := strings.TrimSpace(credentials.HouseID)
		request.Header.Set("houseId", houseID)
		request.Header.Set("house-id", houseID)
	}
	if strings.TrimSpace(credentials.BizType) != "" {
		request.Header.Set("bizType", strings.TrimSpace(credentials.BizType))
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call endpoint: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, HTTPStatusError{StatusCode: response.StatusCode}
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	switch parsed := decoded.(type) {
	case map[string]any:
		return parsed, nil
	case []any:
		return map[string]any{"data": parsed}, nil
	default:
		return map[string]any{"data": parsed}, nil
	}
}
