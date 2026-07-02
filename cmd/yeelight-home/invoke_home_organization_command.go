package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareHomeOrganization(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
	}
	payload, preconditions, summary, err := buildHomeOrganizationPayload(request, houseID)
	if err != nil {
		return homeOrganizationClarificationResponse(request, err.Error()), nil
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
	if reason := resolveHomeOrganizationReferences(request.Intent, payload, entities); reason != "" {
		return homeOrganizationClarificationResponse(request, reason), nil
	}
	if reason := validateHomeOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return homeOrganizationClarificationResponse(request, reason), nil
	}
	var preview map[string]any
	previewCalls := 0
	if request.Intent == "home.sort.configure" {
		sortPreview, calls, err := homeSortPreview(ctx, endpoint, houseID, authorization, clientID, payload)
		if err != nil {
			sortPreview = map[string]any{
				"previewUnavailable": true,
				"warning":            "home_sort_preview_unavailable",
				"detail":             err.Error(),
				"plannedItems":       len(payloadItems(payload)),
			}
			addSemanticSortPayloadPreview(sortPreview, payload)
			calls = 1
		}
		preview = sortPreview
		previewCalls = calls
	} else if request.Intent == "favorite.delete" || request.Intent == "favorite.batch_delete" {
		favoritePreview, calls, reason, err := resolveFavoriteDeleteTargets(ctx, request.Intent, endpoint, houseID, authorization, clientID, payload)
		if err != nil {
			return contract.Response{}, err
		}
		if reason != "" {
			return homeOrganizationClarificationResponse(request, reason), nil
		}
		preview = favoritePreview
		previewCalls = calls
	}
	now := time.Now()
	risk := operation.RiskR2
	if request.Intent == "favorite.delete" || request.Intent == "favorite.batch_delete" {
		risk = operation.RiskR3
	}
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, request.Intent, request.RequestID, summary, risk, payload, preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, previewCalls), nil
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
			"Runtime 根据当前请求构建受控删除 payload",
			"提交后通过 favorite.list 验证全部收藏已移除",
		}, "批量删除首页收藏", err
	default:
		return nil, nil, "", fmt.Errorf("unsupported_home_organization_intent")
	}
}

func buildHomeSortConfigurePayload(request contract.Request, houseID string) (map[string]any, error) {
	items, ok := requestSortItems(request)
	if !ok {
		return nil, fmt.Errorf("invalid_home_sort_configure_payload")
	}
	source := map[string]any{}
	for key, value := range request.Parameters {
		source[key] = value
	}
	source[semantic.FieldItems] = items
	payload, warning := api.NormalizeHomeSortPayload(houseID, source)
	if warning != "" || payload[semantic.FieldType] == nil || payload[semantic.FieldTarget] == nil {
		return nil, fmt.Errorf("invalid_home_sort_configure_payload")
	}
	payload[semantic.FieldHouseID] = houseID
	payload[semantic.FieldItems] = items
	payload = map[string]any{
		semantic.FieldHouseID: houseID,
		semantic.FieldType:    payload[semantic.FieldType],
		semantic.FieldTarget:  payload[semantic.FieldTarget],
		semantic.FieldItems:   items,
	}
	if len(items) == 1 {
		first, _ := items[0].(map[string]any)
		for _, key := range []string{
			semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType),
			semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID),
			semantic.FieldRank,
			semantic.FieldSubIndex,
		} {
			if value, ok := first[key]; ok {
				payload[key] = value
			}
		}
	}
	if value, ok := source[semantic.FieldRoomID]; ok {
		payload[semantic.FieldRoomID] = requestNumberOrString(requestString(value))
	} else if value, ok := payload[semantic.FieldTarget]; ok {
		if sortType, sortOK := requestInt(payload[semantic.FieldType]); sortOK && (sortType == 1 || sortType == 2) {
			payload[semantic.FieldRoomID] = value
		}
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
			semantic.FieldHouseID: houseID,
		},
	}
	for _, key := range []string{
		semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType),
		semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID),
		semantic.FieldRoomID,
		semantic.FieldType,
		semantic.FieldTarget,
		semantic.FieldSubIndex,
	} {
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
	if result.Partial {
		return nil, result.APICalls, fmt.Errorf("home sort preview unavailable: %s", strings.Join(result.Warnings, ","))
	}
	preview := map[string]any{
		semantic.FieldCurrentItems: homeSortResultItemCount(result.Data),
		semantic.FieldPlannedItems: len(payloadItems(payload)),
	}
	addSemanticSortPayloadPreview(preview, payload)
	return preview, result.APICalls, nil
}

