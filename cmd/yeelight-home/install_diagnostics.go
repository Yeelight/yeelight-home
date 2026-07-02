package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func installDiagnostics(online bool) map[string]any {
	executable, _ := os.Executable()
	pathLookup, _ := exec.LookPath("yeelight-home")
	var latestChecker latestVersionChecker
	if online {
		latestChecker = onlineLatestVersions
	}
	return buildInstallDiagnostics(executable, pathLookup, version, runtime.GOOS, runtime.GOARCH, os.Getenv("YEELIGHT_HOME_NPM_WRAPPER_PATH"), runInstallDiagnosticCommand, latestChecker)
}

type installCommandRunner func(command string, args ...string) (string, error)
type latestVersionChecker func(context.Context) map[string]any

func buildInstallDiagnostics(executable string, pathLookup string, version string, goos string, goarch string, npmWrapperPath string, runCommand installCommandRunner, latestChecker latestVersionChecker) map[string]any {
	warnings := []string{}
	executableResolved := canonicalExecutablePath(executable)
	pathLookupResolved := canonicalExecutablePath(pathLookup)
	npmWrapperResolved := canonicalExecutablePath(npmWrapperPath)
	if executable != "" && pathLookup != "" && executableResolved != "" && pathLookupResolved != "" && executableResolved != pathLookupResolved {
		warnings = append(warnings, "path_lookup_differs_from_running_executable")
	}
	npm := npmGlobalDiagnostics(runCommand)
	homebrew := homebrewDiagnostics(runCommand)
	if npmVersion, ok := npm[semantic.FieldVersion].(string); ok && versionMismatch(version, npmVersion) {
		warnings = append(warnings, "npm_global_package_version_differs_from_runtime_version")
	}
	for _, warning := range homebrewVersionMismatchWarnings(version, homebrew) {
		warnings = append(warnings, warning)
	}
	if containsPathSegment(pathLookupResolved, "node_modules/yeelight-home") {
		warnings = append(warnings, "path_lookup_uses_npm_wrapper")
	}
	if npmWrapperResolved != "" && pathLookupResolved != "" && npmWrapperResolved != pathLookupResolved {
		warnings = append(warnings, "npm_wrapper_differs_from_path_lookup")
	}
	latest := map[string]any{}
	if latestChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		latest = latestChecker(ctx)
		warnings = append(warnings, onlineInstallWarnings(npm, homebrew, latest)...)
	}
	remediations := installRemediations(warnings, npm, homebrew)
	result := map[string]any{
		semantic.FieldCLI:                "yeelight-home",
		semantic.FieldPublicRepo:         "Yeelight/yeelight-home",
		semantic.FieldVersion:            version,
		semantic.FieldOS:                 goos,
		semantic.FieldArch:               goarch,
		semantic.FieldExecutable:         executable,
		semantic.FieldExecutableResolved: executableResolved,
		semantic.FieldPathLookup:         pathLookup,
		semantic.FieldPathLookupResolved: pathLookupResolved,
		semantic.FieldNPMWrapper:         npmWrapperPath,
		semantic.FieldNPMWrapperResolved: npmWrapperResolved,
		semantic.FieldPackageManagers: map[string]any{
			semantic.FieldNPM:      npm,
			semantic.FieldHomebrew: homebrew,
		},
		semantic.FieldWarnings:     warnings,
		semantic.FieldRemediations: remediations,
	}
	if latestChecker != nil {
		result[semantic.FieldLatest] = latest
	}
	return result
}

