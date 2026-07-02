package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

const favoriteBatchLimit = 20

func buildFavoritePayload(request contract.Request, houseID string, requireID bool) (map[string]any, error) {
	payload := map[string]any{semantic.FieldHouseID: requestNumberOrString(houseID)}
	if requireID {
		favoriteID := firstRequestString(request.Parameters, semantic.FieldFavoriteID, semantic.FieldID)
		if favoriteID != "" {
			payload[semantic.FieldFavoriteID] = favoriteID
		}
	}
	typeID, ok := favoriteTypeIDFromRequest(request.Parameters)
	if !ok {
		return nil, fmt.Errorf("invalid_favorite_payload")
	}
	resID := firstRequestString(request.Parameters, semantic.FieldTargetID, semantic.FieldEntityID)
	targetName := firstRequestString(request.Parameters, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName)
	if resID == "" && targetName == "" {
		return nil, fmt.Errorf("invalid_favorite_payload")
	}
	payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)] = typeID
	if resID != "" {
		payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)] = requestNumberOrString(resID)
	}
	if targetName != "" {
		payload[semantic.FieldTargetName] = targetName
	}
	if roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldTargetRoomID); roomID != "" {
		payload[semantic.FieldRoomID] = roomID
	}
	if roomName := firstRequestString(request.Parameters, semantic.FieldRoomName, semantic.FieldTargetRoomName); roomName != "" {
		payload[semantic.FieldRoomName] = roomName
	}
	if rank, ok := requestInt(request.Parameters[semantic.FieldRank]); ok {
		payload[semantic.FieldRank] = rank
	}
	if valid, ok := request.Parameters[semantic.FieldValid].(bool); ok {
		payload[semantic.FieldValid] = valid
	}
	return payload, nil
}

func buildFavoriteBatchPayload(request contract.Request, houseID string, requireID bool) (map[string]any, error) {
	rawItems, ok := requestMapList(request.Parameters[semantic.FieldItems])
	if !ok || len(rawItems) == 0 || len(rawItems) > favoriteBatchLimit {
		return nil, fmt.Errorf("invalid_favorite_batch_payload")
	}
	items := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		itemRequest := request
		itemRequest.Parameters = copyRequestParameters(raw)
		itemRequest.Parameters[semantic.FieldHouseID] = houseID
		payload, err := buildFavoritePayload(itemRequest, houseID, requireID)
		if err != nil {
			return nil, fmt.Errorf("invalid_favorite_batch_payload")
		}
		items = append(items, payload)
	}
	return map[string]any{
		semantic.FieldHouseID: requestNumberOrString(houseID),
		semantic.FieldItems:   items,
	}, nil
}

func buildFavoriteDeletePayload(request contract.Request, houseID string) (map[string]any, error) {
	payload := map[string]any{semantic.FieldHouseID: requestNumberOrString(houseID)}
	if favoriteID := firstRequestString(request.Parameters, semantic.FieldFavoriteID, semantic.FieldID); favoriteID != "" {
		payload[semantic.FieldFavoriteID] = favoriteID
	}
	if typeID, ok := favoriteTypeIDFromRequest(request.Parameters); ok {
		payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)] = typeID
	}
	if resID := firstRequestString(request.Parameters, semantic.FieldTargetID, semantic.FieldEntityID); resID != "" {
		payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)] = requestNumberOrString(resID)
	}
	if targetName := firstRequestString(request.Parameters, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName); targetName != "" {
		payload[semantic.FieldTargetName] = targetName
	}
	if roomID := firstRequestString(request.Parameters, semantic.FieldRoomID, semantic.FieldTargetRoomID); roomID != "" {
		payload[semantic.FieldRoomID] = roomID
	}
	if roomName := firstRequestString(request.Parameters, semantic.FieldRoomName, semantic.FieldTargetRoomName); roomName != "" {
		payload[semantic.FieldRoomName] = roomName
	}
	if rank, ok := requestInt(request.Parameters[semantic.FieldRank]); ok {
		payload[semantic.FieldRank] = rank
	}
	if payload[semantic.FieldFavoriteID] == nil &&
		(payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)] == nil ||
			(payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)] == nil && payload[semantic.FieldTargetName] == nil)) {
		return nil, fmt.Errorf("invalid_favorite_delete_payload")
	}
	return payload, nil
}

func buildFavoriteBatchDeletePayload(request contract.Request, houseID string) (map[string]any, error) {
	rawItems, ok := requestMapList(request.Parameters[semantic.FieldItems])
	if !ok || len(rawItems) == 0 || len(rawItems) > favoriteBatchLimit {
		return nil, fmt.Errorf("invalid_favorite_batch_delete_payload")
	}
	items := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		itemRequest := request
		itemRequest.Parameters = copyRequestParameters(raw)
		itemRequest.Parameters[semantic.FieldHouseID] = houseID
		payload, err := buildFavoriteDeletePayload(itemRequest, houseID)
		if err != nil {
			return nil, fmt.Errorf("invalid_favorite_batch_delete_payload")
		}
		items = append(items, payload)
	}
	return map[string]any{
		semantic.FieldHouseID: requestNumberOrString(houseID),
		semantic.FieldItems:   items,
	}, nil
}

