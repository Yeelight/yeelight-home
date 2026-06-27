package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AutomationUpdateCredentials struct {
	Authorization string
	ClientID      string
}

type AutomationUpdateRequest struct {
	HouseID        string
	AutomationID   string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    AutomationUpdateCredentials
}

type AutomationUpdateResult struct {
	Region       string `json:"region"`
	HouseID      string `json:"houseId"`
	AutomationID string `json:"automationId"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	Verified     bool   `json:"verified"`
	VerifiedBy   string `json:"verifiedBy,omitempty"`
	APICalls     int    `json:"apiCalls"`
}

type AutomationUpdateClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewAutomationUpdateClient(endpoint Endpoint, client *http.Client) AutomationUpdateClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return AutomationUpdateClient{endpoint: endpoint, client: client}
}

func (client AutomationUpdateClient) Run(ctx context.Context, request AutomationUpdateRequest) (AutomationUpdateResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return AutomationUpdateResult{}, fmt.Errorf("house id is required")
	}
	automationID := strings.TrimSpace(request.AutomationID)
	if automationID == "" {
		return AutomationUpdateResult{}, fmt.Errorf("automation id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return AutomationUpdateResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	before, preflightCalls, err := findAutomationByID(ctx, client.endpoint, client.client, houseID, automationID, credentials)
	apiCalls += preflightCalls
	if err != nil {
		return AutomationUpdateResult{}, err
	}
	if before.ID == "" {
		return AutomationUpdateResult{}, fmt.Errorf("automation %s not found before write", automationID)
	}
	body := copyAutomationUpdatePayload(request.Payload, automationID, houseID)
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/automations/"+pathSegment(automationID)+"/w/update", body, credentials)
	apiCalls++
	if err != nil {
		return AutomationUpdateResult{}, err
	}
	if !isBusinessOK(response) {
		return AutomationUpdateResult{}, fmt.Errorf("automation.update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	verified, verifyCalls, err := client.verifyAfterWrite(ctx, houseID, automationID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return AutomationUpdateResult{}, err
	}
	if verified.ID == "" {
		return AutomationUpdateResult{}, fmt.Errorf("automation.update write verification mismatch")
	}
	return AutomationUpdateResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		AutomationID: verified.ID,
		Name:         verified.Name,
		Status:       verified.Status,
		Verified:     true,
		VerifiedBy:   "automation.list",
		APICalls:     apiCalls,
	}, nil
}

func (client AutomationUpdateClient) verifyAfterWrite(ctx context.Context, houseID string, automationID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (EntitySummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entity, readCalls, err := findAutomationByID(ctx, client.endpoint, client.client, houseID, automationID, credentials)
		calls += readCalls
		if err != nil {
			return EntitySummary{}, calls, err
		}
		if automationUpdateMatches(entity, payload) {
			return entity, calls, nil
		}
		if attempt == attempts-1 {
			return EntitySummary{}, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return EntitySummary{}, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return EntitySummary{}, calls, nil
}

func findAutomationByID(ctx context.Context, endpoint Endpoint, httpClient *http.Client, houseID string, automationID string, credentials requestCredentials) (EntitySummary, int, error) {
	result, err := NewEntityListClient(endpoint, httpClient).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}, result.APICalls, err
	}
	for _, entity := range result.Entities {
		if entity.Type == "automation" && entity.ID == automationID {
			return entity, result.APICalls, nil
		}
	}
	return EntitySummary{}, result.APICalls, nil
}

func copyAutomationUpdatePayload(payload map[string]any, automationID string, houseID string) map[string]any {
	body := map[string]any{}
	for key, value := range payload {
		body[key] = value
	}
	delete(body, "automationId")
	body["id"] = requestNumberOrStringForAPI(automationID)
	body["houseId"] = requestNumberOrStringForAPI(houseID)
	return body
}

func automationUpdateMatches(entity EntitySummary, payload map[string]any) bool {
	if entity.ID == "" {
		return false
	}
	if expected := strings.TrimSpace(stringFromAny(payload["name"])); expected != "" && entity.Name != expected {
		return false
	}
	if expected := strings.TrimSpace(stringFromAny(payload["status"])); expected != "" && entity.Status != expected {
		return false
	}
	return true
}

func requestNumberOrStringForAPI(value string) any {
	trimmed := strings.TrimSpace(value)
	if parsed, err := parseID(trimmed, "id"); err == nil {
		return parsed
	}
	return trimmed
}
