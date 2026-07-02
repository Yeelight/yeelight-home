package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
	Warning    string `json:"warning,omitempty"`
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
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
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
		return HomeOrganizationResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			Capability: string(request.Kind),
			ItemCount:  homeOrganizationItemCount(request.Kind, request.Payload),
			Verified:   false,
			VerifiedBy: string(request.Kind) + "_read_after_write",
			Warning:    "write_verification_mismatch",
			APICalls:   apiCalls,
		}, nil
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
		normalized, warning := NormalizeHomeSortPayload(houseID, payload)
		if warning != "" {
			return 0, fmt.Errorf("%s", warning)
		}
		sortType := strings.TrimSpace(stringFromAny(normalized[semantic.FieldType]))
		target := strings.TrimSpace(stringFromAny(normalized[semantic.FieldTarget]))
		if sortType == "" || target == "" {
			return 0, fmt.Errorf("sort type and target are required")
		}
		items, ok := normalized[semantic.FieldItems].([]any)
		if !ok || len(items) == 0 {
			return 0, fmt.Errorf("sort items are required")
		}
		body, err := buildHomeSortAddBody(items)
		if err != nil {
			return 0, err
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
		body := mapWithoutKeys(payload, semantic.FieldFavoriteID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/insert", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite add returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteUpdate:
		favoriteID := strings.TrimSpace(stringFromAny(payload[semantic.FieldFavoriteID]))
		if favoriteID != "" {
			body := mapWithoutKeys(payload, semantic.FieldFavoriteID)
			response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/"+favoriteID+"/w/update", body, credentials)
			if err != nil {
				return 1, err
			}
			if !isBusinessOK(response) {
				return 1, fmt.Errorf("favorite update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
			}
			return 1, nil
		}
		body, calls, err := client.favoriteMergedUpdateBody(ctx, houseID, []map[string]any{payload}, credentials)
		if err != nil {
			return calls, err
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/batchupdate", body, credentials)
		if err != nil {
			return calls + 1, err
		}
		if !isBusinessOK(response) {
			return calls + 1, fmt.Errorf("favorite update fallback returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return calls + 1, nil
	case HomeOrganizationFavoriteDelete:
		favoriteID := strings.TrimSpace(stringFromAny(payload[semantic.FieldFavoriteID]))
		if favoriteID != "" {
			response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/"+favoriteID+"/w/delete", nil, credentials)
			if err != nil {
				return 1, err
			}
			if !isBusinessOK(response) {
				return 1, fmt.Errorf("favorite delete returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
			}
			return 1, nil
		}
		body := []any{mapWithoutKeys(payload, semantic.FieldDeleteTarget, semantic.FieldID)}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/batchdelete", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite delete fallback returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteBatchAdd:
		items, err := favoriteBatchItems(payload)
		if err != nil {
			return 0, err
		}
		body := make([]any, 0, len(items))
		for _, item := range items {
			body = append(body, mapWithoutKeys(item, semantic.FieldFavoriteID))
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/batchinsert", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("favorite batch add returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeOrganizationFavoriteBatchUpdate:
		items, err := favoriteBatchItems(payload)
		if err != nil {
			return 0, err
		}
		body, calls, err := client.favoriteMergedUpdateBody(ctx, houseID, items, credentials)
		if err != nil {
			return calls, err
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/w/batchupdate", body, credentials)
		if err != nil {
			return calls + 1, err
		}
		if !isBusinessOK(response) {
			return calls + 1, fmt.Errorf("favorite batch update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return calls + 1, nil
	case HomeOrganizationFavoriteBatchDelete:
		items, err := favoriteBatchItems(payload)
		if err != nil {
			return 0, err
		}
		body := make([]any, 0, len(items))
		for _, item := range items {
			body = append(body, mapWithoutKeys(item, semantic.FieldDeleteTarget))
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

func buildHomeSortAddBody(items []any) ([]any, error) {
	body := make([]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("sort items are required")
		}
		targetType := firstNonNil(item[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)], item[semantic.FieldTargetTypeID], item[semantic.FieldResourceTypeID])
		if strings.TrimSpace(stringFromAny(targetType)) == "" {
			if typeID, ok := homeSortResourceType(item, true); ok {
				targetType = typeID
			}
		}
		targetID := firstNonNil(item[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)], item[semantic.FieldTargetID], item[semantic.FieldEntityID])
		if strings.TrimSpace(stringFromAny(targetID)) == "" {
			targetID = item[semantic.FieldID]
		}
		rank := item[semantic.FieldRank]
		if strings.TrimSpace(stringFromAny(targetType)) == "" || strings.TrimSpace(stringFromAny(targetID)) == "" || strings.TrimSpace(stringFromAny(rank)) == "" {
			return nil, fmt.Errorf("sort item target type, target id, and rank are required")
		}
		row := map[string]any{
			semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType): targetType,
			semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID):   targetID,
			semantic.FieldRank: rank,
		}
		if subIndex, ok := item[semantic.FieldSubIndex]; ok {
			row["subIndex"] = subIndex
		}
		body = append(body, row)
	}
	return body, nil
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
	body, warning := NormalizeHomeSortPayload(houseID, payload)
	if warning != "" {
		return false, 0, fmt.Errorf("%s", warning)
	}
	apiCalls := 0
	if ok, calls, verified, err := client.readSortBySpecificEndpoint(ctx, houseID, body, payload, credentials); verified {
		apiCalls += calls
		if err == nil {
			return ok, apiCalls, nil
		}
	}
	delete(body, semantic.FieldItems)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/sort/r/getSort", body, credentials)
	if err != nil {
		return false, apiCalls + 1, err
	}
	if !isBusinessOK(response) {
		return false, apiCalls + 1, fmt.Errorf("home sort list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return sortItemsPresent(response["data"], payload[semantic.FieldItems]), apiCalls + 1, nil
}

func (client HomeOrganizationClient) readSortBySpecificEndpoint(ctx context.Context, houseID string, normalized map[string]any, payload map[string]any, credentials requestCredentials) (bool, int, bool, error) {
	sortType := strings.TrimSpace(stringFromAny(normalized[semantic.FieldType]))
	target := strings.TrimSpace(stringFromAny(normalized[semantic.FieldTarget]))
	if sortType == "" || target == "" {
		return false, 0, false, nil
	}
	switch sortType {
	case "1":
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/node/r/1/"+pathSegment(target)+"/device", nil, credentials)
		if err != nil {
			return false, 1, true, err
		}
		if !isBusinessOK(response) {
			return false, 1, true, fmt.Errorf("node.sorted_device.list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		data := response["data"]
		rows := projectSortedDeviceRows(data)
		if enriched, err := metadataReadonlyFromHomeOrganization(client).enrichSortedDeviceRows(ctx, houseID, rows, MetadataReadonlyCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		}); err == nil {
			data = enriched
		}
		return nodeSortItemsPresent(data, payload[semantic.FieldItems]), 1, true, nil
	case "2":
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/sort/r/room/scene", map[string]any{
			semantic.FieldIDs: []any{requestNumberOrStringForAPI(target)},
		}, credentials)
		if err != nil {
			return false, 1, true, err
		}
		if !isBusinessOK(response) {
			return false, 1, true, fmt.Errorf("room scene sort list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sceneSortItemsPresent(response["data"], target, payload[semantic.FieldItems]), 1, true, nil
	default:
		return false, 0, false, nil
	}
}

func (client HomeOrganizationClient) readFavorites(ctx context.Context, houseID string, credentials requestCredentials) (any, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/favourite/r/all", map[string]any{semantic.FieldHouseID: houseID}, credentials)
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
	rows := favoriteRowsFromData(data)
	return favoriteRowsContain(rows, payload), calls, nil
}

func homeOrganizationItemCount(kind HomeOrganizationKind, payload map[string]any) int {
	switch kind {
	case HomeOrganizationSortConfigure:
		if items, ok := payload[semantic.FieldItems].([]any); ok {
			return len(items)
		}
	case HomeOrganizationFavoriteBatchAdd, HomeOrganizationFavoriteBatchUpdate, HomeOrganizationFavoriteBatchDelete:
		if items, ok := payload[semantic.FieldItems].([]any); ok {
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

func sceneSortItemsPresent(data any, target string, expected any) bool {
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
		resID := strings.TrimSpace(stringFromAny(expectedMap[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)]))
		rank := strings.TrimSpace(stringFromAny(expectedMap[semantic.FieldRank]))
		if resID == "" || rank == "" {
			return false
		}
		matched := false
		for _, row := range rows {
			item, ok := row.(map[string]any)
			if !ok {
				continue
			}
			if roomID := strings.TrimSpace(firstAnyString(item, semantic.FieldRoomID)); roomID != "" && roomID != target {
				continue
			}
			sceneOrder, ok := item["sceneOrder"].(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(stringFromAny(sceneOrder[resID])) == rank {
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

func nodeSortItemsPresent(data any, expected any) bool {
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
		resID := strings.TrimSpace(stringFromAny(expectedMap[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)]))
		rank := strings.TrimSpace(stringFromAny(expectedMap[semantic.FieldRank]))
		if resID == "" || rank == "" {
			return false
		}
		matched := false
		for _, row := range rows {
			item, ok := row.(map[string]any)
			if !ok {
				continue
			}
			if nodeSortRowID(item) == resID && strings.TrimSpace(firstAnyString(item, semantic.FieldRank)) == rank {
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

func nodeSortRowID(item map[string]any) string {
	return firstAnyString(item, semantic.FieldTargetID, semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID), semantic.FieldDeviceID, semantic.MeshGroupIDField(), semantic.FieldMeshGroupID, semantic.FieldGroupID, semantic.FieldID)
}

func sortItemMatches(item map[string]any, expected map[string]any) bool {
	for _, key := range []string{semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType), semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID), semantic.FieldRank, semantic.FieldSubIndex} {
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
