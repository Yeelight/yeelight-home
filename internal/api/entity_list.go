package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	entityListPageSize             = 100
	houseScopedEntityListCallCount = 6
)

type EntityListCredentials struct {
	Authorization string
	ClientID      string
}

type EntityListRequest struct {
	HouseID     string
	Credentials EntityListCredentials
}

type EntitySummary struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	HouseID string `json:"houseId,omitempty"`
	RoomID  string `json:"roomId,omitempty"`
	Online  *bool  `json:"online,omitempty"`
	Bind    *bool  `json:"bind,omitempty"`
	Virtual *bool  `json:"virtual,omitempty"`
	Status  string `json:"status,omitempty"`
}

type EntityListResult struct {
	Region   string          `json:"region"`
	HouseID  string          `json:"houseId,omitempty"`
	Total    int             `json:"total"`
	Counts   map[string]int  `json:"counts"`
	Entities []EntitySummary `json:"entities"`
	APICalls int             `json:"apiCalls"`
	Partial  bool            `json:"partial,omitempty"`
	Warnings []string        `json:"warnings,omitempty"`
}

type EntityListClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewEntityListClient(endpoint Endpoint, client *http.Client) EntityListClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return EntityListClient{endpoint: endpoint, client: client}
}

func HouseScopedEntityListCallCount() int {
	return houseScopedEntityListCallCount
}

func (client EntityListClient) Run(ctx context.Context, request EntityListRequest) (EntityListResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if houseID == "" {
		return client.listHomes(ctx, credentials)
	}
	result := EntityListResult{
		Region:   client.endpoint.Region,
		HouseID:  houseID,
		Counts:   map[string]int{},
		Entities: []EntitySummary{},
		Warnings: []string{},
	}
	for _, call := range houseScopedEntityListCalls(houseID) {
		entities, apiCalls, err := listHouseScopedEntityPages(ctx, client.client, client.endpoint, houseID, credentials, call)
		result.APICalls += apiCalls
		if err != nil {
			return result, err
		}
		result.Entities = append(result.Entities, entities...)
		result.Counts[call.entityType] += len(entities)
	}
	result.Total = len(result.Entities)
	return result, nil
}

func (client EntityListClient) listHomes(ctx context.Context, credentials requestCredentials) (EntityListResult, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/list", map[string]any{}, credentials)
	if err != nil {
		return EntityListResult{}, err
	}
	if !isBusinessOK(response) {
		return EntityListResult{}, fmt.Errorf("home list returned non-success business response")
	}
	entities := projectEntities("home", "", response)
	return EntityListResult{
		Region:   client.endpoint.Region,
		Total:    len(entities),
		Counts:   map[string]int{"home": len(entities)},
		Entities: entities,
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

type entityListCall struct {
	entityType  string
	method      string
	pathPattern string
	body        map[string]any
	singlePage  bool
}

func houseScopedEntityListCalls(houseID string) []entityListCall {
	return []entityListCall{
		{entityType: "area", method: http.MethodGet, pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/area/r/info/{pageNo}/%d", houseID, entityListPageSize)},
		{entityType: "room", method: http.MethodGet, pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/room/r/info/{pageNo}/%d", houseID, entityListPageSize)},
		{entityType: "device", method: http.MethodPost, pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/device/r/info/{pageNo}/%d", houseID, entityListPageSize), body: map[string]any{}},
		{entityType: "group", method: http.MethodGet, pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/group/r/info/{pageNo}/%d", houseID, entityListPageSize)},
		{entityType: "scene", method: http.MethodPost, pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/scene/r/info/{pageNo}/%d", houseID, entityListPageSize), body: map[string]any{}},
		{entityType: "automation", method: http.MethodPost, pathPattern: "/v1/automations/r/list", body: map[string]any{"houseId": houseID}, singlePage: true},
	}
}

func listHouseScopedEntityPages(ctx context.Context, httpClient *http.Client, endpoint Endpoint, houseID string, credentials requestCredentials, call entityListCall) ([]EntitySummary, int, error) {
	entities := []EntitySummary{}
	apiCalls := 0
	for pageNo := 1; ; pageNo++ {
		path := strings.ReplaceAll(call.pathPattern, "{pageNo}", fmt.Sprintf("%d", pageNo))
		response, err := callJSON(ctx, httpClient, call.method, strings.TrimRight(endpoint.BaseURL, "/")+path, call.body, credentials)
		apiCalls++
		if err != nil {
			return nil, apiCalls, err
		}
		if !isBusinessOK(response) {
			return nil, apiCalls, fmt.Errorf("%s list returned non-success business response: entityType=%s code=%s message=%s dataType=%s", call.entityType, call.entityType, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		pageEntities := projectEntities(call.entityType, houseID, response)
		entities = append(entities, pageEntities...)
		if call.singlePage || len(pageEntities) < entityListPageSize {
			break
		}
	}
	return entities, apiCalls, nil
}

func projectEntities(entityType string, houseID string, response map[string]any) []EntitySummary {
	rows := extractRows(response)
	entities := make([]EntitySummary, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		entity := EntitySummary{
			Type:    entityType,
			ID:      firstAnyString(item, "id", "houseId", "areaId", "roomId", "groupId", "sceneId", "automationId", "deviceId"),
			Name:    firstAnyString(item, "name", "houseName", "areaName", "roomName", "groupName", "sceneName", "automationName"),
			HouseID: firstNonEmpty(firstAnyString(item, "houseId"), houseID),
			RoomID:  firstAnyString(item, "roomId"),
			Status:  firstAnyString(item, "status"),
		}
		if online, ok := item["online"].(bool); ok {
			entity.Online = &online
		}
		if bind, ok := item["bind"].(bool); ok {
			entity.Bind = &bind
		}
		if virtual, ok := item["virtual"].(bool); ok {
			entity.Virtual = &virtual
		}
		entities = append(entities, entity)
	}
	return entities
}

func extractRows(response map[string]any) []any {
	data, ok := response["data"]
	if !ok {
		return []any{}
	}
	switch value := data.(type) {
	case []any:
		return value
	case map[string]any:
		if rows, ok := value["rows"].([]any); ok {
			return rows
		}
		if list, ok := value["list"].([]any); ok {
			return list
		}
	}
	return []any{}
}

func firstAnyString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case float64:
			return fmt.Sprintf("%.0f", typed)
		case int:
			return fmt.Sprintf("%d", typed)
		case int64:
			return fmt.Sprintf("%d", typed)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
