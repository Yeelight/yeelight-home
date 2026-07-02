package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func configureClarificationResponse(request contract.Request, reason string, acceptedFields []string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, acceptedFields, nil)
}

func missingHouseIDAcceptedFields() []string {
	return []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
		"local profile houseId",
	}
}

func configureClarificationResponseWithGuide(request contract.Request, reason string, acceptedFields []string, guide map[string]any) contract.Response {
	clarification := map[string]any{
		semantic.FieldReason:         reason,
		semantic.FieldAcceptedFields: acceptedFields,
	}
	for key, value := range guide {
		clarification[key] = value
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充要配置的必要信息。",
		Clarification:   clarification,
		Warnings:        []string{},
		TraceID:         "configure-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
}

func configureClarificationResponseWithCandidates(request contract.Request, reason string, acceptedFields []string, guide map[string]any, candidates []map[string]any) contract.Response {
	response := configureClarificationResponseWithGuide(request, reason, acceptedFields, guide)
	if len(candidates) == 0 {
		return response
	}
	if response.Clarification == nil {
		response.Clarification = map[string]any{}
	}
	response.Clarification[semantic.FieldCandidates] = candidates
	return response
}

func responseWithVerifiedTopology(response contract.Response, entities api.EntityListResult) contract.Response {
	if entities.Total == 0 {
		return response
	}
	if response.Internal == nil {
		response.Internal = map[string]any{}
	}
	response.Internal[semantic.FieldVerifiedTopology] = entities
	return response
}

func executionPreviewResponse(request contract.Request, record operation.Prepared, entities api.EntityListResult) contract.Response {
	return executionPreviewResponseWithDetails(request, record, entities, nil, 0)
}

func executionPreviewResponseWithDetails(request contract.Request, record operation.Prepared, entities api.EntityListResult, preview map[string]any, extraAPICalls int) contract.Response {
	payloadPreview := executionPayloadPreview(record)
	if len(preview) > 0 {
		payloadPreview[semantic.FieldSemanticPreview] = preview
	}
	previewPayload := map[string]any{
		semantic.FieldRisk:           record.Risk,
		semantic.FieldIntent:         record.Intent,
		semantic.FieldSummary:        record.Summary,
		semantic.FieldExecutionModel: "ordinary_invoke_executes_directly",
		semantic.FieldPreconditions:  record.Preconditions,
		semantic.FieldPayloadPreview: payloadPreview,
	}
	if record.Risk == operation.RiskR3 {
		previewPayload[semantic.FieldDestructive] = true
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成语义校验。dry-run 只返回预览；普通 invoke 会直接执行。",
		Result: map[string]any{
			semantic.FieldPreparedForDirectExecution: true,
			semantic.FieldPreview:                    previewPayload,
		},
		Warnings: []string{},
		TraceID:  "invoke-preview",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + extraAPICalls,
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func executionPayloadPreview(record operation.Prepared) map[string]any {
	preview := map[string]any{}
	if operation.IsAccountScope(record.HouseID) {
		preview[semantic.FieldScope] = "account"
	} else {
		preview[semantic.FieldHouseID] = record.HouseID
	}
	for _, key := range []string{
		semantic.FieldName,
		semantic.FieldFavoriteID,
		semantic.FieldTargetName,
		semantic.FieldType,
		semantic.FieldTarget,
		semantic.FieldRoomID,
		semantic.FieldRoomName,
		semantic.FieldTargetRoomName,
		semantic.FieldDeviceID,
		semantic.FieldDeviceIDs,
		semantic.FieldGroupID,
		semantic.FieldGroupIDs,
		semantic.FieldAreaID,
		semantic.FieldGroupCategory,
		semantic.FieldGroupCapability,
		semantic.FieldSceneID,
		semantic.FieldAutomationID,
		semantic.FieldButtonEventID,
		semantic.FieldIndex,
		semantic.FieldStartTime,
		semantic.FieldEndTime,
		semantic.FieldVersion,
	} {
		if value, ok := record.Payload[key]; ok {
			preview[key] = value
		}
	}
	if repeat := semanticRepeatPreview(record.Payload); repeat != nil {
		preview[semantic.FieldRepeat] = repeat
	}
	if window := semanticActiveWindowPreview(record.Payload); window != nil {
		preview[semantic.FieldActiveWindow] = window
		delete(preview, semantic.FieldStartTime)
		delete(preview, semantic.FieldEndTime)
	}
	if typeID, ok := valueInt(record.Payload[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)]); ok {
		preview[semantic.FieldTargetType] = semanticTypeName(typeID)
	}
	if id := valueIDString(record.Payload[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)]); id != "" {
		preview[semantic.FieldTargetID] = id
	}
	if record.Intent == "home.sort.configure" {
		addSemanticTargetPreview(preview, record.Payload, semantic.DomainSort)
		addSemanticSortPayloadPreview(preview, record.Payload)
	}
	if strings.HasPrefix(record.Intent, "favorite.") {
		addSemanticTargetPreview(preview, record.Payload, semantic.DomainFavorite)
	}
	if rank, ok := record.Payload[semantic.FieldRank]; ok {
		preview[semantic.FieldRank] = rank
	}
	if params, ok := record.Payload[semantic.InternalAutomationParamsField()]; ok {
		preview[semantic.FieldConditions] = semanticConditionPreview(params)
	}
	if items, ok := record.Payload[semantic.FieldItems]; ok {
		switch {
		case record.Intent == "home.sort.configure":
			preview[semantic.FieldItems] = previewList(semanticTargetRowsPreview(items, semantic.DomainSort), 20)
		case strings.HasPrefix(record.Intent, "favorite."):
			preview[semantic.FieldItems] = previewList(semanticTargetRowsPreview(items, semantic.DomainFavorite), 20)
		default:
			preview[semantic.FieldItems] = previewList(items, 20)
		}
	}
	if buttons, ok := record.Payload[semantic.FieldButtons]; ok {
		preview[semantic.FieldButtons] = previewList(semanticPanelRowsPreview(buttons), 20)
	}
	if buttonEvents, ok := record.Payload[semantic.FieldButtonEvents]; ok {
		preview[semantic.FieldButtonEvents] = previewList(semanticPanelEventRowsPreview(buttonEvents), 20)
	}
	if buttonEvent, ok := record.Payload[semantic.FieldButtonEvent]; ok {
		eventPreview := semanticPanelEventRowsPreview([]any{buttonEvent})
		if rows, ok := eventPreview.([]any); ok && len(rows) == 1 {
			preview[semantic.FieldButtonEvent] = rows[0]
		} else {
			preview[semantic.FieldButtonEvent] = buttonEvent
		}
	}
	if details, ok := record.Payload[semantic.FieldDetails]; ok {
		if strings.HasPrefix(record.Intent, "panel.") || strings.HasPrefix(record.Intent, "knob.") {
			preview[semantic.FieldActions] = previewList(semanticPanelRowsPreview(details), 20)
		} else {
			preview[semantic.FieldActions] = previewList(semanticActionPreviewList(details), 20)
		}
	}
	if actions, ok := record.Payload[semantic.FieldActions]; ok {
		preview[semantic.FieldActions] = previewList(semanticPreviewActions(record.Intent, actions), 20)
	}
	return preview
}

func addSemanticSortPayloadPreview(preview map[string]any, source map[string]any) {
	delete(preview, semantic.FieldType)
	delete(preview, semantic.FieldTarget)
	sortType, ok := valueInt(source[semantic.FieldType])
	if !ok {
		return
	}
	preview[semantic.FieldSortType] = semanticSortTypeName(sortType)
	switch sortType {
	case 1, 2:
		if id := valueIDString(source[semantic.FieldRoomID]); id != "" {
			preview[semantic.FieldRoomID] = id
		}
	case 4:
		if id := valueIDString(source[semantic.FieldParentID]); id != "" {
			preview[semantic.FieldParentID] = id
		}
	case 5:
		if id := valueIDString(source[semantic.FieldAreaID]); id != "" {
			preview[semantic.FieldAreaID] = id
		}
	}
}

func semanticSortTypeName(sortType int) string {
	switch sortType {
	case 1:
		return "device_room"
	case 2:
		return "scene_room"
	case 3:
		return "home_rooms"
	case 4:
		return "sub_device"
	case 5:
		return "area_rooms"
	default:
		return "unknown"
	}
}

func semanticTargetRowsPreview(value any, domain string) any {
	rows, ok := value.([]any)
	if !ok {
		return value
	}
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			result = append(result, raw)
			continue
		}
		preview := map[string]any{}
		addSemanticTargetPreview(preview, item, domain)
		for _, key := range []string{
			semantic.FieldID,
			semantic.FieldFavoriteID,
			semantic.FieldTargetName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRank,
			semantic.FieldSubIndex,
			semantic.FieldValid,
		} {
			if value, ok := item[key]; ok {
				preview[key] = value
			}
		}
		result = append(result, preview)
	}
	return result
}

