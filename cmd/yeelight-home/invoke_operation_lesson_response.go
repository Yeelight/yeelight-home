package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func operationLessonRecordResponse(request contract.Request, upsert storage.OperationLessonUpsertResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已保存 1 条能力实操经验。",
		Result: map[string]any{
			semantic.FieldOperationLesson: operationLessonItemMap(upsert.Record),
			semantic.FieldCreated:         upsert.Created,
			semantic.FieldMerged:          upsert.Merged,
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-record-local",
		Metrics:  noAPIMetrics(),
	}
}

func operationLessonRecordPreviewResponse(request contract.Request, record storage.OperationLessonRecord) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已生成能力实操经验预览；dry-run 不会写入本地经验库。",
		Result: map[string]any{
			semantic.FieldDryRun:          true,
			semantic.FieldOperationLesson: operationLessonItemMap(record),
			semantic.FieldPreview: map[string]any{
				semantic.FieldIntent:         "operation.lesson.record",
				semantic.FieldPayloadPreview: operationLessonItemMap(record),
			},
		},
		Warnings: []string{"dry_run_no_local_write"},
		TraceID:  "operation-lesson-record-preview",
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
			semantic.FieldHouseID:          houseID,
			semantic.FieldOperationLessons: items,
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-list-local",
		Metrics:  noAPIMetrics(),
	}
}

func operationLessonItemMap(lesson storage.OperationLessonRecord) map[string]any {
	return map[string]any{
		semantic.FieldID:              lesson.ID,
		semantic.FieldProfile:         lesson.Profile,
		semantic.FieldHouseID:         lesson.HouseID,
		semantic.FieldIntent:          lesson.Intent,
		semantic.FieldLessonType:      lesson.LessonType,
		semantic.FieldSymptom:         lesson.Symptom,
		semantic.FieldCause:           lesson.Cause,
		semantic.FieldRecommendedPath: lesson.RecommendedPath,
		semantic.FieldAvoid:           lesson.Avoid,
		semantic.FieldParametersHint:  lesson.ParametersHint,
		semantic.FieldFallbackIntent:  lesson.FallbackIntent,
		semantic.FieldEvidence:        lesson.Evidence,
		semantic.FieldSource:          lesson.Source,
		semantic.FieldConfidence:      lesson.Confidence,
		semantic.FieldStatus:          lesson.Status,
		semantic.FieldStale:           lesson.Stale,
		semantic.FieldHitCount:        lesson.HitCount,
		semantic.FieldLastValidatedAt: lesson.LastValidatedAt,
		semantic.FieldCreatedAt:       lesson.CreatedAt,
		semantic.FieldUpdatedAt:       lesson.UpdatedAt,
	}
}

func operationLessonClarificationResponse(request contract.Request, reason string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充能力实操经验的结构化字段。",
		Clarification: map[string]any{
			semantic.FieldReason:         reason,
			semantic.FieldRequiredFields: []any{"lesson.intent", "lesson.lessonType", "lesson.symptom", "lesson.recommendedPath"},
			semantic.FieldPayloadShape: map[string]any{
				semantic.FieldIntent:          "scene.update",
				semantic.FieldLessonType:      "parameter_shape",
				semantic.FieldSymptom:         "invalid_scene_update_payload",
				semantic.FieldRecommendedPath: "先 scene.detail.get 取得 editablePayload，再 read-modify-send 完整 actions[]",
				semantic.FieldCause:           "仅 acceptedFields 不足以推断 actions[]、set、trigger 或 conditions 的完整结构",
				semantic.FieldAvoid:           "不要凭空拼大段动作或条件 JSON",
				semantic.FieldFallbackIntent:  "scene.create",
				semantic.FieldEvidence:        "脱敏错误码或简短调用结果",
				semantic.FieldSource:          "ai_skill",
				semantic.FieldConfidence:      "high",
				semantic.FieldStatus:          "confirmed",
				semantic.FieldStale:           false,
				semantic.FieldLastValidatedAt: 1780000000,
			},
		},
		Warnings: []string{},
		TraceID:  "operation-lesson-clarification",
		Metrics:  noAPIMetrics(),
	}
}
