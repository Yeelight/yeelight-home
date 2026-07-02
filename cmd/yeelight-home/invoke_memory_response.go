package main

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func memoryRememberResponse(request contract.Request, upserts []storage.PreferenceUpsertResult) contract.Response {
	items := make([]any, 0, len(upserts))
	createdCount := 0
	mergedCount := 0
	for _, upsert := range upserts {
		if upsert.Created {
			createdCount++
		}
		if upsert.Merged {
			mergedCount++
		}
		items = append(items, memoryItemMap(upsert))
	}
	first := map[string]any{}
	if len(upserts) > 0 {
		first = memoryItemMap(upserts[0])
	}
	for key, value := range map[string]any{
		semantic.FieldCount:        len(items),
		semantic.FieldCreatedCount: createdCount,
		semantic.FieldMergedCount:  mergedCount,
		semantic.FieldItems:        items,
	} {
		first[key] = value
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已保存 %d 条本地偏好。", len(items)),
		Memory:          first,
		Warnings:        []string{},
		TraceID:         "memory-remember-local",
		Metrics:         noAPIMetrics(),
	}
}

func memoryItemMap(upsert storage.PreferenceUpsertResult) map[string]any {
	memory := upsert.Record
	return map[string]any{
		semantic.FieldID:              memory.ID,
		semantic.FieldProfile:         memory.Profile,
		semantic.FieldRegion:          memory.Region,
		semantic.FieldHouseID:         memory.HouseID,
		semantic.FieldKind:            memory.Kind,
		semantic.FieldStatus:          memory.Status,
		semantic.FieldScopeType:       memory.ScopeType,
		semantic.FieldScopeRef:        memory.ScopeRef,
		semantic.FieldPreferenceType:  memory.PreferenceType,
		semantic.FieldPreferenceValue: memory.PreferenceValue,
		semantic.FieldEvidence:        memory.Evidence,
		semantic.FieldCreated:         upsert.Created,
		semantic.FieldMerged:          upsert.Merged,
	}
}

func memoryListResponse(request contract.Request, houseID string, consent storage.ConsentRecord, preferences []storage.PreferenceRecord) contract.Response {
	items := make([]any, 0, len(preferences))
	for _, item := range preferences {
		items = append(items, map[string]any{
			semantic.FieldID:              item.ID,
			semantic.FieldRegion:          item.Region,
			semantic.FieldKind:            item.Kind,
			semantic.FieldStatus:          item.Status,
			semantic.FieldScopeType:       item.ScopeType,
			semantic.FieldScopeRef:        item.ScopeRef,
			semantic.FieldPreferenceType:  item.PreferenceType,
			semantic.FieldPreferenceValue: item.PreferenceValue,
			semantic.FieldEvidence:        item.Evidence,
			semantic.FieldUpdatedAt:       item.UpdatedAt,
		})
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已读取 %d 条本地记忆。", len(items)),
		Memory: map[string]any{
			semantic.FieldHouseID:         houseID,
			semantic.FieldRegion:          consent.Region,
			semantic.FieldNamespace:       memoryResponseNamespace(consent.Profile, consent.Region, houseID),
			semantic.FieldLearningEnabled: consent.LearningEnabled,
			semantic.FieldPaused:          consent.Paused,
			semantic.FieldItems:           items,
		},
		Warnings: []string{},
		TraceID:  "memory-list-local",
		Metrics:  noAPIMetrics(),
	}
}

func memoryPauseResumeResponse(request contract.Request, consent storage.ConsentRecord) contract.Response {
	statusText := "已恢复本地学习。"
	traceID := "memory-resume-local"
	if consent.Paused {
		statusText = "已暂停本地学习。"
		traceID = "memory-pause-local"
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     statusText,
		Memory: map[string]any{
			semantic.FieldHouseID:         consent.HouseID,
			semantic.FieldRegion:          consent.Region,
			semantic.FieldNamespace:       memoryResponseNamespace(consent.Profile, consent.Region, consent.HouseID),
			semantic.FieldLearningEnabled: consent.LearningEnabled,
			semantic.FieldPaused:          consent.Paused,
			semantic.FieldConsentVersion:  consent.ConsentVersion,
		},
		Warnings: []string{},
		TraceID:  traceID,
		Metrics:  noAPIMetrics(),
	}
}

func memoryForgetResponse(request contract.Request, houseID string, exported map[string]any) contract.Response {
	count := 0
	if preferences, ok := exported[semantic.FieldPreferences].([]storage.PreferenceRecord); ok {
		count += len(preferences)
	}
	if recommendations, ok := exported[semantic.FieldRecommendations].([]storage.RecommendationRecord); ok {
		count += len(recommendations)
	}
	userMessage := "已删除当前家庭的本地记忆。"
	if hasTargetedMemoryForgetIDs(request) {
		userMessage = fmt.Sprintf("已删除 %d 条指定本地记忆。", count)
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     userMessage,
		Memory: map[string]any{
			semantic.FieldHouseID:      houseID,
			semantic.FieldDeletedCount: count,
			semantic.FieldExport:       exported,
		},
		Warnings: []string{},
		TraceID:  "memory-forget-local",
		Metrics:  noAPIMetrics(),
	}
}