func addSemanticTargetPreview(preview map[string]any, source map[string]any, domain string) {
	if typeID, ok := valueInt(source[semantic.InternalField(domain, semantic.FieldTargetType)]); ok {
		preview[semantic.FieldTargetType] = semanticTypeName(typeID)
	}
	if id := valueIDString(source[semantic.InternalField(domain, semantic.FieldTargetID)]); id != "" {
		preview[semantic.FieldTargetID] = id
	}
}

func semanticPanelRowsPreview(value any) any {
	rows, ok := value.([]any)
	if !ok {
		return value
	}
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			result = append(result, raw)
			continue
		}
		result = append(result, semanticPanelRowPreview(item))
	}
	return result
}

func semanticPanelEventRowsPreview(value any) any {
	rows, ok := value.([]any)
	if !ok {
		return value
	}
	result := make([]any, 0, len(rows))
	for _, raw := range rows {
		item, ok := raw.(map[string]any)
		if !ok {
			result = append(result, raw)
			continue
		}
		preview := semanticPanelRowPreview(item)
		if details, ok := item[semantic.FieldDetails]; ok {
			preview[semantic.FieldActions] = semanticPanelRowsPreview(details)
		}
		result = append(result, preview)
	}
	return result
}

func semanticPanelRowPreview(item map[string]any) map[string]any {
	preview := map[string]any{}
	for _, key := range []string{
		semantic.FieldID,
		semantic.FieldDeviceID,
		semantic.FieldName,
		semantic.FieldAlias,
		semantic.FieldKeyValue,
		semantic.FieldIndex,
		semantic.FieldVisible,
		semantic.FieldIcon,
		semantic.FieldSort,
		semantic.FieldType,
		semantic.FieldExtend,
		semantic.FieldButtonEventID,
		semantic.FieldRoomID,
		semantic.FieldRank,
		semantic.FieldStartTime,
		semantic.FieldEndTime,
		semantic.FieldConfigType,
		semantic.FieldMode,
		semantic.FieldModel,
		semantic.FieldAction,
		semantic.FieldProperty,
		semantic.FieldValue,
		semantic.FieldDelay,
		semantic.FieldDuration,
	} {
		if value, ok := item[key]; ok {
			preview[key] = value
		}
	}
	if value, ok := item[semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetID)]; ok {
		preview[semantic.FieldTargetID] = value
	}
	if typeID, ok := valueInt(firstNonNil(
		item[semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetType)],
		item[semantic.InternalField(semantic.DomainKnob, semantic.FieldTargetType)],
		item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)],
	)); ok {
		preview[semantic.FieldTargetType] = semanticTypeName(typeID)
	}
	if value, ok := item[semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetName)]; ok {
		preview[semantic.FieldTargetName] = value
	}
	if value, ok := firstNonNilMap(item, semantic.InternalPanelActionParamsField(), semantic.InternalKnobActionParamsField()); ok {
		if params := semanticParamsPreview(value); params != nil {
			preview[semantic.FieldSet] = params
		}
	}
	if value, ok := firstNonNilMap(item, semantic.InternalField(semantic.DomainPanel, semantic.FieldSubIndex), semantic.InternalField(semantic.DomainKnob, semantic.FieldSubIndex)); ok {
		preview[semantic.FieldSubIndex] = value
	}
	if value, ok := item[semantic.InternalField(semantic.DomainKnob, semantic.FieldSensitivity)]; ok {
		preview[semantic.FieldSensitivity] = value
	}
	return preview
}

