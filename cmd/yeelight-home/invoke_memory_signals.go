package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func (app *app) observeMemorySignal(request contract.Request, profile string, region string, houseID string, response contract.Response) error {
	if !shouldObserveMemorySignal(request, houseID, response) {
		return nil
	}
	now := time.Now().Unix()
	consent, err := app.ensureMemoryConsent(profile, region, houseID, now)
	if err != nil {
		return err
	}
	if !memoryConsentActive(consent) {
		return nil
	}
	signal, ok := interactionSignalFromRequest(request, response, profile, region, houseID, now)
	if !ok {
		return nil
	}
	saved, err := app.memoryStore.SaveInteractionSignal(signal)
	if err != nil {
		return err
	}
	_ = saved
	return nil
}

func shouldObserveMemorySignal(request contract.Request, houseID string, response contract.Response) bool {
	if strings.TrimSpace(houseID) == "" || response.Status == "auth_required" || response.Status == "blocked" {
		return false
	}
	switch request.Intent {
	case "memory.remember", "memory.list", "memory.pause", "memory.resume", "memory.forget", "recommendation.list", "recommendation.record", "recommendation.feedback", "operation.lesson.record", "operation.lesson.list":
		return false
	default:
		return true
	}
}

func memoryConsentActive(consent storage.ConsentRecord) bool {
	return consent.LearningEnabled && !consent.Paused
}

func interactionSignalFromRequest(request contract.Request, response contract.Response, profile string, region string, houseID string, now int64) (storage.InteractionSignalRecord, bool) {
	intent := strings.TrimSpace(request.Intent)
	if intent == "" {
		return storage.InteractionSignalRecord{}, false
	}
	signalType := "interaction"
	key := interactionSignalKey(request.Intent, signalType)
	return storage.InteractionSignalRecord{
		ID:          "sig-" + shortHash(profile+"|"+region+"|"+houseID+"|"+key),
		Profile:     profile,
		Region:      region,
		HouseID:     houseID,
		SignalType:  signalType,
		SignalKey:   key,
		Evidence:    interactionEvidence(request, response),
		Count:       1,
		FirstSeenAt: now,
		LastSeenAt:  now,
	}, true
}

func interactionSignalKey(intent string, signalType string) string {
	return strings.Join([]string{intent, signalType}, "|")
}

func interactionEvidence(request contract.Request, response contract.Response) string {
	intent := strings.TrimSpace(request.Intent)
	if intent == "" {
		intent = "unknown"
	}
	status := strings.TrimSpace(response.Status)
	if status == "" {
		status = "unknown"
	}
	return "intent=" + intent + "; status=" + status
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
