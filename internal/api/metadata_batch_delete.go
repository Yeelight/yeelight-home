package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MetadataBatchDeleteKind string

const (
	MetadataBatchDeleteRoom       MetadataBatchDeleteKind = "room.batch_delete"
	MetadataBatchDeleteArea       MetadataBatchDeleteKind = "area.batch_delete"
	MetadataBatchDeleteGroup      MetadataBatchDeleteKind = "group.batch_delete"
	MetadataBatchDeleteScene      MetadataBatchDeleteKind = "scene.batch_delete"
	MetadataBatchDeleteAutomation MetadataBatchDeleteKind = "automation.batch_delete"
)

type MetadataBatchDeleteRequest struct {
	Kind           MetadataBatchDeleteKind
	HouseID        string
	Items          []MetadataBatchDeleteItem
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    MetadataDeleteCredentials
}

type MetadataBatchDeleteItem struct {
	EntityID string
	Name     string
}

type MetadataBatchDeleteResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	EntityType string `json:"entityType"`
	ItemCount  int    `json:"itemCount"`
	Results    []any  `json:"results"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type MetadataBatchDeleteClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewMetadataBatchDeleteClient(endpoint Endpoint, client *http.Client) MetadataBatchDeleteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return MetadataBatchDeleteClient{endpoint: endpoint, client: client}
}

func (client MetadataBatchDeleteClient) Run(ctx context.Context, request MetadataBatchDeleteRequest) (MetadataBatchDeleteResult, error) {
	deleteKind, entityType, err := metadataBatchDeleteTarget(request.Kind)
	if err != nil {
		return MetadataBatchDeleteResult{}, err
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return MetadataBatchDeleteResult{}, fmt.Errorf("house id is required")
	}
	if len(request.Items) == 0 {
		return MetadataBatchDeleteResult{}, fmt.Errorf("batch delete items are required")
	}
	results := make([]any, 0, len(request.Items))
	apiCalls := 0
	for _, item := range request.Items {
		entityID := strings.TrimSpace(item.EntityID)
		if entityID == "" {
			return MetadataBatchDeleteResult{}, fmt.Errorf("%s id is required", entityType)
		}
		result, err := NewMetadataDeleteClient(client.endpoint, client.client).Run(ctx, MetadataDeleteRequest{
			Kind:           deleteKind,
			HouseID:        houseID,
			EntityID:       entityID,
			VerifyAttempts: request.VerifyAttempts,
			VerifyInterval: request.VerifyInterval,
			Credentials:    request.Credentials,
		})
		apiCalls += result.APICalls
		if err != nil {
			return MetadataBatchDeleteResult{}, err
		}
		results = append(results, map[string]any{
			"entityType": result.EntityType,
			"entityId":   result.EntityID,
			"name":       result.Name,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		})
	}
	return MetadataBatchDeleteResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: string(request.Kind),
		EntityType: entityType,
		ItemCount:  len(request.Items),
		Results:    results,
		Verified:   true,
		VerifiedBy: "entity.list",
		APICalls:   apiCalls,
	}, nil
}

func metadataBatchDeleteTarget(kind MetadataBatchDeleteKind) (MetadataDeleteKind, string, error) {
	switch kind {
	case MetadataBatchDeleteRoom:
		return MetadataDeleteRoom, "room", nil
	case MetadataBatchDeleteArea:
		return MetadataDeleteArea, "area", nil
	case MetadataBatchDeleteGroup:
		return MetadataDeleteGroup, "group", nil
	case MetadataBatchDeleteScene:
		return MetadataDeleteScene, "scene", nil
	case MetadataBatchDeleteAutomation:
		return MetadataDeleteAutomation, "automation", nil
	default:
		return "", "", fmt.Errorf("unsupported metadata batch delete kind %q", kind)
	}
}
