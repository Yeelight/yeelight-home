package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func (app *app) invokeOperationLessonRecord(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	} else {
		houseID = ""
	}
	lessonPayload := requestMap(request.Parameters["lesson"])
	if lessonPayload == nil {
		lessonPayload = request.Parameters
	}
	intent := firstNonEmptyString(
		lessonString(lessonPayload, "intent"),
		lessonString(lessonPayload, "operationIntent"),
		lessonString(lessonPayload, "targetIntent"),
	)
	if intent == "" || intent == request.Intent {
		intent = firstNonEmptyString(
			lessonString(request.Parameters, "targetIntent"),
			lessonString(request.Parameters, "operationIntent"),
		)
	}
	now := time.Now().Unix()
	record := storage.OperationLessonRecord{
		Profile:         profile,
		Region:          region,
		HouseID:         houseID,
		Intent:          intent,
		LessonType:      lessonString(lessonPayload, "lessonType"),
		Symptom:         lessonString(lessonPayload, "symptom"),
		Cause:           lessonString(lessonPayload, "cause"),
		RecommendedPath: lessonString(lessonPayload, "recommendedPath"),
		Avoid:           lessonString(lessonPayload, "avoid"),
		ParametersHint:  lessonString(lessonPayload, "parametersHint"),
		FallbackIntent:  lessonString(lessonPayload, "fallbackIntent"),
		Evidence:        lessonString(lessonPayload, "evidence"),
		Source:          lessonString(lessonPayload, "source"),
		Confidence:      lessonString(lessonPayload, "confidence"),
		Status:          lessonString(lessonPayload, "status"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if stale, ok := lessonBool(lessonPayload, "stale"); ok {
		record.Stale = stale
	}
	if lastValidatedAt, ok := requestInteger(lessonPayload["lastValidatedAt"]); ok {
		record.LastValidatedAt = int64(lastValidatedAt)
	}
	if record.Intent == "" || record.LessonType == "" || record.Symptom == "" || record.RecommendedPath == "" {
		return operationLessonClarificationResponse(request, "missing_required_lesson_fields"), nil
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
	if parsed, ok := requestInt(request.Parameters["limit"]); ok {
		limit = parsed
	}
	filter := storage.OperationLessonFilter{
		Profile:         profile,
		Region:          region,
		HouseID:         houseID,
		Intent:          firstRequestString(request.Parameters, "intent", "operationIntent", "targetIntent"),
		LessonType:      firstRequestString(request.Parameters, "lessonType", "type"),
		Query:           firstRequestString(request.Parameters, "query", "keyword", "symptom"),
		Status:          firstRequestString(request.Parameters, "status"),
		Source:          firstRequestString(request.Parameters, "source"),
		MinConfidence:   firstRequestString(request.Parameters, "minConfidence", "confidenceAtLeast"),
		IncludeStale:    requestBool(request.Parameters, "includeStale", "include_stale"),
		IncludeRejected: requestBool(request.Parameters, "includeRejected", "include_rejected"),
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
