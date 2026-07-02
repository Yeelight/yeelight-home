package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

func (store JSONStore) Recommendation(profile string, region string, houseID string, recommendationID string) (RecommendationRecord, bool, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return RecommendationRecord{}, false, err
	}
	for _, record := range document.Recommendations {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID && record.ID == recommendationID {
			return record, true, nil
		}
	}
	return RecommendationRecord{}, false, nil
}

func (store JSONStore) SaveRecommendation(record RecommendationRecord) error {
	if recommendationContainsSensitiveData(record) {
		return errors.New("recommendation must not contain token-like data")
	}
	record = normalizeRecommendationRecord(record)
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("recommendation id, profile and houseId are required")
	}
	return store.mutateScope(record.Profile, record.Region, record.HouseID, func(document *jsonDocument) error {
		replaced := false
		for index, existing := range document.Recommendations {
			if existing.ID == record.ID {
				document.Recommendations[index] = record
				replaced = true
				break
			}
		}
		if !replaced {
			document.Recommendations = append(document.Recommendations, record)
		}
		return nil
	})
}

func (store JSONStore) UpsertRecommendation(record RecommendationRecord) (RecommendationUpsertResult, error) {
	if recommendationContainsSensitiveData(record) {
		return RecommendationUpsertResult{}, errors.New("recommendation must not contain token-like data")
	}
	rawStatus := strings.TrimSpace(record.Status)
	record = normalizeRecommendationRecord(record)
	if record.Profile == "" || record.HouseID == "" {
		return RecommendationUpsertResult{}, errors.New("profile and houseId are required")
	}
	if record.Type == "" || record.Explanation == "" || record.Evidence == "" {
		return RecommendationUpsertResult{}, errors.New("recommendation type, explanation and evidence are required")
	}
	if record.UpdatedAt <= 0 {
		record.UpdatedAt = record.CreatedAt
	}
	if record.CreatedAt <= 0 {
		record.CreatedAt = record.UpdatedAt
	}
	if record.CreatedAt <= 0 {
		return RecommendationUpsertResult{}, errors.New("recommendation timestamp is required")
	}
	var result RecommendationUpsertResult
	err := store.mutateScope(record.Profile, record.Region, record.HouseID, func(document *jsonDocument) error {
		for index, existing := range document.Recommendations {
			if !recommendationEquivalent(existing, record) {
				continue
			}
			if strings.TrimSpace(record.ID) == "" {
				record.ID = existing.ID
			}
			if existing.CreatedAt > 0 && existing.CreatedAt < record.CreatedAt {
				record.CreatedAt = existing.CreatedAt
			}
			record.Evidence = mergeEvidence(existing.Evidence, record.Evidence)
			if record.Source == "" {
				record.Source = existing.Source
			}
			if record.Confidence == "" {
				record.Confidence = existing.Confidence
			}
			if record.Priority == 0 {
				record.Priority = existing.Priority
			}
			if record.ActionHint == nil {
				record.ActionHint = existing.ActionHint
			}
			if record.ParametersHint == nil {
				record.ParametersHint = existing.ParametersHint
			}
			if rawStatus == "" || recommendationFeedbackStatus(existing.Status) {
				record.Status = existing.Status
				record.CooldownUntil = existing.CooldownUntil
			}
			if existing.CooldownUntil > record.CooldownUntil {
				record.CooldownUntil = existing.CooldownUntil
			}
			document.Recommendations[index] = record
			result = RecommendationUpsertResult{Record: record, Created: false, Merged: true}
			return nil
		}
		if strings.TrimSpace(record.ID) == "" {
			record.ID = recommendationStableID(record)
		}
		document.Recommendations = append(document.Recommendations, record)
		result = RecommendationUpsertResult{Record: record, Created: true}
		return nil
	})
	if err != nil {
		return RecommendationUpsertResult{}, err
	}
	return result, nil
}