func firstNonNilMap(values map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != nil {
			return value, true
		}
	}
	return nil, false
}

func semanticPreviewActions(intent string, actions any) any {
	if intent == "lighting.design.apply" {
		return semanticLightingApplyActionsPreview(actions)
	}
	return semanticActionPreviewList(actions)
}

func semanticLightingApplyActionsPreview(value any) any {
	rawItems, ok := value.([]any)
	if !ok {
		return value
	}
	result := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			result = append(result, raw)
			continue
		}
		preview := map[string]any{}
		if id := valueIDString(item[semantic.FieldDeviceID]); id != "" {
			preview[semantic.FieldTargetID] = id
		}
		if name := requestString(item[semantic.FieldDeviceName]); name != "" {
			preview[semantic.FieldTargetName] = name
		}
		preview[semantic.FieldProperty] = semantic.LightPropertyName(requestString(item[semantic.FieldProperty]))
		if value, ok := item[semantic.FieldValue]; ok {
			preview[semantic.FieldValue] = value
		}
		result = append(result, preview)
	}
	return result
}

func semanticLightPropertyName(value string) string {
	switch value {
	case semantic.InternalField(semantic.DomainAction, semantic.FieldPower):
		return semantic.FieldPower
	case semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness):
		return semantic.FieldBrightness
	case semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature):
		return semantic.FieldColorTemperature
	case semantic.InternalField(semantic.DomainAction, semantic.FieldColor):
		return semantic.FieldColor
	default:
		return value
	}
}

