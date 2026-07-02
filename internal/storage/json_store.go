package storage

import (
	"errors"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
	return store.mutateScope(record.Profile, record.Region, record.HouseID, func(document *jsonDocument) error {
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
		return nil
	})
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
	var result PreferenceUpsertResult
	err := store.mutateScope(record.Profile, record.Region, record.HouseID, func(document *jsonDocument) error {
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
			result = PreferenceUpsertResult{Record: record, Created: false, Merged: true}
			return nil
		}
		if strings.TrimSpace(record.ID) == "" {
			record.ID = preferenceStableID(record)
		}
		document.Preferences = append(document.Preferences, record)
		result = PreferenceUpsertResult{Record: record, Created: true, Merged: false}
		return nil
	})
	if err != nil {
		return PreferenceUpsertResult{}, err
	}
	return result, nil
}

func (store JSONStore) SetConsent(record ConsentRecord) error {
	if strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("profile and houseId are required")
	}
	if strings.TrimSpace(record.ConsentVersion) == "" {
		return errors.New("consentVersion is required")
	}
	record.Region = normalizeStorageRegion(record.Region)
	return store.mutateScope(record.Profile, record.Region, record.HouseID, func(document *jsonDocument) error {
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
		return nil
	})
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
		semantic.FieldFormat:          MemoryExportFormatVersion,
		semantic.FieldVersion:         document.Version,
		semantic.FieldNamespace:       storageNamespace(profile, region, houseID, "memory"),
		semantic.FieldProfile:         profile,
		semantic.FieldRegion:          region,
		semantic.FieldHouseID:         houseID,
		semantic.FieldEncryption:      "not_encrypted_local_runtime_export",
		semantic.FieldImportPolicy:    "merge_by_id_replace_existing",
		semantic.FieldRetentionPolicy: RetentionPolicy(),
		semantic.FieldConsents:        consents,
		semantic.FieldPreferences:     preferences,
		semantic.FieldRecommendations: recommendations,
		semantic.FieldSignals:         signals,
		semantic.FieldLessons:         lessons,
	}, nil
}

func RetentionPolicy() map[string]any {
	return map[string]any{
		semantic.FieldExplicitPreferences:              ExplicitPreferenceRetention,
		semantic.FieldRecommendationEvidenceDays:       DefaultRecommendationRetentionDays,
		semantic.FieldRecommendationCompactionScope:    "accepted_dismissed_rejected_only",
		semantic.FieldPendingRecommendations:           "until_feedback_or_replaced",
		semantic.FieldInteractionEventsDays:            DefaultInteractionRetentionDays,
		semantic.FieldInteractionEvidence:              "intent_and_status_only",
		semantic.FieldOperationLessonsRetention:        "until_caller_marks_stale_deprecated_or_rejected",
		semantic.FieldRuntimeSubjectiveInferencePolicy: "disabled",
	}
}
