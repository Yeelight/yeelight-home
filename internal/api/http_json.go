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

type requestCredentials struct {
	Authorization string
	ClientID      string
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
	request.Header.Set("bizType", "1")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(credentials.Authorization) != "" {
		request.Header.Set("Authorization", normalizeAuthorization(credentials.Authorization))
	}
	if strings.TrimSpace(credentials.ClientID) != "" {
		request.Header.Set("Client-Id", strings.TrimSpace(credentials.ClientID))
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
		return nil, fmt.Errorf("endpoint returned HTTP %d", response.StatusCode)
	}
	parsed := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}
	return parsed, nil
}