func homeSortResultItemCount(data any) int {
	wrapper, ok := data.(map[string]any)
	if !ok {
		return 0
	}
	rows, ok := wrapper[semantic.FieldSort].([]any)
	if ok {
		return len(rows)
	}
	return 0
}

func requestSortItems(request contract.Request) ([]any, bool) {
	if rawItems, ok := request.Parameters[semantic.FieldItems].([]any); ok && len(rawItems) > 0 {
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
	typeID, ok := resourceTypeIDFromRequest(source)
	if !ok {
		return nil, false
	}
	resID := requestString(firstNonNil(source[semantic.FieldTargetID], source[semantic.FieldEntityID], source[semantic.FieldID]))
	targetName := firstRequestString(source, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName)
	rank, ok := requestInt(source[semantic.FieldRank])
	if (resID == "" && targetName == "") || !ok {
		return nil, false
	}
	item := map[string]any{
		semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType): typeID,
		semantic.FieldRank: rank,
	}
	if resID != "" {
		item[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] = requestNumberOrString(resID)
	}
	if targetName != "" {
		item[semantic.FieldTargetName] = targetName
	}
	if subIndex, ok := requestInt(source[semantic.FieldSubIndex]); ok {
		item[semantic.FieldSubIndex] = subIndex
	}
	return item, true
}

func resourceTypeIDFromRequest(source map[string]any) (int, bool) {
	entityType := strings.TrimSpace(requestString(firstNonNil(source[semantic.FieldTargetType], source[semantic.FieldEntityType])))
	if typeID, ok := semanticTargetTypeID(entityType, groupTypeCustom); ok {
		return typeID, true
	}
	return 0, false
}

func resolveHomeOrganizationReferences(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "home.sort.configure":
		if reason := resolveHomeSortTargetReference(payload, entities); reason != "" {
			return reason
		}
		items := payloadItems(payload)
		if len(items) == 0 {
			return "invalid_home_sort_configure_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_home_sort_configure_payload"
			}
			inheritHomeSortItemQualifier(payload, item)
			if reason := resolveTypedResourceName(item, semantic.DomainSort, entities, "invalid_sort_resource_type", "invalid_sort_resource_reference", "ambiguous_sort_resource_reference"); reason != "" {
				return reason
			}
		}
		if len(items) == 1 {
			if item, ok := items[0].(map[string]any); ok {
				for _, key := range []string{
					semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType),
					semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID),
					semantic.FieldRank,
					semantic.FieldSubIndex,
				} {
					if value, ok := item[key]; ok {
						payload[key] = value
					}
				}
			}
		}
	case "favorite.add", "favorite.update", "favorite.delete":
		if requestString(firstNonNil(payload[semantic.FieldFavoriteID], payload[semantic.FieldID])) != "" {
			return ""
		}
		return resolveTypedResourceName(payload, semantic.DomainFavorite, entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference", "ambiguous_favorite_resource_reference")
	case "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		items := payloadItems(payload)
		if len(items) == 0 {
			return "invalid_favorite_batch_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_favorite_batch_payload"
			}
			if item[semantic.FieldFavoriteID] != nil || item[semantic.FieldID] != nil {
				continue
			}
			if reason := resolveTypedResourceName(item, semantic.DomainFavorite, entities, "invalid_favorite_resource_type", "invalid_favorite_resource_reference", "ambiguous_favorite_resource_reference"); reason != "" {
				return reason
			}
		}
	}
	return ""
}

