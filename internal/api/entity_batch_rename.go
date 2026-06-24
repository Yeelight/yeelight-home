package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const entityBatchRenameLimit = 20

type EntityBatchRenameRequest struct {
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    SpaceOrganizationCredentials
}

type EntityBatchRenameResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	ItemCount  int    `json:"itemCount"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type EntityBatchRenameClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewEntityBatchRenameClient(endpoint Endpoint, client *http.Client) EntityBatchRenameClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return EntityBatchRenameClient{endpoint: endpoint, client: client}
}

func (client EntityBatchRenameClient) Run(ctx context.Context, request EntityBatchRenameRequest) (EntityBatchRenameResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return EntityBatchRenameResult{}, fmt.Errorf("house id is required")
	}
	items, err := entityBatchRenameItems(request.Payload)
	if err != nil {
		return EntityBatchRenameResult{}, err
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return EntityBatchRenameResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	entities, calls, err := client.listEntities(ctx, houseID, credentials)
	apiCalls += calls
	if err != nil {
		return EntityBatchRenameResult{}, err
	}
	if err := validateEntityBatchRenameItems(items, entities); err != nil {
		return EntityBatchRenameResult{}, err
	}
	if err := client.write(ctx, houseID, items, credentials); err != nil {
		return EntityBatchRenameResult{}, err
	}
	apiCalls++
	ok, calls, err := client.verify(ctx, houseID, items, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return EntityBatchRenameResult{}, err
	}
	if !ok {
		return EntityBatchRenameResult{}, fmt.Errorf("entity.rename.batch write verification mismatch")
	}
	return EntityBatchRenameResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "entity.rename.batch",
		ItemCount:  len(items),
		Verified:   true,
		VerifiedBy: "entity.list",
		APICalls:   apiCalls,
	}, nil
}

type entityBatchRenameItem struct {
	ID         string
	TypeID     int
	EntityType string
	Name       string
	Index      *int
}

func entityBatchRenameItems(payload map[string]any) ([]entityBatchRenameItem, error) {
	rawItems, ok := payload["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return nil, fmt.Errorf("entity rename batch items are required")
	}
	if len(rawItems) > entityBatchRenameLimit {
		return nil, fmt.Errorf("entity rename batch limit exceeded")
	}
	items := make([]entityBatchRenameItem, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid entity rename batch item")
		}
		typeID, ok := renameIntFromAny(item["typeId"])
		if !ok {
			return nil, fmt.Errorf("invalid entity rename resource type")
		}
		entityType, ok := entityTypeForRenameType(typeID)
		if !ok {
			return nil, fmt.Errorf("invalid entity rename resource type")
		}
		id := strings.TrimSpace(firstNonEmpty(stringFromAny(item["id"]), stringFromAny(item["entityId"]), stringFromAny(item["resId"])))
		name := strings.TrimSpace(firstNonEmpty(stringFromAny(item["name"]), stringFromAny(item["newName"])))
		if id == "" || name == "" {
			return nil, fmt.Errorf("invalid entity rename batch item")
		}
		renameItem := entityBatchRenameItem{ID: id, TypeID: typeID, EntityType: entityType, Name: name}
		if index, ok := renameIntFromAny(item["index"]); ok {
			renameItem.Index = &index
		}
		items = append(items, renameItem)
	}
	return items, nil
}

func renameIntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case string:
		var result int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &result); err != nil {
			return 0, false
		}
		return result, true
	default:
		return 0, false
	}
}

func entityTypeForRenameType(typeID int) (string, bool) {
	switch typeID {
	case 2:
		return "device", true
	case 6:
		return "scene", true
	default:
		return "", false
	}
}

func validateEntityBatchRenameItems(items []entityBatchRenameItem, entities EntityListResult) error {
	seen := map[string]bool{}
	namesByType := map[string]map[string]string{}
	for _, entity := range entities.Entities {
		if namesByType[entity.Type] == nil {
			namesByType[entity.Type] = map[string]string{}
		}
		namesByType[entity.Type][entity.Name] = entity.ID
	}
	for _, item := range items {
		key := item.EntityType + ":" + item.ID
		if seen[key] {
			return fmt.Errorf("duplicate entity rename target")
		}
		seen[key] = true
		current, ok := findSpaceEntity(entities, item.EntityType, item.ID)
		if !ok {
			return fmt.Errorf("%s %s not found before write", item.EntityType, item.ID)
		}
		if ownerID := namesByType[item.EntityType][item.Name]; ownerID != "" && ownerID != current.ID {
			return fmt.Errorf("%s name already exists", item.EntityType)
		}
	}
	return nil
}

func (client EntityBatchRenameClient) write(ctx context.Context, houseID string, items []entityBatchRenameItem, credentials requestCredentials) error {
	body := make([]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"id":     requestNumberOrStringForAPI(item.ID),
			"typeId": item.TypeID,
			"name":   item.Name,
		}
		if item.Index != nil {
			row["index"] = *item.Index
		}
		body = append(body, row)
	}
	response, err := callJSONBody(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/ai/"+pathSegment(houseID)+"/name/w/modify", body, credentials)
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("entity.rename.batch returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client EntityBatchRenameClient) verify(ctx context.Context, houseID string, items []entityBatchRenameItem, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entities, readCalls, err := client.listEntities(ctx, houseID, credentials)
		calls += readCalls
		if err != nil {
			return false, calls, err
		}
		if entityBatchRenameMatches(items, entities) || attempt == attempts-1 {
			return entityBatchRenameMatches(items, entities), calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, calls, nil
}

func (client EntityBatchRenameClient) listEntities(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntityListResult{}, result.APICalls, err
	}
	return result, result.APICalls, nil
}

func entityBatchRenameMatches(items []entityBatchRenameItem, entities EntityListResult) bool {
	for _, item := range items {
		entity, ok := findSpaceEntity(entities, item.EntityType, item.ID)
		if !ok || entity.Name != item.Name {
			return false
		}
	}
	return true
}

func sortedEntityBatchRenameItems(items []entityBatchRenameItem) []entityBatchRenameItem {
	result := append([]entityBatchRenameItem{}, items...)
	sort.Slice(result, func(left, right int) bool {
		if result[left].EntityType == result[right].EntityType {
			return result[left].ID < result[right].ID
		}
		return result[left].EntityType < result[right].EntityType
	})
	return result
}
