package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MetadataDeleteKind string

const (
	MetadataDeleteRoom       MetadataDeleteKind = "room.delete"
	MetadataDeleteArea       MetadataDeleteKind = "area.delete"
	MetadataDeleteGroup      MetadataDeleteKind = "group.delete"
	MetadataDeleteScene      MetadataDeleteKind = "scene.delete"
	MetadataDeleteAutomation MetadataDeleteKind = "automation.delete"
)

type MetadataDeleteCredentials struct {
	Authorization string
	ClientID      string
}

type MetadataDeleteRequest struct {
	Kind           MetadataDeleteKind
	HouseID        string
	EntityID       string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    MetadataDeleteCredentials
}

type MetadataDeleteResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	Capability       string           `json:"capability"`
	EntityType       string           `json:"entityType"`
	EntityID         string           `json:"entityId"`
	Name             string           `json:"name,omitempty"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	VerifiedEntities EntityListResult `json:"-"`
}

type MetadataDeleteClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewMetadataDeleteClient(endpoint Endpoint, client *http.Client) MetadataDeleteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return MetadataDeleteClient{endpoint: endpoint, client: client}
}

type metadataDeleteSpec struct {
	kind       MetadataDeleteKind
	entityType string
	path       string
}

var metadataDeleteSpecs = map[MetadataDeleteKind]metadataDeleteSpec{
	MetadataDeleteRoom:       {kind: MetadataDeleteRoom, entityType: "room", path: "/v2/thing/manage/house/{houseId}/room/{id}/w/info"},
	MetadataDeleteArea:       {kind: MetadataDeleteArea, entityType: "area", path: "/v2/thing/manage/house/{houseId}/area/{id}/w/info"},
	MetadataDeleteGroup:      {kind: MetadataDeleteGroup, entityType: "group", path: "/v2/thing/manage/house/{houseId}/group/{id}/w/info"},
	MetadataDeleteScene:      {kind: MetadataDeleteScene, entityType: "scene", path: "/v2/thing/manage/house/{houseId}/scene/{id}/w/info"},
	MetadataDeleteAutomation: {kind: MetadataDeleteAutomation, entityType: "automation", path: "/v2/thing/manage/house/{houseId}/automation/{id}/w/info"},
}

func (client MetadataDeleteClient) Run(ctx context.Context, request MetadataDeleteRequest) (MetadataDeleteResult, error) {
	spec, ok := metadataDeleteSpecs[request.Kind]
	if !ok {
		return MetadataDeleteResult{}, fmt.Errorf("unsupported metadata delete kind %q", request.Kind)
	}
	houseID := strings.TrimSpace(request.HouseID)
	entityID := strings.TrimSpace(request.EntityID)
	if houseID == "" {
		return MetadataDeleteResult{}, fmt.Errorf("house id is required")
	}
	if entityID == "" {
		return MetadataDeleteResult{}, fmt.Errorf("%s id is required", spec.entityType)
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return MetadataDeleteResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	before, _, preflightCalls, err := client.findEntity(ctx, houseID, spec.entityType, entityID, credentials)
	apiCalls += preflightCalls
	if err != nil {
		return MetadataDeleteResult{}, err
	}
	if before.ID == "" {
		return MetadataDeleteResult{}, fmt.Errorf("%s %s not found before delete", spec.entityType, entityID)
	}
	path := strings.ReplaceAll(spec.path, "{houseId}", pathSegment(houseID))
	path = strings.ReplaceAll(path, "{id}", pathSegment(entityID))
	response, err := callJSON(ctx, client.client, http.MethodDelete, strings.TrimRight(client.endpoint.BaseURL, "/")+path, nil, credentials)
	apiCalls++
	if err != nil {
		return MetadataDeleteResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataDeleteResult{}, fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", spec.kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	deleted, verifiedEntities, verifyCalls, err := client.verifyDeleted(ctx, houseID, spec.entityType, entityID, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return MetadataDeleteResult{}, err
	}
	if !deleted {
		return MetadataDeleteResult{}, fmt.Errorf("%s delete verification mismatch", spec.kind)
	}
	return MetadataDeleteResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		Capability:       string(spec.kind),
		EntityType:       spec.entityType,
		EntityID:         before.ID,
		Name:             before.Name,
		Verified:         true,
		VerifiedBy:       "entity.list",
		APICalls:         apiCalls,
		VerifiedEntities: verifiedEntities,
	}, nil
}

func (client MetadataDeleteClient) findEntity(ctx context.Context, houseID string, entityType string, entityID string, credentials requestCredentials) (EntitySummary, EntityListResult, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}, result, result.APICalls, err
	}
	for _, entity := range result.Entities {
		if entity.Type == entityType && entity.ID == entityID {
			return entity, result, result.APICalls, nil
		}
	}
	return EntitySummary{}, result, result.APICalls, nil
}

func (client MetadataDeleteClient) verifyDeleted(ctx context.Context, houseID string, entityType string, entityID string, credentials requestCredentials, attempts int, interval time.Duration) (bool, EntityListResult, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entity, entities, readCalls, err := client.findEntity(ctx, houseID, entityType, entityID, credentials)
		calls += readCalls
		if err != nil {
			return false, entities, calls, err
		}
		if entity.ID == "" {
			return true, entities, calls, nil
		}
		if attempt == attempts-1 {
			return false, entities, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, entities, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, EntityListResult{}, calls, nil
}
