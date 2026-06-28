package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONStoreSavesAndLoadsPreferencesWithoutSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store := NewJSONStore(path)

	record := PreferenceRecord{
		ID:              "pref-1",
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "living-room",
		PreferenceType:  "brightness",
		PreferenceValue: "45",
		UpdatedAt:       123,
	}
	if err := store.SavePreference(record); err != nil {
		t.Fatalf("SavePreference error: %v", err)
	}

	loaded, err := NewJSONStore(path).ListPreferences("family-main", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
	if loaded[0].PreferenceValue != "45" {
		t.Fatalf("PreferenceValue = %s", loaded[0].PreferenceValue)
	}
}

func TestJSONStoreRejectsTokenLikePreferenceType(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))

	err := store.SavePreference(PreferenceRecord{
		ID:              "pref-1",
		Profile:         "family-main",
		HouseID:         "house-1",
		PreferenceType:  "accessToken",
		PreferenceValue: "secret",
	})
	if err == nil {
		t.Fatal("expected token-like preference type to be rejected")
	}
}

func TestJSONStoreUpsertPreferenceMergesNearDuplicateWarmPreference(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "柔和暖光",
		Evidence:        "第一次说喜欢柔和暖光",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("first UpsertPreference error: %v", err)
	}
	if !first.Created || first.Record.PreferenceValue != "prefer_warm" {
		t.Fatalf("first = %#v", first)
	}
	second, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "偏暖一点",
		Evidence:        "第二次说偏暖一点",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("second UpsertPreference error: %v", err)
	}
	if second.Created || !second.Merged || second.Record.ID != first.Record.ID {
		t.Fatalf("second = %#v first = %#v", second, first)
	}
	loaded, err := store.ListPreferences("family-main", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].PreferenceValue != "prefer_warm" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if !strings.Contains(loaded[0].Evidence, "第一次") || !strings.Contains(loaded[0].Evidence, "第二次") {
		t.Fatalf("merged evidence = %q", loaded[0].Evidence)
	}
}

func TestJSONStoreUpsertPreferenceMergesNightDimmerPreference(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "主卧",
		PreferenceType:  "brightness",
		PreferenceValue: "夜里别太亮",
		Evidence:        "夜间偏好",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("first UpsertPreference error: %v", err)
	}
	second, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "主卧",
		PreferenceType:  "brightness",
		PreferenceValue: "暗一点",
		Evidence:        "亮度纠正",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("second UpsertPreference error: %v", err)
	}
	if second.Created || !second.Merged || first.Record.PreferenceValue != "prefer_dimmer" || second.Record.PreferenceValue != "prefer_dimmer" {
		t.Fatalf("first = %#v second = %#v", first, second)
	}
	loaded, err := store.ListPreferences("family-main", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].PreferenceValue != "prefer_dimmer" {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestJSONStoreUpsertPreferenceDeduplicatesEvidenceParts(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "柔和暖光",
		Evidence:        "用户说客厅喜欢柔和暖光；用户说客厅喜欢柔和暖光",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("first UpsertPreference error: %v", err)
	}
	second, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "偏暖一点",
		Evidence:        "用户说客厅喜欢柔和暖光；补充为晚上也偏暖",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("second UpsertPreference error: %v", err)
	}
	if second.Created || !second.Merged || second.Record.ID != first.Record.ID {
		t.Fatalf("second = %#v first = %#v", second, first)
	}
	loaded, err := store.ListPreferences("family-main", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
	if strings.Count(loaded[0].Evidence, "用户说客厅喜欢柔和暖光") != 1 {
		t.Fatalf("duplicate evidence was not removed: %q", loaded[0].Evidence)
	}
	if !strings.Contains(loaded[0].Evidence, "补充为晚上也偏暖") {
		t.Fatalf("new evidence was not preserved: %q", loaded[0].Evidence)
	}
	if len([]rune(loaded[0].Evidence)) > maxMergedEvidenceRunes {
		t.Fatalf("evidence too long: %d", len([]rune(loaded[0].Evidence)))
	}
}

func TestJSONStoreConsentExportAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store := NewJSONStore(path)
	if err := store.SetConsent(ConsentRecord{
		Profile:         "family-main",
		HouseID:         "house-1",
		ConsentVersion:  "memory-v1",
		LearningEnabled: true,
		UpdatedAt:       123,
	}); err != nil {
		t.Fatalf("SetConsent error: %v", err)
	}
	consent, ok, err := store.Consent("family-main", "house-1")
	if err != nil {
		t.Fatalf("Consent error: %v", err)
	}
	if !ok || !consent.LearningEnabled || consent.ConsentVersion != "memory-v1" {
		t.Fatalf("consent = %#v ok=%v", consent, ok)
	}
	if err := store.SaveRecommendation(RecommendationRecord{
		ID:          "rec-1",
		Profile:     "family-main",
		HouseID:     "house-1",
		Type:        "scene_suggestion",
		Explanation: "晚上经常调暗客厅灯",
		Evidence:    "脱敏事件 3 次",
		Status:      "pending",
		CreatedAt:   123,
		UpdatedAt:   123,
	}); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}
	exported, err := store.Export("family-main", "house-1")
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	if exported["format"] != MemoryExportFormatVersion || exported["encryption"] != "not_encrypted_local_runtime_export" {
		t.Fatalf("export envelope = %#v", exported)
	}
	retention, ok := exported["retentionPolicy"].(map[string]any)
	if !ok || retention["explicitPreferences"] != ExplicitPreferenceRetention || retention["implicitCandidatesDays"] != DefaultImplicitCandidateRetentionDays {
		t.Fatalf("retentionPolicy = %#v", exported["retentionPolicy"])
	}
	if exported["importPolicy"] != "merge_by_id_replace_existing" {
		t.Fatalf("importPolicy = %#v", exported["importPolicy"])
	}
	if len(exported["consents"].([]ConsentRecord)) != 1 || len(exported["recommendations"].([]RecommendationRecord)) != 1 {
		t.Fatalf("exported = %#v", exported)
	}
	if err := store.DeleteProfileHouse("family-main", "house-1"); err != nil {
		t.Fatalf("DeleteProfileHouse error: %v", err)
	}
	exported, err = store.Export("family-main", "house-1")
	if err != nil {
		t.Fatalf("Export after delete error: %v", err)
	}
	if len(exported["consents"].([]ConsentRecord)) != 0 || len(exported["recommendations"].([]RecommendationRecord)) != 0 {
		t.Fatalf("exported after delete = %#v", exported)
	}
}

