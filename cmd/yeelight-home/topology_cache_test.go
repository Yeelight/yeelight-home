package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
)

func TestTopologyCacheStoresShardByProfileRegionAndHouse(t *testing.T) {
	root := t.TempDir()
	legacyPath := filepath.Join(root, "topology.json")
	cache := newTopologyCache(legacyPath)
	now := time.Unix(1000, 0)
	result := api.EntityListResult{
		Region:   "cn",
		HouseID:  "200171",
		Total:    1,
		Counts:   map[string]int{"room": 1},
		Entities: []api.EntitySummary{{Type: "room", ID: "room-1", Name: "客厅"}},
		APICalls: 6,
	}

	if err := cache.Save("family/main", "cn", "200171", result, now); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy topology file should not be written, stat err=%v", err)
	}
	shardPath := filepath.Join(root, "topology", "family_main", "cn", "200171.json")
	if _, err := os.Stat(shardPath); err != nil {
		t.Fatalf("expected shard file: %v", err)
	}
	shardData, err := os.ReadFile(shardPath)
	if err != nil {
		t.Fatalf("ReadFile shard error: %v", err)
	}
	var shard topologyCacheEntry
	if err := json.Unmarshal(shardData, &shard); err != nil {
		t.Fatalf("Unmarshal shard error: %v", err)
	}
	if shard.Namespace.Profile != "family/main" || shard.Namespace.Region != "cn" || shard.Namespace.HouseID != "200171" || shard.Namespace.DataType != "topology" {
		t.Fatalf("namespace = %#v", shard.Namespace)
	}

	loaded, ok := cache.Load("family/main", "cn", "200171", now.Add(time.Minute))
	if !ok {
		t.Fatal("expected cache hit")
	}
	if loaded.APICalls != 0 || topologyCacheHits(loaded) != 1 || loaded.Total != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestTopologyCacheMigratesLegacyEntry(t *testing.T) {
	root := t.TempDir()
	legacyPath := filepath.Join(root, "topology.json")
	now := time.Unix(1000, 0)
	entry := topologyCacheEntry{
		UpdatedAt: now.Unix(),
		Result: api.EntityListResult{
			Region:   "cn",
			HouseID:  "200171",
			Total:    1,
			Counts:   map[string]int{"scene": 1},
			Entities: []api.EntitySummary{{Type: "scene", ID: "scene-1", Name: "回家"}},
			APICalls: 6,
		},
	}
	document := topologyCacheDocument{Entries: map[string]topologyCacheEntry{
		topologyCacheKey("default", "cn", "200171"): entry,
	}}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cache := newTopologyCache(legacyPath)
	loaded, ok := cache.Load("default", "cn", "200171", now.Add(time.Minute))
	if !ok || loaded.Total != 1 || loaded.Entities[0].Type != "scene" {
		t.Fatalf("loaded = %#v ok=%v", loaded, ok)
	}
	if _, err := os.Stat(filepath.Join(root, "topology", "default", "cn", "200171.json")); err != nil {
		t.Fatalf("legacy entry was not migrated to shard: %v", err)
	}
}

func TestTopologyCacheInvalidatePreventsLegacyResurrection(t *testing.T) {
	root := t.TempDir()
	legacyPath := filepath.Join(root, "topology.json")
	now := time.Unix(1000, 0)
	document := topologyCacheDocument{Entries: map[string]topologyCacheEntry{
		topologyCacheKey("default", "cn", "200171"): {
			UpdatedAt: now.Unix(),
			Result: api.EntityListResult{
				Region:   "cn",
				HouseID:  "200171",
				Total:    1,
				Counts:   map[string]int{"room": 1},
				Entities: []api.EntitySummary{{Type: "room", ID: "room-1", Name: "客厅"}},
			},
		},
	}}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cache := newTopologyCache(legacyPath)
	if err := cache.Invalidate("default", "cn", "200171"); err != nil {
		t.Fatalf("Invalidate error: %v", err)
	}
	if loaded, ok := cache.Load("default", "cn", "200171", now.Add(time.Minute)); ok {
		t.Fatalf("expected invalidated cache miss, loaded=%#v", loaded)
	}
}
