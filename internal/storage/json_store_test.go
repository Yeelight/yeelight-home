package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestJSONStoreSavesAndLoadsPreferencesWithoutSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store := NewJSONStore(path)

	record := PreferenceRecord{
		ID:              "pref-1",
		Profile:         "family-main",
		Region:          "cn",
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

	loaded, err := NewJSONStore(path).ListPreferences("family-main", "cn", "house-1")
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

func TestJSONStoreShardsByProfileRegionAndHouse(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory.json")
	store := NewJSONStore(path)

	if err := store.SavePreference(PreferenceRecord{
		ID:              "pref-main",
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		PreferenceType:  "brightness",
		PreferenceValue: "45",
		UpdatedAt:       123,
	}); err != nil {
		t.Fatalf("SavePreference main error: %v", err)
	}
	if err := store.SavePreference(PreferenceRecord{
		ID:              "pref-alt",
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-2",
		PreferenceType:  "brightness",
		PreferenceValue: "60",
		UpdatedAt:       124,
	}); err != nil {
		t.Fatalf("SavePreference alt error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("legacy memory file should not be written, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "memory", "family-main", "cn", "house-1.json")); err != nil {
		t.Fatalf("house-1 shard missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "memory", "family-main", "cn", "house-2.json")); err != nil {
		t.Fatalf("house-2 shard missing: %v", err)
	}

	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences house-1 error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "pref-main" {
		t.Fatalf("loaded house-1 = %#v", loaded)
	}
}

func TestJSONStoreConcurrentWritesUseUniqueTempFiles(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for index, value := range []string{"prefer_warm", "prefer_dim"} {
		wg.Add(1)
		go func(index int, value string) {
			defer wg.Done()
			_, err := store.UpsertPreference(PreferenceRecord{
				Profile:         "default",
				Region:          "dev",
				HouseID:         "200171",
				Kind:            "explicit",
				Status:          "confirmed",
				ScopeType:       "profile",
				PreferenceType:  "lighting_preference",
				PreferenceValue: value,
				Evidence:        "concurrent-write-test",
				CreatedAt:       int64(100 + index),
				UpdatedAt:       int64(100 + index),
			})
			errs <- err
		}(index, value)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write error: %v", err)
		}
	}
	loaded, err := store.ListPreferences("default", "dev", "200171")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) == 0 {
		t.Fatalf("preferences should not be empty after concurrent writes")
	}
}

func TestJSONStoreSeparatesSameProfileHouseByRegion(t *testing.T) {
	root := t.TempDir()
	store := NewJSONStore(filepath.Join(root, "memory.json"))
	for _, record := range []PreferenceRecord{
		{ID: "pref-cn", Profile: "family-main", Region: "cn", HouseID: "house-1", PreferenceType: "brightness", PreferenceValue: "45", UpdatedAt: 123},
		{ID: "pref-dev", Profile: "family-main", Region: "dev", HouseID: "house-1", PreferenceType: "brightness", PreferenceValue: "60", UpdatedAt: 124},
	} {
		if err := store.SavePreference(record); err != nil {
			t.Fatalf("SavePreference %#v error: %v", record, err)
		}
	}
	cn, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences cn error: %v", err)
	}
	dev, err := store.ListPreferences("family-main", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences dev error: %v", err)
	}
	if len(cn) != 1 || cn[0].ID != "pref-cn" || len(dev) != 1 || dev[0].ID != "pref-dev" {
		t.Fatalf("cn=%#v dev=%#v", cn, dev)
	}
}