func (store JSONStore) ListRecommendations(profile string, region string, houseID string, now int64, limit int) ([]RecommendationRecord, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1
	}
	result := []RecommendationRecord{}
	for _, record := range document.Recommendations {
		if record.Profile != profile || !sameStorageRegion(record.Region, region) || record.HouseID != houseID || record.Status != "pending" {
			continue
		}
		if record.CooldownUntil > now {
			continue
		}
		result = append(result, record)
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Priority != result[right].Priority {
			return result[left].Priority > result[right].Priority
		}
		leftRank := recommendationConfidenceRank(result[left].Confidence)
		rightRank := recommendationConfidenceRank(result[right].Confidence)
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		if result[left].UpdatedAt != result[right].UpdatedAt {
			return result[left].UpdatedAt > result[right].UpdatedAt
		}
		return result[left].ID < result[right].ID
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (store JSONStore) ApplyRecommendationFeedback(profile string, region string, houseID string, recommendationID string, feedback RecommendationFeedback) (RecommendationRecord, error) {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(houseID) == "" || strings.TrimSpace(recommendationID) == "" {
		return RecommendationRecord{}, errors.New("profile, houseId and recommendation id are required")
	}
	region = normalizeStorageRegion(region)
	if containsSensitiveKey(feedback.Status) {
		return RecommendationRecord{}, errors.New("recommendation feedback must not contain token-like data")
	}
	var updated RecommendationRecord
	err := store.mutateScope(profile, region, houseID, func(document *jsonDocument) error {
		for index, record := range document.Recommendations {
			if record.Profile != profile || !sameStorageRegion(record.Region, region) || record.HouseID != houseID || record.ID != recommendationID {
				continue
			}
			if strings.TrimSpace(feedback.Status) != "" {
				record.Status = strings.TrimSpace(feedback.Status)
			}
			record.CooldownUntil = feedback.CooldownUntil
			if feedback.UpdatedAt > 0 {
				record.UpdatedAt = feedback.UpdatedAt
			}
			document.Recommendations[index] = record
			updated = record
			return nil
		}
		return errors.New("recommendation not found")
	})
	if err != nil {
		return RecommendationRecord{}, err
	}
	return updated, nil
}

func normalizeRecommendationRecord(record RecommendationRecord) RecommendationRecord {
	record.Profile = strings.TrimSpace(record.Profile)
	record.Region = normalizeStorageRegion(record.Region)
	record.HouseID = strings.TrimSpace(record.HouseID)
	record.Type = normalizeMemoryText(record.Type)
	record.Source = normalizeRecommendationSource(record.Source)
	record.TargetIntent = normalizeOperationIntent(record.TargetIntent)
	record.ScopeType = normalizeMemoryText(record.ScopeType)
	record.ScopeRef = strings.TrimSpace(record.ScopeRef)
	record.Confidence = normalizeMemoryText(record.Confidence)
	record.Explanation = strings.TrimSpace(record.Explanation)
	record.Evidence = strings.TrimSpace(record.Evidence)
	record.Status = normalizeRecommendationStatus(record.Status)
	return record
}

func normalizeRecommendationSource(value string) string {
	value = normalizeMemoryText(value)
	if value == "" {
		return "ai_skill"
	}
	return value
}

func normalizeRecommendationStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "pending", "accepted", "dismissed", "rejected":
		return strings.TrimSpace(value)
	default:
		return "pending"
	}
}

func recommendationFeedbackStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "accepted", "dismissed", "rejected":
		return true
	default:
		return false
	}
}

func recommendationConfidenceRank(value string) int {
	switch normalizeMemoryText(value) {
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

func recommendationEquivalent(left RecommendationRecord, right RecommendationRecord) bool {
	return strings.TrimSpace(left.Profile) == strings.TrimSpace(right.Profile) &&
		sameStorageRegion(left.Region, right.Region) &&
		strings.TrimSpace(left.HouseID) == strings.TrimSpace(right.HouseID) &&
		normalizeMemoryText(left.Type) == normalizeMemoryText(right.Type) &&
		normalizeOperationIntent(left.TargetIntent) == normalizeOperationIntent(right.TargetIntent) &&
		normalizeMemoryText(left.ScopeType) == normalizeMemoryText(right.ScopeType) &&
		normalizeMemoryText(left.ScopeRef) == normalizeMemoryText(right.ScopeRef) &&
		normalizeMemoryText(left.Explanation) == normalizeMemoryText(right.Explanation)
}

func recommendationStableID(record RecommendationRecord) string {
	key := strings.Join([]string{
		record.Profile,
		normalizeStorageRegion(record.Region),
		record.HouseID,
		record.Type,
		record.TargetIntent,
		record.ScopeType,
		normalizeMemoryText(record.ScopeRef),
		normalizeMemoryText(record.Explanation),
	}, "|")
	sum := sha1.Sum([]byte(key))
	return "rec-" + hex.EncodeToString(sum[:])[:16]
}

func recommendationContainsSensitiveData(record RecommendationRecord) bool {
	for _, value := range []string{
		record.ID,
		record.Type,
		record.Source,
		record.TargetIntent,
		record.ScopeType,
		record.ScopeRef,
		record.Confidence,
		record.Explanation,
		record.Evidence,
		record.Status,
	} {
		if containsSensitiveKey(value) {
			return true
		}
	}
	return containsSensitiveValue(record.ActionHint) || containsSensitiveValue(record.ParametersHint)
}

func containsSensitiveValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return containsSensitiveKey(typed)
	case map[string]any:
		for key, nested := range typed {
			if containsSensitiveKey(key) || containsSensitiveValue(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsSensitiveValue(nested) {
				return true
			}
		}
	default:
		return containsSensitiveKey(fmt.Sprint(typed))
	}
	return false
}
