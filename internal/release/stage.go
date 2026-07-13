package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type StageResult struct {
	OK          bool           `json:"ok"`
	SourceRoot  string         `json:"sourceRoot"`
	OutputRoot  string         `json:"outputRoot"`
	FilesCopied int            `json:"filesCopied"`
	Files       []ArtifactFile `json:"files"`
	Scan        Result         `json:"scan"`
}

func Stage(sourceRoot string, allowlistPath string, outputRoot string) (StageResult, error) {
	allowlist, err := ReadAllowlist(allowlistPath)
	if err != nil {
		return StageResult{}, err
	}
	if len(allowlist.Includes) == 0 {
		return StageResult{}, fmt.Errorf("release allowlist has no include entries")
	}
	sourceRoot = filepath.Clean(sourceRoot)
	outputRoot = filepath.Clean(outputRoot)
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return StageResult{}, err
	}
	matches, err := expandIncludes(sourceRoot, allowlist)
	if err != nil {
		return StageResult{}, err
	}
	files := make([]ArtifactFile, 0, len(matches))
	for _, relativePath := range matches {
		sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
		targetPath := filepath.Join(outputRoot, filepath.FromSlash(relativePath))
		info, err := os.Stat(sourcePath)
		if err != nil {
			return StageResult{}, err
		}
		if info.IsDir() {
			continue
		}
		sum, err := copyReleaseFile(sourcePath, targetPath, info.Mode())
		if err != nil {
			return StageResult{}, err
		}
		files = append(files, ArtifactFile{
			Path:   relativePath,
			Size:   info.Size(),
			SHA256: sum,
		})
	}
	scan, err := Scan(outputRoot)
	if err != nil {
		return StageResult{}, err
	}
	return StageResult{
		OK:          scan.OK,
		SourceRoot:  sourceRoot,
		OutputRoot:  outputRoot,
		FilesCopied: len(files),
		Files:       files,
		Scan:        scan,
	}, nil
}

func expandIncludes(sourceRoot string, allowlist Allowlist) ([]string, error) {
	seen := map[string]bool{}
	for _, include := range allowlist.Includes {
		matches, err := expandPattern(sourceRoot, include)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("allowlist include matched no files: %s", include)
		}
		for _, relativePath := range matches {
			if excludedByAllowlist(relativePath, allowlist.Excludes) {
				return nil, fmt.Errorf("allowlist include is excluded: %s", relativePath)
			}
			seen[relativePath] = true
		}
	}
	result := make([]string, 0, len(seen))
	for relativePath := range seen {
		result = append(result, relativePath)
	}
	sort.Strings(result)
	return result, nil
}

func expandPattern(sourceRoot string, pattern string) ([]string, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return nil, nil
	}
	if strings.Contains(pattern, "**") {
		return walkPattern(sourceRoot, pattern)
	}
	if !strings.ContainsAny(pattern, "*?[") {
		path := filepath.Join(sourceRoot, filepath.FromSlash(pattern))
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return []string{pattern}, nil
	}
	paths, err := filepath.Glob(filepath.Join(sourceRoot, filepath.FromSlash(pattern)))
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return nil, err
		}
		result = append(result, filepath.ToSlash(relativePath))
	}
	return result, nil
}

func walkPattern(sourceRoot string, pattern string) ([]string, error) {
	result := []string{}
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if allowlistPatternMatch(pattern, relativePath) {
			result = append(result, relativePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}

func excludedByAllowlist(relativePath string, excludes []string) bool {
	for _, exclude := range excludes {
		if allowlistPatternMatch(exclude, relativePath) {
			return true
		}
	}
	return false
}

func allowlistPatternMatch(pattern string, relativePath string) bool {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	relativePath = filepath.ToSlash(strings.TrimSpace(relativePath))
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "**") {
		matched, _ := filepath.Match(pattern, relativePath)
		return matched
	}
	regex := regexp.QuoteMeta(pattern)
	regex = strings.ReplaceAll(regex, `\*\*`, `.*`)
	regex = strings.ReplaceAll(regex, `\*`, `[^/]*`)
	regex = "^" + regex + "$"
	matched, _ := regexp.MatchString(regex, relativePath)
	return matched
}

func copyReleaseFile(sourcePath string, targetPath string, mode os.FileMode) (string, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return "", err
	}
	defer target.Close()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(target, hash), source); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
