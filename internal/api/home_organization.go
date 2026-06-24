package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HomeOrganizationKind string

const (
	HomeOrganizationSortConfigure       HomeOrganizationKind = "home.sort.configure"
	HomeOrganizationFavoriteAdd         HomeOrganizationKind = "favorite.add"
	HomeOrganizationFavoriteUpdate      HomeOrganizationKind = "favorite.update"
	HomeOrganizationFavoriteDelete      HomeOrganizationKind = "favorite.delete"
	HomeOrganizationFavoriteBatchAdd    HomeOrganizationKind = "favorite.batch_add"
	HomeOrganizationFavoriteBatchUpdate HomeOrganizationKind = "favorite.batch_update"
	HomeOrganizationFavoriteBatchDelete HomeOrganizationKind = "favorite.batch_delete"
)

type HomeOrganizationCredentials struct {
	Authorization string
	ClientID      string
}

type HomeOrganizationRequest struct {
	Kind           HomeOrganizationKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    HomeOrganizationCredentials
}

type HomeOrganizationResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	ItemCount  int    `json:"itemCount,omitempty"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type HomeOrganizationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeOrganizationClient(endpoint Endpoint, client *http.Client) HomeOrganizationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeOrganizationClient{endpoint: endpoint, client: client}
}

func (client HomeOrganizationClient) Run(ctx context.Context, request HomeOrganizationRequest) (HomeOrganizationResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return HomeOrganizationResult{}, fmt.Errorf("house id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return HomeOrganizationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	verifyCalls, err := client.verifyBeforeWrite(ctx, request.Kind, houseID, request.Payload, credentials)
	apiCalls += verifyCalls
	if err != nil {
		return HomeOrganizationResult{}, err
	}
	writeCalls, err := client.write(ctx, request.Kind, houseID, request.Payload, credentials)
	if err != nil {
		return HomeOrganizationResult{}, err
	}
	apiCalls += writeCalls
	ok, verifyCalls, err := client.verifyAfterWrite(ctx, request.Kind, houseID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return HomeOrganizationResult{}, err
	}
	if !ok {
		return HomeOrganizationResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return HomeOrganizationResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: string(request.Kind),
		ItemCount:  homeOrganizationItemCount(request.Kind, request.Payload),
		Verified:   true,
		VerifiedBy: string(request.Kind) + "_read_after_write",
		APICalls:   apiCalls,
	}, nil
}

func (client HomeOrganizationClient) verifyBeforeWrite(ctx context.Context, kind HomeOrganizationKind, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	switch kind {
	case HomeOrganizationSortConfigure:
		_, calls, err := client.readSort(ctx, houseID, payload, credentials)
		return calls, err
	case HomeOrganizationFavoriteAdd, HomeOrganizationFavoriteUpdate, HomeOrganizationFavoriteDelete,
		HomeOrganizationFavoriteBatchAdd, HomeOrganizationFavoriteBatchUpdate, HomeOrganizationFavoriteBatchDelete:
		_, calls, err := client.readFavorites(ctx, houseID, credentials)
		return calls, err
	default:
		return 0, fmt.Errorf("unsupported home organization kind %q", kind)
	}
}

func (client HomeOrganizationClient) write(ctx context.Context, kind HomeOrganizationKind, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	switch kind {
	case HomeOrganizationSortConfigure:
		sortType := strings.TrimSpace(stringFromAny(payload["type"]))
		target := strings.TrimSpace(stringFromAny(payload["target"]))
		if sortType == "" || target == "" {
			return 0, fmt.Errorf("sort type and target are required")
		}
		body, ok := payload["items"].([]any)
		if !ok || len(body) == 0 {
			return 0, fmt.Errorf("sort items are required")
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/sort/"+houseID+"/w/"+sortType+"/"+target+"/add", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("home sort configure returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteAdd:
		body := mapWithoutKeys(payload, "favoriteId")
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/insert", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite add returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteUpdate:
		favoriteID := strings.TrimSpace(stringFromAny(payload["favoriteId"]))
		if favoriteID == "" {
			return 0, fmt.Errorf("favorite id is required")
		}
		body := mapWithoutKeys(payload, "favoriteId")
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/"+favoriteID+"/w/update", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteDelete:
		favoriteID := strings.TrimSpace(stringFromAny(payload["favoriteId"]))
		if favoriteID == "" {
			return 0, fmt.Errorf("favorite id is required")
		}
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/"+favoriteID+"/w/delete", nil, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite delete returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteBatchAdd, HomeOrganizationFavoriteBatchUpdate:
		items, err := favoriteBatchItems(payload)
		if err != nil {
			return 0, err
		}
		calls := 0
		for _, item := range items {
			if kind == HomeOrganizationFavoriteBatchAdd {
				writeCalls, err := client.write(ctx, HomeOrganizationFavoriteAdd, houseID, item, credentials)
				calls += writeCalls
				if err != nil {
					return calls, err
				}
				continue
			}
			writeCalls, err := client.write(ctx, HomeOrganizationFavoriteUpdate, houseID, item, credentials)
			calls += writeCalls
			if err != nil {
				return calls, err
			}
		}
		return calls, nil
	case HomeOrganizationFavoriteBatchDelete:
		items, err := favoriteBatchItems(payload)
		if err != nil {
			return 0, err
		}
		body := make([]any, 0, len(items))
		for _, item := range items {
			body = append(body, mapWithoutKeys(item, "deleteTarget"))
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/batchdelete", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite batch delete returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported home organization kind %q", kind)
	}
}

func (client HomeOrganizationClient) verifyAfterWrite(ctx context.Context, kind HomeOrganizationKind, houseID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		var ok bool
		var readCalls int
		var err error
		switch kind {
		case HomeOrganizationSortConfigure:
			ok, readCalls, err = client.readSort(ctx, houseID, payload, credentials)
		case HomeOrganizationFavoriteAdd, HomeOrganizationFavoriteUpdate:
			ok, readCalls, err = client.favoriteExists(ctx, houseID, payload, credentials)
		case HomeOrganizationFavoriteDelete:
			ok, readCalls, err = client.favoriteMissing(ctx, houseID, payload, credentials)
		case HomeOrganizationFavoriteBatchAdd, HomeOrganizationFavoriteBatchUpdate:
			ok, readCalls, err = client.favoriteBatchExists(ctx, houseID, payload, credentials)
		case HomeOrganizationFavoriteBatchDelete:
			ok, readCalls, err = client.favoriteBatchMissing(ctx, houseID, payload, credentials)
		default:
			return false, calls, fmt.Errorf("unsupported home organization kind %q", kind)
		}
		calls += readCalls
		if err != nil || ok || attempt == attempts-1 {
			return ok, calls, err
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

func (client HomeOrganizationClient) readSort(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	body := map[string]any{"houseId": houseID}
	for _, key := range []string{"typeId", "resId", "roomId", "type", "target", "subIndex"} {
		if value, ok := payload[key]; ok {
			body[key] = value
		}
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/sort/r/getSort", body, credentials)
	if err != nil {
		return false, 1, err
	}
	if !isBusinessOK(response) {
		return false, 1, fmt.Errorf("home sort list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return sortItemsPresent(response["data"], payload["items"]), 1, nil
}

func (client HomeOrganizationClient) readFavorites(ctx context.Context, houseID string, credentials requestCredentials) (any, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/r/all", map[string]any{"houseId": houseID}, credentials)
	if err != nil {
		return nil, 1, err
	}
	if !isBusinessOK(response) {
		return nil, 1, fmt.Errorf("favorite list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return response["data"], 1, nil
}

func (client HomeOrganizationClient) favoriteExists(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	rows := rowsFromData(data)
	return favoriteRowsContain(rows, payload), calls, nil
}

func homeOrganizationItemCount(kind HomeOrganizationKind, payload map[string]any) int {
	switch kind {
	case HomeOrganizationSortConfigure:
		if items, ok := payload["items"].([]any); ok {
			return len(items)
		}
	case HomeOrganizationFavoriteBatchAdd, HomeOrganizationFavoriteBatchUpdate, HomeOrganizationFavoriteBatchDelete:
		if items, ok := payload["items"].([]any); ok {
			return len(items)
		}
	case HomeOrganizationFavoriteAdd, HomeOrganizationFavoriteUpdate, HomeOrganizationFavoriteDelete:
		return 1
	}
	return 0
}

func sortItemsPresent(data any, expected any) bool {
	expectedRows, ok := expected.([]any)
	if !ok || len(expectedRows) == 0 {
		return false
	}
	rows := rowsFromData(data)
	if len(rows) == 0 {
		return false
	}
	for _, expectedRow := range expectedRows {
		expectedMap, ok := expectedRow.(map[string]any)
		if !ok {
			return false
		}
		matched := false
		for _, row := range rows {
			item, ok := row.(map[string]any)
			if !ok {
				continue
			}
			if sortItemMatches(item, expectedMap) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func sortItemMatches(item map[string]any, expected map[string]any) bool {
	for _, key := range []string{"typeId", "resId", "rank", "subIndex"} {
		expectedValue, ok := expected[key]
		if !ok {
			continue
		}
		expectedText := strings.TrimSpace(stringFromAny(expectedValue))
		if expectedText != "" && firstAnyString(item, key) != expectedText {
			return false
		}
	}
	return true
}

func mapWithoutKeys(source map[string]any, keys ...string) map[string]any {
	excluded := map[string]bool{}
	for _, key := range keys {
		excluded[key] = true
	}
	result := map[string]any{}
	for key, value := range source {
		if !excluded[key] {
			result[key] = value
		}
	}
	return result
}
