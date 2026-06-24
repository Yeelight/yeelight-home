package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	MemoryExportFormatVersion             = "yeelight-memory-export-v1"
	DefaultInteractionRetentionDays       = 90
	DefaultImplicitCandidateRetentionDays = 30
	DefaultRecommendationRetentionDays    = 90
	ExplicitPreferenceRetention           = "until_user_forgets"
)

type PreferenceRecord struct {
	ID              string `json:"id"`
	Profile         string `json:"profile"`
	HouseID         string `json:"houseId"`
	ScopeType       string `json:"scopeType"`
	ScopeRef        string `json:"scopeRef"`
	PreferenceType  string `json:"preferenceType"`
	PreferenceValue string `json:"preferenceValue"`
	Kind            string `json:"kind,omitempty"`
	Status          string `json:"status,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
	CreatedAt       int64  `json:"createdAt,omitempty"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type ConsentRecord struct {
	Profile         string `json:"profile"`
	HouseID         string `json:"houseId"`
	ConsentVersion  string `json:"consentVersion"`
	LearningEnabled bool   `json:"learningEnabled"`
	Paused          bool   `json:"paused"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type RecommendationRecord struct {
	ID            string `json:"id"`
	Profile       string `json:"profile"`
	HouseID       string `json:"houseId"`
	Type          string `json:"type"`
	Explanation   string `json:"explanation"`
	Evidence      string `json:"evidence"`
	Status        string `json:"status"`
	CooldownUntil int64  `json:"cooldownUntil,omitempty"`
	LastShownAt   int64  `json:"lastShownAt,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type RecommendationFeedback struct {
	Status        string
	CooldownUntil int64
	UpdatedAt     int64
}

type JSONStore struct {
	path string
}

type jsonDocument struct {
	Version         int                    `json:"version"`
	Consents        []ConsentRecord        `json:"consents"`
	Preferences     []PreferenceRecord     `json:"preferences"`
	Recommendations []RecommendationRecord `json:"recommendations"`
}

func NewJSONStore(path string) JSONStore {
	return JSONStore{path: path}
}

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
	record.Kind = normalizeMemoryKind(record.Kind)
	record.Status = normalizeMemoryStatus(record.Status)
	document, err := store.load()
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
	return store.save(document)
}

func (store JSONStore) SetConsent(record ConsentRecord) error {
	if strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return errors.New("profile and houseId are required")
	}
	if strings.TrimSpace(record.ConsentVersion) == "" {
		return errors.New("consentVersion is required")
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range document.Consents {
		if existing.Profile == record.Profile && existing.HouseID == record.HouseID {
			document.Consents[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		document.Consents = append(document.Consents, record)
	}
	return store.save(document)
}

func (store JSONStore) Consent(profile string, houseID string) (ConsentRecord, bool, error) {
	document, err := store.load()
	if err != nil {
		return ConsentRecord{}, false, err
	}
	for _, record := range document.Consents {
		if record.Profile == profile && record.HouseID == houseID {
			return record, true, nil
		}
	}
	return ConsentRecord{}, false, nil
}

func (store JSONStore) ListPreferences(profile string, houseID string) ([]PreferenceRecord, error) {
	document, err := store.load()
	if err != nil {
		return nil, err
	}
	result := []PreferenceRecord{}
	for _, record := range document.Preferences {
		if record.Profile == profile && record.HouseID == houseID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (store JSONStore) DeleteProfileHouse(profile string, houseID string) error {
	document, err := store.load()
	if err != nil {
		return err
	}
	document.Consents = filterConsents(document.Consents, profile, houseID)
	document.Preferences = filterPreferences(document.Preferences, profile, houseID)
	document.Recommendations = filterRecommendations(document.Recommendations, profile, houseID)
	return store.save(document)
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

func (store JSONStore) Export(profile string, houseID string) (map[string]any, error) {
	document, err := store.load()
	if err != nil {
		return nil, err
	}
	preferences := []PreferenceRecord{}
	for _, record := range document.Preferences {
		if record.Profile == profile && record.HouseID == houseID {
			preferences = append(preferences, record)
		}
	}
	recommendations := []RecommendationRecord{}
	for _, record := range document.Recommendations {
		if record.Profile == profile && record.HouseID == houseID {
			recommendations = append(recommendations, record)
		}
	}
	consents := []ConsentRecord{}
	for _, record := range document.Consents {
		if record.Profile == profile && record.HouseID == houseID {
			consents = append(consents, record)
		}
	}
	return map[string]any{
		"format":          MemoryExportFormatVersion,
		"version":         document.Version,
		"profile":         profile,
		"houseId":         houseID,
		"encryption":      "not_encrypted_local_runtime_export",
		"importPolicy":    "merge_by_id_replace_existing",
		"retentionPolicy": RetentionPolicy(),
		"consents":        consents,
		"preferences":     preferences,
		"recommendations": recommendations,
	}, nil
}

func RetentionPolicy() map[string]any {
	return map[string]any{
		"explicitPreferences":        ExplicitPreferenceRetention,
		"implicitCandidatesDays":     DefaultImplicitCandidateRetentionDays,
		"recommendationEvidenceDays": DefaultRecommendationRetentionDays,
		"interactionEventsDays":      DefaultInteractionRetentionDays,
	}
}

func (store JSONStore) load() (jsonDocument, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyDocument(), nil
	}
	if err != nil {
		return jsonDocument{}, err
	}
	document := jsonDocument{}
	if err := json.Unmarshal(data, &document); err != nil {
		return jsonDocument{}, err
	}
	if document.Preferences == nil {
		document.Preferences = []PreferenceRecord{}
	}
	if document.Consents == nil {
		document.Consents = []ConsentRecord{}
	}
	if document.Recommendations == nil {
		document.Recommendations = []RecommendationRecord{}
	}
	return document, nil
}

func (store JSONStore) save(document jsonDocument) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	tempPath := store.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, store.path)
}

func emptyDocument() jsonDocument {
	return jsonDocument{
		Version:         1,
		Consents:        []ConsentRecord{},
		Preferences:     []PreferenceRecord{},
		Recommendations: []RecommendationRecord{},
	}
}

func containsSensitiveKey(value string) bool {
	normalized := strings.ToLower(value)
	for _, forbidden := range []string{"token", "secret", "authorization", "cookie"} {
		if strings.Contains(normalized, forbidden) {
			return true
		}
	}
	return false
}

func normalizeMemoryKind(value string) string {
	switch strings.TrimSpace(value) {
	case "implicit_candidate":
		return "implicit_candidate"
	default:
		return "explicit"
	}
}

func normalizeMemoryStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "candidate", "rejected", "confirmed":
		return strings.TrimSpace(value)
	default:
		return "confirmed"
	}
}

func filterConsents(records []ConsentRecord, profile string, houseID string) []ConsentRecord {
	result := []ConsentRecord{}
	for _, record := range records {
		if record.Profile == profile && record.HouseID == houseID {
			continue
		}
		result = append(result, record)
	}
	return result
}

func filterPreferences(records []PreferenceRecord, profile string, houseID string) []PreferenceRecord {
	result := []PreferenceRecord{}
	for _, record := range records {
		if record.Profile == profile && record.HouseID == houseID {
			continue
		}
		result = append(result, record)
	}
	return result
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
