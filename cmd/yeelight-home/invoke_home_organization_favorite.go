package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

const favoriteBatchLimit = 20

func buildFavoritePayload(request contract.Request, houseID string, requireID bool) (map[string]any, error) {
	payload := map[string]any{"houseId": requestNumberOrString(houseID)}
	if requireID {
		favoriteID := firstRequestString(request.Parameters, "favoriteId", "favouriteId", "id")
		if favoriteID == "" {
			return nil, fmt.Errorf("invalid_favorite_update_payload")
		}
		payload["favoriteId"] = favoriteID
	}
	typeID, ok := requestInt(firstNonNil(request.Parameters["typeId"], request.Parameters["resourceTypeId"]))
	if !ok {
		return nil, fmt.Errorf("invalid_favorite_payload")
	}
	resID := firstRequestString(request.Parameters, "resId", "resourceId", "entityId", "deviceId", "sceneId", "groupId", "roomId")
	if resID == "" {
		return nil, fmt.Errorf("invalid_favorite_payload")
	}
	payload["typeId"] = typeID
	payload["resId"] = requestNumberOrString(resID)
	if rank, ok := requestInt(request.Parameters["rank"]); ok {
		payload["rank"] = rank
	}
	if valid, ok := request.Parameters["valid"].(bool); ok {
		payload["valid"] = valid
	}
	return payload, nil
}

func buildFavoriteBatchPayload(request contract.Request, houseID string, requireID bool) (map[string]any, error) {
	rawItems, ok := requestMapList(request.Parameters["items"])
	if !ok || len(rawItems) == 0 || len(rawItems) > favoriteBatchLimit {
		return nil, fmt.Errorf("invalid_favorite_batch_payload")
	}
	items := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		itemRequest := request
		itemRequest.Parameters = copyRequestParameters(raw)
		itemRequest.Parameters["houseId"] = houseID
		payload, err := buildFavoritePayload(itemRequest, houseID, requireID)
		if err != nil {
			return nil, fmt.Errorf("invalid_favorite_batch_payload")
		}
		items = append(items, payload)
	}
	return map[string]any{
		"houseId": requestNumberOrString(houseID),
		"items":   items,
	}, nil
}

func buildFavoriteDeletePayload(request contract.Request, houseID string) (map[string]any, error) {
	payload := map[string]any{"houseId": requestNumberOrString(houseID)}
	if favoriteID := firstRequestString(request.Parameters, "favoriteId", "favouriteId", "id"); favoriteID != "" {
		payload["favoriteId"] = favoriteID
	}
	if typeID, ok := requestInt(firstNonNil(request.Parameters["typeId"], request.Parameters["resourceTypeId"])); ok {
		payload["typeId"] = typeID
	}
	if resID := firstRequestString(request.Parameters, "resId", "resourceId", "entityId", "deviceId", "sceneId", "groupId", "roomId"); resID != "" {
		payload["resId"] = requestNumberOrString(resID)
	}
	if rank, ok := requestInt(request.Parameters["rank"]); ok {
		payload["rank"] = rank
	}
	if payload["favoriteId"] == nil && (payload["typeId"] == nil || payload["resId"] == nil) {
		return nil, fmt.Errorf("invalid_favorite_delete_payload")
	}
	return payload, nil
}

func buildFavoriteBatchDeletePayload(request contract.Request, houseID string) (map[string]any, error) {
	rawItems, ok := requestMapList(request.Parameters["items"])
	if !ok || len(rawItems) == 0 || len(rawItems) > favoriteBatchLimit {
		return nil, fmt.Errorf("invalid_favorite_batch_delete_payload")
	}
	items := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		itemRequest := request
		itemRequest.Parameters = copyRequestParameters(raw)
		itemRequest.Parameters["houseId"] = houseID
		payload, err := buildFavoriteDeletePayload(itemRequest, houseID)
		if err != nil {
			return nil, fmt.Errorf("invalid_favorite_batch_delete_payload")
		}
		items = append(items, payload)
	}
	return map[string]any{
		"houseId": requestNumberOrString(houseID),
		"items":   items,
	}, nil
}

func resolveFavoriteDeleteTargets(ctx context.Context, intent string, endpoint api.Endpoint, houseID string, authorization string, clientID string, payload map[string]any) (map[string]any, int, string, error) {
	result, err := api.NewMetadataReadonlyClient(endpoint, nil).RunFavoriteList(ctx, api.MetadataReadonlyRequest{
		HouseID:     houseID,
		Parameters:  map[string]any{"houseId": houseID},
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
		payload["favoriteId"] = favoriteIDFromRow(target)
		payload["id"] = favoriteIDFromRow(target)
		payload["deleteTarget"] = favoriteDeletePreviewRow(target)
		return map[string]any{"deleteTarget": favoriteDeletePreviewRow(target)}, result.APICalls, "", nil
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
			favoriteID := favoriteIDFromRow(target)
			if seen[favoriteID] {
				return nil, result.APICalls, "duplicate_favorite_delete_target", nil
			}
			seen[favoriteID] = true
			item["favoriteId"] = favoriteID
			item["id"] = favoriteID
			item["deleteTarget"] = favoriteDeletePreviewRow(target)
			previewItems = append(previewItems, favoriteDeletePreviewRow(target))
		}
		return map[string]any{"deleteTargets": previewItems}, result.APICalls, "", nil
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
		if favoriteIDFromRow(matches[0]) == "" {
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

func favoriteRowsFromReadonlyRows(rows []any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if ok {
			result = append(result, item)
		}
	}
	return result
}

func favoriteRowMatchesPayload(row map[string]any, payload map[string]any) bool {
	if favoriteID := strings.TrimSpace(requestString(firstNonNil(payload["favoriteId"], payload["favouriteId"], payload["id"]))); favoriteID != "" {
		return favoriteIDFromRow(row) == favoriteID
	}
	typeID := strings.TrimSpace(requestString(payload["typeId"]))
	resID := strings.TrimSpace(requestString(payload["resId"]))
	if typeID == "" || resID == "" || favoriteField(row, "typeId") != typeID || favoriteField(row, "resId") != resID {
		return false
	}
	if rank := strings.TrimSpace(requestString(payload["rank"])); rank != "" && favoriteField(row, "rank") != rank {
		return false
	}
	return true
}

func favoriteIDFromRow(row map[string]any) string {
	return firstNonEmptyString(favoriteField(row, "id"), favoriteField(row, "favoriteId"), favoriteField(row, "favouriteId"))
}

func favoriteField(row map[string]any, key string) string {
	if value, ok := row[key]; ok {
		return requestString(value)
	}
	return ""
}

func favoriteDeletePreviewRow(row map[string]any) map[string]any {
	preview := map[string]any{}
	for _, key := range []string{"id", "favoriteId", "favouriteId", "typeId", "resId", "rank", "valid"} {
		if value, ok := row[key]; ok {
			preview[key] = value
		}
	}
	if id := favoriteIDFromRow(row); id != "" {
		preview["favoriteId"] = id
	}
	return preview
}