func semanticActionPreviewList(value any) any {
	items, ok := value.([]map[string]any)
	if !ok {
		rawItems, rawOK := value.([]any)
		if !rawOK {
			return value
		}
		result := make([]any, 0, len(rawItems))
		for _, raw := range rawItems {
			if item, ok := raw.(map[string]any); ok {
				result = append(result, semanticActionPreview(item))
			} else {
				result = append(result, raw)
			}
		}
		return result
	}
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, semanticActionPreview(item))
	}
	return result
}

func semanticActionPreview(item map[string]any) map[string]any {
	preview := map[string]any{}
	if typeID, ok := valueInt(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)]); ok {
		preview[semantic.FieldTargetType] = semanticTypeName(typeID)
	}
	if id := valueIDString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetID)]); id != "" {
		preview[semantic.FieldTargetID] = id
	}
	if tempID := valueIDString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetKey)]); tempID != "" {
		preview[semantic.FieldTargetKey] = tempID
	}
	if name := requestString(item[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)]); name != "" {
		preview[semantic.FieldTargetName] = name
	}
	for _, key := range []string{
		semantic.FieldRank,
		semantic.InternalField(semantic.DomainAction, semantic.FieldSubIndex),
		semantic.FieldAction,
		semantic.FieldRoomID,
		semantic.FieldStartTime,
		semantic.FieldEndTime,
	} {
		if value, ok := item[key]; ok {
			previewKey := key
			if key == semantic.InternalField(semantic.DomainAction, semantic.FieldSubIndex) {
				previewKey = semantic.FieldSubIndex
			}
			preview[previewKey] = value
		}
	}
	if params, ok := item[semantic.InternalActionParamsField()]; ok {
		if params := semanticParamsPreview(params); params != nil {
			preview[semantic.FieldSet] = params
		}
	}
	return preview
}

func semanticTypeName(typeID int) string {
	switch typeID {
	case groupTypeRoom:
		return "room"
	case groupTypeDevice:
		return "device"
	case groupTypeCustom:
		return "group"
	case groupTypeMesh:
		return "meshGroup"
	case groupTypeScene:
		return "scene"
	case groupTypeAutomation:
		return "automation"
	default:
		return "unknown"
	}
}

func semanticParamsPreview(value any) any {
	var params map[string]any
	switch typed := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(strings.TrimSpace(typed)), &params); err != nil {
			return value
		}
	case map[string]any:
		params = typed
	default:
		return value
	}
	result := map[string]any{}
	for key, item := range params {
		result[key] = item
	}
	if set, ok := params[semantic.FieldSet].(map[string]any); ok {
		return semanticLightSetPreview(set)
	}
	return result
}

func semanticConditionPreview(value any) any {
	var params map[string]any
	switch typed := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(strings.TrimSpace(typed)), &params); err != nil {
			return value
		}
	case map[string]any:
		params = typed
	default:
		return value
	}
	return semantic.ToPublicConditionParams(params)
}

func semanticActiveWindowPreview(payload map[string]any) map[string]any {
	start := requestString(payload[semantic.FieldStartTime])
	end := requestString(payload[semantic.FieldEndTime])
	if start == "" && end == "" {
		return nil
	}
	return map[string]any{
		semantic.FieldStart: start,
		semantic.FieldEnd:   end,
	}
}

