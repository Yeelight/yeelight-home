package operation

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	AccountScopeHouseID = "__account__"

	RiskR2 = "R2"
	RiskR3 = "R3"
)

type Prepared struct {
	Profile         string         `json:"profile"`
	Region          string         `json:"region"`
	HouseID         string         `json:"houseId"`
	Intent          string         `json:"intent"`
	Risk            string         `json:"risk"`
	SourceRequestID string         `json:"sourceRequestId"`
	Summary         string         `json:"summary"`
	Payload         map[string]any `json:"payload"`
	Preconditions   []string       `json:"preconditions"`
	CreatedAt       int64          `json:"createdAt"`
}

func NewPrepared(profile string, region string, houseID string, intent string, sourceRequestID string, summary string, payload map[string]any, preconditions []string, now time.Time) (Prepared, error) {
	return NewPreparedWithRisk(profile, region, houseID, intent, sourceRequestID, summary, RiskR2, payload, preconditions, now)
}

func NewAccountPrepared(profile string, region string, intent string, sourceRequestID string, summary string, payload map[string]any, preconditions []string, now time.Time) (Prepared, error) {
	return NewPrepared(profile, region, AccountScopeHouseID, intent, sourceRequestID, summary, payload, preconditions, now)
}

func IsAccountScope(houseID string) bool {
	return strings.TrimSpace(houseID) == AccountScopeHouseID
}

func NewPreparedWithRisk(profile string, region string, houseID string, intent string, sourceRequestID string, summary string, risk string, payload map[string]any, preconditions []string, now time.Time) (Prepared, error) {
	if strings.TrimSpace(profile) == "" {
		return Prepared{}, errors.New("profile is required")
	}
	if strings.TrimSpace(region) == "" {
		return Prepared{}, errors.New("region is required")
	}
	if strings.TrimSpace(houseID) == "" {
		return Prepared{}, errors.New("house id is required")
	}
	if strings.TrimSpace(intent) == "" {
		return Prepared{}, errors.New("intent is required")
	}
	risk = strings.TrimSpace(risk)
	if risk == "" {
		risk = RiskR2
	}
	if risk != RiskR2 && risk != RiskR3 {
		return Prepared{}, fmt.Errorf("unsupported operation risk %q", risk)
	}
	if containsSensitive(payload) {
		return Prepared{}, errors.New("operation payload must not contain token-like data")
	}
	return Prepared{
		Profile:         strings.TrimSpace(profile),
		Region:          strings.TrimSpace(region),
		HouseID:         strings.TrimSpace(houseID),
		Intent:          strings.TrimSpace(intent),
		Risk:            risk,
		SourceRequestID: strings.TrimSpace(sourceRequestID),
		Summary:         strings.TrimSpace(summary),
		Payload:         payload,
		Preconditions:   compactStrings(preconditions),
		CreatedAt:       now.Unix(),
	}, nil
}

func (prepared Prepared) Verify(time.Time) error {
	if containsSensitive(prepared.Payload) {
		return errors.New("operation payload must not contain token-like data")
	}
	return nil
}
