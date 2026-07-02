package main

import (
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

const memoryConsentVersion = "memory-v1"

func (app *app) invokeMemoryRemember(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	candidates := memoryPreferencesFromRequest(request)
	if len(candidates) == 0 {
		return memoryClarificationResponse(request, "missing_preference"), nil
	}
	now := time.Now().Unix()
	consent, err := app.ensureMemoryConsent(profile, region, houseID, now)
	if err != nil {
		return contract.Response{}, err
	}
	if consent.Paused {
		return memoryBlockedResponse(request, "memory_paused", "本地学习已暂停，未写入新记忆。"), nil
	}
	upserts := make([]storage.PreferenceUpsertResult, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.preferenceType == "" || candidate.preferenceValue == "" {
			continue
		}
		memoryRecord := storage.PreferenceRecord{
			Profile:         profile,
			Region:          region,
			HouseID:         houseID,
			ScopeType:       candidate.scopeType,
			ScopeRef:        candidate.scopeRef,
			PreferenceType:  candidate.preferenceType,
			PreferenceValue: candidate.preferenceValue,
			Kind:            candidate.kind,
			Status:          candidate.status,
			Evidence:        candidate.evidence,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		upsert, err := app.memoryStore.UpsertPreference(memoryRecord)
		if err != nil {
			return contract.Response{}, err
		}
		upserts = append(upserts, upsert)
	}
	if len(upserts) == 0 || strings.TrimSpace(upserts[0].Record.ID) == "" {
		return memoryClarificationResponse(request, "missing_preference"), nil
	}
	return memoryRememberResponse(request, upserts), nil
}

func (app *app) invokeMemoryList(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	now := time.Now().Unix()
	preferences, err := app.memoryStore.ListPreferences(profile, region, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	consent, err := app.ensureMemoryConsent(profile, region, houseID, now)
	if err != nil {
		return contract.Response{}, err
	}
	return memoryListResponse(request, houseID, consent, preferences), nil
}

func (app *app) invokeMemoryPauseResume(request contract.Request, profile string, region string, houseID string, paused bool) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	consent, ok, err := app.memoryStore.Consent(profile, region, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	if !ok {
		consent = storage.ConsentRecord{Profile: profile, Region: region, HouseID: houseID, ConsentVersion: memoryConsentVersion, LearningEnabled: true}
	}
	consent.Region = region
	consent.LearningEnabled = true
	consent.Paused = paused
	consent.UpdatedAt = time.Now().Unix()
	if err := app.memoryStore.SetConsent(consent); err != nil {
		return contract.Response{}, err
	}
	return memoryPauseResumeResponse(request, consent), nil
}

func (app *app) invokeMemoryForget(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	preferenceIDs, recommendationIDs := memoryForgetIDs(request)
	if len(preferenceIDs) > 0 || len(recommendationIDs) > 0 {
		deletedPreferences, err := app.memoryStore.DeletePreferencesByID(profile, region, houseID, preferenceIDs)
		if err != nil {
			return contract.Response{}, err
		}
		deletedRecommendations, err := app.memoryStore.DeleteRecommendationsByID(profile, region, houseID, recommendationIDs)
		if err != nil {
			return contract.Response{}, err
		}
		return memoryForgetResponse(request, houseID, selectedMemoryExport(profile, region, houseID, deletedPreferences, deletedRecommendations)), nil
	}
	exported, err := app.memoryStore.Export(profile, region, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.memoryStore.DeleteProfileHouse(profile, region, houseID); err != nil {
		return contract.Response{}, err
	}
	return memoryForgetResponse(request, houseID, exported), nil
}

func memoryForgetIDs(request contract.Request) ([]string, []string) {
	genericIDs := requestStringList(request.Parameters[semantic.FieldIDs], request.Parameters[semantic.FieldID])
	preferenceIDs := requestStringList(request.Parameters[semantic.FieldPreferenceIDs], request.Parameters[semantic.FieldPreferenceID])
	recommendationIDs := requestStringList(request.Parameters[semantic.FieldRecommendationIDs], request.Parameters[semantic.FieldRecommendationID])
	if len(genericIDs) > 0 {
		preferenceIDs = append(preferenceIDs, genericIDs...)
		recommendationIDs = append(recommendationIDs, genericIDs...)
	}
	return preferenceIDs, recommendationIDs
}

func selectedMemoryExport(profile string, region string, houseID string, preferences []storage.PreferenceRecord, recommendations []storage.RecommendationRecord) map[string]any {
	return map[string]any{
		semantic.FieldFormat:          storage.MemoryExportFormatVersion,
		semantic.FieldNamespace:       memoryResponseNamespace(profile, region, houseID),
		semantic.FieldProfile:         profile,
		semantic.FieldRegion:          region,
		semantic.FieldHouseID:         houseID,
		semantic.FieldEncryption:      "not_encrypted_local_runtime_export",
		semantic.FieldImportPolicy:    "merge_by_id_replace_existing",
		semantic.FieldRetentionPolicy: storage.RetentionPolicy(),
		semantic.FieldPreferences:     preferences,
		semantic.FieldRecommendations: recommendations,
	}
}

func (app *app) invokeRecommendationList(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	now := time.Now().Unix()
	if _, err := app.ensureMemoryConsent(profile, region, houseID, now); err != nil {
		return contract.Response{}, err
	}
	recommendations, err := app.memoryStore.ListRecommendations(profile, region, houseID, now, 1)
	if err != nil {
		return contract.Response{}, err
	}
	return recommendationListResponse(request, profile, region, houseID, recommendations), nil
}

func (app *app) invokeRecommendationRecord(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	record, ok := recommendationRecordFromRequest(request, profile, region, houseID, time.Now().Unix())
	if !ok {
		return memoryClarificationResponse(request, "missing_recommendation_candidate"), nil
	}
	upsert, err := app.memoryStore.UpsertRecommendation(record)
	if err != nil {
		return contract.Response{}, err
	}
	return recommendationRecordResponse(request, houseID, upsert), nil
}

func (app *app) invokeRecommendationFeedback(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	recommendationID := firstRequestString(request.Parameters, semantic.FieldRecommendationID, semantic.FieldID)
	if recommendationID == "" {
		return memoryClarificationResponse(request, "missing_recommendation_id"), nil
	}
	feedback := normalizeRecommendationFeedback(firstRequestString(request.Parameters, semantic.FieldFeedback, semantic.FieldAction, semantic.FieldStatus))
	if feedback == "" {
		return memoryClarificationResponse(request, "missing_recommendation_feedback"), nil
	}
	now := time.Now().Unix()
	update := storage.RecommendationFeedback{Status: feedback, UpdatedAt: now}
	if feedback == "cooldown" {
		update.Status = "pending"
		update.CooldownUntil = now + recommendationCooldownSeconds(request)
	}
	record, err := app.memoryStore.ApplyRecommendationFeedback(profile, region, houseID, recommendationID, update)
	if err != nil {
		return recommendationFeedbackBlockedResponse(request, recommendationID, "recommendation_not_found", "未找到要更新的本地推荐。"), nil
	}
	return recommendationFeedbackResponse(request, houseID, record), nil
}

func (app *app) ensureMemoryConsent(profile string, region string, houseID string, now int64) (storage.ConsentRecord, error) {
	consent, ok, err := app.memoryStore.Consent(profile, region, houseID)
	if err != nil {
		return storage.ConsentRecord{}, err
	}
	if !ok {
		consent = storage.ConsentRecord{
			Profile:         profile,
			Region:          region,
			HouseID:         houseID,
			ConsentVersion:  memoryConsentVersion,
			LearningEnabled: true,
			UpdatedAt:       now,
		}
		if err := app.memoryStore.SetConsent(consent); err != nil {
			return storage.ConsentRecord{}, err
		}
		return consent, nil
	}
	changed := false
	if strings.TrimSpace(consent.Region) == "" || consent.Region != region {
		consent.Region = region
		changed = true
	}
	if strings.TrimSpace(consent.ConsentVersion) == "" {
		consent.ConsentVersion = memoryConsentVersion
		changed = true
	}
	if !consent.LearningEnabled {
		consent.LearningEnabled = true
		changed = true
	}
	if changed {
		consent.UpdatedAt = now
		if err := app.memoryStore.SetConsent(consent); err != nil {
			return storage.ConsentRecord{}, err
		}
	}
	return consent, nil
}

func normalizeRecommendationFeedback(value string) string {
	switch strings.ToLower(firstNonEmptyString(value)) {
	case "accept", "accepted", "接受", "采纳":
		return "accepted"
	case "dismiss", "dismissed", "hide", "hidden", "ignore", "ignored", "稍后", "忽略", "隐藏", "不显示", "不要再提醒", "别再提醒":
		return "dismissed"
	case "reject", "rejected", "decline", "declined", "拒绝", "不采纳", "不再推荐", "别再推荐":
		return "rejected"
	case "cooldown", "later", "remind_later", "稍后提醒":
		return "cooldown"
	default:
		return ""
	}
}

func recommendationCooldownSeconds(request contract.Request) int64 {
	value, ok := requestInteger(request.Parameters[semantic.FieldCooldownHours])
	if !ok || value < 1 || value > 24*30 {
		return int64(7 * 24 * 60 * 60)
	}
	return int64(value * 60 * 60)
}

func recommendationRecordFromRequest(request contract.Request, profile string, region string, houseID string, now int64) (storage.RecommendationRecord, bool) {
	source := request.Parameters
	if nested := requestMap(request.Parameters[semantic.FieldRecommendation]); nested != nil {
		source = nested
	}
	recommendationType := firstRequestString(source, semantic.FieldType, semantic.FieldRecommendationType)
	explanation := firstRequestString(source, semantic.FieldExplanation)
	evidence := firstRequestString(source, semantic.FieldEvidence)
	if recommendationType == "" || explanation == "" || evidence == "" {
		return storage.RecommendationRecord{}, false
	}
	priority, _ := requestInteger(source[semantic.FieldPriority])
	record := storage.RecommendationRecord{
		ID:             firstRequestString(source, semantic.FieldRecommendationID, semantic.FieldID),
		Profile:        profile,
		Region:         region,
		HouseID:        houseID,
		Type:           recommendationType,
		Source:         firstRequestString(source, semantic.FieldSource),
		TargetIntent:   firstRequestString(source, semantic.FieldTargetIntent),
		ScopeType:      firstRequestString(source, semantic.FieldScopeType),
		ScopeRef:       firstRequestString(source, semantic.FieldScopeRef),
		Priority:       priority,
		Confidence:     firstRequestString(source, semantic.FieldConfidence),
		ActionHint:     requestMap(source[semantic.FieldActionHint]),
		ParametersHint: requestMap(source[semantic.FieldParametersHint]),
		Explanation:    explanation,
		Evidence:       evidence,
		Status:         firstRequestString(source, semantic.FieldStatus),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return record, true
}
