package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func (client HomeOrganizationClient) favoriteBatchExists(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	rows := favoriteRowsFromData(data)
	items, err := favoriteBatchItems(payload)
	if err != nil {
		return false, calls, err
	}
	for _, item := range items {
		if !favoriteRowsContain(rows, item) {
			return false, calls, nil
		}
	}
	return true, calls, nil
}

func (client HomeOrganizationClient) favoriteMissing(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	return !favoriteRowsContainIdentity(favoriteRowsFromData(data), payload), calls, nil
}

func (client HomeOrganizationClient) favoriteBatchMissing(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	rows := favoriteRowsFromData(data)
	items, err := favoriteBatchItems(payload)
	if err != nil {
		return false, calls, err
	}
	for _, item := range items {
		if favoriteRowsContainIdentity(rows, item) {
			return false, calls, nil
		}
	}
	return true, calls, nil
}

func favoriteFieldsMatch(item map[string]any, payload map[string]any) bool {
	for _, key := range []string{"typeId", "resId", "rank"} {
		expected, ok := payload[key]
		if !ok {
			continue
		}
		expectedText := strings.TrimSpace(stringFromAny(expected))
		if expectedText != "" && firstAnyString(item, key) != expectedText {
			return false
		}
	}
	if expected, ok := payload["valid"].(bool); ok {
		actual, ok := item["valid"].(bool)
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func favoriteRowsFromData(data any) []any {
	rows := []any{}
	switch typed := data.(type) {
	case map[string]any:
		for key, typeID := range map[string]int{
			"devices":    2,
			"meshgroups": 4,
			"meshGroups": 4,
			"userscenes": 6,
			"userScenes": 6,
			"scenes":     6,
		} {
			items, ok := typed[key].([]any)
			if !ok {
				continue
			}
			for _, raw := range items {
				item, ok := raw.(map[string]any)
				if !ok {
					rows = append(rows, raw)
					continue
				}
				normalized := map[string]any{}
				for itemKey, value := range item {
					normalized[itemKey] = value
				}
				if normalized["typeId"] == nil {
					normalized["typeId"] = typeID
				}
				if normalized["resId"] == nil {
					normalized["resId"] = firstNonNilForSort(normalized["deviceId"], normalized["meshGroupId"], normalized["sceneId"], normalized["id"])
				}
				if normalized["favoriteId"] == nil && normalized["favouriteId"] == nil {
					delete(normalized, "id")
				}
				rows = append(rows, normalized)
			}
		}
		if len(rows) > 0 {
			return rows
		}
	}
	return rowsFromData(data)
}

func favoriteRowsContain(rows []any, payload map[string]any) bool {
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if favoriteIdentityMatches(item, payload) && favoriteFieldsMatch(item, payload) {
			return true
		}
	}
	return false
}

func favoriteRowsContainIdentity(rows []any, payload map[string]any) bool {
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if ok && favoriteIdentityMatches(item, payload) {
			return true
		}
	}
	return false
}

func favoriteIdentityMatches(item map[string]any, payload map[string]any) bool {
	favoriteID := strings.TrimSpace(stringFromAny(payload["favoriteId"]))
	if favoriteID != "" {
		return firstAnyString(item, "favoriteId", "favouriteId", "id") == favoriteID
	}
	typeID := strings.TrimSpace(stringFromAny(payload["typeId"]))
	resID := strings.TrimSpace(stringFromAny(payload["resId"]))
	if typeID == "" || resID == "" {
		return false
	}
	return firstAnyString(item, "typeId") == typeID && firstAnyString(item, "resId") == resID
}

func favoriteBatchItems(payload map[string]any) ([]map[string]any, error) {
	rawItems, ok := payload["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return nil, fmt.Errorf("favorite batch items are required")
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("favorite batch item must be an object")
		}
		items = append(items, item)
	}
	return items, nil
}

func (client HomeOrganizationClient) favoriteMergedUpdateBody(ctx context.Context, houseID string, updates []map[string]any, credentials requestCredentials) ([]any, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return nil, calls, err
	}
	typeGroups := map[string]map[string]map[string]any{}
	for _, raw := range favoriteRowsFromData(data) {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typeID := strings.TrimSpace(firstAnyString(row, "typeId"))
		resID := strings.TrimSpace(firstAnyString(row, "resId"))
		if typeID == "" || resID == "" {
			continue
		}
		if typeGroups[typeID] == nil {
			typeGroups[typeID] = map[string]map[string]any{}
		}
		typeGroups[typeID][resID] = favoriteMergeWritableRow(houseID, row)
	}
	for _, update := range updates {
		typeID := strings.TrimSpace(stringFromAny(update["typeId"]))
		resID := strings.TrimSpace(stringFromAny(update["resId"]))
		if typeID == "" || resID == "" {
			return nil, calls, fmt.Errorf("favorite update requires typeId and resId")
		}
		if typeGroups[typeID] == nil {
			typeGroups[typeID] = map[string]map[string]any{}
		}
		typeGroups[typeID][resID] = favoriteMergeWritableRow(houseID, update)
	}
	typeIDs := make([]string, 0, len(typeGroups))
	for typeID := range typeGroups {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Strings(typeIDs)
	body := make([]any, 0)
	for _, typeID := range typeIDs {
		resIDs := make([]string, 0, len(typeGroups[typeID]))
		for resID := range typeGroups[typeID] {
			resIDs = append(resIDs, resID)
		}
		sort.Strings(resIDs)
		for _, resID := range resIDs {
			body = append(body, typeGroups[typeID][resID])
		}
	}
	return body, calls, nil
}

func favoriteMergeWritableRow(houseID string, source map[string]any) map[string]any {
	row := map[string]any{
		"houseId": requestNumberOrStringForSort(houseID),
		"typeId":  requestNumberOrStringForSort(firstAnyString(source, "typeId")),
		"resId":   requestNumberOrStringForSort(firstAnyString(source, "resId")),
	}
	if rank := strings.TrimSpace(firstAnyString(source, "rank")); rank != "" {
		row["rank"] = requestNumberOrStringForSort(rank)
	}
	if valid := strings.TrimSpace(firstAnyString(source, "valid")); valid != "" {
		row["valid"] = requestNumberOrStringForSort(valid)
	}
	if favoriteID := strings.TrimSpace(firstAnyString(source, "favoriteId", "favouriteId", "id")); favoriteID != "" {
		row["id"] = requestNumberOrStringForSort(favoriteID)
	}
	return row
}
