package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type SceneUpdateCredentials struct {
	Authorization string
	ClientID      string
}

type SceneUpdateRequest struct {
	HouseID        string
	SceneID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    SceneUpdateCredentials
}

type SceneUpdateResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	SceneID    string `json:"sceneId"`
	Name       string `json:"name,omitempty"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type SceneUpdateClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewSceneUpdateClient(endpoint Endpoint, client *http.Client) SceneUpdateClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SceneUpdateClient{endpoint: endpoint, client: client}
}

func (client SceneUpdateClient) Run(ctx context.Context, request SceneUpdateRequest) (SceneUpdateResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return SceneUpdateResult{}, fmt.Errorf("house id is required")
	}
	sceneID := strings.TrimSpace(request.SceneID)
	if sceneID == "" {
		return SceneUpdateResult{}, fmt.Errorf("scene id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return SceneUpdateResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	before, preflightCalls, err := findSceneByID(ctx, client.endpoint, client.client, houseID, sceneID, credentials)
	apiCalls += preflightCalls
	if err != nil {
		return SceneUpdateResult{}, err
	}
	if before.ID == "" {
		return SceneUpdateResult{}, fmt.Errorf("scene %s not found before write", sceneID)
	}
	body := copySceneUpdatePayload(request.Payload, sceneID, houseID)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/scene/"+pathSegment(sceneID)+"/w/modify", body, credentials)
	apiCalls++
	if err != nil {
		return SceneUpdateResult{}, err
	}
	if !isBusinessOK(response) {
		return SceneUpdateResult{}, fmt.Errorf("scene.update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	verified, verifyCalls, err := client.verifyAfterWrite(ctx, houseID, sceneID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return SceneUpdateResult{}, err
	}
	if !sceneUpdateMatches(verified, request.Payload) {
		return SceneUpdateResult{}, fmt.Errorf("scene.update write verification mismatch")
	}
	return SceneUpdateResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		SceneID:    sceneIDFromDetail(verified),
		Name:       firstAnyString(verified, semantic.FieldName, semantic.FieldSceneName),
		Verified:   true,
		VerifiedBy: "scene.detail.get",
		APICalls:   apiCalls,
	}, nil
}

func (client SceneUpdateClient) verifyAfterWrite(ctx context.Context, houseID string, sceneID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (map[string]any, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		detail, err := readSceneDetail(ctx, client.endpoint, client.client, sceneID, credentials)
		calls++
		if err != nil {
			return nil, calls, err
		}
		if sceneUpdateMatches(detail, payload) {
			return detail, calls, nil
		}
		if attempt == attempts-1 {
			return detail, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, calls, nil
}

func findSceneByID(ctx context.Context, endpoint Endpoint, httpClient *http.Client, houseID string, sceneID string, credentials requestCredentials) (EntitySummary, int, error) {
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
		if entity.Type == "scene" && entity.ID == sceneID {
			return entity, result.APICalls, nil
		}
	}
	return EntitySummary{}, result.APICalls, nil
}

func readSceneDetail(ctx context.Context, endpoint Endpoint, httpClient *http.Client, sceneID string, credentials requestCredentials) (map[string]any, error) {
	response, err := callJSON(ctx, httpClient, http.MethodPost, strings.TrimRight(endpoint.BaseURL, "/")+"/v1/scene/"+pathSegment(sceneID)+"/r/detail", nil, credentials)
	if err != nil {
		return nil, err
	}
	if !isBusinessOK(response) {
		return nil, metadataReadonlyBusinessError("scene detail", response)
	}
	data, _ := response["data"].(map[string]any)
	return data, nil
}

func copySceneUpdatePayload(payload map[string]any, sceneID string, houseID string) map[string]any {
	body := map[string]any{}
	for key, value := range payload {
		body[key] = value
	}
	body[semantic.FieldID] = requestNumberOrStringForAPI(sceneID)
	body[semantic.FieldHouseID] = requestNumberOrStringForAPI(houseID)
	body[semantic.FieldDetails] = normalizeSceneUpdateDetails(body[semantic.FieldDetails])
	return body
}

func normalizeSceneUpdateDetails(value any) any {
	switch rows := value.(type) {
	case []map[string]any:
		normalized := make([]any, 0, len(rows))
		for _, item := range rows {
			normalized = append(normalized, normalizeSceneUpdateDetail(item))
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(rows))
		for _, raw := range rows {
			item, ok := raw.(map[string]any)
			if !ok {
				normalized = append(normalized, raw)
				continue
			}
			normalized = append(normalized, normalizeSceneUpdateDetail(item))
		}
		return normalized
	default:
		return value
	}
}

func normalizeSceneUpdateDetail(item map[string]any) map[string]any {
	copied := map[string]any{}
	for key, itemValue := range item {
		copied[key] = itemValue
	}
	paramsField := semantic.InternalActionParamsField()
	if params, ok := copied[paramsField]; ok {
		if compact, err := jsonString(params); err == nil {
			copied[paramsField] = compact
		}
	}
	targetNameField := semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)
	targetIDField := semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)
	if strings.TrimSpace(stringFromAny(copied[targetNameField])) == "" {
		copied[targetNameField] = stringFromAny(copied[targetIDField])
	}
	if strings.TrimSpace(stringFromAny(copied[semantic.FieldAction])) == "" {
		copied[semantic.FieldAction] = 0
	}
	if strings.TrimSpace(stringFromAny(copied[semantic.FieldRank])) == "" {
		copied[semantic.FieldRank] = 0
	}
	return copied
}

func sceneUpdateMatches(detail map[string]any, payload map[string]any) bool {
	if len(detail) == 0 {
		return false
	}
	expectedName := strings.TrimSpace(stringFromAny(payload[semantic.FieldName]))
	if expectedName != "" && firstAnyString(detail, semantic.FieldName, semantic.FieldSceneName) != expectedName {
		return false
	}
	if expectedDetails, ok := payload[semantic.FieldDetails]; ok {
		expectedDetails = normalizeSceneUpdateDetails(expectedDetails)
		return sceneUpdateRowsContainExpected(detail[semantic.FieldDetails], expectedDetails)
	}
	return true
}

func sceneIDFromDetail(detail map[string]any) string {
	return firstAnyString(detail, semantic.FieldSceneID, semantic.FieldID)
}

func sceneUpdateRowsContainExpected(actual any, expected any) bool {
	expectedRows := sceneUpdateRowsFromData(expected)
	if len(expectedRows) == 0 {
		return false
	}
	actualRows := sceneUpdateRowsFromData(actual)
	if len(actualRows) == 0 {
		return false
	}
	for _, expectedRow := range expectedRows {
		matched := false
		for _, actualRow := range actualRows {
			if sceneUpdateRowMatches(actualRow, expectedRow) {
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

func sceneUpdateRowsFromData(value any) []map[string]any {
	rawRows := configRowsFromData(value)
	rows := make([]map[string]any, 0, len(rawRows))
	for _, raw := range rawRows {
		if row, ok := raw.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func sceneUpdateRowMatches(actual map[string]any, expected map[string]any) bool {
	if !sceneUpdateSameTargetType(sceneUpdateRowTargetType(actual), sceneUpdateRowTargetType(expected)) {
		return false
	}
	if !sceneUpdateSameText(sceneUpdateRowTargetID(actual), sceneUpdateRowTargetID(expected)) {
		return false
	}
	if !sceneUpdateSameText(sceneUpdateRowScalar(actual, semantic.FieldRank), sceneUpdateRowScalar(expected, semantic.FieldRank)) {
		return false
	}
	if !sceneUpdateSameText(sceneUpdateRowScalar(actual, semantic.FieldAction), sceneUpdateRowScalar(expected, semantic.FieldAction)) {
		return false
	}
	return sceneUpdateSameParams(sceneUpdateRowParams(actual), sceneUpdateRowParams(expected))
}

func sceneUpdateRowTargetType(row map[string]any) string {
	value := firstPresentAny(row, semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType), semantic.FieldTargetType)
	text := strings.TrimSpace(stringFromAny(value))
	if text == "" {
		return ""
	}
	if parsed, err := strconv.Atoi(text); err == nil {
		return strconv.Itoa(parsed)
	}
	if typeID, ok := semantic.TargetTypeID(text, semantic.ResourceMeshGroup); ok {
		return strconv.Itoa(typeID)
	}
	return text
}

func sceneUpdateRowTargetID(row map[string]any) string {
	return strings.TrimSpace(stringFromAny(firstPresentAny(row, semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID), semantic.FieldTargetID)))
}

func sceneUpdateRowScalar(row map[string]any, key string) string {
	text := strings.TrimSpace(stringFromAny(row[key]))
	if text == "" && (key == semantic.FieldRank || key == semantic.FieldAction) {
		return "0"
	}
	return text
}

func sceneUpdateRowParams(row map[string]any) any {
	if value, ok := row[semantic.InternalActionParamsField()]; ok {
		return value
	}
	if params, ok := semantic.ActionParamsFromRow(row); ok {
		return params
	}
	return nil
}

func sceneUpdateSameTargetType(actual string, expected string) bool {
	return expected == "" || actual == expected
}

func sceneUpdateSameText(actual string, expected string) bool {
	return expected == "" || actual == expected
}

func sceneUpdateSameParams(actual any, expected any) bool {
	expectedJSON := sceneUpdateCanonicalParams(expected)
	if expectedJSON == "" {
		return true
	}
	return expectedJSON == sceneUpdateCanonicalParams(actual)
}

func sceneUpdateCanonicalParams(value any) string {
	normalized, ok := sceneUpdateNormalizeParamValue(value)
	if !ok {
		return ""
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(data)
}

func sceneUpdateNormalizeParamValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, false
		}
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return sceneUpdateNormalizeParamValue(decoded)
		}
		return trimmed, true
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if key == semantic.FieldSet {
				if set, ok := item.(map[string]any); ok {
					result[key] = sceneUpdateNormalizeSet(set)
					continue
				}
			}
			if normalized, ok := sceneUpdateNormalizeParamValue(item); ok {
				result[key] = normalized
			} else {
				result[key] = item
			}
		}
		return result, true
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			if normalized, ok := sceneUpdateNormalizeParamValue(item); ok {
				result = append(result, normalized)
			} else {
				result = append(result, item)
			}
		}
		return result, true
	default:
		return typed, true
	}
}

func sceneUpdateNormalizeSet(set map[string]any) map[string]any {
	result := make(map[string]any, len(set))
	for key, value := range set {
		result[sceneUpdateSetKey(key)] = value
	}
	return result
}

func sceneUpdateSetKey(key string) string {
	switch key {
	case semantic.FieldPower, semantic.InternalField(semantic.DomainAction, semantic.FieldPower):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldPower)
	case semantic.FieldBrightness, semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness)
	case semantic.FieldColorTemperature, semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature)
	case semantic.FieldColor, semantic.InternalField(semantic.DomainAction, semantic.FieldColor):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldColor)
	case semantic.FieldTargetPercent, semantic.InternalField(semantic.DomainAction, semantic.FieldTargetPercent):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldTargetPercent)
	case semantic.FieldSwitchPower, semantic.InternalField(semantic.DomainAction, semantic.FieldSwitchPower):
		return semantic.InternalField(semantic.DomainAction, semantic.FieldSwitchPower)
	default:
		return key
	}
}

func firstPresentAny(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}
