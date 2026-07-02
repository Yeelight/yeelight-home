package api

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
	for _, key := range []string{
		semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType),
		semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID),
		semantic.FieldRank,
	} {
		expected, ok := payload[key]
		if !ok {
			continue
		}
		expectedText := strings.TrimSpace(stringFromAny(expected))
		if expectedText != "" && firstAnyString(item, key) != expectedText {
			return false
		}
	}
	if expected, ok := payload[semantic.FieldValid].(bool); ok {
		actual, ok := item[semantic.FieldValid].(bool)
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
		type container struct {
			key    string
			typeID int
		}
		containers := []container{
			{key: "devices", typeID: semantic.ResourceDevice},
			{key: "meshgroups", typeID: semantic.ResourceMeshGroup},
			{key: "meshGroups", typeID: semantic.ResourceMeshGroup},
			{key: "userscenes", typeID: semantic.ResourceScene},
			{key: "userScenes", typeID: semantic.ResourceScene},
			{key: "scenes", typeID: semantic.ResourceScene},
		}
		for _, spec := range containers {
			items, ok := typed[spec.key].([]any)
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
				typeField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)
				idField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)
				if normalized[typeField] == nil {
					normalized[typeField] = spec.typeID
				}
				if normalized[idField] == nil {
					normalized[idField] = firstNonNilForSort(normalized[semantic.FieldDeviceID], normalized[semantic.FieldMeshGroupID], normalized[semantic.FieldSceneID], normalized[semantic.FieldID])
				}
				if normalized[semantic.FieldFavoriteID] == nil {
					delete(normalized, semantic.FieldID)
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

func projectFavoriteRows(data any) []any {
	rows := favoriteRowsFromData(data)
	favorites := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		favorite := map[string]any{}
		if value := firstCloudAny(item, semantic.FieldFavoriteID, semantic.FieldID); value != nil {
			favorite[semantic.FieldFavoriteID] = sanitizeCloudData(value)
		}
		if targetType := projectFavoriteTargetType(item); targetType != "" {
			favorite[semantic.FieldTargetType] = targetType
		}
		if targetID := firstCloudAny(item,
			semantic.FieldTargetID,
			semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID),
			semantic.FieldDeviceID,
			semantic.FieldGroupID,
			semantic.FieldMeshGroupID,
			semantic.FieldSceneID,
			semantic.FieldAutomationID,
			semantic.FieldRoomID,
		); targetID != nil {
			favorite[semantic.FieldTargetID] = sanitizeCloudData(targetID)
		}
		if name := firstAnyString(item,
			semantic.FieldName,
			semantic.FieldTargetName,
			semantic.FieldDeviceName,
			semantic.FieldGroupName,
			semantic.FieldSceneName,
			semantic.FieldAutomationName,
			semantic.FieldRoomName,
		); name != "" {
			favorite[semantic.FieldName] = name
		}
		for _, key := range []string{semantic.FieldRank, semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldValid} {
			if value, ok := item[key]; ok {
				favorite[key] = sanitizeCloudData(value)
			}
		}
		if len(favorite) == 0 {
			continue
		}
		favorites = append(favorites, favorite)
	}
	return favorites
}

func projectFavoriteTargetType(item map[string]any) string {
	if value := firstAnyString(item, semantic.FieldTargetType); value != "" {
		return value
	}
	typeValue := firstCloudAny(item,
		semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType),
		semantic.FieldTargetTypeID,
	)
	if targetType := semantic.ResourceTypeName(typeValue); targetType != "" {
		return targetType
	}
	return ""
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
	favoriteID := strings.TrimSpace(stringFromAny(payload[semantic.FieldFavoriteID]))
	if favoriteID != "" {
		return firstAnyString(item, semantic.FieldFavoriteID, semantic.FieldID) == favoriteID
	}
	typeID := strings.TrimSpace(stringFromAny(payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)]))
	resID := strings.TrimSpace(stringFromAny(payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)]))
	if typeID == "" || resID == "" {
		return false
	}
	return firstAnyString(item, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)) == typeID &&
		firstAnyString(item, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)) == resID
}

func favoriteBatchItems(payload map[string]any) ([]map[string]any, error) {
	rawItems, ok := payload[semantic.FieldItems].([]any)
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
		typeID := strings.TrimSpace(firstAnyString(row, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)))
		resID := strings.TrimSpace(firstAnyString(row, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)))
		if typeID == "" || resID == "" {
			continue
		}
		if typeGroups[typeID] == nil {
			typeGroups[typeID] = map[string]map[string]any{}
		}
		typeGroups[typeID][resID] = favoriteMergeWritableRow(houseID, row)
	}
	for _, update := range updates {
		typeID := strings.TrimSpace(stringFromAny(update[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)]))
		resID := strings.TrimSpace(stringFromAny(update[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)]))
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
		semantic.FieldHouseID: requestNumberOrStringForSort(houseID),
		semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType): requestNumberOrStringForSort(firstAnyString(source, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType))),
		semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID):   requestNumberOrStringForSort(firstAnyString(source, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID))),
	}
	if rank := strings.TrimSpace(firstAnyString(source, semantic.FieldRank)); rank != "" {
		row[semantic.FieldRank] = requestNumberOrStringForSort(rank)
	}
	if valid := strings.TrimSpace(firstAnyString(source, semantic.FieldValid)); valid != "" {
		row[semantic.FieldValid] = requestNumberOrStringForSort(valid)
	}
	if favoriteID := strings.TrimSpace(firstAnyString(source, semantic.FieldFavoriteID, semantic.FieldID)); favoriteID != "" {
		row[semantic.FieldID] = requestNumberOrStringForSort(favoriteID)
	}
	return row
}