func TestJSONStoreListsRecommendationsWithCooldown(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	records := []RecommendationRecord{
		{ID: "rec-1", Profile: "p", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", CreatedAt: 1, UpdatedAt: 1},
		{ID: "rec-2", Profile: "p", HouseID: "h", Type: "scene", Explanation: "冷却", Evidence: "e2", Status: "pending", CooldownUntil: 200, CreatedAt: 1, UpdatedAt: 1},
		{ID: "rec-3", Profile: "p", HouseID: "h", Type: "scene", Explanation: "拒绝", Evidence: "e3", Status: "rejected", CreatedAt: 1, UpdatedAt: 1},
	}
	for _, record := range records {
		if err := store.SaveRecommendation(record); err != nil {
			t.Fatalf("SaveRecommendation error: %v", err)
		}
	}
	list, err := store.ListRecommendations("p", "h", 100, 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "rec-1" {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreAggregatesInteractionSignals(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := InteractionSignalRecord{
		ID:              "sig-1",
		Profile:         "p",
		HouseID:         "h",
		SignalType:      "preference_hint",
		SignalKey:       "light.brightness.adjust|room|客厅|brightness|prefer_dimmer",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "brightness",
		PreferenceValue: "prefer_dimmer",
		Evidence:        "用户交互信号：客厅太亮了",
		Count:           1,
		FirstSeenAt:     10,
		LastSeenAt:      10,
	}
	saved, err := store.SaveInteractionSignal(record)
	if err != nil {
		t.Fatalf("SaveInteractionSignal first error: %v", err)
	}
	if saved.Count != 1 || saved.FirstSeenAt != 10 {
		t.Fatalf("first saved = %#v", saved)
	}
	record.LastSeenAt = 20
	record.Evidence = "用户交互信号：客厅还是太亮"
	saved, err = store.SaveInteractionSignal(record)
	if err != nil {
		t.Fatalf("SaveInteractionSignal second error: %v", err)
	}
	if saved.Count != 2 || saved.FirstSeenAt != 10 || saved.LastSeenAt != 20 {
		t.Fatalf("second saved = %#v", saved)
	}
	signals, err := store.ListInteractionSignals("p", "h")
	if err != nil {
		t.Fatalf("ListInteractionSignals error: %v", err)
	}
	if len(signals) != 1 || signals[0].Count != 2 {
		t.Fatalf("signals = %#v", signals)
	}
}

func TestJSONStoreAppliesRecommendationFeedback(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := RecommendationRecord{ID: "rec-1", Profile: "p", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", CreatedAt: 1, UpdatedAt: 1}
	if err := store.SaveRecommendation(record); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}
	updated, err := store.ApplyRecommendationFeedback("p", "h", "rec-1", RecommendationFeedback{Status: "rejected", UpdatedAt: 100})
	if err != nil {
		t.Fatalf("ApplyRecommendationFeedback error: %v", err)
	}
	if updated.Status != "rejected" || updated.UpdatedAt != 100 {
		t.Fatalf("updated = %#v", updated)
	}
	list, err := store.ListRecommendations("p", "h", 101, 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreRecommendationFeedbackCooldown(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := RecommendationRecord{ID: "rec-1", Profile: "p", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", CreatedAt: 1, UpdatedAt: 1}
	if err := store.SaveRecommendation(record); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}
	updated, err := store.ApplyRecommendationFeedback("p", "h", "rec-1", RecommendationFeedback{Status: "pending", CooldownUntil: 200, UpdatedAt: 100})
	if err != nil {
		t.Fatalf("ApplyRecommendationFeedback error: %v", err)
	}
	if updated.Status != "pending" || updated.CooldownUntil != 200 {
		t.Fatalf("updated = %#v", updated)
	}
	list, err := store.ListRecommendations("p", "h", 150, 1)
	if err != nil {
		t.Fatalf("ListRecommendations before cooldown error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list before cooldown = %#v", list)
	}
	list, err = store.ListRecommendations("p", "h", 201, 1)
	if err != nil {
		t.Fatalf("ListRecommendations after cooldown error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "rec-1" {
		t.Fatalf("list after cooldown = %#v", list)
	}
}
