package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
	"github.com/yeelight/yeelight-home/internal/storage"
)

const memoryConsentVersion = "memory-v1"

func (app *app) invokeMemoryRememberPlan(request contract.Request, profile string, region string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	candidate := memoryPreferenceFromRequest(request)
	if candidate.preferenceType == "" || candidate.preferenceValue == "" {
		return memoryClarificationResponse(request, "missing_preference"), nil
	}
	payload := map[string]any{
		"preferenceType":  candidate.preferenceType,
		"preferenceValue": candidate.preferenceValue,
		"scopeType":       candidate.scopeType,
		"scopeRef":        candidate.scopeRef,
		"kind":            candidate.kind,
		"evidence":        candidate.evidence,
	}
	record, err := plan.NewRecord(profile, region, houseID, "memory.remember", request.RequestID, fmt.Sprintf("记住偏好 %s=%s", candidate.preferenceType, candidate.preferenceValue), payload, []string{
		"仅写入本地 JSON 记忆",
		"不会创建情景、自动化或设备配置",
		"token-like 字段会被拒绝",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	return pendingMemoryPlanResponse(request, record), nil
}

func (app *app) commitMemoryRememberPlan(_ context.Context, request contract.Request, record plan.Record) (contract.Response, error) {
	now := time.Now().Unix()
	consent, err := app.ensureMemoryConsent(record.Profile, record.HouseID, now)
	if err != nil {
		return contract.Response{}, err
	}
	if consent.Paused {
		return memoryBlockedResponse(request, "memory_paused", "本地学习已暂停，未写入新记忆。"), nil
	}
	memoryRecord := storage.PreferenceRecord{
		ID:              record.ID,
		Profile:         record.Profile,
		HouseID:         record.HouseID,
		ScopeType:       planPayloadString(record.Payload, "scopeType"),
		ScopeRef:        planPayloadString(record.Payload, "scopeRef"),
		PreferenceType:  planPayloadString(record.Payload, "preferenceType"),
		PreferenceValue: planPayloadString(record.Payload, "preferenceValue"),
		Kind:            planPayloadString(record.Payload, "kind"),
		Evidence:        planPayloadString(record.Payload, "evidence"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := app.memoryStore.SavePreference(memoryRecord); err != nil {
		return contract.Response{}, err
	}
	if err := app.ensurePreferenceRecommendation(memoryRecord, now); err != nil {
		return contract.Response{}, err
	}
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return memoryRememberCommitResponse(request, record, memoryRecord), nil
}

func (app *app) invokeMemoryList(request contract.Request, profile string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	now := time.Now().Unix()
	if err := app.recoverCommittedMemoryPlans(profile, houseID, now); err != nil {
		return contract.Response{}, err
	}
	preferences, err := app.memoryStore.ListPreferences(profile, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	consent, err := app.ensureMemoryConsent(profile, houseID, now)
	if err != nil {
		return contract.Response{}, err
	}
	return memoryListResponse(request, houseID, consent, preferences), nil
}

func (app *app) invokeMemoryPauseResume(request contract.Request, profile string, houseID string, paused bool) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	consent, ok, err := app.memoryStore.Consent(profile, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	if !ok {
		consent = storage.ConsentRecord{Profile: profile, HouseID: houseID, ConsentVersion: memoryConsentVersion, LearningEnabled: true}
	}
	consent.LearningEnabled = true
	consent.Paused = paused
	consent.UpdatedAt = time.Now().Unix()
	if err := app.memoryStore.SetConsent(consent); err != nil {
		return contract.Response{}, err
	}
	return memoryPauseResumeResponse(request, consent), nil
}

func (app *app) invokeMemoryForget(request contract.Request, profile string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	exported, err := app.memoryStore.Export(profile, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.memoryStore.DeleteProfileHouse(profile, houseID); err != nil {
		return contract.Response{}, err
	}
	return memoryForgetResponse(request, houseID, exported), nil
}

func (app *app) invokeRecommendationList(request contract.Request, profile string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	now := time.Now().Unix()
	if err := app.recoverCommittedMemoryPlans(profile, houseID, now); err != nil {
		return contract.Response{}, err
	}
	if _, err := app.ensureMemoryConsent(profile, houseID, now); err != nil {
		return contract.Response{}, err
	}
	recommendations, err := app.memoryStore.ListRecommendations(profile, houseID, now, 1)
	if err != nil {
		return contract.Response{}, err
	}
	if len(recommendations) == 0 {
		if err := app.ensurePreferenceRecommendations(profile, houseID, now); err != nil {
			return contract.Response{}, err
		}
		recommendations, err = app.memoryStore.ListRecommendations(profile, houseID, now, 1)
		if err != nil {
			return contract.Response{}, err
		}
	}
	return recommendationListResponse(request, houseID, recommendations), nil
}

func (app *app) invokeRecommendationFeedback(request contract.Request, profile string, houseID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if houseID == "" {
		return memoryClarificationResponse(request, "missing_house_id"), nil
	}
	recommendationID := firstRequestString(request.Parameters, "recommendationId", "recommendationID", "id")
	if recommendationID == "" {
		return memoryClarificationResponse(request, "missing_recommendation_id"), nil
	}
	feedback := normalizeRecommendationFeedback(firstRequestString(request.Parameters, "feedback", "action", "status"))
	if feedback == "" {
		return memoryClarificationResponse(request, "missing_recommendation_feedback"), nil
	}
	now := time.Now().Unix()
	update := storage.RecommendationFeedback{Status: feedback, UpdatedAt: now}
	if feedback == "cooldown" {
		update.Status = "pending"
		update.CooldownUntil = now + recommendationCooldownSeconds(request)
	}
	record, err := app.memoryStore.ApplyRecommendationFeedback(profile, houseID, recommendationID, update)
	if err != nil {
		return recommendationFeedbackBlockedResponse(request, recommendationID, "recommendation_not_found", "未找到要更新的本地推荐。"), nil
	}
	return recommendationFeedbackResponse(request, houseID, record), nil
}

func (app *app) ensureMemoryConsent(profile string, houseID string, now int64) (storage.ConsentRecord, error) {
	consent, ok, err := app.memoryStore.Consent(profile, houseID)
	if err != nil {
		return storage.ConsentRecord{}, err
	}
	if !ok {
		consent = storage.ConsentRecord{
			Profile:         profile,
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
	case "dismiss", "dismissed", "ignore", "ignored", "稍后", "忽略":
		return "dismissed"
	case "reject", "rejected", "decline", "declined", "拒绝", "不再推荐":
		return "rejected"
	case "cooldown", "later", "remind_later", "稍后提醒":
		return "cooldown"
	default:
		return ""
	}
}

func recommendationCooldownSeconds(request contract.Request) int64 {
	value, ok := requestInteger(request.Parameters["cooldownHours"])
	if !ok || value < 1 || value > 24*30 {
		return int64(7 * 24 * 60 * 60)
	}
	return int64(value * 60 * 60)
}

func (app *app) ensurePreferenceRecommendations(profile string, houseID string, now int64) error {
	preferences, err := app.memoryStore.ListPreferences(profile, houseID)
	if err != nil {
		return err
	}
	for _, preference := range preferences {
		if err := app.ensurePreferenceRecommendation(preference, now); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) ensurePreferenceRecommendation(preference storage.PreferenceRecord, now int64) error {
	if strings.TrimSpace(preference.ID) == "" || strings.TrimSpace(preference.Profile) == "" || strings.TrimSpace(preference.HouseID) == "" {
		return nil
	}
	recommendationID := preferenceRecommendationID(preference.ID)
	if _, ok, err := app.memoryStore.Recommendation(preference.Profile, preference.HouseID, recommendationID); err != nil || ok {
		return err
	}
	return app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          recommendationID,
		Profile:     preference.Profile,
		HouseID:     preference.HouseID,
		Type:        "preference_based",
		Explanation: preferenceRecommendationExplanation(preference),
		Evidence:    preferenceRecommendationEvidence(preference),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func preferenceRecommendationID(preferenceID string) string {
	return "pref-" + strings.TrimSpace(preferenceID)
}

func preferenceRecommendationExplanation(preference storage.PreferenceRecord) string {
	scope := strings.TrimSpace(preference.ScopeRef)
	if scope == "" {
		scope = strings.TrimSpace(preference.ScopeType)
	}
	if scope == "" {
		scope = "当前家庭"
	}
	return fmt.Sprintf("可以按你保存的偏好调整 %s 的默认建议：%s=%s。", scope, preference.PreferenceType, preference.PreferenceValue)
}

func preferenceRecommendationEvidence(preference storage.PreferenceRecord) string {
	evidence := strings.TrimSpace(preference.Evidence)
	if evidence == "" {
		evidence = "本地已确认偏好"
	}
	return fmt.Sprintf("来源：%s；scope=%s/%s；kind=%s", evidence, preference.ScopeType, preference.ScopeRef, preference.Kind)
}