func TestJSONStoreMigratesLegacyMemoryDocument(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory.json")
	legacy := jsonDocument{
		Version: 1,
		Consents: []ConsentRecord{{
			Profile:         "family-main",
			Region:          "cn",
			HouseID:         "house-1",
			ConsentVersion:  "memory-v1",
			LearningEnabled: true,
			UpdatedAt:       123,
		}},
		Preferences: []PreferenceRecord{{
			ID:              "pref-1",
			Profile:         "family-main",
			Region:          "cn",
			HouseID:         "house-1",
			PreferenceType:  "brightness",
			PreferenceValue: "45",
			UpdatedAt:       123,
		}, {
			ID:              "pref-other",
			Profile:         "family-main",
			Region:          "cn",
			HouseID:         "house-2",
			PreferenceType:  "brightness",
			PreferenceValue: "60",
			UpdatedAt:       123,
		}},
		Recommendations: []RecommendationRecord{{
			ID:          "rec-1",
			Profile:     "family-main",
			Region:      "cn",
			HouseID:     "house-1",
			Type:        "scene",
			Explanation: "可用",
			Evidence:    "e1",
			Status:      "pending",
			CreatedAt:   1,
			UpdatedAt:   1,
		}},
		Signals: []InteractionSignalRecord{{
			ID:         "sig-1",
			Profile:    "family-main",
			Region:     "cn",
			HouseID:    "house-1",
			SignalType: "interaction",
			SignalKey:  "k",
			Count:      1,
		}},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	store := NewJSONStore(path)
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "pref-1" {
		t.Fatalf("loaded = %#v", loaded)
	}
	consent, ok, err := store.Consent("family-main", "cn", "house-1")
	if err != nil || !ok || !consent.LearningEnabled {
		t.Fatalf("consent = %#v ok=%v err=%v", consent, ok, err)
	}
	recommendations, err := store.ListRecommendations("family-main", "cn", "house-1", 2, 1)
	if err != nil || len(recommendations) != 1 || recommendations[0].ID != "rec-1" {
		t.Fatalf("recommendations = %#v err=%v", recommendations, err)
	}
	signals, err := store.ListInteractionSignals("family-main", "cn", "house-1")
	if err != nil || len(signals) != 1 || signals[0].ID != "sig-1" {
		t.Fatalf("signals = %#v err=%v", signals, err)
	}
	shardPath := filepath.Join(root, "memory", "family-main", "cn", "house-1.json")
	shardData, err := os.ReadFile(shardPath)
	if err != nil {
		t.Fatalf("migrated shard missing: %v", err)
	}
	if strings.Contains(string(shardData), "pref-other") {
		t.Fatalf("migrated shard contains another house: %s", string(shardData))
	}
}

func TestJSONStoreDeleteProfileHousePreventsLegacyResurrection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory.json")
	legacy := jsonDocument{
		Version: 1,
		Preferences: []PreferenceRecord{{
			ID:              "pref-1",
			Profile:         "family-main",
			Region:          "cn",
			HouseID:         "house-1",
			PreferenceType:  "brightness",
			PreferenceValue: "45",
			UpdatedAt:       123,
		}},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	store := NewJSONStore(path)
	if err := store.DeleteProfileHouse("family-main", "cn", "house-1"); err != nil {
		t.Fatalf("DeleteProfileHouse error: %v", err)
	}
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("legacy data resurrected after delete: %#v", loaded)
	}
}

func TestJSONStoreRejectsTokenLikePreferenceType(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))

	err := store.SavePreference(PreferenceRecord{
		ID:              "pref-1",
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		PreferenceType:  "accessToken",
		PreferenceValue: "secret",
	})
	if err == nil {
		t.Fatal("expected token-like preference type to be rejected")
	}
}

func TestJSONStoreUpsertPreferenceMergesSameStructuredPreference(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "prefer_warm",
		Evidence:        "第一次结构化写入暖色温偏好",
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
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "prefer_warm",
		Evidence:        "第二次结构化写入同一偏好",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("second UpsertPreference error: %v", err)
	}
	if second.Created || !second.Merged || second.Record.ID != first.Record.ID {
		t.Fatalf("second = %#v first = %#v", second, first)
	}
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].PreferenceValue != "prefer_warm" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if !strings.Contains(loaded[0].Evidence, "第一次结构化") || !strings.Contains(loaded[0].Evidence, "第二次结构化") {
		t.Fatalf("merged evidence = %q", loaded[0].Evidence)
	}
}