func favoriteTypeIDFromRequest(source map[string]any) (int, bool) {
	typeID, ok := resourceTypeIDFromRequest(source)
	if !ok {
		return 0, false
	}
	switch typeID {
	case groupTypeDevice, groupTypeCustom, groupTypeMesh, groupTypeScene:
		return typeID, true
	default:
		return 0, false
	}
}

func resolveFavoriteDeleteTargets(ctx context.Context, intent string, endpoint api.Endpoint, houseID string, authorization string, clientID string, payload map[string]any) (map[string]any, int, string, error) {
	result, err := api.NewMetadataReadonlyClient(endpoint, nil).RunFavoriteList(ctx, api.MetadataReadonlyRequest{
		HouseID:     houseID,
		Parameters:  map[string]any{semantic.FieldHouseID: houseID},
		Credentials: api.MetadataReadonlyCredentials{Authorization: authorization, ClientID: clientID},
	})
	if err != nil {
		return nil, 1, "", err
	}
	rows := favoriteRowsFromReadonlyData(result.Data)
	switch intent {
	case "favorite.delete":
		target, reason := resolveFavoriteDeleteItem(payload, rows)
		if reason != "" {
			return nil, result.APICalls, reason, nil
		}
		if favoriteID := favoriteIDFromRow(target); favoriteID != "" {
			payload[semantic.FieldFavoriteID] = favoriteID
			payload[semantic.FieldID] = favoriteID
		}
		payload[semantic.FieldDeleteTarget] = favoriteDeletePreviewRow(target)
		return map[string]any{semantic.FieldDeleteTarget: favoriteDeletePreviewRow(target)}, result.APICalls, "", nil
	case "favorite.batch_delete":
		items := payloadItems(payload)
		seen := map[string]bool{}
		previewItems := make([]any, 0, len(items))
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return nil, result.APICalls, "invalid_favorite_batch_delete_payload", nil
			}
			target, reason := resolveFavoriteDeleteItem(item, rows)
			if reason != "" {
				return nil, result.APICalls, reason, nil
			}
			deleteID := favoriteDeleteIdentityFromRow(target)
			if seen[deleteID] {
				return nil, result.APICalls, "duplicate_favorite_delete_target", nil
			}
			seen[deleteID] = true
			if favoriteID := favoriteIDFromRow(target); favoriteID != "" {
				item[semantic.FieldFavoriteID] = favoriteID
				item[semantic.FieldID] = favoriteID
			}
			item[semantic.FieldDeleteTarget] = favoriteDeletePreviewRow(target)
			previewItems = append(previewItems, favoriteDeletePreviewRow(target))
		}
		return map[string]any{semantic.FieldDeleteTargets: previewItems}, result.APICalls, "", nil
	default:
		return nil, result.APICalls, "unsupported_home_organization_intent", nil
	}
}

func resolveFavoriteDeleteItem(payload map[string]any, rows []map[string]any) (map[string]any, string) {
	matches := make([]map[string]any, 0, 1)
	for _, row := range rows {
		if favoriteRowMatchesPayload(row, payload) {
			matches = append(matches, row)
		}
	}
	switch len(matches) {
	case 0:
		return nil, "favorite_not_found"
	case 1:
		if favoriteDeleteIdentityFromRow(matches[0]) == "" {
			return nil, "favorite_id_missing"
		}
		return matches[0], ""
	default:
		return nil, "ambiguous_favorite_target"
	}
}

func favoriteRowsFromReadonlyData(data any) []map[string]any {
	wrapper, ok := data.(map[string]any)
	if ok {
		if favorites, ok := wrapper["favorites"]; ok {
			return favoriteRowsFromReadonlyData(favorites)
		}
		if rows := favoriteRowsFromReadonlyContainers(wrapper); len(rows) > 0 {
			return rows
		}
		for _, key := range []string{"rows", "list", "data"} {
			if rows, ok := wrapper[key].([]any); ok {
				return favoriteRowsFromReadonlyRows(rows)
			}
		}
		return []map[string]any{wrapper}
	}
	rows, ok := data.([]any)
	if !ok {
		return nil
	}
	return favoriteRowsFromReadonlyRows(rows)
}

