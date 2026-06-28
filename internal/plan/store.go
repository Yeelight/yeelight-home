package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusCommitted = "committed"
	StatusCanceled  = "canceled"

	AccountScopeHouseID = "__account__"

	RiskR2 = "R2"
	RiskR3 = "R3"

	DefaultTerminalRetention       = 7 * 24 * time.Hour
	DefaultExpiredPendingRetention = 24 * time.Hour
	DefaultMaxStoredPlans          = 200
)

type Record struct {
	ID                string         `json:"id"`
	Hash              string         `json:"hash"`
	Profile           string         `json:"profile"`
	Region            string         `json:"region"`
	HouseID           string         `json:"houseId"`
	Intent            string         `json:"intent"`
	Status            string         `json:"status"`
	Risk              string         `json:"risk"`
	ApprovalRequired  bool           `json:"approvalRequired,omitempty"`
	ApprovalChallenge string         `json:"approvalChallenge,omitempty"`
	SourceRequestID   string         `json:"sourceRequestId"`
	Summary           string         `json:"summary"`
	Payload           map[string]any `json:"payload"`
	Preconditions     []string       `json:"preconditions"`
	CreatedAt         int64          `json:"createdAt"`
	ExpiresAt         int64          `json:"expiresAt"`
	ApprovedAt        int64          `json:"approvedAt,omitempty"`
	CommittedAt       int64          `json:"committedAt,omitempty"`
	CanceledAt        int64          `json:"canceledAt,omitempty"`
}

type Store struct {
	path                    string
	now                     func() time.Time
	terminalRetention       time.Duration
	expiredPendingRetention time.Duration
	maxStoredPlans          int
	beforeCompact           func([]Record) error
}

type document struct {
	Version int      `json:"version"`
	Plans   []Record `json:"plans"`
}

func NewStore(path string) Store {
	return Store{
		path:                    path,
		now:                     time.Now,
		terminalRetention:       DefaultTerminalRetention,
		expiredPendingRetention: DefaultExpiredPendingRetention,
		maxStoredPlans:          DefaultMaxStoredPlans,
	}
}

func (store Store) WithClock(now func() time.Time) Store {
	store.now = now
	return store
}

func (store Store) WithRetention(terminalRetention time.Duration, expiredPendingRetention time.Duration, maxStoredPlans int) Store {
	store.terminalRetention = terminalRetention
	store.expiredPendingRetention = expiredPendingRetention
	store.maxStoredPlans = maxStoredPlans
	return store
}

func (store Store) WithBeforeCompact(hook func([]Record) error) Store {
	store.beforeCompact = hook
	return store
}

func NewRecord(profile string, region string, houseID string, intent string, sourceRequestID string, summary string, payload map[string]any, preconditions []string, now time.Time, ttl time.Duration) (Record, error) {
	return NewRecordWithRisk(profile, region, houseID, intent, sourceRequestID, summary, RiskR2, "", payload, preconditions, now, ttl)
}

func NewAccountRecord(profile string, region string, intent string, sourceRequestID string, summary string, payload map[string]any, preconditions []string, now time.Time, ttl time.Duration) (Record, error) {
	return NewRecord(profile, region, AccountScopeHouseID, intent, sourceRequestID, summary, payload, preconditions, now, ttl)
}

func IsAccountScope(houseID string) bool {
	return strings.TrimSpace(houseID) == AccountScopeHouseID
}

func NewRecordWithRisk(profile string, region string, houseID string, intent string, sourceRequestID string, summary string, risk string, approvalChallenge string, payload map[string]any, preconditions []string, now time.Time, ttl time.Duration) (Record, error) {
	if strings.TrimSpace(profile) == "" {
		return Record{}, errors.New("profile is required")
	}
	if strings.TrimSpace(region) == "" {
		return Record{}, errors.New("region is required")
	}
	if strings.TrimSpace(houseID) == "" {
		return Record{}, errors.New("house id is required")
	}
	if strings.TrimSpace(intent) == "" {
		return Record{}, errors.New("intent is required")
	}
	risk = strings.TrimSpace(risk)
	if risk == "" {
		risk = RiskR2
	}
	if risk != RiskR2 && risk != RiskR3 {
		return Record{}, fmt.Errorf("unsupported plan risk %q", risk)
	}
	approvalChallenge = strings.TrimSpace(approvalChallenge)
	approvalRequired := risk == RiskR3
	if approvalRequired && approvalChallenge == "" {
		return Record{}, errors.New("approval challenge is required for R3 plan")
	}
	if containsSensitive(payload) {
		return Record{}, errors.New("plan payload must not contain token-like data")
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	id, err := randomID()
	if err != nil {
		return Record{}, err
	}
	record := Record{
		ID:                id,
		Profile:           strings.TrimSpace(profile),
		Region:            strings.TrimSpace(region),
		HouseID:           strings.TrimSpace(houseID),
		Intent:            strings.TrimSpace(intent),
		Status:            StatusPending,
		Risk:              risk,
		ApprovalRequired:  approvalRequired,
		ApprovalChallenge: approvalChallenge,
		SourceRequestID:   strings.TrimSpace(sourceRequestID),
		Summary:           strings.TrimSpace(summary),
		Payload:           payload,
		Preconditions:     compactStrings(preconditions),
		CreatedAt:         now.Unix(),
		ExpiresAt:         now.Add(ttl).Unix(),
	}
	record.Hash = ComputeHash(record)
	return record, nil
}

func (store Store) Save(record Record) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("plan id is required")
	}
	if strings.TrimSpace(record.Hash) == "" {
		return errors.New("plan hash is required")
	}
	if containsSensitive(record.Payload) {
		return errors.New("plan payload must not contain token-like data")
	}
	doc, err := store.load()
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range doc.Plans {
		if existing.ID == record.ID {
			doc.Plans[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		doc.Plans = append(doc.Plans, record)
	}
	if err := store.runBeforeCompact(doc.Plans); err != nil {
		return err
	}
	doc.Plans = store.compact(doc.Plans)
	return store.save(doc)
}

func (store Store) Load(id string) (Record, bool, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return Record{}, false, errors.New("plan id is required")
	}
	doc, err := store.load()
	if err != nil {
		return Record{}, false, err
	}
	for index := len(doc.Plans) - 1; index >= 0; index-- {
		if doc.Plans[index].ID == trimmed {
			return doc.Plans[index], true, nil
		}
	}
	return Record{}, false, nil
}

