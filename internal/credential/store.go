package credential

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store interface {
	Save(record TokenRecord) error
	Load(profile string) (TokenRecord, bool, error)
	Delete(profile string) error
}

type TokenRecord struct {
	Profile      string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

type TokenMetadata struct {
	Profile             string `json:"profile"`
	AccessTokenPresent  bool   `json:"accessTokenPresent"`
	RefreshTokenPresent bool   `json:"refreshTokenPresent"`
	ExpiresAt           int64  `json:"expiresAt,omitempty"`
}

type ProfileMetadata struct {
	Profile     string `json:"profile"`
	Region      string `json:"region"`
	ClientID    string `json:"clientId,omitempty"`
	HouseID     string `json:"houseId,omitempty"`
	BizType     string `json:"bizType,omitempty"`
	QRDevice    string `json:"qrDevice,omitempty"`
	Language    string `json:"language,omitempty"`
	ControlMode string `json:"controlMode,omitempty"`
	GatewayIP   string `json:"gatewayIp,omitempty"`
	LANEndpoint string `json:"lanEndpoint,omitempty"`
}

func (record TokenRecord) Metadata() TokenMetadata {
	return TokenMetadata{
		Profile:             record.Profile,
		AccessTokenPresent:  strings.TrimSpace(record.AccessToken) != "",
		RefreshTokenPresent: strings.TrimSpace(record.RefreshToken) != "",
		ExpiresAt:           record.ExpiresAt,
	}
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]TokenRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		records: map[string]TokenRecord{},
	}
}

func (store *MemoryStore) Save(record TokenRecord) error {
	profile, err := normalizeProfile(record.Profile)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record.Profile = profile
	store.records[profile] = record
	return nil
}

func (store *MemoryStore) Load(profile string) (TokenRecord, bool, error) {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return TokenRecord{}, false, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.records[normalized]
	return record, ok, nil
}

func (store *MemoryStore) Delete(profile string) error {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.records, normalized)
	return nil
}

func normalizeProfile(profile string) (string, error) {
	normalized := strings.TrimSpace(profile)
	if normalized == "" {
		return "", errors.New("profile is required")
	}
	return normalized, nil
}

type FileMetadataStore struct {
	path string
}

type metadataDocument struct {
	Version       int               `json:"version"`
	ActiveProfile string            `json:"activeProfile,omitempty"`
	Profiles      []ProfileMetadata `json:"profiles"`
}

func NewFileMetadataStore(path string) FileMetadataStore {
	return FileMetadataStore{path: path}
}

func (store FileMetadataStore) Path() string {
	return store.path
}

func (store FileMetadataStore) Save(metadata ProfileMetadata) error {
	profile, err := normalizeProfile(metadata.Profile)
	if err != nil {
		return err
	}
	metadata.Profile = profile
	metadata.Region = strings.TrimSpace(metadata.Region)
	metadata.ClientID = strings.TrimSpace(metadata.ClientID)
	metadata.HouseID = strings.TrimSpace(metadata.HouseID)
	metadata.BizType = strings.TrimSpace(metadata.BizType)
	metadata.QRDevice = strings.TrimSpace(metadata.QRDevice)
	metadata.Language = strings.TrimSpace(metadata.Language)
	metadata.ControlMode = strings.TrimSpace(metadata.ControlMode)
	metadata.GatewayIP = strings.TrimSpace(metadata.GatewayIP)
	metadata.LANEndpoint = strings.TrimSpace(metadata.LANEndpoint)
	document, err := store.load()
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range document.Profiles {
		if existing.Profile == profile {
			document.Profiles[index] = metadata
			replaced = true
			break
		}
	}
	if !replaced {
		document.Profiles = append(document.Profiles, metadata)
	}
	return store.save(document)
}

func (store FileMetadataStore) Delete(profile string) error {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return err
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	filtered := document.Profiles[:0]
	for _, metadata := range document.Profiles {
		if metadata.Profile != normalized {
			filtered = append(filtered, metadata)
		}
	}
	document.Profiles = filtered
	if document.ActiveProfile == normalized {
		document.ActiveProfile = ""
	}
	return store.save(document)
}

func (store FileMetadataStore) Load(profile string) (ProfileMetadata, bool, error) {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return ProfileMetadata{}, false, err
	}
	document, err := store.load()
	if err != nil {
		return ProfileMetadata{}, false, err
	}
	for _, metadata := range document.Profiles {
		if metadata.Profile == normalized {
			return metadata, true, nil
		}
	}
	return ProfileMetadata{}, false, nil
}

func (store FileMetadataStore) List() ([]ProfileMetadata, error) {
	document, err := store.load()
	if err != nil {
		return nil, err
	}
	profiles := make([]ProfileMetadata, len(document.Profiles))
	copy(profiles, document.Profiles)
	return profiles, nil
}

func (store FileMetadataStore) ActiveProfile() (string, error) {
	document, err := store.load()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(document.ActiveProfile), nil
}

func (store FileMetadataStore) SetActiveProfile(profile string) error {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return err
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	document.ActiveProfile = normalized
	return store.save(document)
}

func (store FileMetadataStore) load() (metadataDocument, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return metadataDocument{Version: 1, Profiles: []ProfileMetadata{}}, nil
	}
	if err != nil {
		return metadataDocument{}, err
	}
	document := metadataDocument{}
	if err := json.Unmarshal(data, &document); err != nil {
		return metadataDocument{}, err
	}
	if document.Profiles == nil {
		document.Profiles = []ProfileMetadata{}
	}
	return document, nil
}

func (store FileMetadataStore) save(document metadataDocument) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	tempPath := store.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, store.path)
}