func hasTargetedMemoryForgetIDs(request contract.Request) bool {
	preferenceIDs, recommendationIDs := memoryForgetIDs(request)
	return len(preferenceIDs) > 0 || len(recommendationIDs) > 0
}

func recommendationListResponse(request contract.Request, profile string, region string, houseID string, recommendations []storage.RecommendationRecord) contract.Response {
	if len(recommendations) == 0 {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "success",
			UserMessage:     "当前没有新的本地推荐。",
			Recommendation: map[string]any{
				semantic.FieldHouseID:   houseID,
				semantic.FieldRegion:    region,
				semantic.FieldNamespace: memoryResponseNamespace(profile, region, houseID),
				semantic.FieldItems:     []any{},
			},
			Warnings: []string{},
			TraceID:  "recommendation-list-local",
			Metrics:  noAPIMetrics(),
		}
	}
	item := recommendations[0]
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已返回 1 条本地推荐。",
		Recommendation: map[string]any{
			semantic.FieldHouseID:      houseID,
			semantic.FieldRegion:       item.Region,
			semantic.FieldNamespace:    memoryResponseNamespace(item.Profile, item.Region, houseID),
			semantic.FieldItems:        []any{recommendationItemMap(item)},
			semantic.FieldSessionLimit: 1,
		},
		Warnings: []string{},
		TraceID:  "recommendation-list-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationRecordResponse(request contract.Request, houseID string, upsert storage.RecommendationUpsertResult) contract.Response {
	item := recommendationItemMap(upsert.Record)
	item[semantic.FieldCreated] = upsert.Created
	item[semantic.FieldMerged] = upsert.Merged
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已保存本地推荐候选。",
		Recommendation: map[string]any{
			semantic.FieldHouseID: houseID,
			semantic.FieldItem:    item,
		},
		Warnings: []string{},
		TraceID:  "recommendation-record-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationItemMap(item storage.RecommendationRecord) map[string]any {
	result := map[string]any{
		semantic.FieldID:          item.ID,
		semantic.FieldRegion:      item.Region,
		semantic.FieldType:        item.Type,
		semantic.FieldSource:      item.Source,
		semantic.FieldExplanation: item.Explanation,
		semantic.FieldEvidence:    item.Evidence,
		semantic.FieldStatus:      item.Status,
		semantic.FieldUpdatedAt:   item.UpdatedAt,
	}
	if item.TargetIntent != "" {
		result[semantic.FieldTargetIntent] = item.TargetIntent
	}
	if item.ScopeType != "" {
		result[semantic.FieldScopeType] = item.ScopeType
	}
	if item.ScopeRef != "" {
		result[semantic.FieldScopeRef] = item.ScopeRef
	}
	if item.Priority != 0 {
		result[semantic.FieldPriority] = item.Priority
	}
	if item.Confidence != "" {
		result[semantic.FieldConfidence] = item.Confidence
	}
	if item.ActionHint != nil {
		result[semantic.FieldActionHint] = item.ActionHint
	}
	if item.ParametersHint != nil {
		result[semantic.FieldParametersHint] = item.ParametersHint
	}
	return result
}

func memoryResponseNamespace(profile string, region string, houseID string) map[string]any {
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		region = "default"
	}
	return map[string]any{
		semantic.FieldAccountProfile: profile,
		semantic.FieldProfile:        profile,
		semantic.FieldRegion:         region,
		semantic.FieldHouseID:        houseID,
		semantic.FieldDataType:       "memory",
	}
}

func recommendationFeedbackResponse(request contract.Request, houseID string, recommendation storage.RecommendationRecord) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已更新本地推荐反馈。",
		Recommendation: map[string]any{
			semantic.FieldHouseID:          houseID,
			semantic.FieldID:               recommendation.ID,
			semantic.FieldStatus:           recommendation.Status,
			semantic.FieldCooldownUntil:    recommendation.CooldownUntil,
			semantic.FieldUpdatedAt:        recommendation.UpdatedAt,
			semantic.FieldFeedbackRecorded: true,
		},
		Warnings: []string{},
		TraceID:  "recommendation-feedback-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationFeedbackBlockedResponse(request contract.Request, recommendationID string, code string, message string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Recommendation: map[string]any{
			semantic.FieldID: recommendationID,
		},
		Warnings: []string{code},
		TraceID:  "recommendation-feedback-blocked",
		Metrics:  noAPIMetrics(),
		Error: &contract.Error{
			Code:    code,
			Message: message,
		},
	}
}

func memoryClarificationResponse(request contract.Request, reason string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充本地记忆所需的信息。",
		Clarification: map[string]any{
			semantic.FieldReason: reason,
		},
		Warnings: []string{},
		TraceID:  "memory-clarification",
		Metrics:  noAPIMetrics(),
	}
}

func memoryBlockedResponse(request contract.Request, code string, message string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Warnings:        []string{code},
		TraceID:         "memory-blocked",
		Metrics:         noAPIMetrics(),
		Error:           &contract.Error{Code: code, Message: message},
	}
}

func noAPIMetrics() map[string]any {
	return map[string]any{
		semantic.FieldAPICalls:  0,
		semantic.FieldCacheHits: 0,
	}
}
