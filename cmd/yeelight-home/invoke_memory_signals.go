package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/storage"
)

const implicitSignalPromotionThreshold = 2

func (app *app) observeMemorySignal(request contract.Request, profile string, houseID string, response contract.Response) error {
	if !shouldObserveMemorySignal(request, houseID, response) {
		return nil
	}
	now := time.Now().Unix()
	consent, err := app.ensureMemoryConsent(profile, houseID, now)
	if err != nil {
		return err
	}
	if !memoryConsentActive(consent) {
		return nil
	}
	signal, ok := interactionSignalFromRequest(request, profile, houseID, now)
	if !ok {
		return nil
	}
	saved, err := app.memoryStore.SaveInteractionSignal(signal)
	if err != nil {
		return err
	}
	if saved.SignalType != "preference_hint" || saved.Count < implicitSignalPromotionThreshold {
		return nil
	}
	return app.ensureImplicitSignalRecommendation(saved, now)
}

func shouldObserveMemorySignal(request contract.Request, houseID string, response contract.Response) bool {
	if strings.TrimSpace(houseID) == "" || response.Status == "auth_required" || response.Status == "blocked" {
		return false
	}
	switch request.Intent {
	case "memory.remember", "memory.list", "memory.pause", "memory.resume", "memory.forget", "recommendation.list", "recommendation.feedback":
		return false
	default:
		return true
	}
}

func memoryConsentActive(consent storage.ConsentRecord) bool {
	return consent.LearningEnabled && !consent.Paused
}

func interactionSignalFromRequest(request contract.Request, profile string, houseID string, now int64) (storage.InteractionSignalRecord, bool) {
	utterance := strings.TrimSpace(request.Utterance)
	if utterance == "" {
		return storage.InteractionSignalRecord{}, false
	}
	candidate := memoryPreferenceFromUtterance(request, memoryPreferenceCandidate{
		scopeType: firstNonEmptyString(firstRequestString(request.Parameters, "scopeType"), "home"),
		scopeRef:  firstRequestString(request.Parameters, "scopeRef"),
		kind:      "implicit_candidate",
	})
	if implicitSignalAllowsExtraction(utterance) {
		candidate = enrichImplicitSignalCandidate(utterance, candidate)
	}
	signalType := "interaction"
	if candidate.preferenceType != "" && candidate.preferenceValue != "" && implicitSignalAllowsExtraction(utterance) {
		signalType = "preference_hint"
	}
	if candidate.scopeRef == "" {
		candidate.scopeRef = inferMemoryScopeRef(utterance)
	}
	if candidate.scopeType == "" || candidate.scopeType == "home" {
		candidate.scopeType = inferMemoryScopeType(candidate.scopeRef)
	}
	key := interactionSignalKey(request.Intent, signalType, candidate)
	return storage.InteractionSignalRecord{
		ID:              "sig-" + shortHash(profile+"|"+houseID+"|"+key),
		Profile:         profile,
		HouseID:         houseID,
		SignalType:      signalType,
		SignalKey:       key,
		ScopeType:       candidate.scopeType,
		ScopeRef:        candidate.scopeRef,
		PreferenceType:  candidate.preferenceType,
		PreferenceValue: candidate.preferenceValue,
		Evidence:        interactionEvidence(utterance),
		Count:           1,
		FirstSeenAt:     now,
		LastSeenAt:      now,
	}, true
}

func enrichImplicitSignalCandidate(utterance string, candidate memoryPreferenceCandidate) memoryPreferenceCandidate {
	if candidate.preferenceType == "" {
		candidate.preferenceType = inferMemoryPreferenceType(utterance)
	}
	if candidate.scopeRef == "" {
		candidate.scopeRef = inferMemoryScopeRef(utterance)
	}
	if candidate.scopeType == "" || candidate.scopeType == "home" {
		candidate.scopeType = inferMemoryScopeType(candidate.scopeRef)
	}
	if candidate.preferenceValue == "" {
		candidate.preferenceValue = normalizeSignalPreferenceValue(utterance)
	}
	if candidate.evidence == "" {
		candidate.evidence = interactionEvidence(utterance)
	}
	return candidate
}

func implicitSignalAllowsExtraction(value string) bool {
	if memoryUtteranceAllowsExtraction(value) {
		return true
	}
	for _, marker := range []string{"太亮", "太暗", "调暗", "亮一点", "暗一点", "暖一点", "冷一点", "暖光", "冷光", "不要彩光", "不喜欢彩色", "柔和"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func interactionSignalKey(intent string, signalType string, candidate memoryPreferenceCandidate) string {
	if signalType != "preference_hint" {
		return strings.Join([]string{intent, signalType}, "|")
	}
	return strings.Join([]string{
		intent,
		signalType,
		candidate.scopeType,
		candidate.scopeRef,
		candidate.preferenceType,
		normalizeSignalPreferenceValue(candidate.preferenceValue),
	}, "|")
}

func normalizeSignalPreferenceValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, marker := range []string{"太亮", "调暗", "暗一点"} {
		if strings.Contains(value, marker) {
			return "prefer_dimmer"
		}
	}
	for _, marker := range []string{"太暗", "亮一点"} {
		if strings.Contains(value, marker) {
			return "prefer_brighter"
		}
	}
	for _, marker := range []string{"暖一点", "暖光", "柔和暖光"} {
		if strings.Contains(value, marker) {
			return "prefer_warm"
		}
	}
	for _, marker := range []string{"冷一点", "冷光"} {
		if strings.Contains(value, marker) {
			return "prefer_cool"
		}
	}
	for _, marker := range []string{"不要彩光", "不喜欢彩色"} {
		if strings.Contains(value, marker) {
			return "avoid_colorful"
		}
	}
	return value
}

func interactionEvidence(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 80 {
		runes := []rune(value)
		value = string(runes[:80])
	}
	return "用户交互信号：" + value
}

func (app *app) ensureImplicitSignalRecommendation(signal storage.InteractionSignalRecord, now int64) error {
	if strings.TrimSpace(signal.PreferenceType) == "" || strings.TrimSpace(signal.PreferenceValue) == "" {
		return nil
	}
	recommendationID := "sigrec-" + strings.TrimPrefix(signal.ID, "sig-")
	if _, ok, err := app.memoryStore.Recommendation(signal.Profile, signal.HouseID, recommendationID); err != nil || ok {
		return err
	}
	scope := firstNonEmptyString(signal.ScopeRef, signal.ScopeType, "当前家庭")
	return app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          recommendationID,
		Profile:     signal.Profile,
		HouseID:     signal.HouseID,
		Type:        "implicit_candidate",
		Explanation: fmt.Sprintf("多次观察到你可能希望 %s 的建议更贴近：%s=%s。", scope, signal.PreferenceType, signal.PreferenceValue),
		Evidence:    fmt.Sprintf("%s；count=%d；signal=%s", signal.Evidence, signal.Count, signal.SignalType),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
