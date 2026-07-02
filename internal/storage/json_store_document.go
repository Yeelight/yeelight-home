package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var jsonDocumentLocks sync.Map

func jsonDocumentLock(path string) *sync.Mutex {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	value, _ := jsonDocumentLocks.LoadOrStore(absolute, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (store JSONStore) load() (jsonDocument, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyDocument(), nil
	}
	if err != nil {
		return jsonDocument{}, err
	}
	document := jsonDocument{}
	if err := json.Unmarshal(data, &document); err != nil {
		return jsonDocument{}, err
	}
	if document.Preferences == nil {
		document.Preferences = []PreferenceRecord{}
	}
	if document.Consents == nil {
		document.Consents = []ConsentRecord{}
	}
	if document.Recommendations == nil {
		document.Recommendations = []RecommendationRecord{}
	}
	if document.Signals == nil {
		document.Signals = []InteractionSignalRecord{}
	}
	if document.Lessons == nil {
		document.Lessons = []OperationLessonRecord{}
	}
	return document, nil
}

func (store JSONStore) loadScope(profile string, region string, houseID string) (jsonDocument, error) {
	region = normalizeStorageRegion(region)
	path := store.scopePath(profile, region, houseID)
	document, err := readJSONDocument(path)
	if err == nil {
		return document, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return jsonDocument{}, err
	}
	scoped, err := store.loadLegacyScope(profile, region, houseID)
	if err != nil {
		return jsonDocument{}, err
	}
	if scoped.isEmpty() {
		return scoped, nil
	}
	if err := store.saveScope(profile, region, houseID, scoped); err != nil {
		return jsonDocument{}, err
	}
	return scoped, nil
}

func (store JSONStore) mutateScope(profile string, region string, houseID string, mutate func(*jsonDocument) error) error {
	region = normalizeStorageRegion(region)
	path := store.scopePath(profile, region, houseID)
	lock := jsonDocumentLock(path)
	lock.Lock()
	defer lock.Unlock()
	document, err := store.loadScopeForMutation(profile, region, houseID)
	if err != nil {
		return err
	}
	if err := mutate(&document); err != nil {
		return err
	}
	document = normalizeDocument(document)
	document.Namespace = storageNamespace(profile, region, houseID, "memory")
	document = compactScopedDocument(document, 0)
	return writeJSONDocument(path, document)
}

func (store JSONStore) loadScopeForMutation(profile string, region string, houseID string) (jsonDocument, error) {
	path := store.scopePath(profile, region, houseID)
	document, err := readJSONDocument(path)
	if err == nil {
		return document, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return jsonDocument{}, err
	}
	return store.loadLegacyScope(profile, region, houseID)
}

func (store JSONStore) saveScope(profile string, region string, houseID string, document jsonDocument) error {
	region = normalizeStorageRegion(region)
	document = normalizeDocument(document)
	document.Namespace = storageNamespace(profile, region, houseID, "memory")
	document = compactScopedDocument(document, 0)
	path := store.scopePath(profile, region, houseID)
	lock := jsonDocumentLock(path)
	lock.Lock()
	defer lock.Unlock()
	return writeJSONDocument(path, document)
}

func (store JSONStore) scopePath(profile string, region string, houseID string) string {
	return filepath.Join(
		filepath.Dir(store.path),
		strings.TrimSuffix(filepath.Base(store.path), filepath.Ext(store.path)),
		safeStorageSegment(profile),
		safeStorageSegment(normalizeStorageRegion(region)),
		safeStorageSegment(houseID)+".json",
	)
}

func (store JSONStore) legacyScopePath(profile string, houseID string) string {
	return filepath.Join(
		filepath.Dir(store.path),
		strings.TrimSuffix(filepath.Base(store.path), filepath.Ext(store.path)),
		safeStorageSegment(profile),
		safeStorageSegment(houseID)+".json",
	)
}

func (store JSONStore) loadLegacyScope(profile string, region string, houseID string) (jsonDocument, error) {
	legacyScoped, err := readJSONDocument(store.legacyScopePath(profile, houseID))
	if err == nil {
		return legacyScoped.scope(profile, region, houseID), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return jsonDocument{}, err
	}
	legacy, err := store.load()
	if err != nil {
		return jsonDocument{}, err
	}
	return legacy.scope(profile, region, houseID), nil
}

func (store JSONStore) save(document jsonDocument) error {
	lock := jsonDocumentLock(store.path)
	lock.Lock()
	defer lock.Unlock()
	return writeJSONDocument(store.path, normalizeDocument(document))
}

func readJSONDocument(path string) (jsonDocument, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return jsonDocument{}, err
	}
	if err != nil {
		return jsonDocument{}, err
	}
	document := jsonDocument{}
	if err := json.Unmarshal(data, &document); err != nil {
		return jsonDocument{}, err
	}
	return normalizeDocument(document), nil
}

func writeJSONDocument(path string, document jsonDocument) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	removeTemp = false
	return os.Rename(tempPath, path)
}

func emptyDocument() jsonDocument {
	return jsonDocument{
		Version:         1,
		Consents:        []ConsentRecord{},
		Preferences:     []PreferenceRecord{},
		Recommendations: []RecommendationRecord{},
		Signals:         []InteractionSignalRecord{},
		Lessons:         []OperationLessonRecord{},
	}
}

func normalizeDocument(document jsonDocument) jsonDocument {
	if document.Version == 0 {
		document.Version = 1
	}
	if document.Preferences == nil {
		document.Preferences = []PreferenceRecord{}
	}
	if document.Consents == nil {
		document.Consents = []ConsentRecord{}
	}
	if document.Recommendations == nil {
		document.Recommendations = []RecommendationRecord{}
	}
	if document.Signals == nil {
		document.Signals = []InteractionSignalRecord{}
	}
	if document.Lessons == nil {
		document.Lessons = []OperationLessonRecord{}
	}
	return document
}

func (document jsonDocument) scope(profile string, region string, houseID string) jsonDocument {
	region = normalizeStorageRegion(region)
	scoped := emptyDocument()
	scoped.Namespace = storageNamespace(profile, region, houseID, "memory")
	if document.Version != 0 {
		scoped.Version = document.Version
	}
	for _, record := range document.Consents {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			record.Region = region
			scoped.Consents = append(scoped.Consents, record)
		}
	}
	for _, record := range document.Preferences {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			record.Region = region
			scoped.Preferences = append(scoped.Preferences, record)
		}
	}
	for _, record := range document.Recommendations {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			record.Region = region
			scoped.Recommendations = append(scoped.Recommendations, record)
		}
	}
	for _, record := range document.Signals {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID {
			record.Region = region
			scoped.Signals = append(scoped.Signals, record)
		}
	}
	for _, record := range document.Lessons {
		if record.Profile == profile && sameStorageRegion(record.Region, region) && lessonScopeMatches(record.HouseID, houseID) {
			record.Region = region
			scoped.Lessons = append(scoped.Lessons, record)
		}
	}
	return scoped
}

func (document jsonDocument) isEmpty() bool {
	return len(document.Consents) == 0 &&
		len(document.Preferences) == 0 &&
		len(document.Recommendations) == 0 &&
		len(document.Signals) == 0 &&
		len(document.Lessons) == 0
}
