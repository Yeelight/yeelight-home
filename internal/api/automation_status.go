package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AutomationStatusKind string

const (
	AutomationStatusEnable  AutomationStatusKind = "automation.enable"
	AutomationStatusDisable AutomationStatusKind = "automation.disable"
)

const (
	automationStatusEnabled  = "1"
	automationStatusDisabled = "0"
)

type AutomationStatusCredentials struct {
	Authorization string
	ClientID      string
}

type AutomationStatusRequest struct {
	Kind           AutomationStatusKind
	HouseID        string
	AutomationID   string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    AutomationStatusCredentials
}

type AutomationStatusResult struct {
	Region       string `json:"region"`
	HouseID      string `json:"houseId"`
	AutomationID string `json:"automationId"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	Capability   string `json:"capability"`
	Verified     bool   `json:"verified"`
	VerifiedBy   string `json:"verifiedBy,omitempty"`
	APICalls     int    `json:"apiCalls"`
}

type AutomationStatusClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewAutomationStatusClient(endpoint Endpoint, client *http.Client) AutomationStatusClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return AutomationStatusClient{endpoint: endpoint, client: client}
}

func (client AutomationStatusClient) Run(ctx context.Context, request AutomationStatusRequest) (AutomationStatusResult, error) {
	spec, ok := automationStatusSpecs[request.Kind]
	if !ok {
		return AutomationStatusResult{}, fmt.Errorf("unsupported automation status kind %q", request.Kind)
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return AutomationStatusResult{}, fmt.Errorf("house id is required")
	}
	automationID := strings.TrimSpace(request.AutomationID)
	if automationID == "" {
		return AutomationStatusResult{}, fmt.Errorf("automation id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return AutomationStatusResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	before, preflightCalls, err := client.findAutomation(ctx, houseID, automationID, credentials)
	apiCalls += preflightCalls
	if err != nil {
		return AutomationStatusResult{}, err
	}
	if before.ID == "" {
		return AutomationStatusResult{}, fmt.Errorf("automation %s not found before write", automationID)
	}
	if err := client.write(ctx, spec, automationID, credentials); err != nil {
		return AutomationStatusResult{}, err
	}
	apiCalls++
	verified, verifyCalls, err := client.verifyAfterWrite(ctx, spec, houseID, automationID, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return AutomationStatusResult{}, err
	}
	if verified.ID == "" {
		return AutomationStatusResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return AutomationStatusResult{
		Region:       client.endpoint.Region,
		HouseID:      houseID,
		AutomationID: verified.ID,
		Name:         verified.Name,
		Status:       verified.Status,
		Capability:   string(request.Kind),
		Verified:     true,
		VerifiedBy:   "automation.list.status",
		APICalls:     apiCalls,
	}, nil
}

type automationStatusSpec struct {
	kind           AutomationStatusKind
	pathPattern    string
	expectedStatus string
}

var automationStatusSpecs = map[AutomationStatusKind]automationStatusSpec{
	AutomationStatusEnable: {
		kind:           AutomationStatusEnable,
		pathPattern:    "/v1/automations/w/enable/{id}",
		expectedStatus: automationStatusEnabled,
	},
	AutomationStatusDisable: {
		kind:           AutomationStatusDisable,
		pathPattern:    "/v1/automations/w/disable/{id}",
		expectedStatus: automationStatusDisabled,
	},
}

func (client AutomationStatusClient) write(ctx context.Context, spec automationStatusSpec, automationID string, credentials requestCredentials) error {
	path := strings.ReplaceAll(spec.pathPattern, "{id}", pathSegment(automationID))
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, nil, credentials)
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", spec.kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client AutomationStatusClient) verifyAfterWrite(ctx context.Context, spec automationStatusSpec, houseID string, automationID string, credentials requestCredentials, attempts int, interval time.Duration) (EntitySummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entity, readCalls, err := client.findAutomation(ctx, houseID, automationID, credentials)
		calls += readCalls
		if err != nil {
			return EntitySummary{}, calls, err
		}
		if automationStatusMatches(entity, spec.expectedStatus) {
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

func (client AutomationStatusClient) findAutomation(ctx context.Context, houseID string, automationID string, credentials requestCredentials) (EntitySummary, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
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

func automationStatusMatches(entity EntitySummary, expectedStatus string) bool {
	if entity.ID == "" {
		return false
	}
	return strings.TrimSpace(entity.Status) == expectedStatus
}
