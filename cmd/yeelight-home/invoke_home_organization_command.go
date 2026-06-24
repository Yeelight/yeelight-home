package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func (app *app) invokeHomeOrganizationPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, preconditions, summary, err := buildHomeOrganizationPayload(request, houseID)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeOrganizationAcceptedFields(request.Intent)), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	if reason := validateHomeOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return configureClarificationResponse(request, reason, homeOrganizationAcceptedFields(request.Intent)), nil
	}
	var preview map[string]any
	previewCalls := 0
	if request.Intent == "home.sort.configure" {
		sortPreview, calls, err := homeSortPreview(ctx, endpoint, houseID, authorization, clientID, payload)
		if err != nil {
			return contract.Response{}, err
		}
		preview = sortPreview
		previewCalls = calls
	} else if request.Intent == "favorite.delete" || request.Intent == "favorite.batch_delete" {
		favoritePreview, calls, reason, err := resolveFavoriteDeleteTargets(ctx, request.Intent, endpoint, houseID, authorization, clientID, payload)
		if err != nil {
			return contract.Response{}, err
		}
		if reason != "" {
			return configureClarificationResponse(request, reason, homeOrganizationAcceptedFields(request.Intent)), nil
		}
		preview = favoritePreview
		previewCalls = calls
	}
	now := time.Now()
	record, err := plan.NewRecord(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now, pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingPlanResponseWithPreview(request, record, entities, preview, previewCalls), nil
}

func buildHomeOrganizationPayload(request contract.Request, houseID string) (map[string]any, []string, string, error) {
	switch request.Intent {
	case "home.sort.configure":
		payload, err := buildHomeSortConfigurePayload(request, houseID)
		return payload, []string{
			"提交前重新读取当前排序",
			"排序资源必须属于当前家庭",
			"提交后通过 home.sort.list 验证排序结果",
		}, "配置首页排序", err
	case "favorite.add":
		payload, err := buildFavoritePayload(request, houseID, false)
		return payload, []string{
			"提交前重新读取当前收藏",
			"收藏资源必须属于当前家庭",
			"提交后通过 favorite.list 验证收藏存在",
		}, "新增首页收藏", err
	case "favorite.update":
		payload, err := buildFavoritePayload(request, houseID, true)
		return payload, []string{
			"提交前重新读取当前收藏",
			"收藏资源必须属于当前家庭",
			"提交后通过 favorite.list 验证收藏更新",
		}, "更新首页收藏", err
	case "favorite.delete":
		payload, err := buildFavoriteDeletePayload(request, houseID)
		return payload, []string{
			"提交前重新读取当前收藏并确认目标仍存在",
			"只删除本计划中已解析的单个收藏",
			"提交后通过 favorite.list 验证收藏已移除",
		}, "删除首页收藏", err
	case "favorite.batch_add":
		payload, err := buildFavoriteBatchPayload(request, houseID, false)
		return payload, []string{
			"提交前重新读取当前收藏",
			"收藏资源必须全部属于当前家庭",
			"提交后通过 favorite.list 验证全部收藏存在",
		}, "批量新增首页收藏", err
	case "favorite.batch_update":
		payload, err := buildFavoriteBatchPayload(request, houseID, true)
		return payload, []string{
			"提交前重新读取当前收藏",
			"收藏资源必须全部属于当前家庭",
			"提交后通过 favorite.list 验证全部收藏更新",
		}, "批量更新首页收藏", err
	case "favorite.batch_delete":
		payload, err := buildFavoriteBatchDeletePayload(request, houseID)
		return payload, []string{
			"提交前重新读取当前收藏并确认全部目标仍存在",
			"单次计划最多删除 20 个收藏",
			"plan.commit 只接受 planId，忽略提交时附带的删除字段",
			"提交后通过 favorite.list 验证全部收藏已移除",
		}, "批量删除首页收藏", err
	default:
		return nil, nil, "", fmt.Errorf("unsupported_home_organization_intent")
	}
}

func buildHomeSortConfigurePayload(request contract.Request, houseID string) (map[string]any, error) {
	sortType := firstRequestString(request.Parameters, "type", "sortType")
	target := firstRequestString(request.Parameters, "target", "roomId", "areaId")
	if sortType == "" || target == "" {
		return nil, fmt.Errorf("invalid_home_sort_configure_payload")
	}
	items, ok := requestSortItems(request)
	if !ok {
		return nil, fmt.Errorf("invalid_home_sort_configure_payload")
	}
	payload := map[string]any{
		"houseId": houseID,
		"type":    sortType,
		"target":  target,
		"items":   items,
	}
	if len(items) == 1 {
		first, _ := items[0].(map[string]any)
		for _, key := range []string{"typeId", "resId", "rank", "subIndex"} {
			if value, ok := first[key]; ok {
				payload[key] = value
			}
		}
	}
	if value := firstRequestString(request.Parameters, "roomId"); value != "" {
		payload["roomId"] = value
	}
	return payload, nil
}