func installRemediations(warnings []string, npm map[string]any, homebrew map[string]any) []string {
	actions := []string{}
	if containsDiagnostic(warnings, "path_lookup_differs_from_running_executable") {
		actions = append(actions, "Run `command -v yeelight-home` and restart the shell or Skill host so PATH resolves the intended binary.")
	}
	if containsDiagnostic(warnings, "path_lookup_uses_npm_wrapper") || containsDiagnostic(warnings, "npm_global_package_version_differs_from_runtime_version") {
		if boolFromMap(npm, semantic.FieldInstalled) {
			actions = append(actions, "Upgrade the npm wrapper with `npm install -g yeelight-home@latest`, then restart the shell or Skill host.")
		}
	}
	if containsDiagnostic(warnings, "npm_global_package_behind_latest") {
		actions = append(actions, "The npm registry has a newer yeelight-home package; run `npm install -g yeelight-home@latest` and restart the shell or Skill host.")
	}
	if containsDiagnostic(warnings, "homebrew_package_version_differs_from_runtime_version") {
		if boolFromMap(homebrew, semantic.FieldInstalled) {
			actions = append(actions, "Upgrade Homebrew with `brew update && brew upgrade yeelight-home`, then restart the shell or Skill host.")
		}
	}
	if containsDiagnostic(warnings, "homebrew_formula_version_differs_from_runtime_version") && nestedBoolFromMap(homebrew, semantic.FieldFormula, semantic.FieldInstalled) {
		actions = append(actions, "Upgrade the Homebrew formula with `brew update && brew upgrade yeelight-home`, then restart the shell or Skill host.")
	}
	if containsDiagnostic(warnings, "homebrew_cask_version_differs_from_runtime_version") && nestedBoolFromMap(homebrew, semantic.FieldCask, semantic.FieldInstalled) {
		actions = append(actions, "Upgrade the Homebrew cask with `brew update && brew upgrade --cask yeelight-home`, then restart the shell or Skill host.")
	}
	if containsDiagnostic(warnings, "homebrew_formula_behind_latest") {
		actions = append(actions, "Refresh Homebrew formula metadata with `brew update`, then run `brew upgrade yeelight-home` or reinstall from Yeelight/tap.")
	}
	if containsDiagnostic(warnings, "homebrew_cask_behind_latest") {
		actions = append(actions, "Refresh Homebrew cask metadata with `brew update`, then run `brew upgrade --cask yeelight-home` or reinstall from Yeelight/tap.")
	}
	if containsDiagnostic(warnings, "npm_wrapper_differs_from_path_lookup") {
		actions = append(actions, "Remove or unlink stale npm/Homebrew entries so only one `yeelight-home` appears on PATH.")
	}
	if len(actions) == 0 && boolFromMap(npm, semantic.FieldInstalled) && boolFromMap(homebrew, semantic.FieldInstalled) {
		actions = append(actions, "Prefer one primary install channel per machine; remove the unused npm or Homebrew install to avoid PATH drift.")
	}
	return uniqueStrings(actions)
}

func onlineInstallWarnings(npm map[string]any, homebrew map[string]any, latest map[string]any) []string {
	warnings := []string{}
	npmLatest := latestChannelVersion(latest, semantic.FieldNPM)
	if boolFromMap(npm, semantic.FieldInstalled) && versionNewerThan(npmLatest, stringFromMap(npm, semantic.FieldVersion)) {
		warnings = append(warnings, "npm_global_package_behind_latest")
	}
	homebrewLatest := latestChannelVersion(latest, semantic.FieldHomebrew)
	if nestedBoolFromMap(homebrew, semantic.FieldFormula, semantic.FieldInstalled) && versionNewerThan(homebrewLatest, nestedStringFromMap(homebrew, semantic.FieldFormula, semantic.FieldVersion)) {
		warnings = append(warnings, "homebrew_formula_behind_latest")
	}
	homebrewCaskLatest := firstNonEmpty(latestChannelVersion(latest, semantic.FieldHomebrewCask), homebrewLatest)
	if nestedBoolFromMap(homebrew, semantic.FieldCask, semantic.FieldInstalled) && versionNewerThan(homebrewCaskLatest, nestedStringFromMap(homebrew, semantic.FieldCask, semantic.FieldVersion)) {
		warnings = append(warnings, "homebrew_cask_behind_latest")
	}
	return warnings
}

func latestChannelVersion(latest map[string]any, channel string) string {
	channels, ok := latest[semantic.FieldChannels].(map[string]any)
	if !ok {
		return ""
	}
	info, ok := channels[channel].(map[string]any)
	if !ok {
		return ""
	}
	return stringFromMap(info, semantic.FieldVersion)
}

func stringFromMap(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func nestedStringFromMap(values map[string]any, section string, key string) string {
	nested, ok := values[section].(map[string]any)
	if !ok {
		return ""
	}
	return stringFromMap(nested, key)
}

func runInstallDiagnosticCommand(command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", ctx.Err()
	}
	return strings.TrimSpace(string(output)), err
}

func npmGlobalDiagnostics(runCommand installCommandRunner) map[string]any {
	info := map[string]any{semantic.FieldAvailable: false, semantic.FieldInstalled: false}
	if runCommand == nil {
		return info
	}
	root, rootErr := runCommand("npm", "root", "-g")
	if rootErr != nil {
		info[semantic.FieldError] = "npm_not_available"
		return info
	}
	info[semantic.FieldAvailable] = true
	if root != "" {
		info[semantic.FieldGlobalRoot] = root
	}
	output, err := runCommand("npm", "list", "-g", "yeelight-home", "--depth=0", "--json")
	if err != nil && strings.TrimSpace(output) == "" {
		info[semantic.FieldError] = "package_not_listed"
		return info
	}
	version := parseNPMGlobalVersion(output)
	if version == "" {
		return info
	}
	info[semantic.FieldInstalled] = true
	info[semantic.FieldVersion] = version
	if root != "" {
		info[semantic.FieldPackagePath] = filepath.Join(root, "yeelight-home")
	}
	return info
}

