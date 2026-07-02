package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

const (
	LightingDesignImportCapability = "lighting.design.import"
	DeviceSlotCreateCapability     = "device.slot.create"

	lightingDesignMaxRooms       = 50
	lightingDesignMaxDevices     = 120
	lightingDesignMaxGroups      = 50
	lightingDesignMaxScenes      = 30
	lightingDesignMaxAutomations = 30
)

type LightingDesignImportCredentials struct {
	Authorization string
	ClientID      string
}

type LightingDesignImportRequest struct {
	HouseID        string
	Intent         string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    LightingDesignImportCredentials
}

type LightingDesignImportResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	Capability       string           `json:"capability"`
	Mode             string           `json:"mode"`
	RequestKey       string           `json:"requestKey,omitempty"`
	Counts           map[string]int   `json:"counts"`
	Mappings         map[string]any   `json:"mappings,omitempty"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	Warnings         []string         `json:"warnings,omitempty"`
	VerifiedEntities EntityListResult `json:"-"`
}

type LightingDesignImportClient struct {
	endpoint Endpoint
	client   *http.Client
}

type lightingDesignMetaIndex struct {
	RoomsByTempID   map[string]string
	DevicesByTempID map[string]string
	GroupsByTempID  map[string]string
	ScenesByTempID  map[string]string
}

func NewLightingDesignImportClient(endpoint Endpoint, client *http.Client) LightingDesignImportClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return LightingDesignImportClient{endpoint: endpoint, client: client}
}

func (client LightingDesignImportClient) Run(ctx context.Context, request LightingDesignImportRequest) (LightingDesignImportResult, error) {
	houseID := strings.TrimSpace(firstNonEmpty(request.HouseID, stringFromMap(request.Payload, semantic.FieldHouseID)))
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return LightingDesignImportResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	payload := copyLightingDesignDeepMap(request.Payload)
	response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/meta/import", payload, credentials)
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	if !isBusinessOK(response) {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	requestKey := lightingDesignMetaImportRequestKey(response["data"])
	if requestKey == "" {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import missing meta import request key")
	}
	result := LightingDesignImportResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: firstNonEmpty(strings.TrimSpace(request.Intent), LightingDesignImportCapability),
		Mode:       lightingDesignImportMode(payload),
		RequestKey: requestKey,
		Counts:     lightingDesignImportCounts(payload),
		Verified:   true,
		VerifiedBy: "meta.import",
		APICalls:   1,
		Warnings:   nil,
	}
	status, calls, err := client.waitForMetaImport(ctx, requestKey, credentials, request.VerifyAttempts, request.VerifyInterval)
	result.APICalls += calls
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	if importedHouseID := lightingDesignMetaImportHouseID(status); importedHouseID != "" {
		result.HouseID = importedHouseID
		credentials.HouseID = importedHouseID
	}
	if strings.TrimSpace(result.HouseID) == "" {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import status missing houseId for verification")
	}
	verified, verifiedEntities, calls, err := client.verify(ctx, result.HouseID, credentials, result.Counts, request.VerifyAttempts, request.VerifyInterval)
	result.APICalls += calls
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	if !verified {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import write verification mismatch")
	}
	result.VerifiedBy = "entity.list"
	result.VerifiedEntities = verifiedEntities
	return result, nil
}
