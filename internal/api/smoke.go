package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SmokeCredentials struct {
	Authorization string
	ClientID      string
	HouseID       string
}

type SmokeResult struct {
	Region      string `json:"region"`
	AccountOK   bool   `json:"accountOk"`
	HouseListOK bool   `json:"houseListOk"`
	HouseCount  int    `json:"houseCount"`
}

type SmokeClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewSmokeClient(endpoint Endpoint, client *http.Client) SmokeClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SmokeClient{endpoint: endpoint, client: client}
}

func (client SmokeClient) Run(ctx context.Context, credentials SmokeCredentials) (SmokeResult, error) {
	account, err := client.call(ctx, http.MethodGet, client.endpoint.AccountBaseURL()+"/apis/account/user/info", nil, credentials)
	if err != nil {
		return SmokeResult{}, err
	}
	houses, err := client.call(ctx, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/list", map[string]any{}, credentials)
	if err != nil {
		return SmokeResult{}, err
	}
	return SmokeResult{
		Region:      client.endpoint.Region,
		AccountOK:   isBusinessOK(account),
		HouseListOK: isBusinessOK(houses),
		HouseCount:  countDataRows(houses),
	}, nil
}

func (client SmokeClient) call(ctx context.Context, method string, url string, body map[string]any, credentials SmokeCredentials) (map[string]any, error) {
	response, err := callJSON(ctx, client.client, method, url, body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
	if err != nil {
		return nil, fmt.Errorf("call smoke endpoint: %w", err)
	}
	return response, nil
}

func isBusinessOK(response map[string]any) bool {
	if value, ok := response["success"].(bool); ok {
		return value
	}
	if value, ok := response["code"].(string); ok {
		return value == "200" || value == "0"
	}
	if value, ok := response["code"].(float64); ok {
		return value == 200 || value == 0
	}
	return true
}

func countDataRows(response map[string]any) int {
	data, ok := response["data"]
	if !ok {
		return 0
	}
	switch rows := data.(type) {
	case []any:
		return len(rows)
	case map[string]any:
		if nestedRows, ok := rows["rows"].([]any); ok {
			return len(nestedRows)
		}
	}
	return 0
}
