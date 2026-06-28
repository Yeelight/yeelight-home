package storage

import (
	"errors"
	"strings"
)

type InteractionSignalRecord struct {
	ID              string `json:"id"`
	Profile         string `json:"profile"`
	HouseID         string `json:"houseId"`
	SignalType      string `json:"signalType"`
	SignalKey       string `json:"signalKey"`
	ScopeType       string `json:"scopeType,omitempty"`
	ScopeRef        string `json:"scopeRef,omitempty"`
	PreferenceType  string `json:"preferenceType,omitempty"`
	PreferenceValue string `json:"preferenceValue,omitempty"`
	Evidence        string `json:"evidence,omitempty"`
	Count           int    `json:"count"`
	FirstSeenAt     int64  `json:"firstSeenAt"`
	LastSeenAt      int64  `json:"lastSeenAt"`
}

func (store JSONStore) SaveInteractionSignal(record InteractionSignalRecord) (InteractionSignalRecord, error) {
	if containsSensitiveKey(record.SignalType) || containsSensitiveKey(record.SignalKey) || containsSensitiveKey(record.Evidence) {
		return InteractionSignalRecord{}, errors.New("interaction signal must not contain token-like data")
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return InteractionSignalRecord{}, errors.New("interaction signal id, profile and houseId are required")
	}
	if record.Count <= 0 {
		record.Count = 1
	}
	document, err := store.load()
	if err != nil {
		return InteractionSignalRecord{}, err
	}
	for index, existing := range document.Signals {
		if existing.ID != record.ID {
			continue
		}
		if record.FirstSeenAt == 0 {
			record.FirstSeenAt = existing.FirstSeenAt
		}
		if record.Count <= existing.Count {
			record.Count = existing.Count + 1
		}
		document.Signals[index] = record
		if err := store.save(document); err != nil {
			return InteractionSignalRecord{}, err
		}
		return record, nil
	}
	document.Signals = append(document.Signals, record)
	if err := store.save(document); err != nil {
		return InteractionSignalRecord{}, err
	}
	return record, nil
}

func (store JSONStore) ListInteractionSignals(profile string, houseID string) ([]InteractionSignalRecord, error) {
	document, err := store.load()
	if err != nil {
		return nil, err
	}
	result := []InteractionSignalRecord{}
	for _, record := range document.Signals {
		if record.Profile == profile && record.HouseID == houseID {
			result = append(result, record)
		}
	}
	return result, nil
}

func filterSignals(records []InteractionSignalRecord, profile string, houseID string) []InteractionSignalRecord {
	result := []InteractionSignalRecord{}
	for _, record := range records {
		if record.Profile == profile && record.HouseID == houseID {
			continue
		}
		result = append(result, record)
	}
	return result
}
