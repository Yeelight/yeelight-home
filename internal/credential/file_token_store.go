package credential

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type FileTokenStore struct {
	path string
}

type tokenDocument struct {
	Version int           `json:"version"`
	Tokens  []TokenRecord `json:"tokens"`
}

func NewFileTokenStore(path string) FileTokenStore {
	return FileTokenStore{path: path}
}

func (store FileTokenStore) Save(record TokenRecord) error {
	profile, err := normalizeProfile(record.Profile)
	if err != nil {
		return err
	}
	record.Profile = profile
	document, err := store.load()
	if err != nil {
		return err
	}
	replaced := false
	for index, existing := range document.Tokens {
		if existing.Profile == profile {
			document.Tokens[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		document.Tokens = append(document.Tokens, record)
	}
	return store.save(document)
}

func (store FileTokenStore) Load(profile string) (TokenRecord, bool, error) {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return TokenRecord{}, false, err
	}
	document, err := store.load()
	if err != nil {
		return TokenRecord{}, false, err
	}
	for _, record := range document.Tokens {
		if record.Profile == normalized {
			return record, true, nil
		}
	}
	return TokenRecord{}, false, nil
}

func (store FileTokenStore) Delete(profile string) error {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return err
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	filtered := document.Tokens[:0]
	for _, record := range document.Tokens {
		if record.Profile != normalized {
			filtered = append(filtered, record)
		}
	}
	document.Tokens = filtered
	return store.save(document)
}

func (store FileTokenStore) load() (tokenDocument, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return tokenDocument{Version: 1, Tokens: []TokenRecord{}}, nil
	}
	if err != nil {
		return tokenDocument{}, err
	}
	document := tokenDocument{}
	if err := json.Unmarshal(data, &document); err != nil {
		return tokenDocument{}, err
	}
	if document.Tokens == nil {
		document.Tokens = []TokenRecord{}
	}
	return document, nil
}

func (store FileTokenStore) save(document tokenDocument) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
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

type FallbackStore struct {
	primary  Store
	fallback Store
}

func NewFallbackStore(primary Store, fallback Store) FallbackStore {
	return FallbackStore{primary: primary, fallback: fallback}
}

func (store FallbackStore) Save(record TokenRecord) error {
	if err := store.primary.Save(record); err == nil {
		return nil
	}
	return store.fallback.Save(record)
}

func (store FallbackStore) Load(profile string) (TokenRecord, bool, error) {
	record, ok, err := store.primary.Load(profile)
	if err == nil && ok {
		return record, true, nil
	}
	if err != nil && store.fallback == nil {
		return TokenRecord{}, false, err
	}
	return store.fallback.Load(profile)
}

func (store FallbackStore) Delete(profile string) error {
	primaryErr := store.primary.Delete(profile)
	fallbackErr := store.fallback.Delete(profile)
	if fallbackErr != nil {
		return fallbackErr
	}
	return primaryErr
}
