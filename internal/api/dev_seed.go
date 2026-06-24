package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DevSeedCredentials struct {
	Authorization string
	ClientID      string
}

type DevSeedHouseRequest struct {
	Name             string
	Description      string
	AreaCode         string
	AreaName         string
	CandidateHouseID string
	AllowWriteDev    bool
	VerifyAttempts   int
	VerifyInterval   time.Duration
	Credentials      DevSeedCredentials
}

type DevSeedHouseResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
}

type DevSeedClient struct {
	endpoint Endpoint
	client   *http.Client
}

type writeProbeResult struct {
	id       string
	dataType string
	code     string
	message  string
}

func NewDevSeedClient(endpoint Endpoint, client *http.Client) DevSeedClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DevSeedClient{endpoint: endpoint, client: client}
}

func (client DevSeedClient) EnsureHouse(ctx context.Context, request DevSeedHouseRequest) (DevSeedHouseResult, error) {
	if client.endpoint.Region != "dev" {
		return DevSeedHouseResult{}, fmt.Errorf("dev seed is only allowed for dev region")
	}
	if !request.AllowWriteDev {
		return DevSeedHouseResult{}, fmt.Errorf("dev seed requires --allow-write-dev")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return DevSeedHouseResult{}, fmt.Errorf("house name is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DevSeedHouseResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	if candidateHouseID := strings.TrimSpace(request.CandidateHouseID); candidateHouseID != "" {
		if err := client.verifyHouseScopedEntityList(ctx, candidateHouseID, credentials); err == nil {
			return DevSeedHouseResult{
				Region:     client.endpoint.Region,
				HouseID:    candidateHouseID,
				Name:       name,
				Verified:   true,
				VerifiedBy: "entity_list_candidate",
			}, nil
		}
	}
	existing, err := client.findHouseByName(ctx, name, credentials)
	if err != nil {
		return DevSeedHouseResult{}, err
	}
	if existing.ID != "" {
		return DevSeedHouseResult{
			Region:     client.endpoint.Region,
			HouseID:    existing.ID,
			Name:       existing.Name,
			Verified:   true,
			VerifiedBy: existing.Source,
		}, nil
	}
	created, err := client.createHouse(ctx, request, credentials)
	if err != nil {
		return DevSeedHouseResult{}, err
	}
	verified, err := client.verifyHouseByName(ctx, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	if err != nil {
		return DevSeedHouseResult{}, err
	}
	if verified.ID == "" {
		if created.id != "" {
			if err := client.verifyHouseScopedEntityList(ctx, created.id, credentials); err == nil {
				return DevSeedHouseResult{
					Region:     client.endpoint.Region,
					HouseID:    created.id,
					Name:       name,
					Created:    true,
					Verified:   true,
					VerifiedBy: "entity_list",
				}, nil
			}
		}
		return DevSeedHouseResult{}, fmt.Errorf("unknown write result: created house was not found during verification; createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
	}
	return DevSeedHouseResult{
		Region:     client.endpoint.Region,
		HouseID:    verified.ID,
		Name:       verified.Name,
		Created:    true,
		Verified:   true,
		VerifiedBy: verified.Source,
	}, nil
}

func (client DevSeedClient) verifyHouseByName(ctx context.Context, name string, credentials requestCredentials, attempts int, interval time.Duration) (HouseSummary, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	for attempt := 0; attempt < attempts; attempt++ {
		house, err := client.findHouseByName(ctx, name, credentials)
		if err != nil || house.ID != "" || attempt == attempts-1 {
			return house, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return HouseSummary{}, ctx.Err()
		case <-timer.C:
		}
	}
	return HouseSummary{}, nil
}

func (client DevSeedClient) findHouseByName(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, error) {
	if house, err := client.findHouseByNameV1(ctx, name, credentials); err != nil || house.ID != "" {
		return house, err
	}
	return client.findHouseByNameV2(ctx, name, credentials)
}

func (client DevSeedClient) findHouseByNameV1(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/list", map[string]any{}, credentials)
	if err != nil {
		return HouseSummary{}, err
	}
	if !isBusinessOK(response) {
		return HouseSummary{}, fmt.Errorf("home list returned non-success business response")
	}
	for _, house := range extractHouseSummaries(response) {
		if house.Name == name {
			house.Source = "home_list"
			return house, nil
		}
	}
	return HouseSummary{}, nil
}

func (client DevSeedClient) findHouseByNameV2(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/r/info/1/100", nil, credentials)
	if err != nil {
		return HouseSummary{}, err
	}
	if !isBusinessOK(response) {
		return HouseSummary{}, fmt.Errorf("v2 house list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	for _, house := range extractHouseSummaries(response) {
		if house.Name == name {
			house.Source = "v2_house_page"
			return house, nil
		}
	}
	return HouseSummary{}, nil
}

func (client DevSeedClient) verifyHouseScopedEntityList(ctx context.Context, houseID string, credentials requestCredentials) error {
	_, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	return err
}

func (client DevSeedClient) createHouse(ctx context.Context, request DevSeedHouseRequest, credentials requestCredentials) (writeProbeResult, error) {
	body := map[string]any{"name": strings.TrimSpace(request.Name)}
	if value := strings.TrimSpace(request.Description); value != "" {
		body["desc"] = value
	}
	if value := strings.TrimSpace(request.AreaCode); value != "" {
		body["areaCode"] = value
	}
	if value := strings.TrimSpace(request.AreaName); value != "" {
		body["areaName"] = value
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/w/create", body, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("house create returned non-success business response")
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}

func responseID(response map[string]any) string {
	switch value := response["data"].(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return fmt.Sprintf("%.0f", value)
	case map[string]any:
		return firstAnyString(value, "id", "houseId", "roomId", "sceneId")
	}
	return ""
}

func responseDataType(response map[string]any) string {
	value, ok := response["data"]
	if !ok || value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", value)
}

func responseScalar(response map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := response[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		case float64:
			return fmt.Sprintf("%.0f", typed)
		case bool:
			return fmt.Sprintf("%t", typed)
		}
	}
	return ""
}
