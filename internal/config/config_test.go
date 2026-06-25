package config

import "testing"

func TestResolveUsesExplicitHomeDir(t *testing.T) {
	paths := Resolve("/tmp/yeelight-home-test")

	if paths.HomeDir != "/tmp/yeelight-home-test" {
		t.Fatalf("HomeDir = %s", paths.HomeDir)
	}
	if paths.ConfigDir != "/tmp/yeelight-home-test/config" {
		t.Fatalf("ConfigDir = %s", paths.ConfigDir)
	}
	if paths.DataDir != "/tmp/yeelight-home-test/data" {
		t.Fatalf("DataDir = %s", paths.DataDir)
	}
	if paths.CacheDir != "/tmp/yeelight-home-test/cache" {
		t.Fatalf("CacheDir = %s", paths.CacheDir)
	}
}

func TestResolveFromLookupUsesOverride(t *testing.T) {
	paths := ResolveFromLookup(func(key string) (string, bool) {
		if key == "YEELIGHT_HOME_DIR" {
			return "/tmp/override-home", true
		}
		return "", false
	})

	if paths.HomeDir != "/tmp/override-home" {
		t.Fatalf("HomeDir = %s", paths.HomeDir)
	}
}
