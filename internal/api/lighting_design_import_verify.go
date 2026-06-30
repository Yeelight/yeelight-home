package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (client LightingDesignImportClient) verify(ctx context.Context, houseID string, credentials requestCredentials, counts map[string]int, attempts int, interval time.Duration) (bool, EntityListResult, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	totalCalls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entities, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
			HouseID: houseID,
			Credentials: EntityListCredentials{
				Authorization: credentials.Authorization,
				ClientID:      credentials.ClientID,
			},
		})
		totalCalls += HouseScopedEntityListCallCount()
		if err != nil || lightingDesignVerificationPasses(entities, counts) || attempt == attempts-1 {
			return err == nil && lightingDesignVerificationPasses(entities, counts), entities, totalCalls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, entities, totalCalls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, EntityListResult{}, totalCalls, nil
}

func (client LightingDesignImportClient) waitForMetaImport(ctx context.Context, requestKey string, credentials requestCredentials, attempts int, interval time.Duration) (map[string]any, int, error) {
	if attempts <= 0 {
		attempts = 5
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	statusURL := strings.TrimRight(client.endpoint.BaseURL, "/") + "/v1/meta/status?requestKey=" + requestKey
	for attempt := 0; attempt < attempts; attempt++ {
		response, err := callJSONBody(ctx, client.client, http.MethodGet, statusURL, nil, credentials)
		if err != nil {
			return nil, attempt + 1, err
		}
		if !isBusinessOK(response) {
			return nil, attempt + 1, fmt.Errorf("lighting.design.import status returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		status, _ := mapFromAny(response["data"])
		switch lightingDesignMetaImportStatus(status) {
		case "1":
			return status, attempt + 1, nil
		case "-1":
			reason := firstNonEmpty(stringFromMap(status, "reason"), "家庭元数据导入失败")
			return status, attempt + 1, fmt.Errorf("lighting.design.import failed: %s", reason)
		}
		if attempt == attempts-1 {
			return status, attempt + 1, fmt.Errorf("lighting.design.import status still importing after %d attempts", attempts)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return status, attempt + 1, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, attempts, nil
}

func lightingDesignVerificationPasses(entities EntityListResult, counts map[string]int) bool {
	if counts["rooms"] > 0 && entities.Counts["room"] <= 0 {
		return false
	}
	if counts["devices"] > 0 && entities.Counts["device"] <= 0 {
		return false
	}
	if counts["groups"] > 0 && entities.Counts["group"] <= 0 {
		return false
	}
	if counts["scenes"] > 0 && entities.Counts["scene"] <= 0 {
		return false
	}
	return true
}

func lightingDesignMetaImportRequestKey(data any) string {
	switch typed := data.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return firstNonEmpty(stringFromMap(typed, "requestKey"), stringFromMap(typed, "key"))
	default:
		return lightingDesignStringFromAny(typed)
	}
}

func lightingDesignMetaImportStatus(data map[string]any) string {
	return firstNonEmpty(stringFromMap(data, "status"), stringFromMap(data, "state"))
}

func lightingDesignMetaImportHouseID(data map[string]any) string {
	return firstNonEmpty(stringFromMap(data, "houseId"), stringFromMap(data, "houseID"))
}

func lightingDesignImportMode(payload map[string]any) string {
	if _, ok := mapFromAny(payload["gateway"]); ok {
		return "house_meta_import"
	}
	return "house_meta_import_unknown"
}

func lightingDesignImportCounts(payload map[string]any) map[string]int {
	counts := map[string]int{
		"gateways":    0,
		"rooms":       0,
		"devices":     0,
		"groups":      0,
		"areas":       len(anyListFromMap(payload, "areaList")),
		"scenes":      len(anyListFromMap(payload, "sceneList")),
		"automations": len(anyListFromMap(payload, "automationList")),
	}
	gateway, ok := mapFromAny(payload["gateway"])
	if !ok {
		return counts
	}
	counts["gateways"] = 1
	rooms, _ := mapListFromAny(gateway["roomList"])
	counts["rooms"] = len(rooms)
	for _, room := range rooms {
		counts["devices"] += len(anyListFromMap(room, "deviceList"))
		counts["groups"] += len(anyListFromMap(room, "groupList"))
	}
	return counts
}

func lightingDesignImportMappings(any) map[string]any {
	return nil
}