func inheritHomeSortItemQualifier(payload map[string]any, item map[string]any) {
	sortType, ok := valueInt(payload[semantic.FieldType])
	if !ok || (sortType != 1 && sortType != 2) {
		return
	}
	for _, key := range []string{semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName} {
		if item[key] != nil {
			continue
		}
		if value, ok := payload[key]; ok {
			item[key] = value
		}
	}
}

func resolveHomeSortTargetReference(payload map[string]any, entities api.EntityListResult) string {
	sortType, ok := valueInt(payload[semantic.FieldType])
	if !ok {
		return "invalid_home_sort_configure_payload"
	}
	entityType := ""
	idField := semantic.FieldTarget
	switch sortType {
	case 1, 2:
		entityType = "room"
		idField = semantic.FieldRoomID
	case 4:
		entityType = "device"
	case 5:
		entityType = "area"
		idField = semantic.FieldAreaID
	default:
		return ""
	}
	targetIDOrName := firstNonEmptyString(
		requestString(payload[idField]),
		requestString(payload[semantic.FieldTarget]),
	)
	targetName := firstNonEmptyString(
		requestString(payload[semantic.FieldTargetName]),
		requestString(payload[semantic.FieldEntityName]),
		requestString(payload[semantic.FieldName]),
		requestString(payload[semantic.FieldRoomName]),
		requestString(payload[semantic.FieldTargetRoomName]),
		requestString(payload[semantic.FieldAreaName]),
		requestString(payload[semantic.FieldDeviceName]),
	)
	match, ambiguous := resolveHomeSortTargetEntity(entities, entityType, targetIDOrName, targetName)
	if ambiguous {
		return "ambiguous_sort_target_reference"
	}
	if match.ID == "" {
		return "invalid_sort_target_reference"
	}
	payload[semantic.FieldTarget] = requestNumberOrString(match.ID)
	switch entityType {
	case "room":
		payload[semantic.FieldRoomID] = requestNumberOrString(match.ID)
	case "area":
		payload[semantic.FieldAreaID] = requestNumberOrString(match.ID)
	case "device":
		payload[semantic.FieldParentID] = requestNumberOrString(match.ID)
	}
	return ""
}

func resolveHomeSortTargetEntity(entities api.EntityListResult, entityType string, idOrName string, name string) (api.EntitySummary, bool) {
	if strings.TrimSpace(idOrName) != "" {
		match, candidates, _ := findEntity(entityGetTarget{id: strings.TrimSpace(idOrName), entityType: entityType}, entities.Entities)
		if match.ID != "" {
			return match, false
		}
		if len(candidates) > 0 {
			return api.EntitySummary{}, true
		}
		if strings.TrimSpace(name) == "" {
			name = idOrName
		}
	}
	if strings.TrimSpace(name) == "" {
		return api.EntitySummary{}, false
	}
	match, candidates, _ := findEntity(entityGetTarget{name: strings.TrimSpace(name), entityType: entityType}, entities.Entities)
	if match.ID != "" {
		return match, false
	}
	return api.EntitySummary{}, len(candidates) > 0
}

