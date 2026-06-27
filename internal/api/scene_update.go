package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
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
		Name:       firstAnyString(verified, "name", "sceneName"),
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
	body["id"] = requestNumberOrStringForAPI(sceneID)
	body["houseId"] = requestNumberOrStringForAPI(houseID)
	body["details"] = normalizeSceneUpdateDetails(body["details"])
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
	if params, ok := copied["params"]; ok {
		if compact, err := jsonString(params); err == nil {
			copied["params"] = compact
		}
	}
	if strings.TrimSpace(stringFromAny(copied["resName"])) == "" {
		copied["resName"] = stringFromAny(copied["resId"])
	}
	if strings.TrimSpace(stringFromAny(copied["action"])) == "" {
		copied["action"] = 0
	}
	if strings.TrimSpace(stringFromAny(copied["rank"])) == "" {
		copied["rank"] = 0
	}
	return copied
}

func sceneUpdateMatches(detail map[string]any, payload map[string]any) bool {
	if len(detail) == 0 {
		return false
	}
	expectedName := strings.TrimSpace(stringFromAny(payload["name"]))
	if expectedName != "" && firstAnyString(detail, "name", "sceneName") != expectedName {
		return false
	}
	if expectedDetails, ok := payload["details"]; ok {
		expectedDetails = normalizeSceneUpdateDetails(expectedDetails)
		return configRowsContainExpected(detail["details"], expectedDetails, []string{"typeId", "resId", "rank", "action", "params"})
	}
	return true
}

func sceneIDFromDetail(detail map[string]any) string {
	return firstAnyString(detail, "sceneId", "id")
}
