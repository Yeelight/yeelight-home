package main

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
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
		"count":        len(items),
		"createdCount": createdCount,
		"mergedCount":  mergedCount,
		"items":        items,
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
		"id":              memory.ID,
		"profile":         memory.Profile,
		"region":          memory.Region,
		"houseId":         memory.HouseID,
		"kind":            memory.Kind,
		"status":          memory.Status,
		"scopeType":       memory.ScopeType,
		"scopeRef":        memory.ScopeRef,
		"preferenceType":  memory.PreferenceType,
		"preferenceValue": memory.PreferenceValue,
		"evidence":        memory.Evidence,
		"created":         upsert.Created,
		"merged":          upsert.Merged,
	}
}

func memoryListResponse(request contract.Request, houseID string, consent storage.ConsentRecord, preferences []storage.PreferenceRecord) contract.Response {
	items := make([]any, 0, len(preferences))
	for _, item := range preferences {
		items = append(items, map[string]any{
			"id":              item.ID,
			"region":          item.Region,
			"kind":            item.Kind,
			"status":          item.Status,
			"scopeType":       item.ScopeType,
			"scopeRef":        item.ScopeRef,
			"preferenceType":  item.PreferenceType,
			"preferenceValue": item.PreferenceValue,
			"evidence":        item.Evidence,
			"updatedAt":       item.UpdatedAt,
		})
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已读取 %d 条本地记忆。", len(items)),
		Memory: map[string]any{
			"houseId":         houseID,
			"region":          consent.Region,
			"namespace":       memoryResponseNamespace(consent.Profile, consent.Region, houseID),
			"learningEnabled": consent.LearningEnabled,
			"paused":          consent.Paused,
			"items":           items,
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
			"houseId":         consent.HouseID,
			"region":          consent.Region,
			"namespace":       memoryResponseNamespace(consent.Profile, consent.Region, consent.HouseID),
			"learningEnabled": consent.LearningEnabled,
			"paused":          consent.Paused,
			"consentVersion":  consent.ConsentVersion,
		},
		Warnings: []string{},
		TraceID:  traceID,
		Metrics:  noAPIMetrics(),
	}
}

func memoryForgetResponse(request contract.Request, houseID string, exported map[string]any) contract.Response {
	count := 0
	if preferences, ok := exported["preferences"].([]storage.PreferenceRecord); ok {
		count += len(preferences)
	}
	if recommendations, ok := exported["recommendations"].([]storage.RecommendationRecord); ok {
		count += len(recommendations)
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已删除当前家庭的本地记忆。",
		Memory: map[string]any{
			"houseId":      houseID,
			"deletedCount": count,
			"export":       exported,
		},
		Warnings: []string{},
		TraceID:  "memory-forget-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationListResponse(request contract.Request, profile string, region string, houseID string, recommendations []storage.RecommendationRecord) contract.Response {
	if len(recommendations) == 0 {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "success",
			UserMessage:     "当前没有新的本地推荐。",
			Recommendation: map[string]any{
				"houseId":   houseID,
				"region":    region,
				"namespace": memoryResponseNamespace(profile, region, houseID),
				"items":     []any{},
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
			"houseId":      houseID,
			"region":       item.Region,
			"namespace":    memoryResponseNamespace(item.Profile, item.Region, houseID),
			"items":        []any{recommendationItemMap(item)},
			"sessionLimit": 1,
		},
		Warnings: []string{},
		TraceID:  "recommendation-list-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationRecordResponse(request contract.Request, houseID string, upsert storage.RecommendationUpsertResult) contract.Response {
	item := recommendationItemMap(upsert.Record)
	item["created"] = upsert.Created
	item["merged"] = upsert.Merged
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已保存本地推荐候选。",
		Recommendation: map[string]any{
			"houseId": houseID,
			"item":    item,
		},
		Warnings: []string{},
		TraceID:  "recommendation-record-local",
		Metrics:  noAPIMetrics(),
	}
}

func recommendationItemMap(item storage.RecommendationRecord) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"region":      item.Region,
		"type":        item.Type,
		"source":      item.Source,
		"explanation": item.Explanation,
		"evidence":    item.Evidence,
		"status":      item.Status,
		"updatedAt":   item.UpdatedAt,
	}
	if item.TargetIntent != "" {
		result["targetIntent"] = item.TargetIntent
	}
	if item.ScopeType != "" {
		result["scopeType"] = item.ScopeType
	}
	if item.ScopeRef != "" {
		result["scopeRef"] = item.ScopeRef
	}
	if item.Priority != 0 {
		result["priority"] = item.Priority
	}
	if item.Confidence != "" {
		result["confidence"] = item.Confidence
	}
	if item.ActionHint != nil {
		result["actionHint"] = item.ActionHint
	}
	if item.ParametersHint != nil {
		result["parametersHint"] = item.ParametersHint
	}
	return result
}

func memoryResponseNamespace(profile string, region string, houseID string) map[string]any {
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		region = "default"
	}
	return map[string]any{
		"accountProfile": profile,
		"profile":        profile,
		"region":         region,
		"houseId":        houseID,
		"dataType":       "memory",
	}
}

func recommendationFeedbackResponse(request contract.Request, houseID string, recommendation storage.RecommendationRecord) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已更新本地推荐反馈。",
		Recommendation: map[string]any{
			"houseId":          houseID,
			"id":               recommendation.ID,
			"status":           recommendation.Status,
			"cooldownUntil":    recommendation.CooldownUntil,
			"updatedAt":        recommendation.UpdatedAt,
			"feedbackRecorded": true,
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
			"id": recommendationID,
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
			"reason": reason,
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
		"apiCalls":  0,
		"cacheHits": 0,
	}
}
