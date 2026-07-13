package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/release"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func runRelease(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home release <build|scan|scan-allowlist|stage|manifest|verify-manifest> <path> [output-dir]")
		return exitInvalidInput
	}
	switch args[0] {
	case "build":
		if len(args) != 2 {
			return releaseUsage(stderr)
		}
		result, err := release.BuildRuntimeBinary(args[1])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "release build: %v\n", err)
			return exitInternalError
		}
		return writeJSON(stdout, stderr, result)
	case "scan":
		if len(args) != 2 {
			return releaseUsage(stderr)
		}
		return writeReleaseScan(stdout, stderr, func() (release.Result, error) {
			return release.Scan(args[1])
		})
	case "scan-allowlist":
		if len(args) != 2 {
			return releaseUsage(stderr)
		}
		return writeReleaseScan(stdout, stderr, func() (release.Result, error) {
			return release.ScanAllowlist(args[1])
		})
	case "stage":
		if len(args) != 3 {
			return releaseUsage(stderr)
		}
		result, err := release.Stage(".", args[1], args[2])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "release stage: %v\n", err)
			return exitInvalidInput
		}
		if !result.OK {
			_, _ = fmt.Fprintln(stderr, "release stage scan failed")
			return exitInvalidInput
		}
		return writeJSON(stdout, stderr, result)
	case "manifest":
		if len(args) != 2 {
			return releaseUsage(stderr)
		}
		manifest, err := release.BuildArtifactManifest(args[1], os.Getenv("YEELIGHT_RELEASE_SIGNING_KEY"), time.Now())
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "release manifest: %v\n", err)
			return exitInvalidInput
		}
		return writeJSON(stdout, stderr, manifest)
	case "verify-manifest":
		if len(args) != 2 {
			return releaseUsage(stderr)
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "release verify-manifest: %v\n", err)
			return exitInternalError
		}
		var manifest release.ArtifactManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			_, _ = fmt.Fprintf(stderr, "release verify-manifest: %v\n", err)
			return exitInvalidInput
		}
		if err := release.VerifyArtifactManifest(manifest); err != nil {
			_, _ = fmt.Fprintf(stderr, "release verify-manifest: %v\n", err)
			return exitInvalidInput
		}
		return writeJSON(stdout, stderr, map[string]any{semantic.FieldOK: true, semantic.FieldRootSHA256: manifest.RootSHA256})
	default:
		return releaseUsage(stderr)
	}
}

func releaseUsage(stderr io.Writer) int {
	_, _ = fmt.Fprintln(stderr, "usage: yeelight-home release <build|scan|scan-allowlist|stage|manifest|verify-manifest> <path> [output-dir]")
	return exitInvalidInput
}

func writeReleaseScan(stdout io.Writer, stderr io.Writer, run func() (release.Result, error)) int {
	result, err := run()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "release scan: %v\n", err)
		return exitInternalError
	}
	if result.OK {
		return writeJSON(stdout, stderr, result)
	}
	parts := make([]string, 0, len(result.Violations))
	for _, violation := range result.Violations {
		if violation.Rule == "raw-docs" {
			parts = append(parts, "raw docs")
			continue
		}
		if violation.Pattern != "" {
			parts = append(parts, violation.Pattern)
			continue
		}
		parts = append(parts, violation.Rule)
	}
	_, _ = fmt.Fprintf(stderr, "release scan failed: %s\n", strings.Join(parts, ", "))
	return exitInvalidInput
}
