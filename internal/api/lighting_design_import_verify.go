package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
			reason := firstNonEmpty(stringFromMap(status, semantic.FieldReason), "家庭元数据导入失败")
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
	if expected := counts[semantic.FieldRooms]; expected > 0 && entities.Counts["room"] < expected {
		return false
	}
	if expected := counts[semantic.FieldDevices]; expected > 0 && entities.Counts["device"] < expected {
		return false
	}
	if expected := counts[semantic.FieldGroups]; expected > 0 && entities.Counts["group"] < expected {
		return false
	}
	if expected := counts[semantic.FieldAreas]; expected > 0 && entities.Counts["area"] < expected {
		return false
	}
	if expected := counts[semantic.FieldScenes]; expected > 0 && entities.Counts["scene"] < expected {
		return false
	}
	if expected := counts[semantic.FieldAutomations]; expected > 0 && entities.Counts["automation"] < expected {
		return false
	}
	return true
}

func lightingDesignMetaImportRequestKey(data any) string {
	switch typed := data.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return firstNonEmpty(stringFromMap(typed, semantic.FieldRequestKey), stringFromMap(typed, semantic.FieldKey))
	default:
		return lightingDesignStringFromAny(typed)
	}
}

func lightingDesignMetaImportStatus(data map[string]any) string {
	return firstNonEmpty(stringFromMap(data, semantic.FieldStatus), stringFromMap(data, semantic.InternalMetaImportStateField()))
}

func lightingDesignMetaImportHouseID(data map[string]any) string {
	return firstNonEmpty(stringFromMap(data, semantic.FieldHouseID), stringFromMap(data, semantic.InternalUpperHouseIDField()))
}

func lightingDesignImportMode(payload map[string]any) string {
	if _, ok := mapFromAny(payload[semantic.FieldGateway]); ok {
		return "house_meta_import"
	}
	return "house_meta_import_unknown"
}

func lightingDesignImportCounts(payload map[string]any) map[string]int {
	counts := map[string]int{
		semantic.FieldGateways:    0,
		semantic.FieldRooms:       0,
		semantic.FieldDevices:     0,
		semantic.FieldGroups:      0,
		semantic.FieldAreas:       len(anyListFromMap(payload, semantic.InternalField(semantic.DomainImport, semantic.FieldAreas))),
		semantic.FieldScenes:      len(anyListFromMap(payload, semantic.InternalField(semantic.DomainImport, semantic.FieldScenes))),
		semantic.FieldAutomations: len(anyListFromMap(payload, semantic.InternalField(semantic.DomainImport, semantic.FieldAutomations))),
	}
	gateway, ok := mapFromAny(payload[semantic.FieldGateway])
	if !ok {
		return counts
	}
	counts[semantic.FieldGateways] = 1
	rooms, _ := mapListFromAny(gateway[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)])
	counts[semantic.FieldRooms] = len(rooms)
	for _, room := range rooms {
		counts[semantic.FieldDevices] += len(anyListFromMap(room, semantic.InternalField(semantic.DomainImport, semantic.FieldDeviceSlots)))
		counts[semantic.FieldGroups] += len(anyListFromMap(room, semantic.InternalField(semantic.DomainImport, semantic.FieldGroups)))
	}
	return counts
}

func lightingDesignImportMappings(any) map[string]any {
	return nil
}
