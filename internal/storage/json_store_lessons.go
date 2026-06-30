package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

const defaultOperationLessonLimit = 10

type OperationLessonFilter struct {
	Profile         string
	Region          string
	HouseID         string
	Intent          string
	LessonType      string
	Query           string
	Status          string
	Source          string
	MinConfidence   string
	IncludeStale    bool
	IncludeRejected bool
	Limit           int
}

func (store JSONStore) UpsertOperationLesson(record OperationLessonRecord) (OperationLessonUpsertResult, error) {
	if operationLessonContainsSensitiveData(record) {
		return OperationLessonUpsertResult{}, errors.New("operation lesson must not contain token-like data")
	}
	record = normalizeOperationLesson(record)
	if record.Profile == "" {
		return OperationLessonUpsertResult{}, errors.New("profile is required")
	}
	if record.Intent == "" || record.LessonType == "" {
		return OperationLessonUpsertResult{}, errors.New("lesson intent and lessonType are required")
	}
	if record.Symptom == "" || record.RecommendedPath == "" {
		return OperationLessonUpsertResult{}, errors.New("lesson symptom and recommendedPath are required")
	}
	if record.UpdatedAt <= 0 {
		record.UpdatedAt = record.CreatedAt
	}
	if record.CreatedAt <= 0 {
		record.CreatedAt = record.UpdatedAt
	}
	if record.CreatedAt <= 0 {
		return OperationLessonUpsertResult{}, errors.New("lesson timestamp is required")
	}
	if record.HitCount <= 0 {
		record.HitCount = 1
	}
	scopeHouseID := lessonStorageHouseID(record.HouseID)
	document, err := store.loadScope(record.Profile, record.Region, scopeHouseID)
	if err != nil {
		return OperationLessonUpsertResult{}, err
	}
	for index, existing := range document.Lessons {
		if !operationLessonEquivalent(existing, record) {
			continue
		}
		if strings.TrimSpace(record.ID) == "" {
			record.ID = existing.ID
		}
		if existing.CreatedAt > 0 && existing.CreatedAt < record.CreatedAt {
			record.CreatedAt = existing.CreatedAt
		}
		record.HitCount += existing.HitCount
		record.Evidence = mergeEvidence(existing.Evidence, record.Evidence)
		if record.Cause == "" {
			record.Cause = existing.Cause
		}
		if record.Avoid == "" {
			record.Avoid = existing.Avoid
		}
		if record.ParametersHint == "" {
			record.ParametersHint = existing.ParametersHint
		}
		if record.FallbackIntent == "" {
			record.FallbackIntent = existing.FallbackIntent
		}
		if record.Source == "" {
			record.Source = existing.Source
		}
		if record.Confidence == "" {
			record.Confidence = existing.Confidence
		}
		if record.LastValidatedAt <= 0 {
			record.LastValidatedAt = existing.LastValidatedAt
		}
		document.Lessons[index] = record
		if err := store.saveScope(record.Profile, record.Region, scopeHouseID, document); err != nil {
			return OperationLessonUpsertResult{}, err
		}
		return OperationLessonUpsertResult{Record: record, Created: false, Merged: true}, nil
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = operationLessonStableID(record)
	}
	document.Lessons = append(document.Lessons, record)
	if err := store.saveScope(record.Profile, record.Region, scopeHouseID, document); err != nil {
		return OperationLessonUpsertResult{}, err
	}
	return OperationLessonUpsertResult{Record: record, Created: true}, nil
}

