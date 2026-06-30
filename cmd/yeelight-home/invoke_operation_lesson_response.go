package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func operationLessonRecordResponse(request contract.Request, upsert storage.OperationLessonUpsertResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已保存 1 条能力实操经验。",
		Result: map[string]any{
			"operationLesson": operationLessonItemMap(upsert.Record),
			"created":         upsert.Created,
			"merged":          upsert.Merged,
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-record-local",
		Metrics:  noAPIMetrics(),
	}
}

func operationLessonListResponse(request contract.Request, houseID string, lessons []storage.OperationLessonRecord) contract.Response {
	items := make([]any, 0, len(lessons))
	for _, lesson := range lessons {
		items = append(items, operationLessonItemMap(lesson))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已读取 %d 条能力实操经验。", len(items)),
		Result: map[string]any{
			"houseId":          houseID,
			"operationLessons": items,
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-list-local",
		Metrics:  noAPIMetrics(),
	}
}

func operationLessonItemMap(lesson storage.OperationLessonRecord) map[string]any {
	return map[string]any{
		"id":              lesson.ID,
		"profile":         lesson.Profile,
		"houseId":         lesson.HouseID,
		"intent":          lesson.Intent,
		"lessonType":      lesson.LessonType,
		"symptom":         lesson.Symptom,
		"cause":           lesson.Cause,
		"recommendedPath": lesson.RecommendedPath,
		"avoid":           lesson.Avoid,
		"parametersHint":  lesson.ParametersHint,
		"fallbackIntent":  lesson.FallbackIntent,
		"evidence":        lesson.Evidence,
		"source":          lesson.Source,
		"confidence":      lesson.Confidence,
		"status":          lesson.Status,
		"stale":           lesson.Stale,
		"hitCount":        lesson.HitCount,
		"lastValidatedAt": lesson.LastValidatedAt,
		"createdAt":       lesson.CreatedAt,
		"updatedAt":       lesson.UpdatedAt,
	}
}

func operationLessonClarificationResponse(request contract.Request, reason string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充能力实操经验的结构化字段。",
		Clarification: map[string]any{
			"reason":         reason,
			"requiredFields": []any{"lesson.intent", "lesson.lessonType", "lesson.symptom", "lesson.recommendedPath"},
			"payloadShape": map[string]any{
				"intent":          "scene.update",
				"lessonType":      "parameter_shape",
				"symptom":         "invalid_scene_update_payload",
				"recommendedPath": "先 scene.detail.get 取得 editablePayload，再 read-modify-send 完整 details",
				"cause":           "仅 acceptedFields 不足以推断 details/params 内部结构",
				"avoid":           "不要凭空拼大段 JSON",
				"fallbackIntent":  "scene.create",
				"evidence":        "脱敏错误码或简短调用结果",
				"source":          "ai_skill",
				"confidence":      "high",
				"status":          "confirmed",
				"stale":           false,
				"lastValidatedAt": 1780000000,
			},
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-clarification",
		Metrics:  noAPIMetrics(),
	}
}
