package api

import (
	"context"
	"fmt"
	"strings"
)

func (client HomeOrganizationClient) favoriteBatchExists(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	rows := rowsFromData(data)
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
	return !favoriteRowsContainIdentity(rowsFromData(data), payload), calls, nil
}

func (client HomeOrganizationClient) favoriteBatchMissing(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	data, calls, err := client.readFavorites(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	rows := rowsFromData(data)
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
		return firstAnyString(item, "id", "favouriteId", "favoriteId") == favoriteID
	}
	typeID := strings.TrimSpace(stringFromAny(payload["typeId"]))
	resID := strings.TrimSpace(stringFromAny(payload["resId"]))
	if typeID == "" || resID == "" || firstAnyString(item, "typeId") != typeID || firstAnyString(item, "resId") != resID {
		return false
	}
	if expectedRank := strings.TrimSpace(stringFromAny(payload["rank"])); expectedRank != "" && firstAnyString(item, "rank") != expectedRank {
		return false
	}
	return true
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