func parseNPMGlobalVersion(output string) string {
	var payload struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ""
	}
	if dependency, ok := payload.Dependencies["yeelight-home"]; ok {
		return strings.TrimSpace(dependency.Version)
	}
	return ""
}

func homebrewDiagnostics(runCommand installCommandRunner) map[string]any {
	info := map[string]any{semantic.FieldAvailable: false, semantic.FieldInstalled: false}
	if runCommand == nil {
		return info
	}
	prefix, prefixErr := runCommand("brew", "--prefix")
	if prefixErr != nil {
		info[semantic.FieldError] = "brew_not_available"
		return info
	}
	info[semantic.FieldAvailable] = true
	if prefix != "" {
		info[semantic.FieldPrefix] = prefix
	}
	formula := homebrewFormulaDiagnostics(runCommand)
	cask := homebrewCaskDiagnostics(runCommand)
	info[semantic.FieldFormula] = formula
	info[semantic.FieldCask] = cask
	if boolFromMap(formula, semantic.FieldInstalled) || boolFromMap(cask, semantic.FieldInstalled) {
		info[semantic.FieldInstalled] = true
	}
	if version := stringFromMap(formula, semantic.FieldVersion); version != "" {
		info[semantic.FieldVersion] = version
		info[semantic.FieldChannel] = semantic.FieldFormula
	} else if version := stringFromMap(cask, semantic.FieldVersion); version != "" {
		info[semantic.FieldVersion] = version
		info[semantic.FieldChannel] = semantic.FieldCask
	}
	return info
}

func homebrewFormulaDiagnostics(runCommand installCommandRunner) map[string]any {
	info := map[string]any{semantic.FieldInstalled: false}
	output, err := runCommand("brew", "list", "--versions", "yeelight-home")
	if err != nil && strings.TrimSpace(output) == "" {
		return info
	}
	version := parseHomebrewVersion(output)
	if version == "" {
		return info
	}
	info[semantic.FieldInstalled] = true
	info[semantic.FieldVersion] = version
	return info
}

func homebrewCaskDiagnostics(runCommand installCommandRunner) map[string]any {
	info := map[string]any{semantic.FieldInstalled: false}
	output, err := runCommand("brew", "list", "--cask", "--versions", "yeelight-home")
	if err != nil && strings.TrimSpace(output) == "" {
		return info
	}
	version := parseHomebrewVersion(output)
	if version == "" {
		return info
	}
	info[semantic.FieldInstalled] = true
	info[semantic.FieldVersion] = version
	return info
}

func homebrewVersionMismatchWarnings(runtimeVersion string, homebrew map[string]any) []string {
	warnings := []string{}
	formulaVersion := nestedStringFromMap(homebrew, semantic.FieldFormula, semantic.FieldVersion)
	caskVersion := nestedStringFromMap(homebrew, semantic.FieldCask, semantic.FieldVersion)
	if formulaVersion == "" && caskVersion == "" {
		if brewVersion, ok := homebrew[semantic.FieldVersion].(string); ok && versionMismatch(runtimeVersion, brewVersion) {
			return []string{"homebrew_package_version_differs_from_runtime_version"}
		}
		return nil
	}
	if formulaVersion != "" && versionMismatch(runtimeVersion, formulaVersion) {
		warnings = append(warnings, "homebrew_formula_version_differs_from_runtime_version")
	}
	if caskVersion != "" && versionMismatch(runtimeVersion, caskVersion) {
		warnings = append(warnings, "homebrew_cask_version_differs_from_runtime_version")
	}
	return warnings
}

func parseHomebrewVersion(output string) string {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) >= 2 && fields[0] == "yeelight-home" {
		return fields[1]
	}
	return ""
}

func versionMismatch(runtimeVersion string, packageVersion string) bool {
	runtimeVersion = strings.TrimPrefix(strings.TrimSpace(runtimeVersion), "v")
	packageVersion = strings.TrimPrefix(strings.TrimSpace(packageVersion), "v")
	return runtimeVersion != "" && runtimeVersion != "dev" && packageVersion != "" && runtimeVersion != packageVersion
}

