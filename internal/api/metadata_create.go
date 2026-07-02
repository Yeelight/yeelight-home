package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type MetadataCreateCredentials struct {
	Authorization string
	ClientID      string
}

type MetadataKind string

const (
	MetadataKindArea       MetadataKind = "area"
	MetadataKindGroup      MetadataKind = "group"
	MetadataKindScene      MetadataKind = "scene"
	MetadataKindAutomation MetadataKind = "automation"
)

type MetadataCreateRequest struct {
	Kind           MetadataKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    MetadataCreateCredentials
}

type MetadataCreateResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	EntityType       string           `json:"entityType"`
	EntityID         string           `json:"entityId"`
	Name             string           `json:"name"`
	Created          bool             `json:"created"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	VerifiedEntities EntityListResult `json:"-"`
}

type MetadataCreateClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewMetadataCreateClient(endpoint Endpoint, client *http.Client) MetadataCreateClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return MetadataCreateClient{endpoint: endpoint, client: client}
}

func (client MetadataCreateClient) Run(ctx context.Context, request MetadataCreateRequest) (MetadataCreateResult, error) {
	spec, ok := metadataCreateSpecs[request.Kind]
	if !ok {
		return MetadataCreateResult{}, fmt.Errorf("unsupported metadata kind %q", request.Kind)
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return MetadataCreateResult{}, fmt.Errorf("house id is required")
	}
	name := payloadString(request.Payload, "name")
	if name == "" {
		return MetadataCreateResult{}, fmt.Errorf("%s name is required", request.Kind)
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return MetadataCreateResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	preflightEntities, verifyCalls, err := client.verifyHouseScopedEntityList(ctx, houseID, credentials)
	if err != nil {
		return MetadataCreateResult{}, fmt.Errorf("verify house before %s create: %w", request.Kind, err)
	}
	apiCalls += verifyCalls
	existing, findCalls, err := client.findMetadataByNameWithCallCount(ctx, spec, houseID, name, credentials)
	apiCalls += findCalls
	if err != nil {
		return MetadataCreateResult{}, err
	}
	if existing.ID != "" {
		return MetadataCreateResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			EntityType: string(request.Kind),
			EntityID:   existing.ID,
			Name:       existing.Name,
			Verified:   true,
			VerifiedBy: existing.Source,
			APICalls:   apiCalls,
			VerifiedEntities: upsertEntityListSummary(preflightEntities, EntitySummary{
				Type:    string(request.Kind),
				ID:      existing.ID,
				Name:    existing.Name,
				HouseID: houseID,
			}),
		}, nil
	}
	created, err := client.createMetadata(ctx, spec, houseID, request.Payload, credentials)
	apiCalls++
	if err != nil {
		return MetadataCreateResult{}, err
	}
	verified, verifyCalls, err := client.verifyMetadataByName(ctx, spec, houseID, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return MetadataCreateResult{}, err
	}
	if verified.ID == "" {
		return MetadataCreateResult{}, fmt.Errorf("unknown write result: created %s was not found during verification; createId=%q code=%q message=%q dataType=%s", request.Kind, created.id, created.code, created.message, created.dataType)
	}
	return MetadataCreateResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		EntityType: string(request.Kind),
		EntityID:   verified.ID,
		Name:       verified.Name,
		Created:    true,
		Verified:   true,
		VerifiedBy: verified.Source,
		APICalls:   apiCalls,
		VerifiedEntities: upsertEntityListSummary(preflightEntities, EntitySummary{
			Type:    string(request.Kind),
			ID:      verified.ID,
			Name:    verified.Name,
			HouseID: houseID,
		}),
	}, nil
}

func (client MetadataCreateClient) verifyHouseScopedEntityList(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return result, result.APICalls, err
	}
	if result.APICalls > 0 {
		return result, result.APICalls, nil
	}
	return result, houseScopedEntityListCallCount, nil
}

func (client MetadataCreateClient) findMetadataByName(ctx context.Context, spec metadataCreateSpec, houseID string, name string, credentials requestCredentials) (metadataSummary, error) {
	summary, _, err := client.findMetadataByNameWithCallCount(ctx, spec, houseID, name, credentials)
	return summary, err
}

func (client MetadataCreateClient) findMetadataByNameWithCallCount(ctx context.Context, spec metadataCreateSpec, houseID string, name string, credentials requestCredentials) (metadataSummary, int, error) {
	body := metadataListBody(spec, houseID)
	entities, apiCalls, err := listHouseScopedEntityPages(ctx, client.client, client.endpoint, houseID, credentials, entityListCall{
		entityType:  string(spec.kind),
		method:      spec.listMethod,
		pathPattern: strings.ReplaceAll(spec.listPath, "{houseId}", houseID),
		body:        body,
		singlePage:  spec.singlePage,
	})
	if err != nil {
		return metadataSummary{}, apiCalls, err
	}
	for _, entity := range entities {
		summary := metadataSummary{
			ID:     entity.ID,
			Name:   entity.Name,
			Source: spec.source,
		}
		if summary.Name == name {
			return summary, apiCalls, nil
		}
	}
	return metadataSummary{}, apiCalls, nil
}

func metadataListBody(spec metadataCreateSpec, houseID string) map[string]any {
	if spec.kind == MetadataKindAutomation {
		return map[string]any{semantic.FieldHouseID: houseID}
	}
	return spec.listBody
}

func (client MetadataCreateClient) verifyMetadataByName(ctx context.Context, spec metadataCreateSpec, houseID string, name string, credentials requestCredentials, attempts int, interval time.Duration) (metadataSummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		summary, findCalls, err := client.findMetadataByNameWithCallCount(ctx, spec, houseID, name, credentials)
		calls += findCalls
		if err != nil || summary.ID != "" || attempt == attempts-1 {
			return summary, calls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return metadataSummary{}, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return metadataSummary{}, calls, nil
}

func (client MetadataCreateClient) createMetadata(ctx context.Context, spec metadataCreateSpec, houseID string, payload map[string]any, credentials requestCredentials) (writeProbeResult, error) {
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+strings.ReplaceAll(spec.createPath, "{houseId}", houseID), payload, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("%s create returned non-success business response: code=%s message=%s dataType=%s", spec.kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}
