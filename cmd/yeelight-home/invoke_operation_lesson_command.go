package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func (app *app) invokeOperationLessonRecord(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	} else {
		houseID = ""
	}
	lessonPayload := requestMap(request.Parameters[semantic.FieldLesson])
	if lessonPayload == nil {
		lessonPayload = request.Parameters
	}
	intent := firstNonEmptyString(
		lessonString(lessonPayload, semantic.FieldIntent),
		lessonString(lessonPayload, semantic.FieldOperationIntent),
		lessonString(lessonPayload, semantic.FieldTargetIntent),
	)
	if intent == "" || intent == request.Intent {
		intent = firstNonEmptyString(
			lessonString(request.Parameters, semantic.FieldTargetIntent),
			lessonString(request.Parameters, semantic.FieldOperationIntent),
		)
	}
	now := time.Now().Unix()
	record := storage.OperationLessonRecord{
		Profile:         profile,
		Region:          region,
		HouseID:         houseID,
		Intent:          intent,
		LessonType:      lessonString(lessonPayload, semantic.FieldLessonType),
		Symptom:         lessonString(lessonPayload, semantic.FieldSymptom),
		Cause:           lessonString(lessonPayload, semantic.FieldCause),
		RecommendedPath: lessonString(lessonPayload, semantic.FieldRecommendedPath),
		Avoid:           lessonString(lessonPayload, semantic.FieldAvoid),
		ParametersHint:  lessonString(lessonPayload, semantic.FieldParametersHint),
		FallbackIntent:  lessonString(lessonPayload, semantic.FieldFallbackIntent),
		Evidence:        lessonString(lessonPayload, semantic.FieldEvidence),
		Source:          lessonString(lessonPayload, semantic.FieldSource),
		Confidence:      lessonString(lessonPayload, semantic.FieldConfidence),
		Status:          lessonString(lessonPayload, semantic.FieldStatus),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if stale, ok := lessonBool(lessonPayload, semantic.FieldStale); ok {
		record.Stale = stale
	}
	if lastValidatedAt, ok := requestInteger(lessonPayload[semantic.FieldLastValidatedAt]); ok {
		record.LastValidatedAt = int64(lastValidatedAt)
	}
	if record.Intent == "" || record.LessonType == "" || record.Symptom == "" || record.RecommendedPath == "" {
		return operationLessonClarificationResponse(request, "missing_required_lesson_fields"), nil
	}
	if operationLessonRecordDryRunRequested(request) {
		return operationLessonRecordPreviewResponse(request, record), nil
	}
	upsert, err := app.memoryStore.UpsertOperationLesson(record)
	if err != nil {
		return contract.Response{}, err
	}
	return operationLessonRecordResponse(request, upsert), nil
}

func (app *app) invokeOperationLessonList(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	} else {
		houseID = ""
	}
	limit := 0
	if parsed, ok := requestInt(request.Parameters[semantic.FieldLimit]); ok {
		limit = parsed
	}
	filter := storage.OperationLessonFilter{
		Profile:         profile,
		Region:          region,
		HouseID:         houseID,
		Intent:          firstRequestString(request.Parameters, semantic.FieldIntent, semantic.FieldOperationIntent, semantic.FieldTargetIntent),
		LessonType:      firstRequestString(request.Parameters, semantic.FieldLessonType),
		Query:           firstRequestString(request.Parameters, semantic.FieldQuery, semantic.FieldKeyword, semantic.FieldSymptom),
		Status:          firstRequestString(request.Parameters, semantic.FieldStatus),
		Source:          firstRequestString(request.Parameters, semantic.FieldSource),
		MinConfidence:   firstRequestString(request.Parameters, semantic.FieldMinConfidence, semantic.FieldConfidenceAtLeast),
		IncludeStale:    requestBool(request.Parameters, semantic.FieldIncludeStale),
		IncludeRejected: requestBool(request.Parameters, semantic.FieldIncludeRejected),
		Limit:           limit,
	}
	lessons, err := app.memoryStore.ListOperationLessons(filter)
	if err != nil {
		return contract.Response{}, err
	}
	return operationLessonListResponse(request, houseID, lessons), nil
}

func lessonString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if text := requestString(value); text != "" {
			return text
		}
		switch value.(type) {
		case map[string]any, []any:
			data, err := json.Marshal(value)
			if err == nil {
				return string(data)
			}
		}
	}
	return ""
}

func lessonBool(values map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1", "yes", "y":
				return true, true
			case "false", "0", "no", "n":
				return false, true
			}
		}
	}
	return false, false
}

func isOperationLessonIntent(intent string) bool {
	return strings.HasPrefix(intent, "operation.lesson.")
}

func operationLessonRecordDryRunRequested(request contract.Request) bool {
	return requestBool(request.Options, semantic.FieldDryRun, semantic.FieldPreviewOnly) ||
		requestBool(request.Parameters, semantic.FieldDryRun, semantic.FieldPreviewOnly)
}
