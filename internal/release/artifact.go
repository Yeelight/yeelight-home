package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const manifestVersion = "yeelight-smart-home-release-v1"

type ArtifactManifest struct {
	Version     string            `json:"version"`
	GeneratedAt string            `json:"generatedAt"`
	RootSHA256  string            `json:"rootSha256"`
	Files       []ArtifactFile    `json:"files"`
	Signature   ArtifactSignature `json:"signature"`
	Scan        Result            `json:"scan"`
}

type ArtifactFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type ArtifactSignature struct {
	Algorithm string `json:"algorithm"`
	Payload   string `json:"payload"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}

func BuildArtifactManifest(root string, signingKey string, now time.Time) (ArtifactManifest, error) {
	scanResult, err := Scan(root)
	if err != nil {
		return ArtifactManifest{}, err
	}
	if !scanResult.OK {
		return ArtifactManifest{}, fmt.Errorf("release scan failed with %d violation(s)", len(scanResult.Violations))
	}
	privateKey, err := parseSigningKey(signingKey)
	if err != nil {
		return ArtifactManifest{}, err
	}
	files, err := fileChecksums(root)
	if err != nil {
		return ArtifactManifest{}, err
	}
	if len(files) == 0 {
		return ArtifactManifest{}, errors.New("release artifact root contains no files")
	}
	rootHash := rootChecksum(files)
	payload := signingPayload(rootHash)
	signature := ed25519.Sign(privateKey, []byte(payload))
	return ArtifactManifest{
		Version:     manifestVersion,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		RootSHA256:  rootHash,
		Files:       files,
		Signature: ArtifactSignature{
			Algorithm: "Ed25519",
			Payload:   "root-sha256-v1",
			PublicKey: base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)),
			Signature: base64.StdEncoding.EncodeToString(signature),
		},
		Scan: scanResult,
	}, nil
}

func VerifyArtifactManifest(manifest ArtifactManifest) error {
	if manifest.Version != manifestVersion {
		return fmt.Errorf("unsupported manifest version %q", manifest.Version)
	}
	if manifest.Signature.Algorithm != "Ed25519" || manifest.Signature.Payload != "root-sha256-v1" {
		return errors.New("unsupported release signature metadata")
	}
	publicKey, err := base64.StdEncoding.DecodeString(manifest.Signature.PublicKey)
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(manifest.Signature.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(signingPayload(manifest.RootSHA256)), signature) {
		return errors.New("release manifest signature verification failed")
	}
	return nil
}

func parseSigningKey(value string) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("missing YEELIGHT_RELEASE_SIGNING_KEY")
	}
	var data []byte
	var err error
	if len(trimmed) == ed25519.SeedSize*2 || len(trimmed) == ed25519.PrivateKeySize*2 {
		data, err = hex.DecodeString(trimmed)
		if err != nil {
			return nil, errors.New("release signing key must be base64 or hex encoded Ed25519 seed/private key")
		}
	} else {
		data, err = base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, errors.New("release signing key must be base64 or hex encoded Ed25519 seed/private key")
		}
	}
	switch len(data) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(data), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(data), nil
	default:
		return nil, fmt.Errorf("release signing key has %d bytes, want %d byte seed or %d byte private key", len(data), ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

func fileChecksums(root string) ([]ArtifactFile, error) {
	root = filepath.Clean(root)
	files := []ArtifactFile{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, ArtifactFile{
			Path:   filepath.ToSlash(relative),
			Size:   info.Size(),
			SHA256: hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func rootChecksum(files []ArtifactFile) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(file.SHA256))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(fmt.Sprintf("%d", file.Size)))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func signingPayload(rootHash string) string {
	return manifestVersion + "\n" + rootHash + "\n"
}
