package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func memoryRememberResponse(request contract.Request, upsert storage.PreferenceUpsertResult) contract.Response {
	memory := upsert.Record
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已保存本地偏好。",
		Memory: map[string]any{
			"id":              memory.ID,
			"profile":         memory.Profile,
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
		},
		Warnings: []string{},
		TraceID:  "memory-remember-local",
		Metrics:  noAPIMetrics(),
	}
}

func memoryListResponse(request contract.Request, houseID string, consent storage.ConsentRecord, preferences []storage.PreferenceRecord) contract.Response {
	items := make([]any, 0, len(preferences))
	for _, item := range preferences {
		items = append(items, map[string]any{
			"id":              item.ID,
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

func recommendationListResponse(request contract.Request, houseID string, recommendations []storage.RecommendationRecord) contract.Response {
	if len(recommendations) == 0 {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "success",
			UserMessage:     "当前没有新的本地推荐。",
			Recommendation: map[string]any{
				"houseId": houseID,
				"items":   []any{},
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
			"houseId": houseID,
			"items": []any{map[string]any{
				"id":          item.ID,
				"type":        item.Type,
				"explanation": item.Explanation,
				"evidence":    item.Evidence,
				"status":      item.Status,
			}},
			"sessionLimit": 1,
		},
		Warnings: []string{},
		TraceID:  "recommendation-list-local",
		Metrics:  noAPIMetrics(),
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
