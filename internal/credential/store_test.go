package credential

import (
	"os"
	"strings"
	"testing"
)

func TestMemoryStoreSavesLoadsAndDeletesToken(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Save(TokenRecord{Profile: "family-main", AccessToken: "access-token", RefreshToken: "refresh-token"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	record, ok, err := store.Load("family-main")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !ok {
		t.Fatal("expected token record")
	}
	if record.AccessToken != "access-token" || record.RefreshToken != "refresh-token" {
		t.Fatalf("record = %#v", record)
	}

	if err := store.Delete("family-main"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, ok, err = store.Load("family-main")
	if err != nil {
		t.Fatalf("Load after delete error: %v", err)
	}
	if ok {
		t.Fatal("expected token record to be deleted")
	}
}

func TestMemoryStoreRejectsEmptyProfile(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Save(TokenRecord{AccessToken: "access-token"}); err == nil {
		t.Fatal("expected empty profile to be rejected")
	}
	if _, _, err := store.Load(""); err == nil {
		t.Fatal("expected empty profile load to be rejected")
	}
}

func TestTokenMetadataDoesNotExposeSecretValues(t *testing.T) {
	record := TokenRecord{
		Profile:      "family-main",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    123,
	}

	metadata := record.Metadata()
	if metadata.Profile != "family-main" {
		t.Fatalf("profile = %s", metadata.Profile)
	}
	if !metadata.AccessTokenPresent || !metadata.RefreshTokenPresent {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata.ExpiresAt != 123 {
		t.Fatalf("expiresAt = %d", metadata.ExpiresAt)
	}
}

func TestFileMetadataStoreSavesMetadataWithoutToken(t *testing.T) {
	path := t.TempDir() + "/profiles.json"
	store := NewFileMetadataStore(path)

	if err := store.Save(ProfileMetadata{
		Profile:  "default",
		Region:   "dev",
		ClientID: "client-qr-123456",
		HouseID:  "house-qr-123456",
		QRDevice: "F8:24:41:AA:BB:CC",
	}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	metadata, ok, err := store.Load("default")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !ok {
		t.Fatal("expected profile metadata")
	}
	if metadata.Region != "dev" || metadata.ClientID != "client-qr-123456" || metadata.HouseID != "house-qr-123456" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata.QRDevice != "F8:24:41:AA:BB:CC" {
		t.Fatalf("qrDevice = %q", metadata.QRDevice)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{"accessToken", "authorization", "Bearer", "secret-token"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metadata file leaked token-like field %q: %s", forbidden, text)
		}
	}
}

func TestFileMetadataStoreTracksActiveProfile(t *testing.T) {
	path := t.TempDir() + "/profiles.json"
	store := NewFileMetadataStore(path)

	if err := store.Save(ProfileMetadata{Profile: "family", Region: "cn"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if err := store.SetActiveProfile("family"); err != nil {
		t.Fatalf("SetActiveProfile error: %v", err)
	}
	active, err := store.ActiveProfile()
	if err != nil {
		t.Fatalf("ActiveProfile error: %v", err)
	}
	if active != "family" {
		t.Fatalf("active profile = %q", active)
	}
	if err := store.Delete("family"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	active, err = store.ActiveProfile()
	if err != nil {
		t.Fatalf("ActiveProfile after delete error: %v", err)
	}
	if active != "" {
		t.Fatalf("active profile after delete = %q", active)
	}
}

func TestFileTokenStoreSavesLoadsAndDeletesToken(t *testing.T) {
	path := t.TempDir() + "/tokens.json"
	store := NewFileTokenStore(path)

	if err := store.Save(TokenRecord{Profile: "default", AccessToken: "Bearer token-secret-123456"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	record, ok, err := store.Load("default")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !ok || record.AccessToken != "Bearer token-secret-123456" {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
	if err := store.Delete("default"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, ok, err = store.Load("default")
	if err != nil {
		t.Fatalf("Load after delete error: %v", err)
	}
	if ok {
		t.Fatal("expected token to be deleted")
	}
}