func containsPathSegment(path string, segment string) bool {
	if path == "" {
		return false
	}
	normalizedPath := filepath.ToSlash(path)
	return strings.Contains(normalizedPath, segment)
}

func containsDiagnostic(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func boolFromMap(values map[string]any, key string) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return false
}

func nestedBoolFromMap(values map[string]any, section string, key string) bool {
	nested, ok := values[section].(map[string]any)
	if !ok {
		return false
	}
	return boolFromMap(nested, key)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func canonicalExecutablePath(path string) string {
	if path == "" {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		absolute = resolved
	}
	return filepath.Clean(absolute)
}

func onlineLatestVersions(ctx context.Context) map[string]any {
	checks := map[string]any{
		semantic.FieldGitHubRelease: latestGitHubRelease(ctx),
		semantic.FieldNPM:           latestNPMVersion(ctx),
		semantic.FieldHomebrew:      latestHomebrewFormulaVersion(ctx),
		semantic.FieldHomebrewCask:  latestHomebrewCaskVersion(ctx),
	}
	return map[string]any{
		semantic.FieldChecked:  true,
		semantic.FieldChannels: checks,
	}
}

func latestGitHubRelease(ctx context.Context) map[string]any {
	var payload struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
	}
	if err := getJSON(ctx, "https://api.github.com/repos/Yeelight/yeelight-home/releases/latest", &payload); err != nil {
		return map[string]any{semantic.FieldOK: false, semantic.FieldError: err.Error()}
	}
	return map[string]any{
		semantic.FieldOK:          true,
		semantic.FieldVersion:     strings.TrimPrefix(strings.TrimSpace(payload.TagName), "v"),
		semantic.FieldTag:         payload.TagName,
		semantic.FieldPublishedAt: payload.PublishedAt,
		semantic.FieldURL:         payload.HTMLURL,
	}
}

func latestNPMVersion(ctx context.Context) map[string]any {
	var payload struct {
		Version string `json:"version"`
	}
	if err := getJSON(ctx, "https://registry.npmjs.org/yeelight-home/latest", &payload); err != nil {
		return map[string]any{semantic.FieldOK: false, semantic.FieldError: err.Error()}
	}
	version := strings.TrimSpace(payload.Version)
	return map[string]any{semantic.FieldOK: version != "", semantic.FieldVersion: version}
}

func latestHomebrewFormulaVersion(ctx context.Context) map[string]any {
	body, err := getText(ctx, "https://raw.githubusercontent.com/Yeelight/homebrew-tap/main/Formula/yeelight-home.rb")
	if err != nil {
		return map[string]any{semantic.FieldOK: false, semantic.FieldError: err.Error()}
	}
	version := parseHomebrewFormulaVersion(body)
	return map[string]any{
		semantic.FieldOK:      version != "",
		semantic.FieldVersion: version,
		semantic.FieldName:    "yeelight/tap/yeelight-home",
	}
}

func latestHomebrewCaskVersion(ctx context.Context) map[string]any {
	body, err := getText(ctx, "https://raw.githubusercontent.com/Yeelight/homebrew-tap/main/Casks/yeelight-home.rb")
	if err != nil {
		return map[string]any{semantic.FieldOK: false, semantic.FieldError: err.Error()}
	}
	version := parseHomebrewFormulaVersion(body)
	return map[string]any{
		semantic.FieldOK:      version != "",
		semantic.FieldVersion: version,
		semantic.FieldName:    "yeelight/tap/yeelight-home",
	}
}

func parseHomebrewFormulaVersion(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "version ") {
			continue
		}
		parts := strings.Split(line, `"`)
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func getJSON(ctx context.Context, url string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return err
	}
	return nil
}

func getText(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func versionNewerThan(candidate string, current string) bool {
	candidateParts := semanticVersionParts(candidate)
	currentParts := semanticVersionParts(current)
	if candidateParts == nil || currentParts == nil {
		return false
	}
	for index := 0; index < len(candidateParts); index++ {
		if candidateParts[index] > currentParts[index] {
			return true
		}
		if candidateParts[index] < currentParts[index] {
			return false
		}
	}
	return false
}

func semanticVersionParts(value string) []int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if value == "" || value == "dev" {
		return nil
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return nil
	}
	result := make([]int, 3)
	for index, part := range parts {
		number := 0
		for _, char := range part {
			if char < '0' || char > '9' {
				return nil
			}
			number = number*10 + int(char-'0')
		}
		result[index] = number
	}
	return result
}
