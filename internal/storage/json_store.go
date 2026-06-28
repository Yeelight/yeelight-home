package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	MemoryExportFormatVersion             = "yeelight-memory-export-v1"
	DefaultInteractionRetentionDays       = 90
	DefaultImplicitCandidateRetentionDays = 30
	DefaultRecommendationRetentionDays    = 90
	ExplicitPreferenceRetention           = "until_user_forgets"
	maxMergedEvidenceRunes                = 240
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

type PreferenceUpsertResult struct {
	Record  PreferenceRecord
	Created bool
	Merged  bool
}

type jsonDocument struct {
	Version         int                       `json:"version"`
	Consents        []ConsentRecord           `json:"consents"`
	Preferences     []PreferenceRecord        `json:"preferences"`
	Recommendations []RecommendationRecord    `json:"recommendations"`
	Signals         []InteractionSignalRecord `json:"signals,omitempty"`
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

func (store JSONStore) UpsertPreference(record PreferenceRecord) (PreferenceUpsertResult, error) {
	if containsSensitiveKey(record.PreferenceType) {
		return PreferenceUpsertResult{}, errors.New("preference type must not contain token-like data")
	}
	if containsSensitiveKey(record.PreferenceValue) || containsSensitiveKey(record.Evidence) {
		return PreferenceUpsertResult{}, errors.New("preference value and evidence must not contain token-like data")
	}
	record.Profile = strings.TrimSpace(record.Profile)
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
	document, err := store.load()
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
		if err := store.save(document); err != nil {
			return PreferenceUpsertResult{}, err
		}
		return PreferenceUpsertResult{Record: record, Created: false, Merged: true}, nil
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = preferenceStableID(record)
	}
	document.Preferences = append(document.Preferences, record)
	if err := store.save(document); err != nil {
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
	document.Signals = filterSignals(document.Signals, profile, houseID)
	return store.save(document)
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
	signals := []InteractionSignalRecord{}
	for _, record := range document.Signals {
		if record.Profile == profile && record.HouseID == houseID {
			signals = append(signals, record)
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
		"signals":         signals,
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
	if document.Signals == nil {
		document.Signals = []InteractionSignalRecord{}
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
		Signals:         []InteractionSignalRecord{},
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

func preferenceEquivalent(left PreferenceRecord, right PreferenceRecord) bool {
	if strings.TrimSpace(left.Profile) != strings.TrimSpace(right.Profile) || strings.TrimSpace(left.HouseID) != strings.TrimSpace(right.HouseID) {
		return false
	}
	if normalizeMemoryText(left.ScopeType) != normalizeMemoryText(right.ScopeType) {
		return false
	}
	if normalizeMemoryText(left.ScopeRef) != normalizeMemoryText(right.ScopeRef) {
		return false
	}
	if normalizeMemoryText(left.PreferenceType) != normalizeMemoryText(right.PreferenceType) {
		return false
	}
	return normalizePreferenceValue(left.PreferenceValue) == normalizePreferenceValue(right.PreferenceValue)
}

func normalizeMemoryText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"，", " ", "。", " ", "、", " ", "；", " ", "：", " ",
		",", " ", ".", " ", ";", " ", ":", " ", "!", " ", "！", " ",
		"?", " ", "？", " ", "（", " ", "）", " ", "(", " ", ")", " ",
		"“", " ", "”", " ", "\"", " ", "'", " ",
	)
	normalized = replacer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func normalizePreferenceValue(value string) string {
	normalized := normalizeMemoryText(value)
	compact := compactMemoryText(normalized)
	for _, marker := range []string{
		"preferdimmer", "dimmer", "preferdarker", "darker",
		"太亮", "调暗", "暗一点", "暗一些", "暗点", "别太亮", "不要太亮", "夜里别太亮", "夜晚别太亮", "晚上别太亮", "柔和一点", "柔和些", "别刺眼", "不要刺眼", "不刺眼",
	} {
		if strings.Contains(compact, marker) {
			return "prefer_dimmer"
		}
	}
	for _, marker := range []string{
		"preferbrighter", "brighter", "亮一点", "亮一些", "亮点", "太暗", "不够亮",
	} {
		if strings.Contains(compact, marker) {
			return "prefer_brighter"
		}
	}
	for _, marker := range []string{
		"preferwarm", "warm", "暖一点", "暖一些", "暖点", "偏暖", "暖光", "柔和暖光", "温暖一点", "温暖些", "暖白",
	} {
		if strings.Contains(compact, marker) {
			return "prefer_warm"
		}
	}
	for _, marker := range []string{
		"prefercool", "cool", "冷一点", "冷一些", "冷点", "偏冷", "冷光", "冷白",
	} {
		if strings.Contains(compact, marker) {
			return "prefer_cool"
		}
	}
	for _, marker := range []string{
		"avoidcolorful", "avoidcolor", "不要彩光", "不喜欢彩色", "别用彩光", "不要彩色", "少彩光",
	} {
		if strings.Contains(compact, marker) {
			return "avoid_colorful"
		}
	}
	replacements := map[string]string{
		"prefer dimmer":   "prefer_dimmer",
		"dimmer":          "prefer_dimmer",
		"prefer darker":   "prefer_dimmer",
		"dark":            "prefer_dimmer",
		"调暗":              "prefer_dimmer",
		"暗一点":             "prefer_dimmer",
		"太亮":              "prefer_dimmer",
		"prefer brighter": "prefer_brighter",
		"brighter":        "prefer_brighter",
		"亮一点":             "prefer_brighter",
		"太暗":              "prefer_brighter",
		"prefer warm":     "prefer_warm",
		"warm":            "prefer_warm",
		"暖一点":             "prefer_warm",
		"暖光":              "prefer_warm",
		"prefer cool":     "prefer_cool",
		"cool":            "prefer_cool",
		"冷一点":             "prefer_cool",
		"冷光":              "prefer_cool",
		"avoid colorful":  "avoid_colorful",
		"avoid color":     "avoid_colorful",
		"不要彩光":            "avoid_colorful",
		"不喜欢彩色":           "avoid_colorful",
	}
	if replacement, ok := replacements[normalized]; ok {
		return replacement
	}
	return normalized
}

var memorySpacePattern = regexp.MustCompile(`\s+`)

func compactMemoryText(value string) string {
	return memorySpacePattern.ReplaceAllString(value, "")
}

func mergeEvidence(existing string, incoming string) string {
	parts := dedupeEvidenceParts(existing, incoming)
	if len(parts) == 0 {
		return ""
	}
	merged := strings.Join(parts, "；")
	runes := []rune(merged)
	if len(runes) <= maxMergedEvidenceRunes {
		return merged
	}
	return string(runes[:maxMergedEvidenceRunes])
}

func dedupeEvidenceParts(values ...string) []string {
	parts := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, "；") {
			part = strings.TrimSpace(part)
			if part == "" || evidencePartExists(parts, part) {
				continue
			}
			parts = append(parts, part)
		}
	}
	return parts
}

func evidencePartExists(parts []string, incoming string) bool {
	normalizedIncoming := normalizeMemoryText(incoming)
	for _, existing := range parts {
		normalizedExisting := normalizeMemoryText(existing)
		if normalizedExisting == normalizedIncoming {
			return true
		}
		if strings.Contains(normalizedExisting, normalizedIncoming) || strings.Contains(normalizedIncoming, normalizedExisting) {
			return true
		}
	}
	return false
}

func preferenceStableID(record PreferenceRecord) string {
	key := strings.Join([]string{
		record.Profile,
		record.HouseID,
		record.ScopeType,
		record.ScopeRef,
		record.PreferenceType,
		record.PreferenceValue,
	}, "|")
	sum := sha1.Sum([]byte(key))
	return "mem-" + hex.EncodeToString(sum[:])[:16]
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
