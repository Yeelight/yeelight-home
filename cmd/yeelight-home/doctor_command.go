package main

import (
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/config"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func (app *app) runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "doctor: %v\n", err)
		return exitInvalidInput
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "doctor: %v\n", err)
		return exitInvalidInput
	}
	authStatus := auth.StatusFromEnv()
	status := "ok"
	warnings := []string{}
	if !context.TokenPresent && !authStatus.Authenticated {
		status = "warning"
		warnings = append(warnings, "auth_required")
	}
	paths := config.ResolveFromEnv()
	response := map[string]any{
		"status":        status,
		"warnings":      warnings,
		"authenticated": context.TokenPresent || authStatus.Authenticated,
		"profile":       context.Profile,
		"region":        context.Region,
		"houseId":       context.HouseID,
		"tokenPresent":  context.TokenPresent,
		"tokenSource":   context.TokenSource,
		"homeDir":       paths.HomeDir,
		"configDir":     paths.ConfigDir,
		"dataDir":       paths.DataDir,
		"cacheDir":      paths.CacheDir,
		"install":       installDiagnostics(flags.bool("online")),
		"memoryMigrations": map[string]any{
			"status": "available",
			"count":  len(storage.MemoryMigrations()),
		},
	}
	if !flags.bool("json") {
		return writeDoctorText(stdout, response)
	}
	return writeJSON(stdout, stderr, response)
}

func writeDoctorText(stdout io.Writer, response map[string]any) int {
	_, _ = fmt.Fprintf(stdout, "Yeelight Home Doctor\n")
	_, _ = fmt.Fprintf(stdout, "Status: %s\n", stringFromDiagnostic(response, "status"))
	_, _ = fmt.Fprintf(stdout, "Authenticated: %t\n", boolFromDiagnostic(response, "authenticated"))
	_, _ = fmt.Fprintf(stdout, "Profile: %s\n", stringFromDiagnostic(response, "profile"))
	_, _ = fmt.Fprintf(stdout, "Region: %s\n", stringFromDiagnostic(response, "region"))
	houseID := stringFromDiagnostic(response, "houseId")
	if houseID == "" {
		houseID = "(not selected)"
	}
	_, _ = fmt.Fprintf(stdout, "House ID: %s\n", houseID)
	_, _ = fmt.Fprintf(stdout, "Home dir: %s\n", stringFromDiagnostic(response, "homeDir"))
	if install, ok := response["install"].(map[string]any); ok {
		_, _ = fmt.Fprintf(stdout, "Runtime version: %s\n", stringFromDiagnostic(install, "version"))
		_, _ = fmt.Fprintf(stdout, "Executable: %s\n", stringFromDiagnostic(install, "executable"))
		_, _ = fmt.Fprintf(stdout, "PATH lookup: %s\n", stringFromDiagnostic(install, "pathLookup"))
		writeInstallSourceSummary(stdout, install)
		if packageManagers, ok := install["packageManagers"].(map[string]any); ok {
			writePackageManagerText(stdout, "npm", packageManagers["npm"])
			writePackageManagerText(stdout, "homebrew", packageManagers["homebrew"])
		}
		writeLatestVersionsText(stdout, install["latest"])
		writeWarningsText(stdout, "Install warnings", install["warnings"])
		writeWarningsText(stdout, "Suggested fixes", install["remediations"])
	}
	writeWarningsText(stdout, "Warnings", response["warnings"])
	return exitOK
}

func writeInstallSourceSummary(stdout io.Writer, install map[string]any) {
	summary := []string{}
	pathLookup := stringFromDiagnostic(install, "pathLookupResolved")
	if pathLookup == "" {
		pathLookup = stringFromDiagnostic(install, "pathLookup")
	}
	if pathLookup != "" {
		if containsPathSegment(pathLookup, "node_modules/yeelight-home") {
			summary = append(summary, "PATH channel: npm wrapper")
		} else {
			summary = append(summary, "PATH channel: direct binary or package-manager shim")
		}
	}
	if wrapper := stringFromDiagnostic(install, "npmWrapperResolved"); wrapper != "" {
		summary = append(summary, "Running through npm wrapper: true")
	} else if wrapper := stringFromDiagnostic(install, "npmWrapper"); wrapper != "" {
		summary = append(summary, "Running through npm wrapper: true")
	}
	if packageManagers, ok := install["packageManagers"].(map[string]any); ok {
		if npm, ok := packageManagers["npm"].(map[string]any); ok && boolFromDiagnostic(npm, "installed") {
			summary = append(summary, "npm global version: "+fallbackVersion(stringFromDiagnostic(npm, "version")))
		}
		if homebrew, ok := packageManagers["homebrew"].(map[string]any); ok {
			if formula, ok := homebrew["formula"].(map[string]any); ok && boolFromDiagnostic(formula, "installed") {
				summary = append(summary, "Homebrew formula version: "+fallbackVersion(stringFromDiagnostic(formula, "version")))
			}
			if cask, ok := homebrew["cask"].(map[string]any); ok && boolFromDiagnostic(cask, "installed") {
				summary = append(summary, "Homebrew cask version: "+fallbackVersion(stringFromDiagnostic(cask, "version")))
			}
		}
	}
	if len(summary) == 0 {
		return
	}
	_, _ = fmt.Fprintln(stdout, "Install source summary:")
	for _, item := range summary {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", item)
	}
}

func writePackageManagerText(stdout io.Writer, name string, value any) {
	info, ok := value.(map[string]any)
	if !ok {
		return
	}
	available := boolFromDiagnostic(info, "available")
	installed := boolFromDiagnostic(info, "installed")
	version := stringFromDiagnostic(info, "version")
	if version == "" {
		version = "-"
	}
	_, _ = fmt.Fprintf(stdout, "%s: available=%t installed=%t version=%s\n", name, available, installed, version)
	if name == "homebrew" {
		writeHomebrewChannelText(stdout, "formula", info["formula"])
		writeHomebrewChannelText(stdout, "cask", info["cask"])
	}
}

func writeHomebrewChannelText(stdout io.Writer, name string, value any) {
	info, ok := value.(map[string]any)
	if !ok {
		return
	}
	version := stringFromDiagnostic(info, "version")
	if version == "" {
		version = "-"
	}
	_, _ = fmt.Fprintf(stdout, "  %s: installed=%t version=%s\n", name, boolFromDiagnostic(info, "installed"), version)
}

func writeLatestVersionsText(stdout io.Writer, value any) {
	latest, ok := value.(map[string]any)
	if !ok || !boolFromDiagnostic(latest, "checked") {
		return
	}
	channels, ok := latest["channels"].(map[string]any)
	if !ok {
		return
	}
	_, _ = fmt.Fprintln(stdout, "Public latest:")
	for _, name := range []string{"githubRelease", "npm", "homebrew", "homebrewCask"} {
		info, ok := channels[name].(map[string]any)
		if !ok {
			continue
		}
		version := stringFromDiagnostic(info, "version")
		if version == "" {
			version = "-"
		}
		_, _ = fmt.Fprintf(stdout, "  - %s: ok=%t version=%s\n", name, boolFromDiagnostic(info, "ok"), version)
	}
}

func fallbackVersion(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func writeWarningsText(stdout io.Writer, label string, value any) {
	warnings := diagnosticStrings(value)
	if len(warnings) == 0 {
		return
	}
	_, _ = fmt.Fprintf(stdout, "%s:\n", label)
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", warning)
	}
}

func stringFromDiagnostic(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func boolFromDiagnostic(values map[string]any, key string) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return false
}

func diagnosticStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}