func favoriteRowsFromReadonlyContainers(wrapper map[string]any) []map[string]any {
	type container struct {
		key    string
		typeID int
	}
	containers := []container{
		{key: "devices", typeID: groupTypeDevice},
		{key: "meshgroups", typeID: groupTypeMesh},
		{key: "meshGroups", typeID: groupTypeMesh},
		{key: "userscenes", typeID: groupTypeScene},
		{key: "userScenes", typeID: groupTypeScene},
		{key: "scenes", typeID: groupTypeScene},
	}
	rows := []map[string]any{}
	for _, spec := range containers {
		rawRows, ok := wrapper[spec.key].([]any)
		if !ok {
			continue
		}
		for _, raw := range rawRows {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			normalized := map[string]any{}
			for key, value := range row {
				normalized[key] = value
			}
			typeField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)
			idField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)
			if normalized[typeField] == nil {
				normalized[typeField] = spec.typeID
			}
			if normalized[idField] == nil {
				normalized[idField] = firstNonNil(normalized[semantic.FieldDeviceID], normalized[semantic.FieldMeshGroupID], normalized[semantic.FieldSceneID], normalized[semantic.FieldID])
			}
			if normalized[semantic.FieldFavoriteID] == nil {
				delete(normalized, semantic.FieldID)
			}
			rows = append(rows, normalized)
		}
	}
	return rows
}

func favoriteRowsFromReadonlyRows(rows []any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if ok {
			result = append(result, normalizeReadonlyFavoriteRow(item))
		}
	}
	return result
}

func normalizeReadonlyFavoriteRow(row map[string]any) map[string]any {
	normalized := map[string]any{}
	for key, value := range row {
		normalized[key] = value
	}
	typeField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)
	idField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)
	if normalized[typeField] == nil {
		if targetType := favoriteField(normalized, semantic.FieldTargetType); targetType != "" {
			if typeID, ok := semanticTargetTypeID(targetType, groupTypeMesh); ok {
				normalized[typeField] = typeID
			}
		}
	}
	if normalized[idField] == nil {
		if targetID := favoriteField(normalized, semantic.FieldTargetID); targetID != "" {
			normalized[idField] = targetID
		}
	}
	if normalized[semantic.FieldFavoriteID] == nil {
		if favoriteID := favoriteField(normalized, semantic.FieldID); favoriteID != "" {
			normalized[semantic.FieldFavoriteID] = favoriteID
		}
	}
	return normalized
}

func favoriteRowMatchesPayload(row map[string]any, payload map[string]any) bool {
	if favoriteID := strings.TrimSpace(requestString(firstNonNil(payload[semantic.FieldFavoriteID], payload[semantic.FieldID]))); favoriteID != "" {
		return favoriteIDFromRow(row) == favoriteID
	}
	typeField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)
	idField := semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)
	typeID := strings.TrimSpace(requestString(payload[typeField]))
	resID := strings.TrimSpace(requestString(payload[idField]))
	if typeID == "" || resID == "" || favoriteField(row, typeField) != typeID || favoriteField(row, idField) != resID {
		return false
	}
	if rank := strings.TrimSpace(requestString(payload[semantic.FieldRank])); rank != "" && favoriteField(row, semantic.FieldRank) != rank {
		return false
	}
	return true
}

func favoriteIDFromRow(row map[string]any) string {
	return firstNonEmptyString(favoriteField(row, semantic.FieldFavoriteID), favoriteField(row, semantic.FieldID))
}

func favoriteDeleteIdentityFromRow(row map[string]any) string {
	if favoriteID := favoriteIDFromRow(row); favoriteID != "" {
		return "favorite:" + favoriteID
	}
	typeID := favoriteField(row, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType))
	resID := favoriteField(row, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID))
	if typeID == "" || resID == "" {
		return ""
	}
	return "resource:" + typeID + ":" + resID
}

func favoriteField(row map[string]any, key string) string {
	if value, ok := row[key]; ok {
		return requestString(value)
	}
	return ""
}

func favoriteDeletePreviewRow(row map[string]any) map[string]any {
	preview := map[string]any{}
	for _, key := range []string{semantic.FieldID, semantic.FieldFavoriteID, semantic.FieldRank, semantic.FieldValid} {
		if value, ok := row[key]; ok {
			preview[key] = value
		}
	}
	if id := favoriteIDFromRow(row); id != "" {
		preview[semantic.FieldFavoriteID] = id
	}
	if targetType, ok := favoriteTargetTypeFromRow(row); ok {
		preview[semantic.FieldTargetType] = targetType
	}
	if targetID := favoriteField(row, semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)); targetID != "" {
		preview[semantic.FieldTargetID] = targetID
	}
	return preview
}

func favoriteTargetTypeFromRow(row map[string]any) (string, bool) {
	typeID, ok := requestInt(row[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)])
	if !ok {
		return "", false
	}
	switch typeID {
	case groupTypeRoom:
		return "room", true
	case groupTypeDevice:
		return "device", true
	case groupTypeCustom, groupTypeMesh:
		return "group", true
	case groupTypeScene:
		return "scene", true
	case groupTypeAutomation:
		return "automation", true
	default:
		return "", false
	}
}
