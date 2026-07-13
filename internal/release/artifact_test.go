package release

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildArtifactManifestSignsReleaseRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "SKILL.md"), "public skill")
	writeFile(t, filepath.Join(root, "specifications", "api-registry.yaml"), "version: test")
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}

	manifest, err := BuildArtifactManifest(root, base64.StdEncoding.EncodeToString(seed), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("BuildArtifactManifest error: %v", err)
	}
	if manifest.Version != manifestVersion || manifest.RootSHA256 == "" || len(manifest.Files) != 2 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.Files[0].Path != "skill/yeelight-smart-home/SKILL.md" {
		t.Fatalf("files = %#v", manifest.Files)
	}
	if manifest.Signature.Algorithm != "Ed25519" || manifest.Signature.PublicKey == "" || manifest.Signature.Signature == "" {
		t.Fatalf("signature = %#v", manifest.Signature)
	}
	if err := VerifyArtifactManifest(manifest); err != nil {
		t.Fatalf("VerifyArtifactManifest error: %v", err)
	}
}

func TestBuildArtifactManifestRejectsUnsignedAndDirtyReleaseRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "internal.md"), "internal")
	if _, err := BuildArtifactManifest(root, "", time.Unix(1, 0)); err == nil {
		t.Fatal("expected missing key or dirty root to fail")
	}
	seed := make([]byte, ed25519.SeedSize)
	if _, err := BuildArtifactManifest(root, base64.StdEncoding.EncodeToString(seed), time.Unix(1, 0)); err == nil {
		t.Fatal("expected raw docs to fail")
	}
}

func TestVerifyArtifactManifestRejectsTamperedRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "public skill")
	seed := make([]byte, ed25519.SeedSize)
	manifest, err := BuildArtifactManifest(root, base64.StdEncoding.EncodeToString(seed), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("BuildArtifactManifest error: %v", err)
	}
	manifest.RootSHA256 = "tampered"
	if err := VerifyArtifactManifest(manifest); err == nil {
		t.Fatal("expected tampered manifest to fail verification")
	}
}

func TestParseSigningKeyAcceptsHexSeed(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	key, err := parseSigningKey("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("parseSigningKey error: %v", err)
	}
	expected := ed25519.NewKeyFromSeed(seed)
	if len(key) != len(expected) {
		t.Fatalf("key length = %d", len(key))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