func resolveTypedResourceName(payload map[string]any, domain string, entities api.EntityListResult, typeReason string, referenceReason string, ambiguousReason string) string {
	typeKey := semantic.InternalField(domain, semantic.FieldTargetType)
	idKey := semantic.InternalField(domain, semantic.FieldTargetID)
	typeID, ok := valueInt(payload[typeKey])
	if !ok {
		return typeReason
	}
	if valueIDString(payload[idKey]) != "" {
		return ""
	}
	targetName := firstNonEmptyString(
		requestString(payload[semantic.FieldTargetName]),
		requestString(payload[semantic.FieldEntityName]),
		requestString(payload[semantic.FieldName]),
	)
	if targetName == "" {
		return referenceReason
	}
	entityType, ok := entityTypeForGroupType(typeID)
	if !ok {
		return typeReason
	}
	target := entityGetTarget{
		name:       targetName,
		entityType: entityType,
		roomID:     firstNonEmptyString(requestString(payload[semantic.FieldRoomID]), requestString(payload[semantic.FieldTargetRoomID])),
		roomName:   firstNonEmptyString(requestString(payload[semantic.FieldRoomName]), requestString(payload[semantic.FieldTargetRoomName])),
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if len(candidates) > 1 {
		return ambiguousReason
	}
	if match.ID == "" {
		return referenceReason
	}
	payload[idKey] = requestNumberOrString(match.ID)
	return ""
}

func validateHomeOrganizationPayload(intent string, payload map[string]any, entities api.EntityListResult) string {
	switch intent {
	case "home.sort.configure":
		items, ok := payload[semantic.FieldItems].([]any)
		if !ok || len(items) == 0 {
			return "invalid_home_sort_configure_payload"
		}
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return "invalid_home_sort_configure_payload"
			}
			if reason := validateResourceReference(
				item[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)],
				item[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)],
				entities,
				"invalid_sort_resource_type",
				"invalid_sort_resource_reference",
			); reason != "" {
				return reason
			}
		}
	case "favorite.add", "favorite.update":
		if reason := validateResourceReference(
			payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)],
			payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)],
			entities,
			"invalid_favorite_resource_type",
			"invalid_favorite_resource_reference",
		); reason != "" {
			return reason
		}
	case "favorite.delete":
		if payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)] != nil || payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)] != nil {
			if reason := validateResourceReference(
				payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)],
				payload[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)],
				entities,
				"invalid_favorite_resource_type",
				"invalid_favorite_resource_reference",
			); reason != "" {
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
			if reason := validateResourceReference(
				item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)],
				item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)],
				entities,
				"invalid_favorite_resource_type",
				"invalid_favorite_resource_reference",
			); reason != "" {
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
			if item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)] == nil && item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)] == nil {
				continue
			}
			if reason := validateResourceReference(
				item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetType)],
				item[semantic.InternalField(semantic.DomainFavorite, semantic.FieldTargetID)],
				entities,
				"invalid_favorite_resource_type",
				"invalid_favorite_resource_reference",
			); reason != "" {
				return reason
			}
		}
	}
	return ""
}

func homeOrganizationAcceptedFields(intent string) []string {
	switch intent {
	case "home.sort.configure":
		return append(semanticParameterPaths(semantic.FieldHouseID, semantic.FieldSortType, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldAreaID, semantic.FieldAreaName, semantic.FieldItems),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRank),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldSubIndex),
		)
	case "favorite.add":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldRank)
	case "favorite.update":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldFavoriteID, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldRank)
	case "favorite.delete":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldFavoriteID, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldRank, semantic.FieldConfirmed)
	case "favorite.batch_add":
		return append(semanticParameterPaths(semantic.FieldHouseID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRank),
		)
	case "favorite.batch_update":
		return append(semanticParameterPaths(semantic.FieldHouseID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldFavoriteID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRank),
		)
	case "favorite.batch_delete":
		return append(semanticParameterPaths(semantic.FieldHouseID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldFavoriteID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityType),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldTargetName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomID),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRoomName),
			semanticParameterArrayPath(semantic.FieldItems, semantic.FieldRank),
			semantic.ParameterPath(semantic.FieldConfirmed),
		)
	default:
		return semanticParameterPaths(semantic.FieldHouseID)
	}
}

func homeOrganizationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, homeOrganizationAcceptedFields(request.Intent), payloadGuideForIntent(request.Intent))
}

func payloadItems(payload map[string]any) []any {
	items, ok := payload[semantic.FieldItems].([]any)
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
