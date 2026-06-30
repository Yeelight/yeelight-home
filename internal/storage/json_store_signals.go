package storage

import (
	"errors"
	"strings"
)

type InteractionSignalRecord struct {
	ID          string `json:"id"`
	Profile     string `json:"profile"`
	Region      string `json:"region,omitempty"`
	HouseID     string `json:"houseId"`
	SignalType  string `json:"signalType"`
	SignalKey   string `json:"signalKey"`
	Evidence    string `json:"evidence,omitempty"`
	Count       int    `json:"count"`
	FirstSeenAt int64  `json:"firstSeenAt"`
	LastSeenAt  int64  `json:"lastSeenAt"`
}

func (store JSONStore) SaveInteractionSignal(record InteractionSignalRecord) (InteractionSignalRecord, error) {
	if containsSensitiveKey(record.SignalType) || containsSensitiveKey(record.SignalKey) || containsSensitiveKey(record.Evidence) {
		return InteractionSignalRecord{}, errors.New("interaction signal must not contain token-like data")
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Profile) == "" || strings.TrimSpace(record.HouseID) == "" {
		return InteractionSignalRecord{}, errors.New("interaction signal id, profile and houseId are required")
	}
	record.Region = normalizeStorageRegion(record.Region)
	if record.Count <= 0 {
		record.Count = 1
	}
	document, err := store.loadScope(record.Profile, record.Region, record.HouseID)
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
		if err := store.saveScope(record.Profile, record.Region, record.HouseID, document); err != nil {
			return InteractionSignalRecord{}, err
		}
		return record, nil
	}
	document.Signals = append(document.Signals, record)
	if err := store.saveScope(record.Profile, record.Region, record.HouseID, document); err != nil {
		return InteractionSignalRecord{}, err
	}
	return record, nil
}

func (store JSONStore) ListInteractionSignals(profile string, region string, houseID string) ([]InteractionSignalRecord, error) {
	region = normalizeStorageRegion(region)
	document, err := store.loadScope(profile, region, houseID)
	if err != nil {
		return nil, err
	}
	result := []InteractionSignalRecord{}
	for _, record := range document.Signals {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			result = append(result, record)
		}
	}
	return result, nil
}