func semanticRepeatPreview(payload map[string]any) any {
	repeatType, ok := valueInt(payload[semantic.InternalRepeatTypeField()])
	if !ok {
		return nil
	}
	repeatValue := requestString(payload[semantic.InternalRepeatValueField()])
	switch repeatType {
	case 1:
		return "once"
	case 2:
		return "daily"
	case 3:
		return "weekdays"
	case 5:
		return "weekend"
	case 6:
		return "legal_holiday"
	case 7:
		return "legal_workday"
	default:
		if repeatValue != "" {
			return map[string]any{semantic.FieldType: "custom", semantic.FieldRepeatDays: repeatValue}
		}
		return map[string]any{semantic.FieldType: "custom"}
	}
}

func semanticLightSetPreview(set map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range set {
		result[key] = value
	}
	for semanticKey, rawKey := range map[string]string{
		semantic.FieldPower:            semantic.InternalField(semantic.DomainAction, semantic.FieldPower),
		semantic.FieldBrightness:       semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness),
		semantic.FieldColorTemperature: semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature),
		semantic.FieldColor:            semantic.InternalField(semantic.DomainAction, semantic.FieldColor),
	} {
		if value, ok := set[rawKey]; ok {
			result[semanticKey] = value
			delete(result, rawKey)
		}
	}
	return result
}

func previewList(value any, limit int) any {
	items, ok := value.([]any)
	if !ok || limit <= 0 || len(items) <= limit {
		return value
	}
	return map[string]any{
		semantic.FieldCount: len(items),
		semantic.FieldItems: items[:limit],
	}
}

func homeOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeOrganizationResult) contract.Response {
	userMessage := "已提交并验证首页组织配置。"
	status := "success"
	warnings := []string{}
	var responseError *contract.Error
	if !result.Verified {
		status = "partial"
		userMessage = "已提交首页组织配置，但读后验证未完全匹配。"
		if result.Warning != "" {
			warnings = append(warnings, result.Warning)
		}
		responseError = &contract.Error{
			Code:    "write_verification_mismatch",
			Message: "home organization write did not fully match read-after-write verification",
		}
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     userMessage,
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldItemCount:  result.ItemCount,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: warnings,
		TraceID:  "home-organization-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
		Error: responseError,
	}
}

func homeCreateAlreadyExistsResponse(request contract.Request, house api.HouseSummary, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("家庭 %s 已存在，无需创建。", house.Name),
		Result: map[string]any{
			semantic.FieldRegion:     "",
			semantic.FieldHouseID:    house.ID,
			semantic.FieldName:       house.Name,
			semantic.FieldCreated:    false,
			semantic.FieldVerified:   true,
			semantic.FieldVerifiedBy: firstNonEmptyString(house.Source, "home.summary"),
		},
		Warnings: []string{},
		TraceID:  "home-create-idempotent",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func homeCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeCreateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证家庭创建操作。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldName:       result.Name,
			semantic.FieldCreated:    result.Created,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "home-create-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func homeMemberExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeMemberResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成家庭成员操作，并通过成员列表回读验证。",
		Result: map[string]any{
			semantic.FieldIntent:           record.Intent,
			semantic.FieldRisk:             record.Risk,
			semantic.FieldRegion:           result.Region,
			semantic.FieldHouseID:          result.HouseID,
			semantic.FieldCapability:       result.Capability,
			semantic.FieldVerified:         result.Verified,
			semantic.FieldVerifiedBy:       result.VerifiedBy,
			semantic.FieldResultData:       result.Data,
			semantic.FieldPersistentWrites: true,
		},
		Warnings: []string{},
		TraceID:  "home-member-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func homeLockExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeLockResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证整屋重置锁定配置。",
		Result: map[string]any{
			semantic.FieldRegion:      result.Region,
			semantic.FieldHouseID:     result.HouseID,
			semantic.FieldCapability:  result.Capability,
			semantic.FieldDeviceCount: result.DeviceCount,
			semantic.FieldVerified:    result.Verified,
			semantic.FieldVerifiedBy:  result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "home-lock-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func entityBatchRenameExecuteResponse(request contract.Request, record operation.Prepared, result api.EntityBatchRenameResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量重命名。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldItemCount:  result.ItemCount,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "entity-batch-rename-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func homeSpaceConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeSpaceConfigurationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证家庭空间配置。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldItemCount:  result.ItemCount,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "home-space-configuration-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func lightingDesignApplyExecuteResponse(request contract.Request, record operation.Prepared, entities api.EntityListResult, results []any, apiCalls int) contract.Response {
	allVerified := true
	for _, item := range results {
		row, ok := item.(map[string]any)
		if !ok || row[semantic.FieldVerified] != true {
			allVerified = false
			break
		}
	}
	status := "success"
	traceID := "lighting-design-apply-execute"
	warnings := append([]string{}, entities.Warnings...)
	var responseError *contract.Error
	if !allVerified {
		status = "partial"
		traceID = "lighting-design-apply-verification-mismatch"
		warnings = append(warnings, "write_verification_mismatch")
		responseError = &contract.Error{
			Code:    "write_verification_mismatch",
			Message: "one or more lighting design actions did not match expected values after write",
		}
	}
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     "已提交受限照明设计应用操作，并完成设备状态验证。",
		Result: map[string]any{
			semantic.FieldRegion:           entities.Region,
			semantic.FieldHouseID:          record.HouseID,
			semantic.FieldCapability:       "lighting.design.apply",
			semantic.FieldPersistentWrites: true,
			semantic.FieldCreatedArtifacts: []string{},
			semantic.FieldActionCount:      len(results),
			semantic.FieldResults:          results,
			semantic.FieldVerified:         allVerified,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
		Error: responseError,
	}, entities)
}

