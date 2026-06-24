package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HomeCreateCredentials struct {
	Authorization string
	ClientID      string
}

type HomeCreateRequest struct {
	Name           string
	Description    string
	Icon           string
	AreaCode       string
	AreaName       string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    HomeCreateCredentials
}

type HomeCreateResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type HomeCreateClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeCreateClient(endpoint Endpoint, client *http.Client) HomeCreateClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeCreateClient{endpoint: endpoint, client: client}
}

func (client HomeCreateClient) FindHouseByNameForPlan(ctx context.Context, name string, credentials HomeCreateCredentials) (HouseSummary, int, error) {
	if strings.TrimSpace(name) == "" {
		return HouseSummary{}, 0, fmt.Errorf("home name is required")
	}
	requestCredentials := requestCredentials{Authorization: credentials.Authorization, ClientID: credentials.ClientID}
	if strings.TrimSpace(requestCredentials.Authorization) == "" {
		return HouseSummary{}, 0, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	return client.findHouseByName(ctx, strings.TrimSpace(name), requestCredentials)
}

func (client HomeCreateClient) Run(ctx context.Context, request HomeCreateRequest) (HomeCreateResult, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return HomeCreateResult{}, fmt.Errorf("home name is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return HomeCreateResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	existing, calls, err := client.findHouseByName(ctx, name, credentials)
	apiCalls += calls
	if err != nil {
		return HomeCreateResult{}, err
	}
	if existing.ID != "" {
		return HomeCreateResult{
			Region:     client.endpoint.Region,
			HouseID:    existing.ID,
			Name:       existing.Name,
			Created:    false,
			Verified:   true,
			VerifiedBy: existing.Source,
			APICalls:   apiCalls,
		}, nil
	}
	created, calls, err := client.createHouse(ctx, request, credentials)
	apiCalls += calls
	if err != nil {
		return HomeCreateResult{}, err
	}
	verified, calls, err := client.verifyHouseByName(ctx, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return HomeCreateResult{}, err
	}
	if verified.ID != "" {
		return HomeCreateResult{
			Region:     client.endpoint.Region,
			HouseID:    verified.ID,
			Name:       verified.Name,
			Created:    true,
			Verified:   true,
			VerifiedBy: verified.Source,
			APICalls:   apiCalls,
		}, nil
	}
	if created.id != "" {
		if err := client.verifyHouseScopedEntityList(ctx, created.id, credentials); err == nil {
			apiCalls += HouseScopedEntityListCallCount()
			return HomeCreateResult{
				Region:     client.endpoint.Region,
				HouseID:    created.id,
				Name:       name,
				Created:    true,
				Verified:   true,
				VerifiedBy: "entity_list",
				APICalls:   apiCalls,
			}, nil
		}
		apiCalls += HouseScopedEntityListCallCount()
	}
	return HomeCreateResult{}, fmt.Errorf("home.create verification mismatch: createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
}

func (client HomeCreateClient) findHouseByName(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, int, error) {
	house, calls, err := client.findHouseByNameV1(ctx, name, credentials)
	if err != nil || house.ID != "" {
		return house, calls, err
	}
	next, nextCalls, err := client.findHouseByNameV2(ctx, name, credentials)
	return next, calls + nextCalls, err
}

func (client HomeCreateClient) findHouseByNameV1(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/list", map[string]any{}, credentials)
	if err != nil {
		return HouseSummary{}, 1, err
	}
	if !isBusinessOK(response) {
		return HouseSummary{}, 1, fmt.Errorf("home.summary returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	for _, house := range extractHouseSummaries(response) {
		if house.Name == name {
			house.Source = "home.summary"
			return house, 1, nil
		}
	}
	return HouseSummary{}, 1, nil
}

func (client HomeCreateClient) findHouseByNameV2(ctx context.Context, name string, credentials requestCredentials) (HouseSummary, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/r/info/1/100", nil, credentials)
	if err != nil {
		return HouseSummary{}, 1, err
	}
	if !isBusinessOK(response) {
		return HouseSummary{}, 1, fmt.Errorf("v2 home list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	for _, house := range extractHouseSummaries(response) {
		if house.Name == name {
			house.Source = "v2_home_page"
			return house, 1, nil
		}
	}
	return HouseSummary{}, 1, nil
}

func (client HomeCreateClient) verifyHouseByName(ctx context.Context, name string, credentials requestCredentials, attempts int, interval time.Duration) (HouseSummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	totalCalls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		house, calls, err := client.findHouseByName(ctx, name, credentials)
		totalCalls += calls
		if err != nil || house.ID != "" || attempt == attempts-1 {
			return house, totalCalls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return HouseSummary{}, totalCalls, ctx.Err()
		case <-timer.C:
		}
	}
	return HouseSummary{}, totalCalls, nil
}

func (client HomeCreateClient) verifyHouseScopedEntityList(ctx context.Context, houseID string, credentials requestCredentials) error {
	_, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	return err
}

func (client HomeCreateClient) createHouse(ctx context.Context, request HomeCreateRequest, credentials requestCredentials) (writeProbeResult, int, error) {
	body := map[string]any{"name": strings.TrimSpace(request.Name)}
	if value := strings.TrimSpace(request.Description); value != "" {
		body["desc"] = value
	}
	if value := strings.TrimSpace(request.Icon); value != "" {
		body["icon"] = value
	}
	if value := strings.TrimSpace(request.AreaCode); value != "" {
		body["areaCode"] = value
	}
	if value := strings.TrimSpace(request.AreaName); value != "" {
		body["areaName"] = value
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/w/create", body, credentials)
	if err != nil {
		return writeProbeResult{}, 1, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, 1, fmt.Errorf("home.create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, 1, nil
}
