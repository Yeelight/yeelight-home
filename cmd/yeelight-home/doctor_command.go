package main

import (
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/config"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
		semantic.FieldStatus:        status,
		semantic.FieldWarnings:      warnings,
		semantic.FieldAuthenticated: context.TokenPresent || authStatus.Authenticated,
		semantic.FieldProfile:       context.Profile,
		semantic.FieldRegion:        context.Region,
		semantic.FieldHouseID:       context.HouseID,
		semantic.FieldTokenPresent:  context.TokenPresent,
		semantic.FieldTokenSource:   context.TokenSource,
		semantic.FieldHomeDir:       paths.HomeDir,
		semantic.FieldConfigDir:     paths.ConfigDir,
		semantic.FieldDataDir:       paths.DataDir,
		semantic.FieldCacheDir:      paths.CacheDir,
		semantic.FieldInstall:       installDiagnostics(flags.bool("online")),
		semantic.FieldMemoryMigrations: map[string]any{
			semantic.FieldStatus: "available",
			semantic.FieldCount:  len(storage.MemoryMigrations()),
		},
	}
	if !flags.bool("json") {
		return writeDoctorText(stdout, response)
	}
	return writeJSON(stdout, stderr, response)
}

func writeDoctorText(stdout io.Writer, response map[string]any) int {
	_, _ = fmt.Fprintf(stdout, "Yeelight Home Doctor\n")
	_, _ = fmt.Fprintf(stdout, "Status: %s\n", stringFromDiagnostic(response, semantic.FieldStatus))
	_, _ = fmt.Fprintf(stdout, "Authenticated: %t\n", boolFromDiagnostic(response, semantic.FieldAuthenticated))
	_, _ = fmt.Fprintf(stdout, "Profile: %s\n", stringFromDiagnostic(response, semantic.FieldProfile))
	_, _ = fmt.Fprintf(stdout, "Region: %s\n", stringFromDiagnostic(response, semantic.FieldRegion))
	houseID := stringFromDiagnostic(response, semantic.FieldHouseID)
	if houseID == "" {
		houseID = "(not selected)"
	}
	_, _ = fmt.Fprintf(stdout, "House ID: %s\n", houseID)
	_, _ = fmt.Fprintf(stdout, "Home dir: %s\n", stringFromDiagnostic(response, semantic.FieldHomeDir))
	if install, ok := response[semantic.FieldInstall].(map[string]any); ok {
		_, _ = fmt.Fprintf(stdout, "Runtime version: %s\n", stringFromDiagnostic(install, semantic.FieldVersion))
		_, _ = fmt.Fprintf(stdout, "Executable: %s\n", stringFromDiagnostic(install, semantic.FieldExecutable))
		_, _ = fmt.Fprintf(stdout, "PATH lookup: %s\n", stringFromDiagnostic(install, semantic.FieldPathLookup))
		writeInstallSourceSummary(stdout, install)
		if packageManagers, ok := install[semantic.FieldPackageManagers].(map[string]any); ok {
			writePackageManagerText(stdout, semantic.FieldNPM, packageManagers[semantic.FieldNPM])
			writePackageManagerText(stdout, semantic.FieldHomebrew, packageManagers[semantic.FieldHomebrew])
		}
		writeLatestVersionsText(stdout, install[semantic.FieldLatest])
		writeWarningsText(stdout, "Install warnings", install[semantic.FieldWarnings])
		writeWarningsText(stdout, "Suggested fixes", install[semantic.FieldRemediations])
	}
	writeWarningsText(stdout, "Warnings", response[semantic.FieldWarnings])
	return exitOK
}

func writeInstallSourceSummary(stdout io.Writer, install map[string]any) {
	summary := []string{}
	pathLookup := stringFromDiagnostic(install, semantic.FieldPathLookupResolved)
	if pathLookup == "" {
		pathLookup = stringFromDiagnostic(install, semantic.FieldPathLookup)
	}
	if pathLookup != "" {
		if containsPathSegment(pathLookup, "node_modules/yeelight-home") {
			summary = append(summary, "PATH channel: npm wrapper")
		} else {
			summary = append(summary, "PATH channel: direct binary or package-manager shim")
		}
	}
	if wrapper := stringFromDiagnostic(install, semantic.FieldNPMWrapperResolved); wrapper != "" {
		summary = append(summary, "Running through npm wrapper: true")
	} else if wrapper := stringFromDiagnostic(install, semantic.FieldNPMWrapper); wrapper != "" {
		summary = append(summary, "Running through npm wrapper: true")
	}
	if packageManagers, ok := install[semantic.FieldPackageManagers].(map[string]any); ok {
		if npm, ok := packageManagers[semantic.FieldNPM].(map[string]any); ok && boolFromDiagnostic(npm, semantic.FieldInstalled) {
			summary = append(summary, "npm global version: "+fallbackVersion(stringFromDiagnostic(npm, semantic.FieldVersion)))
		}
		if homebrew, ok := packageManagers[semantic.FieldHomebrew].(map[string]any); ok {
			if formula, ok := homebrew[semantic.FieldFormula].(map[string]any); ok && boolFromDiagnostic(formula, semantic.FieldInstalled) {
				summary = append(summary, "Homebrew formula version: "+fallbackVersion(stringFromDiagnostic(formula, semantic.FieldVersion)))
			}
			if cask, ok := homebrew[semantic.FieldCask].(map[string]any); ok && boolFromDiagnostic(cask, semantic.FieldInstalled) {
				summary = append(summary, "Homebrew cask version: "+fallbackVersion(stringFromDiagnostic(cask, semantic.FieldVersion)))
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
	available := boolFromDiagnostic(info, semantic.FieldAvailable)
	installed := boolFromDiagnostic(info, semantic.FieldInstalled)
	version := stringFromDiagnostic(info, semantic.FieldVersion)
	if version == "" {
		version = "-"
	}
	_, _ = fmt.Fprintf(stdout, "%s: available=%t installed=%t version=%s\n", name, available, installed, version)
	if name == semantic.FieldHomebrew {
		writeHomebrewChannelText(stdout, semantic.FieldFormula, info[semantic.FieldFormula])
		writeHomebrewChannelText(stdout, semantic.FieldCask, info[semantic.FieldCask])
	}
}

func writeHomebrewChannelText(stdout io.Writer, name string, value any) {
	info, ok := value.(map[string]any)
	if !ok {
		return
	}
	version := stringFromDiagnostic(info, semantic.FieldVersion)
	if version == "" {
		version = "-"
	}
	_, _ = fmt.Fprintf(stdout, "  %s: installed=%t version=%s\n", name, boolFromDiagnostic(info, semantic.FieldInstalled), version)
}

func writeLatestVersionsText(stdout io.Writer, value any) {
	latest, ok := value.(map[string]any)
	if !ok || !boolFromDiagnostic(latest, semantic.FieldChecked) {
		return
	}
	channels, ok := latest[semantic.FieldChannels].(map[string]any)
	if !ok {
		return
	}
	_, _ = fmt.Fprintln(stdout, "Public latest:")
	for _, name := range []string{semantic.FieldGitHubRelease, semantic.FieldNPM, semantic.FieldHomebrew, semantic.FieldHomebrewCask} {
		info, ok := channels[name].(map[string]any)
		if !ok {
			continue
		}
		version := stringFromDiagnostic(info, semantic.FieldVersion)
		if version == "" {
			version = "-"
		}
		_, _ = fmt.Fprintf(stdout, "  - %s: ok=%t version=%s\n", name, boolFromDiagnostic(info, semantic.FieldOK), version)
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