func automationStatusExecuteResponse(request contract.Request, record operation.Prepared, result api.AutomationStatusResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证自动化状态。",
		Result: map[string]any{
			semantic.FieldRegion:       result.Region,
			semantic.FieldHouseID:      result.HouseID,
			semantic.FieldAutomationID: result.AutomationID,
			semantic.FieldName:         result.Name,
			semantic.FieldStatus:       result.Status,
			semantic.FieldStatusLabel:  automationStatusLabel(result.Status),
			semantic.FieldCapability:   result.Capability,
			semantic.FieldVerified:     result.Verified,
			semantic.FieldVerifiedBy:   result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "automation-status-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func automationStatusLabel(status string) string {
	switch status {
	case "1":
		return "enabled"
	case "0":
		return "disabled"
	default:
		return "unknown"
	}
}

func automationUpdateExecuteResponse(request contract.Request, record operation.Prepared, result api.AutomationUpdateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证自动化更新。",
		Result: map[string]any{
			semantic.FieldRegion:       result.Region,
			semantic.FieldHouseID:      result.HouseID,
			semantic.FieldAutomationID: result.AutomationID,
			semantic.FieldName:         result.Name,
			semantic.FieldStatus:       result.Status,
			semantic.FieldVerified:     result.Verified,
			semantic.FieldVerifiedBy:   result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "automation-update-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func sceneUpdateExecuteResponse(request contract.Request, record operation.Prepared, result api.SceneUpdateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证情景更新。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldSceneID:    result.SceneID,
			semantic.FieldName:       result.Name,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "scene-update-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func metadataDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证删除操作。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldEntityType: result.EntityType,
			semantic.FieldEntityID:   result.EntityID,
			semantic.FieldName:       result.Name,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-delete-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func metadataBatchDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataBatchDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量删除操作。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldEntityType: result.EntityType,
			semantic.FieldItemCount:  result.ItemCount,
			semantic.FieldResults:    result.Results,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-batch-delete-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func destructiveDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.DestructiveDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证高影响删除操作。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldEntityType: result.EntityType,
			semantic.FieldEntityID:   result.EntityID,
			semantic.FieldName:       result.Name,
			semantic.FieldRisk:       record.Risk,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "destructive-delete-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func deviceUnbindExecuteResponse(request contract.Request, record operation.Prepared, result api.DeviceUnbindResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成设备解绑，并通过实体列表回读验证。",
		Result: map[string]any{
			semantic.FieldIntent:               record.Intent,
			semantic.FieldRisk:                 record.Risk,
			semantic.FieldRegion:               result.Region,
			semantic.FieldHouseID:              result.HouseID,
			semantic.FieldDeviceID:             result.DeviceID,
			semantic.FieldName:                 result.Name,
			semantic.FieldClearMAC:             result.ClearMac,
			semantic.FieldUnbindRelatedDevices: result.UnbindRelDevices,
			semantic.FieldVerified:             result.Verified,
			semantic.FieldVerifiedBy:           result.VerifiedBy,
			semantic.FieldPersistentWrites:     true,
		},
		Warnings: []string{},
		TraceID:  "device-unbind-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func spaceOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.SpaceOrganizationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证空间组织配置。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldEntityType: result.EntityType,
			semantic.FieldEntityID:   result.EntityID,
			semantic.FieldName:       result.Name,
			semantic.FieldRoomID:     result.RoomID,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "space-organization-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func gatewayConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.GatewayConfigurationResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已更新网关配置，并通过网关详情回读验证。",
		Result: map[string]any{
			semantic.FieldIntent:           record.Intent,
			semantic.FieldRisk:             record.Risk,
			semantic.FieldRegion:           result.Region,
			semantic.FieldHouseID:          result.HouseID,
			semantic.FieldCapability:       result.Capability,
			semantic.FieldGatewayID:        result.GatewayID,
			semantic.FieldName:             result.Name,
			semantic.FieldVerified:         result.Verified,
			semantic.FieldVerifiedBy:       result.VerifiedBy,
			semantic.FieldPersistentWrites: true,
		},
		Warnings: []string{},
		TraceID:  "gateway-configuration-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func spaceBatchOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.SpaceBatchOrganizationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量空间组织配置。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldItemCount:  result.ItemCount,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "space-batch-organization-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func panelConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.PanelConfigurationResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证面板/旋钮配置。",
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldDeviceID:   result.DeviceID,
			semantic.FieldCapability: result.Capability,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "panel-configuration-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func roomCreateAlreadyExistsResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("房间 %s 已存在，无需创建。", entity.Name),
		Result: map[string]any{
			semantic.FieldRegion:   entities.Region,
			semantic.FieldHouseID:  entities.HouseID,
			semantic.FieldEntity:   entitySummaryMap(entity),
			semantic.FieldCreated:  false,
			semantic.FieldVerified: true,
		},
		Warnings: []string{},
		TraceID:  "room-create-idempotent",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities),
			semantic.FieldCacheHits: 0,
		},
	}
}

func metadataCreateAlreadyExistsResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, label string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("%s %s 已存在，无需创建。", label, entity.Name),
		Result: map[string]any{
			semantic.FieldRegion:   entities.Region,
			semantic.FieldHouseID:  entities.HouseID,
			semantic.FieldEntity:   entitySummaryMap(entity),
			semantic.FieldCreated:  false,
			semantic.FieldVerified: true,
		},
		Warnings: []string{},
		TraceID:  "metadata-create-idempotent",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities),
			semantic.FieldCacheHits: 0,
		},
	}
}

func roomCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.RoomCreateResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已创建并验证房间 %s。", result.Name),
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldRoomID:     result.RoomID,
			semantic.FieldName:       result.Name,
			semantic.FieldCreated:    result.Created,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "room-create-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func metadataCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataCreateResult, label string) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已创建并验证%s %s。", label, result.Name),
		Result: map[string]any{
			semantic.FieldRegion:     result.Region,
			semantic.FieldHouseID:    result.HouseID,
			semantic.FieldEntityType: result.EntityType,
			semantic.FieldEntityID:   result.EntityID,
			semantic.FieldName:       result.Name,
			semantic.FieldCreated:    result.Created,
			semantic.FieldVerified:   result.Verified,
			semantic.FieldVerifiedBy: result.VerifiedBy,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-create-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func executionBlockedResponse(request contract.Request, code string, message string) contract.Response {
	return executionBlockedResponseWithResult(request, code, message, nil)
}

func executionVerifyBlockedResponse(request contract.Request, record operation.Prepared, verifyErr error) contract.Response {
	code, safeNextStep := executionVerifyCode(verifyErr)
	result := map[string]any{
		semantic.FieldIntent: record.Intent,
		semantic.FieldRecovery: map[string]any{
			semantic.FieldSuggestedIntent: record.Intent,
			semantic.FieldSafeNextStep:    safeNextStep,
			semantic.FieldCanRegenerate:   true,
			semantic.FieldSafeToRetry:     false,
		},
	}
	return executionBlockedResponseWithResult(request, code, verifyErr.Error(), result)
}

func executionBlockedResponseWithResult(request contract.Request, code string, message string, extra map[string]any) contract.Response {
	result := map[string]any{}
	for key, value := range extra {
		result[key] = value
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Result:          result,
		Warnings:        []string{code},
		TraceID:         "execution-blocked",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
		Error: &contract.Error{
			Code:    code,
			Message: message,
		},
	}
}
