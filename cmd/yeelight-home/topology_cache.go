package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/storage"
)

const topologyCacheTTL = 5 * time.Minute

type topologyCache struct {
	legacyPath string
	rootDir    string
}

type topologyCacheDocument struct {
	Entries map[string]topologyCacheEntry `json:"entries"`
}

type topologyCacheEntry struct {
	Namespace storage.StorageNamespace `json:"namespace,omitempty"`
	UpdatedAt int64                    `json:"updatedAt"`
	Result    api.EntityListResult     `json:"result"`
}

func newTopologyCache(path string) topologyCache {
	if strings.TrimSpace(path) == "" {
		return topologyCache{}
	}
	return topologyCache{
		legacyPath: path,
		rootDir:    filepath.Join(filepath.Dir(path), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
	}
}

func (cache topologyCache) Load(profile string, region string, houseID string, now time.Time) (api.EntityListResult, bool) {
	if strings.TrimSpace(cache.rootDir) == "" || strings.TrimSpace(houseID) == "" {
		return api.EntityListResult{}, false
	}
	entry, ok := cache.loadEntry(profile, region, houseID)
	if !ok {
		return api.EntityListResult{}, false
	}
	if entry.Result.Total == 0 {
		return api.EntityListResult{}, false
	}
	if now.Unix()-entry.UpdatedAt > int64(topologyCacheTTL.Seconds()) {
		return api.EntityListResult{}, false
	}
	result := entry.Result
	result.APICalls = 0
	result.Warnings = appendWarning(result.Warnings, "topology_cache_hit")
	return result, true
}

func (cache topologyCache) Save(profile string, region string, houseID string, result api.EntityListResult, now time.Time) error {
	if strings.TrimSpace(cache.rootDir) == "" || strings.TrimSpace(houseID) == "" || result.Total == 0 {
		return nil
	}
	return cache.saveEntry(profile, region, houseID, topologyCacheEntry{
		UpdatedAt: now.Unix(),
		Result:    result,
	})
}

func (cache topologyCache) Invalidate(profile string, region string, houseID string) error {
	if strings.TrimSpace(cache.rootDir) == "" || strings.TrimSpace(houseID) == "" {
		return nil
	}
	return cache.saveEntry(profile, region, houseID, topologyCacheEntry{})
}

func (cache topologyCache) loadEntry(profile string, region string, houseID string) (topologyCacheEntry, bool) {
	entry, err := cache.readEntry(cache.entryPath(profile, region, houseID))
	if err == nil {
		return entry, true
	}
	if !os.IsNotExist(err) {
		return topologyCacheEntry{}, false
	}
	entry, ok := cache.loadLegacyEntry(profile, region, houseID)
	if !ok {
		return topologyCacheEntry{}, false
	}
	if entry.Result.Total > 0 {
		_ = cache.saveEntry(profile, region, houseID, entry)
	}
	return entry, true
}

func (cache topologyCache) readEntry(path string) (topologyCacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return topologyCacheEntry{}, err
	}
	if len(data) == 0 {
		return topologyCacheEntry{}, os.ErrNotExist
	}
	var entry topologyCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return topologyCacheEntry{}, err
	}
	return entry, nil
}

func (cache topologyCache) saveEntry(profile string, region string, houseID string, entry topologyCacheEntry) error {
	path := cache.entryPath(profile, region, houseID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	entry.Namespace = storage.StorageNamespace{
		AccountProfile: strings.TrimSpace(profile),
		Profile:        strings.TrimSpace(profile),
		Region:         strings.ToLower(strings.TrimSpace(region)),
		HouseID:        strings.TrimSpace(houseID),
		DataType:       "topology",
	}
	if entry.Namespace.Region == "" {
		entry.Namespace.Region = "default"
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (cache topologyCache) entryPath(profile string, region string, houseID string) string {
	return filepath.Join(
		cache.rootDir,
		safeCacheSegment(profile),
		safeCacheSegment(region),
		safeCacheSegment(houseID)+".json",
	)
}

func (cache topologyCache) loadLegacyEntry(profile string, region string, houseID string) (topologyCacheEntry, bool) {
	document, err := cache.loadLegacy()
	if err != nil {
		return topologyCacheEntry{}, false
	}
	entry, ok := document.Entries[topologyCacheKey(profile, region, houseID)]
	return entry, ok
}

func (cache topologyCache) loadLegacy() (topologyCacheDocument, error) {
	data, err := os.ReadFile(cache.legacyPath)
	if os.IsNotExist(err) {
		return topologyCacheDocument{Entries: map[string]topologyCacheEntry{}}, nil
	}
	if err != nil {
		return topologyCacheDocument{}, err
	}
	if len(data) == 0 {
		return topologyCacheDocument{Entries: map[string]topologyCacheEntry{}}, nil
	}
	var document topologyCacheDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return topologyCacheDocument{Entries: map[string]topologyCacheEntry{}}, nil
	}
	if document.Entries == nil {
		document.Entries = map[string]topologyCacheEntry{}
	}
	return document, nil
}

func safeCacheSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	segment := strings.Trim(builder.String(), ".")
	if segment == "" {
		return "default"
	}
	if len(segment) > 80 {
		return segment[:80]
	}
	return segment
}

func topologyCacheKey(profile string, region string, houseID string) string {
	return strings.TrimSpace(profile) + "|" + strings.TrimSpace(region) + "|" + strings.TrimSpace(houseID)
}
