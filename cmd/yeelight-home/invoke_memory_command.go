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
	preferenceType := firstRequestString(request.Parameters, "preferenceType", "type")
	preferenceValue := firstRequestString(request.Parameters, "preferenceValue", "value")
	if preferenceType == "" || preferenceValue == "" {
		return memoryClarificationResponse(request, "missing_preference"), nil
	}
	payload := map[string]any{
		"preferenceType":  preferenceType,
		"preferenceValue": preferenceValue,
		"scopeType":       firstNonEmptyString(firstRequestString(request.Parameters, "scopeType"), "home"),
		"scopeRef":        firstRequestString(request.Parameters, "scopeRef"),
		"kind":            firstNonEmptyString(firstRequestString(request.Parameters, "kind"), "explicit"),
		"evidence":        firstRequestString(request.Parameters, "evidence"),
	}
	record, err := plan.NewRecord(profile, region, houseID, "memory.remember", request.RequestID, fmt.Sprintf("记住偏好 %s=%s", preferenceType, preferenceValue), payload, []string{
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
	consent, ok, err := app.memoryStore.Consent(record.Profile, record.HouseID)
	if err != nil {
		return contract.Response{}, err
	}
	if !ok {
		consent = storage.ConsentRecord{
			Profile:         record.Profile,
			HouseID:         record.HouseID,
			ConsentVersion:  memoryConsentVersion,
			LearningEnabled: true,
			UpdatedAt:       now,
		}
		if err := app.memoryStore.SetConsent(consent); err != nil {
			return contract.Response{}, err
		}
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
	preferences, err := app.memoryStore.ListPreferences(profile, houseID)
	if err != nil {
		return contract.Response{}, err
	}
	consent, _, err := app.memoryStore.Consent(profile, houseID)
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
	recommendations, err := app.memoryStore.ListRecommendations(profile, houseID, time.Now().Unix(), 1)
	if err != nil {
		return contract.Response{}, err
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
