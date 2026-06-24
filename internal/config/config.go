package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	HomeDir   string `json:"homeDir"`
	ConfigDir string `json:"configDir"`
	DataDir   string `json:"dataDir"`
	CacheDir  string `json:"cacheDir"`
}

func Resolve(homeDir string) Paths {
	cleanHome := filepath.Clean(homeDir)
	return Paths{
		HomeDir:   cleanHome,
		ConfigDir: filepath.Join(cleanHome, "config"),
		DataDir:   filepath.Join(cleanHome, "data"),
		CacheDir:  filepath.Join(cleanHome, "cache"),
	}
}

func ResolveFromEnv() Paths {
	return ResolveFromLookup(os.LookupEnv)
}

func ResolveFromLookup(lookup func(string) (string, bool)) Paths {
	if value, ok := lookup("YEELIGHT_HOME_DIR"); ok && strings.TrimSpace(value) != "" {
		return Resolve(value)
	}
	if value, ok := lookup("HOME"); ok && strings.TrimSpace(value) != "" {
		return Resolve(filepath.Join(value, ".yeelight-home"))
	}
	return Resolve(".yeelight-home")
}