func homeSortPreview(ctx context.Context, endpoint api.Endpoint, houseID string, authorization string, clientID string, payload map[string]any) (map[string]any, int, error) {
	previewRequest := contract.Request{
		ContractVersion: contract.Version,
		RequestID:       "home-sort-preview",
		Intent:          "home.sort.list",
		Locale:          "zh-CN",
		Utterance:       "preview home sort",
		Parameters: map[string]any{
			"houseId": houseID,
		},
	}
	for _, key := range []string{"typeId", "resId", "roomId", "type", "target", "subIndex"} {
		if value, ok := payload[key]; ok {
			previewRequest.Parameters[key] = value
		}
	}
	result, err := api.NewMetadataReadonlyClient(endpoint, nil).RunHomeSortList(ctx, api.MetadataReadonlyRequest{
		HouseID:     houseID,
		Parameters:  previewRequest.Parameters,
		Credentials: api.MetadataReadonlyCredentials{Authorization: authorization, ClientID: clientID},
	})
	if err != nil {
		return nil, 1, err
	}
	return map[string]any{
		"currentItems": homeSortResultItemCount(result.Data),
		"plannedItems": len(payloadItems(payload)),
		"type":         payload["type"],
		"target":       payload["target"],
	}, result.APICalls, nil
}

func homeSortResultItemCount(data any) int {
	wrapper, ok := data.(map[string]any)
	if !ok {
		return 0
	}
	rows, ok := wrapper["sort"].([]any)
	if ok {
		return len(rows)
	}
	return 0
}

func requestSortItems(request contract.Request) ([]any, bool) {
	if rawItems, ok := request.Parameters["items"].([]any); ok && len(rawItems) > 0 {
		items := make([]any, 0, len(rawItems))
		for _, raw := range rawItems {
			item, ok := raw.(map[string]any)
			if !ok {
				return nil, false
			}
			normalized, ok := normalizeSortItem(item)
			if !ok {
				return nil, false
			}
			items = append(items, normalized)
		}
		return items, true
	}
	item, ok := normalizeSortItem(request.Parameters)
	if !ok {
		return nil, false
	}
	return []any{item}, true
}

func normalizeSortItem(source map[string]any) (map[string]any, bool) {
	typeID, ok := requestInt(source["typeId"])
	if !ok {
		return nil, false
	}
	resID := requestString(firstNonNil(source["resId"], source["entityId"], source["deviceId"], source["sceneId"], source["groupId"], source["roomId"]))
	rank, ok := requestInt(source["rank"])
	if resID == "" || !ok {
		return nil, false
	}
	item := map[string]any{
		"typeId": typeID,
		"resId":  requestNumberOrString(resID),
		"rank":   rank,
	}
	if subIndex, ok := requestInt(source["subIndex"]); ok {
		item["subIndex"] = subIndex
	}
	return item, true
}

func validateHomeOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "home.sort.configure":
		items, ok := payload["items"].([]any)
		if !ok || len(items) == 0 {
			return "invalid_home_sort_configure_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_home_sort_configure_payload"
			}
			if reason := validateResourceReference(item["typeId"], item["resId"], entities, "invalid_sort_resource_type", "invalid_sort_resource_reference"); reason != "" {
				return reason
			}
		}
	case "favorite.add", "favorite.update":
		if reason := validateResourceReference(payload["typeId"], payload["resId"], entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference"); reason != "" {
			return reason
		}
	case "favorite.delete":
		if payload["typeId"] != nil || payload["resId"] != nil {
			if reason := validateResourceReference(payload["typeId"], payload["resId"], entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference"); reason != "" {
				return reason
			}
		}
	case "favorite.batch_add", "favorite.batch_update":
		items := payloadItems(payload)
		if len(items) == 0 || len(items) > favoriteBatchLimit {
			return "invalid_favorite_batch_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_favorite_batch_payload"
			}
			if reason := validateResourceReference(item["typeId"], item["resId"], entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference"); reason != "" {
				return reason
			}
		}
	case "favorite.batch_delete":
		items := payloadItems(payload)
		if len(items) == 0 || len(items) > favoriteBatchLimit {
			return "invalid_favorite_batch_delete_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_favorite_batch_delete_payload"
			}
			if item["typeId"] == nil && item["resId"] == nil {
				continue
			}
			if reason := validateResourceReference(item["typeId"], item["resId"], entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference"); reason != "" {
				return reason
			}
		}
	}
	return ""
}

func homeOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "home.sort.configure":
		return []string{"parameters.houseId", "parameters.type", "parameters.target", "parameters.items", "parameters.typeId", "parameters.resId", "parameters.rank"}
	case "favorite.add":
		return []string{"parameters.houseId", "parameters.typeId", "parameters.resId", "parameters.rank"}
	case "favorite.update":
		return []string{"parameters.houseId", "parameters.favoriteId", "parameters.typeId", "parameters.resId", "parameters.rank"}
	case "favorite.delete":
		return []string{"parameters.houseId", "parameters.favoriteId", "parameters.typeId", "parameters.resId", "parameters.rank"}
	case "favorite.batch_add":
		return []string{"parameters.houseId", "parameters.items[].typeId", "parameters.items[].resId", "parameters.items[].rank"}
	case "favorite.batch_update":
		return []string{"parameters.houseId", "parameters.items[].favoriteId", "parameters.items[].typeId", "parameters.items[].resId", "parameters.items[].rank"}
	case "favorite.batch_delete":
		return []string{"parameters.houseId", "parameters.items[].favoriteId", "parameters.items[].typeId", "parameters.items[].resId", "parameters.items[].rank"}
	default:
		return []string{"parameters.houseId"}
	}
}

func payloadItems(payload map[string]any) []any {
	items, ok := payload["items"].([]any)
	if !ok {
		return nil
	}
	return items
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