func TestJSONStoreUpsertPreferenceDoesNotMergeSemanticSynonyms(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
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
		Region:          "cn",
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
	if !first.Created || !second.Created || first.Record.ID == second.Record.ID {
		t.Fatalf("first = %#v second = %#v", first, second)
	}
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestJSONStoreUpsertPreferenceDeduplicatesEvidenceParts(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "prefer_warm",
		Evidence:        "用户说客厅喜欢柔和暖光；用户说客厅喜欢柔和暖光",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("first UpsertPreference error: %v", err)
	}
	second, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "color_temperature",
		PreferenceValue: "prefer_warm",
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
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
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

func TestJSONStoreUpsertPreferenceMergesCallerProvidedCanonicalValues(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	romantic, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "home",
		PreferenceType:  "ambience",
		PreferenceValue: "prefer_romantic_warm",
		Evidence:        "用户说喜欢浪漫色调",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("romantic UpsertPreference error: %v", err)
	}
	romanticAgain, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "home",
		PreferenceType:  "ambience",
		PreferenceValue: "prefer_romantic_warm",
		Evidence:        "用户重复说温馨浪漫",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("romanticAgain UpsertPreference error: %v", err)
	}
	premium, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "home",
		PreferenceType:  "product_preference",
		PreferenceValue: "prefer_premium_luxury",
		Evidence:        "用户说喜欢高端奢华",
		CreatedAt:       30,
		UpdatedAt:       30,
	})
	if err != nil {
		t.Fatalf("premium UpsertPreference error: %v", err)
	}
	premiumAgain, err := store.UpsertPreference(PreferenceRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ScopeType:       "home",
		PreferenceType:  "product_preference",
		PreferenceValue: "prefer_premium_luxury",
		Evidence:        "用户重复说旗舰优先",
		CreatedAt:       40,
		UpdatedAt:       40,
	})
	if err != nil {
		t.Fatalf("premiumAgain UpsertPreference error: %v", err)
	}
	if !romantic.Created || !romanticAgain.Merged || romantic.Record.ID != romanticAgain.Record.ID {
		t.Fatalf("romantic=%#v romanticAgain=%#v", romantic, romanticAgain)
	}
	if !premium.Created || !premiumAgain.Merged || premium.Record.ID != premiumAgain.Record.ID {
		t.Fatalf("premium=%#v premiumAgain=%#v", premium, premiumAgain)
	}
	loaded, err := store.ListPreferences("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded = %#v", loaded)
	}
	valuesByType := map[string]string{}
	for _, item := range loaded {
		valuesByType[item.PreferenceType] = item.PreferenceValue
	}
	if valuesByType["ambience"] != "prefer_romantic_warm" || valuesByType["product_preference"] != "prefer_premium_luxury" {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestJSONStoreConsentExportAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store := NewJSONStore(path)
	if err := store.SetConsent(ConsentRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		ConsentVersion:  "memory-v1",
		LearningEnabled: true,
		UpdatedAt:       123,
	}); err != nil {
		t.Fatalf("SetConsent error: %v", err)
	}
	consent, ok, err := store.Consent("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("Consent error: %v", err)
	}
	if !ok || !consent.LearningEnabled || consent.ConsentVersion != "memory-v1" {
		t.Fatalf("consent = %#v ok=%v", consent, ok)
	}
	if err := store.SaveRecommendation(RecommendationRecord{
		ID:          "rec-1",
		Profile:     "family-main",
		Region:      "cn",
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
	exported, err := store.Export("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	if exported["format"] != MemoryExportFormatVersion || exported["encryption"] != "not_encrypted_local_runtime_export" {
		t.Fatalf("export envelope = %#v", exported)
	}
	retention, ok := exported["retentionPolicy"].(map[string]any)
	if !ok ||
		retention["explicitPreferences"] != ExplicitPreferenceRetention ||
		retention["recommendationEvidenceDays"] != DefaultRecommendationRetentionDays ||
		retention["interactionEventsDays"] != DefaultInteractionRetentionDays ||
		retention["runtimeSubjectiveInferencePolicy"] != "disabled" {
		t.Fatalf("retentionPolicy = %#v", exported["retentionPolicy"])
	}
	namespace, ok := exported["namespace"].(StorageNamespace)
	if !ok ||
		namespace.Profile != "family-main" ||
		namespace.AccountProfile != "family-main" ||
		namespace.Region != "cn" ||
		namespace.HouseID != "house-1" ||
		namespace.DataType != "memory" {
		t.Fatalf("namespace = %#v", exported["namespace"])
	}
	if _, exists := retention["implicitCandidatesDays"]; exists {
		t.Fatalf("retentionPolicy must not expose implicit candidates: %#v", retention)
	}
	if exported["importPolicy"] != "merge_by_id_replace_existing" {
		t.Fatalf("importPolicy = %#v", exported["importPolicy"])
	}
	if len(exported["consents"].([]ConsentRecord)) != 1 || len(exported["recommendations"].([]RecommendationRecord)) != 1 {
		t.Fatalf("exported = %#v", exported)
	}
	if err := store.DeleteProfileHouse("family-main", "cn", "house-1"); err != nil {
		t.Fatalf("DeleteProfileHouse error: %v", err)
	}
	exported, err = store.Export("family-main", "cn", "house-1")
	if err != nil {
		t.Fatalf("Export after delete error: %v", err)
	}
	if len(exported["consents"].([]ConsentRecord)) != 0 || len(exported["recommendations"].([]RecommendationRecord)) != 0 {
		t.Fatalf("exported after delete = %#v", exported)
	}
}

func TestJSONStoreUpsertOperationLessonMergesDuplicate(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first, err := store.UpsertOperationLesson(OperationLessonRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		Intent:          "scene.update",
		LessonType:      "parameter_shape",
		Symptom:         "invalid_scene_update_payload",
		RecommendedPath: "先 scene.detail.get 再完整更新 details",
		Evidence:        "第一次失败",
		CreatedAt:       10,
		UpdatedAt:       10,
	})
	if err != nil {
		t.Fatalf("first UpsertOperationLesson error: %v", err)
	}
	second, err := store.UpsertOperationLesson(OperationLessonRecord{
		Profile:         "family-main",
		Region:          "cn",
		HouseID:         "house-1",
		Intent:          "scene.update",
		LessonType:      "parameter_shape",
		Symptom:         "invalid_scene_update_payload",
		RecommendedPath: "先 scene.detail.get 再完整更新 details",
		Evidence:        "第一次失败；第二次确认",
		CreatedAt:       20,
		UpdatedAt:       20,
	})
	if err != nil {
		t.Fatalf("second UpsertOperationLesson error: %v", err)
	}
	if !first.Created || second.Created || !second.Merged || first.Record.ID != second.Record.ID {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	lessons, err := store.ListOperationLessons(OperationLessonFilter{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update"})
	if err != nil {
		t.Fatalf("ListOperationLessons error: %v", err)
	}
	if len(lessons) != 1 || lessons[0].HitCount != 2 {
		t.Fatalf("lessons = %#v", lessons)
	}
	if strings.Count(lessons[0].Evidence, "第一次失败") != 1 || !strings.Contains(lessons[0].Evidence, "第二次确认") {
		t.Fatalf("merged evidence = %q", lessons[0].Evidence)
	}
}

func TestJSONStoreListsHouseAndGlobalOperationLessons(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	for _, lesson := range []OperationLessonRecord{
		{
			Profile:         "family-main",
			Region:          "cn",
			Intent:          "light.power.set",
			LessonType:      "fast_path",
			Symptom:         "用户直接开灯时不应先 entity.list",
			RecommendedPath: "直接 light.power.set 并传自然语言目标",
			CreatedAt:       10,
			UpdatedAt:       10,
		},
		{
			Profile:         "family-main",
			Region:          "cn",
			HouseID:         "house-1",
			Intent:          "light.power.set",
			LessonType:      "resource_resolution",
			Symptom:         "孩子屋主灯存在重名",
			RecommendedPath: "带 roomName=孩子屋",
			CreatedAt:       20,
			UpdatedAt:       20,
		},
	} {
		if _, err := store.UpsertOperationLesson(lesson); err != nil {
			t.Fatalf("UpsertOperationLesson error: %v", err)
		}
	}
	lessons, err := store.ListOperationLessons(OperationLessonFilter{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "light.power.set", Limit: 10})
	if err != nil {
		t.Fatalf("ListOperationLessons error: %v", err)
	}
	if len(lessons) != 2 {
		t.Fatalf("lessons = %#v", lessons)
	}
	if lessons[0].HouseID != "house-1" || lessons[1].HouseID != "" {
		t.Fatalf("lesson order/scope = %#v", lessons)
	}
}

func TestJSONStoreOperationLessonLifecycleFiltersStaleAndRejected(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	for _, lesson := range []OperationLessonRecord{
		{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update", LessonType: "fast_path", Symptom: "可用路径", RecommendedPath: "直接使用 editablePayload", Source: "ai_skill", Confidence: "high", Status: "confirmed", LastValidatedAt: 100, CreatedAt: 100, UpdatedAt: 100},
		{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update", LessonType: "fallback", Symptom: "旧路径", RecommendedPath: "旧 fallback", Source: "runtime_response", Confidence: "medium", Status: "confirmed", Stale: true, CreatedAt: 90, UpdatedAt: 90},
		{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update", LessonType: "failure_pattern", Symptom: "错误经验", RecommendedPath: "不要使用", Source: "ai_skill", Confidence: "low", Status: "rejected", CreatedAt: 80, UpdatedAt: 80},
	} {
		if _, err := store.UpsertOperationLesson(lesson); err != nil {
			t.Fatalf("UpsertOperationLesson error: %v", err)
		}
	}
	active, err := store.ListOperationLessons(OperationLessonFilter{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update", MinConfidence: "medium", Limit: 10})
	if err != nil {
		t.Fatalf("ListOperationLessons active error: %v", err)
	}
	if len(active) != 1 || active[0].Symptom != "可用路径" || active[0].Source != "ai_skill" || active[0].LastValidatedAt != 100 {
		t.Fatalf("active = %#v", active)
	}
	all, err := store.ListOperationLessons(OperationLessonFilter{Profile: "family-main", Region: "cn", HouseID: "house-1", Intent: "scene.update", IncludeStale: true, IncludeRejected: true, Limit: 10})
	if err != nil {
		t.Fatalf("ListOperationLessons all error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("all = %#v", all)
	}
}

func TestJSONStoreListsRecommendationsWithCooldown(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	records := []RecommendationRecord{
		{ID: "rec-1", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", Priority: 10, Confidence: "medium", CreatedAt: 1, UpdatedAt: 1},
		{ID: "rec-2", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "冷却", Evidence: "e2", Status: "pending", CooldownUntil: 200, CreatedAt: 1, UpdatedAt: 1},
		{ID: "rec-3", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "拒绝", Evidence: "e3", Status: "rejected", CreatedAt: 1, UpdatedAt: 1},
		{ID: "rec-4", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "高优先级", Evidence: "e4", Status: "pending", Priority: 90, Confidence: "high", CreatedAt: 1, UpdatedAt: 2},
	}
	for _, record := range records {
		if err := store.SaveRecommendation(record); err != nil {
			t.Fatalf("SaveRecommendation error: %v", err)
		}
	}
	list, err := store.ListRecommendations("p", "cn", "h", 100, 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "rec-4" {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreSeparatesMemoryByRegion(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	if err := store.SavePreference(PreferenceRecord{
		ID:              "pref-default",
		Profile:         "p",
		Region:          "",
		HouseID:         "h",
		ScopeType:       "home",
		PreferenceType:  "brightness",
		PreferenceValue: "default",
		UpdatedAt:       1,
	}); err != nil {
		t.Fatalf("SavePreference default error: %v", err)
	}
	if err := store.SavePreference(PreferenceRecord{
		ID:              "pref-cn",
		Profile:         "p",
		Region:          "cn",
		HouseID:         "h",
		ScopeType:       "home",
		PreferenceType:  "brightness",
		PreferenceValue: "cn",
		UpdatedAt:       1,
	}); err != nil {
		t.Fatalf("SavePreference cn error: %v", err)
	}
	cn, err := store.ListPreferences("p", "cn", "h")
	if err != nil {
		t.Fatalf("ListPreferences cn error: %v", err)
	}
	if len(cn) != 1 || cn[0].ID != "pref-cn" {
		t.Fatalf("cn = %#v", cn)
	}
	def, err := store.ListPreferences("p", "", "h")
	if err != nil {
		t.Fatalf("ListPreferences default error: %v", err)
	}
	if len(def) != 1 || def[0].ID != "pref-default" {
		t.Fatalf("default = %#v", def)
	}
}

func TestJSONStoreUpsertsRecommendationByStructuredCandidate(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	first := RecommendationRecord{
		Profile:      "p",
		Region:       "cn",
		HouseID:      "h",
		Type:         "automation",
		Source:       "ai_skill",
		TargetIntent: "automation.create",
		ScopeType:    "room",
		ScopeRef:     "主卧",
		Explanation:  "可以创建主卧晚间自动化。",
		Evidence:     "证据一",
		CreatedAt:    10,
		UpdatedAt:    10,
	}
	saved, err := store.UpsertRecommendation(first)
	if err != nil {
		t.Fatalf("UpsertRecommendation first error: %v", err)
	}
	if !saved.Created || saved.Merged || saved.Record.ID == "" || saved.Record.Status != "pending" {
		t.Fatalf("first saved = %#v", saved)
	}
	second := first
	second.Evidence = "证据二"
	second.CreatedAt = 20
	second.UpdatedAt = 20
	saved, err = store.UpsertRecommendation(second)
	if err != nil {
		t.Fatalf("UpsertRecommendation second error: %v", err)
	}
	if saved.Created || !saved.Merged || saved.Record.ID == "" {
		t.Fatalf("second saved = %#v", saved)
	}
	list, err := store.ListRecommendations("p", "cn", "h", 21, 10)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 1 || !strings.Contains(list[0].Evidence, "证据一") || !strings.Contains(list[0].Evidence, "证据二") {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreRecommendationRejectSuppressesDuplicateUpsert(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := RecommendationRecord{
		Profile:      "p",
		Region:       "cn",
		HouseID:      "h",
		Type:         "automation",
		TargetIntent: "automation.create",
		ScopeType:    "room",
		ScopeRef:     "主卧",
		Explanation:  "可以创建主卧晚间自动化。",
		Evidence:     "证据一",
		CreatedAt:    10,
		UpdatedAt:    10,
	}
	saved, err := store.UpsertRecommendation(record)
	if err != nil {
		t.Fatalf("UpsertRecommendation error: %v", err)
	}
	if _, err := store.ApplyRecommendationFeedback("p", "cn", "h", saved.Record.ID, RecommendationFeedback{Status: "rejected", UpdatedAt: 20}); err != nil {
		t.Fatalf("ApplyRecommendationFeedback error: %v", err)
	}
	record.Evidence = "证据二"
	record.CreatedAt = 30
	record.UpdatedAt = 30
	saved, err = store.UpsertRecommendation(record)
	if err != nil {
		t.Fatalf("UpsertRecommendation duplicate error: %v", err)
	}
	if saved.Record.Status != "rejected" {
		t.Fatalf("duplicate revived rejected recommendation: %#v", saved)
	}
	list, err := store.ListRecommendations("p", "cn", "h", 31, 10)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreAggregatesInteractionSignals(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := InteractionSignalRecord{
		ID:          "sig-1",
		Profile:     "p",
		Region:      "cn",
		HouseID:     "h",
		SignalType:  "interaction",
		SignalKey:   "light.brightness.adjust|interaction",
		Evidence:    "intent=light.brightness.adjust; status=success",
		Count:       1,
		FirstSeenAt: 10,
		LastSeenAt:  10,
	}
	saved, err := store.SaveInteractionSignal(record)
	if err != nil {
		t.Fatalf("SaveInteractionSignal first error: %v", err)
	}
	if saved.Count != 1 || saved.FirstSeenAt != 10 {
		t.Fatalf("first saved = %#v", saved)
	}
	record.LastSeenAt = 20
	record.Evidence = "intent=light.brightness.adjust; status=success"
	saved, err = store.SaveInteractionSignal(record)
	if err != nil {
		t.Fatalf("SaveInteractionSignal second error: %v", err)
	}
	if saved.Count != 2 || saved.FirstSeenAt != 10 || saved.LastSeenAt != 20 {
		t.Fatalf("second saved = %#v", saved)
	}
	signals, err := store.ListInteractionSignals("p", "cn", "h")
	if err != nil {
		t.Fatalf("ListInteractionSignals error: %v", err)
	}
	if len(signals) != 1 || signals[0].Count != 2 {
		t.Fatalf("signals = %#v", signals)
	}
}

func TestJSONStoreCompactsExpiredRecommendationAndSignalRecords(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	old := int64(100)
	now := old + int64(DefaultRecommendationRetentionDays+1)*secondsPerDay
	for _, record := range []RecommendationRecord{
		{ID: "accepted-old", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "old", Evidence: "old", Status: "accepted", CreatedAt: old, UpdatedAt: old},
		{ID: "pending-old", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "pending", Evidence: "pending", Status: "pending", CreatedAt: old, UpdatedAt: old},
		{ID: "pending-new", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "new", Evidence: "new", Status: "pending", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.SaveRecommendation(record); err != nil {
			t.Fatalf("SaveRecommendation error: %v", err)
		}
	}
	if _, err := store.SaveInteractionSignal(InteractionSignalRecord{
		ID:          "sig-old",
		Profile:     "p",
		Region:      "cn",
		HouseID:     "h",
		SignalType:  "interaction",
		SignalKey:   "old|interaction",
		Evidence:    "intent=old; status=success",
		Count:       1,
		FirstSeenAt: old,
		LastSeenAt:  old,
	}); err != nil {
		t.Fatalf("SaveInteractionSignal old error: %v", err)
	}
	if _, err := store.SaveInteractionSignal(InteractionSignalRecord{
		ID:          "sig-new",
		Profile:     "p",
		Region:      "cn",
		HouseID:     "h",
		SignalType:  "interaction",
		SignalKey:   "new|interaction",
		Evidence:    "intent=new; status=success",
		Count:       1,
		FirstSeenAt: now,
		LastSeenAt:  now,
	}); err != nil {
		t.Fatalf("SaveInteractionSignal new error: %v", err)
	}
	list, err := store.ListRecommendations("p", "cn", "h", now, 10)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("pending recommendations should be retained: %#v", list)
	}
	signals, err := store.ListInteractionSignals("p", "cn", "h")
	if err != nil {
		t.Fatalf("ListInteractionSignals error: %v", err)
	}
	if len(signals) != 1 || signals[0].ID != "sig-new" {
		t.Fatalf("signals = %#v", signals)
	}
	exported, err := store.Export("p", "cn", "h")
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	recommendations := exported["recommendations"].([]RecommendationRecord)
	for _, record := range recommendations {
		if record.ID == "accepted-old" {
			t.Fatalf("expired accepted recommendation was retained: %#v", recommendations)
		}
	}
}

func TestJSONStoreAppliesRecommendationFeedback(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := RecommendationRecord{ID: "rec-1", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", CreatedAt: 1, UpdatedAt: 1}
	if err := store.SaveRecommendation(record); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}
	updated, err := store.ApplyRecommendationFeedback("p", "cn", "h", "rec-1", RecommendationFeedback{Status: "rejected", UpdatedAt: 100})
	if err != nil {
		t.Fatalf("ApplyRecommendationFeedback error: %v", err)
	}
	if updated.Status != "rejected" || updated.UpdatedAt != 100 {
		t.Fatalf("updated = %#v", updated)
	}
	list, err := store.ListRecommendations("p", "cn", "h", 101, 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestJSONStoreRecommendationFeedbackCooldown(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "memory.json"))
	record := RecommendationRecord{ID: "rec-1", Profile: "p", Region: "cn", HouseID: "h", Type: "scene", Explanation: "可用", Evidence: "e1", Status: "pending", CreatedAt: 1, UpdatedAt: 1}
	if err := store.SaveRecommendation(record); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}
	updated, err := store.ApplyRecommendationFeedback("p", "cn", "h", "rec-1", RecommendationFeedback{Status: "pending", CooldownUntil: 200, UpdatedAt: 100})
	if err != nil {
		t.Fatalf("ApplyRecommendationFeedback error: %v", err)
	}
	if updated.Status != "pending" || updated.CooldownUntil != 200 {
		t.Fatalf("updated = %#v", updated)
	}
	list, err := store.ListRecommendations("p", "cn", "h", 150, 1)
	if err != nil {
		t.Fatalf("ListRecommendations before cooldown error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list before cooldown = %#v", list)
	}
	list, err = store.ListRecommendations("p", "cn", "h", 201, 1)
	if err != nil {
		t.Fatalf("ListRecommendations after cooldown error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "rec-1" {
		t.Fatalf("list after cooldown = %#v", list)
	}
}
