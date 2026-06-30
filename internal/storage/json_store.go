package storage

import (
	"errors"
	"strings"
)

func (store JSONStore) SavePreference(record PreferenceRecord) error {
	if containsSensitiveKey(record.PreferenceType) {
		return errors.New("preference type must not contain token-like data")
	}
	if containsSensitiveKey(record.PreferenceValue) || containsSensitiveKey(record.Evidence) {
		return errors.New("preference value and evidence must not contain token-like data")
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("preference id, profile and houseId are required")
	}
	record.Region = normalizeStorageRegion(record.Region)
	record.Kind = normalizeMemoryKind(record.Kind)
	record.Status = normalizeMemoryStatus(record.Status)
	document, err := store.loadScope(record.Profile, record.Region, record.HouseID)
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range document.Preferences {
		if existing.ID == record.ID {
			document.Preferences[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		document.Preferences = append(document.Preferences, record)
	}
	return store.saveScope(record.Profile, record.Region, record.HouseID, document)
}

func (store JSONStore) UpsertPreference(record PreferenceRecord) (PreferenceUpsertResult, error) {
	if containsSensitiveKey(record.PreferenceType) {
		return PreferenceUpsertResult{}, errors.New("preference type must not contain token-like data")
	}
	if containsSensitiveKey(record.PreferenceValue) || containsSensitiveKey(record.Evidence) {
		return PreferenceUpsertResult{}, errors.New("preference value and evidence must not contain token-like data")
	}
	record.Profile = strings.TrimSpace(record.Profile)
	record.Region = normalizeStorageRegion(record.Region)
	record.HouseID = strings.TrimSpace(record.HouseID)
	record.ScopeType = normalizeMemoryText(record.ScopeType)
	record.ScopeRef = normalizeMemoryText(record.ScopeRef)
	record.PreferenceType = normalizeMemoryText(record.PreferenceType)
	record.PreferenceValue = normalizePreferenceValue(record.PreferenceValue)
	record.Kind = normalizeMemoryKind(record.Kind)
	record.Status = normalizeMemoryStatus(record.Status)
	if record.Profile == "" || record.HouseID == "" {
		return PreferenceUpsertResult{}, errors.New("profile and houseId are required")
	}
	if record.PreferenceType == "" || record.PreferenceValue == "" {
		return PreferenceUpsertResult{}, errors.New("preference type and value are required")
	}
	if record.UpdatedAt <= 0 {
		record.UpdatedAt = record.CreatedAt
	}
	if record.CreatedAt <= 0 {
		record.CreatedAt = record.UpdatedAt
	}
	if record.CreatedAt <= 0 {
		return PreferenceUpsertResult{}, errors.New("preference timestamp is required")
	}
	document, err := store.loadScope(record.Profile, record.Region, record.HouseID)
	if err != nil {
		return PreferenceUpsertResult{}, err
	}
	for index, existing := range document.Preferences {
		if !preferenceEquivalent(existing, record) {
			continue
		}
		if strings.TrimSpace(record.ID) == "" {
			record.ID = existing.ID
		}
		if existing.CreatedAt > 0 && existing.CreatedAt < record.CreatedAt {
			record.CreatedAt = existing.CreatedAt
		}
		record.Evidence = mergeEvidence(existing.Evidence, record.Evidence)
		document.Preferences[index] = record
		if err := store.saveScope(record.Profile, record.Region, record.HouseID, document); err != nil {
			return PreferenceUpsertResult{}, err
		}
		return PreferenceUpsertResult{Record: record, Created: false, Merged: true}, nil
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = preferenceStableID(record)
	}
	document.Preferences = append(document.Preferences, record)
	if err := store.saveScope(record.Profile, record.Region, record.HouseID, document); err != nil {
		return PreferenceUpsertResult{}, err
	}
	return PreferenceUpsertResult{Record: record, Created: true, Merged: false}, nil
}

func (store JSONStore) SetConsent(record ConsentRecord) error {
	if strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("profile and houseId are required")
	}
	if strings.TrimSpace(record.ConsentVersion) == "" {
		return errors.New("consentVersion is required")
	}
	record.Region = normalizeStorageRegion(record.Region)
	document, err := store.loadScope(record.Profile, record.Region, record.HouseID)
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range document.Consents {
		if existing.Profile == record.Profile && sameStorageRegion(existing.Region, record.Region) && existing.HouseID == record.HouseID {
			document.Consents[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		document.Consents = append(document.Consents, record)
	}
	return store.saveScope(record.Profile, record.Region, record.HouseID, document)
}

func (store JSONStore) Consent(profile string, region string, houseID string) (ConsentRecord, bool, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return ConsentRecord{}, false, err
	}
	for _, record := range document.Consents {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			return record, true, nil
		}
	}
	return ConsentRecord{}, false, nil
}

func (store JSONStore) ListPreferences(profile string, region string, houseID string) ([]PreferenceRecord, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return nil, err
	}
	result := []PreferenceRecord{}
	for _, record := range document.Preferences {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (store JSONStore) DeleteProfileHouse(profile string, region string, houseID string) error {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(houseID) == "" {
		return errors.New("profile and houseId are required")
	}
	return store.saveScope(profile, normalizeStorageRegion(region), houseID, emptyDocument())
}

func (store JSONStore) Export(profile string, region string, houseID string) (map[string]any, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return nil, err
	}
	preferences := []PreferenceRecord{}
	for _, record := range document.Preferences {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			preferences = append(preferences, record)
		}
	}
	recommendations := []RecommendationRecord{}
	for _, record := range document.Recommendations {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			recommendations = append(recommendations, record)
		}
	}
	signals := []InteractionSignalRecord{}
	for _, record := range document.Signals {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			signals = append(signals, record)
		}
	}
	lessons := []OperationLessonRecord{}
	for _, record := range document.Lessons {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && lessonScopeMatches(record.HouseID, houseID) {
			lessons = append(lessons, record)
		}
	}
	consents := []ConsentRecord{}
	for _, record := range document.Consents {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			consents = append(consents, record)
		}
	}
	return map[string]any{
		"format":          MemoryExportFormatVersion,
		"version":         document.Version,
		"namespace":       storageNamespace(profile, region, houseID, "memory"),
		"profile":         profile,
		"region":          region,
		"houseId":         houseID,
		"encryption":      "not_encrypted_local_runtime_export",
		"importPolicy":    "merge_by_id_replace_existing",
		"retentionPolicy": RetentionPolicy(),
		"consents":        consents,
		"preferences":     preferences,
		"recommendations": recommendations,
		"signals":         signals,
		"lessons":         lessons,
	}, nil
}

func RetentionPolicy() map[string]any {
	return map[string]any{
		"explicitPreferences":              ExplicitPreferenceRetention,
		"recommendationEvidenceDays":       DefaultRecommendationRetentionDays,
		"recommendationCompactionScope":    "accepted_dismissed_rejected_only",
		"pendingRecommendations":           "until_feedback_or_replaced",
		"interactionEventsDays":            DefaultInteractionRetentionDays,
		"interactionEvidence":              "intent_and_status_only",
		"operationLessons":                 "until_caller_marks_stale_deprecated_or_rejected",
		"runtimeSubjectiveInferencePolicy": "disabled",
	}
}