func (store JSONStore) ListOperationLessons(filter OperationLessonFilter) ([]OperationLessonRecord, error) {
	filter.Profile = strings.TrimSpace(filter.Profile)
	if filter.Profile == "" {
		return nil, errors.New("profile is required")
	}
	filter.Region = normalizeStorageRegion(filter.Region)
	filter.HouseID = strings.TrimSpace(filter.HouseID)
	filter.Intent = normalizeOperationIntent(filter.Intent)
	filter.LessonType = normalizeMemoryText(filter.LessonType)
	filter.Query = normalizeMemoryText(filter.Query)
	filter.Status = normalizeOperationLessonStatusFilter(filter.Status)
	filter.Source = normalizeMemoryText(filter.Source)
	filter.MinConfidence = normalizeOperationLessonConfidence(filter.MinConfidence)
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultOperationLessonLimit
	}
	if limit > 50 {
		limit = 50
	}
	documents := []jsonDocument{}
	document, err := store.loadScope(filter.Profile, filter.Region, lessonStorageHouseID(filter.HouseID))
	if err != nil {
		return nil, err
	}
	documents = append(documents, document)
	if filter.HouseID != "" {
		globalDocument, err := store.loadScope(filter.Profile, filter.Region, lessonStorageHouseID(""))
		if err != nil {
			return nil, err
		}
		documents = append(documents, globalDocument)
	}
	result := []OperationLessonRecord{}
	for _, document := range documents {
		for _, record := range document.Lessons {
			if record.Profile != filter.Profile || !sameStorageRegion(record.Region, filter.Region) || !lessonScopeMatches(record.HouseID, filter.HouseID) {
				continue
			}
			if filter.Intent != "" && normalizeOperationIntent(record.Intent) != filter.Intent {
				continue
			}
			if filter.LessonType != "" && normalizeMemoryText(record.LessonType) != filter.LessonType {
				continue
			}
			if filter.Query != "" && !operationLessonMatchesQuery(record, filter.Query) {
				continue
			}
			if !filter.IncludeRejected && record.Status == "rejected" {
				continue
			}
			if !filter.IncludeStale && record.Stale {
				continue
			}
			if filter.Status != "" && normalizeOperationLessonStatus(record.Status) != filter.Status {
				continue
			}
			if filter.Source != "" && normalizeMemoryText(record.Source) != filter.Source {
				continue
			}
			if filter.MinConfidence != "" && operationLessonConfidenceRank(record.Confidence) < operationLessonConfidenceRank(filter.MinConfidence) {
				continue
			}
			result = append(result, record)
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].UpdatedAt == result[right].UpdatedAt {
			return result[left].ID < result[right].ID
		}
		return result[left].UpdatedAt > result[right].UpdatedAt
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func normalizeOperationLesson(record OperationLessonRecord) OperationLessonRecord {
	record.Profile = strings.TrimSpace(record.Profile)
	record.Region = normalizeStorageRegion(record.Region)
	record.HouseID = strings.TrimSpace(record.HouseID)
	record.Intent = normalizeOperationIntent(record.Intent)
	record.LessonType = normalizeMemoryText(record.LessonType)
	record.Symptom = strings.TrimSpace(record.Symptom)
	record.Cause = strings.TrimSpace(record.Cause)
	record.RecommendedPath = strings.TrimSpace(record.RecommendedPath)
	record.Avoid = strings.TrimSpace(record.Avoid)
	record.ParametersHint = strings.TrimSpace(record.ParametersHint)
	record.FallbackIntent = normalizeOperationIntent(record.FallbackIntent)
	record.Evidence = strings.TrimSpace(record.Evidence)
	record.Source = normalizeOperationLessonSource(record.Source)
	record.Confidence = normalizeOperationLessonConfidence(record.Confidence)
	record.Status = normalizeOperationLessonStatus(record.Status)
	if record.LastValidatedAt < 0 {
		record.LastValidatedAt = 0
	}
	return record
}

func normalizeOperationLessonStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "candidate", "confirmed", "rejected", "deprecated":
		return strings.TrimSpace(value)
	default:
		return "confirmed"
	}
}

func normalizeOperationLessonStatusFilter(value string) string {
	switch strings.TrimSpace(value) {
	case "candidate", "confirmed", "rejected", "deprecated":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeOperationLessonSource(value string) string {
	value = normalizeMemoryText(value)
	if value == "" {
		return "ai_skill"
	}
	return value
}

func normalizeOperationLessonConfidence(value string) string {
	switch normalizeMemoryText(value) {
	case "low", "medium", "high":
		return normalizeMemoryText(value)
	default:
		return ""
	}
}

func operationLessonConfidenceRank(value string) int {
	switch normalizeOperationLessonConfidence(value) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func operationLessonEquivalent(left OperationLessonRecord, right OperationLessonRecord) bool {
	return strings.TrimSpace(left.Profile) == strings.TrimSpace(right.Profile) &&
		sameStorageRegion(left.Region, right.Region) &&
		lessonStorageHouseID(left.HouseID) == lessonStorageHouseID(right.HouseID) &&
		normalizeOperationIntent(left.Intent) == normalizeOperationIntent(right.Intent) &&
		normalizeMemoryText(left.LessonType) == normalizeMemoryText(right.LessonType) &&
		normalizeMemoryText(left.Symptom) == normalizeMemoryText(right.Symptom) &&
		normalizeMemoryText(left.RecommendedPath) == normalizeMemoryText(right.RecommendedPath)
}

func operationLessonMatchesQuery(record OperationLessonRecord, query string) bool {
	haystack := normalizeMemoryText(strings.Join([]string{
		record.Intent,
		record.LessonType,
		record.Symptom,
		record.Cause,
		record.RecommendedPath,
		record.Avoid,
		record.ParametersHint,
		record.FallbackIntent,
		record.Evidence,
	}, " "))
	return strings.Contains(haystack, query)
}

func normalizeOperationIntent(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func operationLessonContainsSensitiveData(record OperationLessonRecord) bool {
	for _, value := range []string{
		record.Intent,
		record.LessonType,
		record.Symptom,
		record.Cause,
		record.RecommendedPath,
		record.Avoid,
		record.ParametersHint,
		record.FallbackIntent,
		record.Evidence,
		record.Source,
		record.Confidence,
		record.Status,
	} {
		if containsSensitiveKey(value) {
			return true
		}
	}
	return false
}

func lessonScopeMatches(recordHouseID string, queryHouseID string) bool {
	recordHouseID = strings.TrimSpace(recordHouseID)
	queryHouseID = strings.TrimSpace(queryHouseID)
	return recordHouseID == queryHouseID || recordHouseID == ""
}

func lessonStorageHouseID(houseID string) string {
	if strings.TrimSpace(houseID) == "" {
		return "global"
	}
	return strings.TrimSpace(houseID)
}

func operationLessonStableID(record OperationLessonRecord) string {
	key := strings.Join([]string{
		record.Profile,
		normalizeStorageRegion(record.Region),
		lessonStorageHouseID(record.HouseID),
		record.Intent,
		record.LessonType,
		normalizeMemoryText(record.Symptom),
		normalizeMemoryText(record.RecommendedPath),
	}, "|")
	return "les-" + shortStableHash(key)
}

func shortStableHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}