func (store Store) List() ([]Record, error) {
	doc, err := store.load()
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(doc.Plans))
	copy(records, doc.Plans)
	return records, nil
}

func (store Store) MarkCommitted(id string) (Record, error) {
	record, ok, err := store.Load(id)
	if err != nil {
		return Record{}, err
	}
	if !ok {
		return Record{}, errors.New("plan not found")
	}
	record.Status = StatusCommitted
	record.CommittedAt = store.clock().Unix()
	if err := store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (store Store) MarkCanceled(id string) (Record, error) {
	record, ok, err := store.Load(id)
	if err != nil {
		return Record{}, err
	}
	if !ok {
		return Record{}, errors.New("plan not found")
	}
	record.Status = StatusCanceled
	record.CanceledAt = store.clock().Unix()
	if err := store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (store Store) MarkApproved(id string, challenge string) (Record, error) {
	record, ok, err := store.Load(id)
	if err != nil {
		return Record{}, err
	}
	if !ok {
		return Record{}, errors.New("plan not found")
	}
	if err := record.Verify(store.clock()); err != nil {
		return Record{}, err
	}
	if !record.ApprovalRequired || record.Risk != RiskR3 {
		return Record{}, errors.New("plan does not require local approval")
	}
	if strings.TrimSpace(challenge) != record.ApprovalChallenge {
		return Record{}, errors.New("approval challenge mismatch")
	}
	record.ApprovedAt = store.clock().Unix()
	if err := store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (record Record) Verify(now time.Time) error {
	if record.Status != StatusPending {
		return fmt.Errorf("plan status is %s", record.Status)
	}
	if record.ExpiresAt > 0 && now.Unix() > record.ExpiresAt {
		return errors.New("plan expired")
	}
	if record.Hash != ComputeHash(record) {
		return errors.New("plan hash mismatch")
	}
	if containsSensitive(record.Payload) {
		return errors.New("plan payload must not contain token-like data")
	}
	return nil
}

func ComputeHash(record Record) string {
	canonical := struct {
		Profile           string         `json:"profile"`
		Region            string         `json:"region"`
		HouseID           string         `json:"houseId"`
		Intent            string         `json:"intent"`
		Risk              string         `json:"risk"`
		ApprovalRequired  bool           `json:"approvalRequired,omitempty"`
		ApprovalChallenge string         `json:"approvalChallenge,omitempty"`
		SourceRequestID   string         `json:"sourceRequestId"`
		Summary           string         `json:"summary"`
		Payload           map[string]any `json:"payload"`
		Preconditions     []string       `json:"preconditions"`
		CreatedAt         int64          `json:"createdAt"`
		ExpiresAt         int64          `json:"expiresAt"`
	}{
		Profile:           record.Profile,
		Region:            record.Region,
		HouseID:           record.HouseID,
		Intent:            record.Intent,
		Risk:              record.Risk,
		ApprovalRequired:  record.ApprovalRequired,
		ApprovalChallenge: record.ApprovalChallenge,
		SourceRequestID:   record.SourceRequestID,
		Summary:           record.Summary,
		Payload:           record.Payload,
		Preconditions:     record.Preconditions,
		CreatedAt:         record.CreatedAt,
		ExpiresAt:         record.ExpiresAt,
	}
	data, _ := json.Marshal(canonical)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (store Store) load() (document, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return document{Version: 1, Plans: []Record{}}, nil
	}
	if err != nil {
		return document{}, err
	}
	doc := document{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return document{}, err
	}
	if doc.Plans == nil {
		doc.Plans = []Record{}
	}
	return doc, nil
}

func (store Store) save(doc document) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tempPath := store.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, store.path)
}

func (store Store) clock() time.Time {
	if store.now == nil {
		return time.Now()
	}
	return store.now()
}
