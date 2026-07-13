package release

import (
	"os"
	"path/filepath"
	"strings"
)

type Result struct {
	OK           bool        `json:"ok"`
	FilesScanned int         `json:"filesScanned"`
	Violations   []Violation `json:"violations"`
}

type Violation struct {
	Path    string `json:"path"`
	Rule    string `json:"rule"`
	Pattern string `json:"pattern,omitempty"`
}

var forbiddenPatterns = []string{
	"Bearer",
	"accessToken",
	"refreshToken",
	"Authorization",
	"client_secret",
	"internal-api",
	"confluence.yeedev",
	"192.168.",
}

func Scan(root string) (Result, error) {
	result := Result{OK: true, Violations: []Violation{}}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "docs" {
				result.Violations = append(result.Violations, Violation{
					Path: path,
					Rule: "raw-docs",
				})
			}
			return nil
		}

		result.FilesScanned++
		if isRuntimeBinary(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, pattern := range forbiddenPatterns {
			if strings.Contains(content, pattern) {
				result.Violations = append(result.Violations, Violation{
					Path:    path,
					Rule:    "forbidden-sensitive-text",
					Pattern: pattern,
				})
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	result.OK = len(result.Violations) == 0
	return result, nil
}

func isRuntimeBinary(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/runtime/bin/yeelight-home-") || strings.HasPrefix(normalized, "runtime/bin/yeelight-home-")
}

func ScanAllowlist(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	content := string(data)
	result := Result{OK: true, FilesScanned: 1, Violations: []Violation{}}
	requiredIncludes := []string{
		"skill/yeelight-smart-home/SKILL.md",
		"skill/yeelight-smart-home/agents/openai.yaml",
		"skill/yeelight-smart-home/assets/intent-catalog.json",
		"skill/yeelight-smart-home/assets/catalog/*.json",
		"skill/yeelight-smart-home/assets/schemas/*.json",
		"skill/yeelight-smart-home/references/*.md",
		"skill/yeelight-smart-home/scripts/invoke",
		"skill/yeelight-smart-home/scripts/invoke.sh",
		"skill/yeelight-smart-home/scripts/invoke.ps1",
		"skill/yeelight-smart-home/scripts/runtime-manifest.json",
	}
	for _, include := range requiredIncludes {
		if !strings.Contains(content, include) {
			result.Violations = append(result.Violations, Violation{
				Path:    path,
				Rule:    "missing-required-allowlist-include",
				Pattern: include,
			})
		}
	}
	includeContent := allowlistSectionContent(content, "include:")
	for _, forbiddenInclude := range []string{"docs/", "specifications/", "tools/"} {
		if strings.Contains(includeContent, "- "+forbiddenInclude) {
			result.Violations = append(result.Violations, Violation{
				Path:    path,
				Rule:    "forbidden-allowlist-include",
				Pattern: forbiddenInclude,
			})
		}
	}
	for _, include := range allowlistEntries(includeContent) {
		if strings.HasPrefix(include, "runtime/") {
			result.Violations = append(result.Violations, Violation{
				Path:    path,
				Rule:    "forbidden-allowlist-include",
				Pattern: include,
			})
		}
	}
	requiredExcludes := []string{
		"docs/**",
		"specifications/**",
		"runtime/bin/**",
		"runtime/**/*.go",
		"tools/**",
	}
	for _, exclude := range requiredExcludes {
		if !strings.Contains(content, exclude) {
			result.Violations = append(result.Violations, Violation{
				Path:    path,
				Rule:    "missing-required-allowlist-exclude",
				Pattern: exclude,
			})
		}
	}
	scanContent := allowlistSensitiveScanContent(content)
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(scanContent, pattern) {
			result.Violations = append(result.Violations, Violation{
				Path:    path,
				Rule:    "forbidden-sensitive-text",
				Pattern: pattern,
			})
		}
	}
	result.OK = len(result.Violations) == 0
	return result, nil
}

func allowlistSectionContent(content string, section string) string {
	lines := strings.Split(content, "\n")
	capturing := false
	var sectionLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == section {
			capturing = true
			continue
		}
		if capturing && trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		if capturing {
			sectionLines = append(sectionLines, line)
		}
	}
	return strings.Join(sectionLines, "\n")
}

func allowlistEntries(sectionContent string) []string {
	lines := strings.Split(sectionContent, "\n")
	entries := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		entries = append(entries, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}
	return entries
}

func allowlistSensitiveScanContent(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	skipForbiddenPatternItems := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if trimmed == "forbiddenPatterns:" {
			skipForbiddenPatternItems = true
			filtered = append(filtered, line)
			continue
		}
		if skipForbiddenPatternItems {
			if trimmed == "" || indent > 2 {
				continue
			}
			skipForbiddenPatternItems = false
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}
