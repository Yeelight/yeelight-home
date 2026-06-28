package storage

import (
	"errors"
	"strings"
)

func (store JSONStore) Recommendation(profile string, houseID string, recommendationID string) (RecommendationRecord, bool, error) {
	document, err := store.load()
	if err != nil {
		return RecommendationRecord{}, false, err
	}
	for _, record := range document.Recommendations {
		if record.Profile == profile && record.HouseID == houseID && record.ID == recommendationID {
			return record, true, nil
		}
	}
	return RecommendationRecord{}, false, nil
}

func (store JSONStore) SaveRecommendation(record RecommendationRecord) error {
	if containsSensitiveKey(record.Type) || containsSensitiveKey(record.Explanation) || containsSensitiveKey(record.Evidence) {
		return errors.New("recommendation must not contain token-like data")
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("recommendation id, profile and houseId are required")
	}
	if strings.TrimSpace(record.Status) == "" {
		record.Status = "pending"
	}
	document, err := store.load()
	if err != nil {
		return err
	}
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
	return store.save(document)
}

func (store JSONStore) ListRecommendations(profile string, houseID string, now int64, limit int) ([]RecommendationRecord, error) {
	document, err := store.load()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1
	}
	result := []RecommendationRecord{}
	for _, record := range document.Recommendations {
		if record.Profile != profile || record.HouseID != houseID || record.Status != "pending" {
			continue
		}
		if record.CooldownUntil > now {
			continue
		}
		result = append(result, record)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (store JSONStore) ApplyRecommendationFeedback(profile string, houseID string, recommendationID string, feedback RecommendationFeedback) (RecommendationRecord, error) {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(houseID) == "" || strings.TrimSpace(recommendationID) == "" {
		return RecommendationRecord{}, errors.New("profile, houseId and recommendation id are required")
	}
	if containsSensitiveKey(feedback.Status) {
		return RecommendationRecord{}, errors.New("recommendation feedback must not contain token-like data")
	}
	document, err := store.load()
	if err != nil {
		return RecommendationRecord{}, err
	}
	for index, record := range document.Recommendations {
		if record.Profile != profile || record.HouseID != houseID || record.ID != recommendationID {
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
		if err := store.save(document); err != nil {
			return RecommendationRecord{}, err
		}
		return record, nil
	}
	return RecommendationRecord{}, errors.New("recommendation not found")
}

func filterRecommendations(records []RecommendationRecord, profile string, houseID string) []RecommendationRecord {
	result := []RecommendationRecord{}
	for _, record := range records {
		if record.Profile == profile && record.HouseID == houseID {
			continue
		}
		result = append(result, record)
	}
	return result
}
