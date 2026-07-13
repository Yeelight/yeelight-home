package release

import (
	"bufio"
	"os"
	"strings"
)

type Allowlist struct {
	Includes []string
	Excludes []string
}

func ReadAllowlist(path string) (Allowlist, error) {
	file, err := os.Open(path)
	if err != nil {
		return Allowlist{}, err
	}
	defer file.Close()
	var result Allowlist
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		value := cleanAllowlistItem(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		if value == "" {
			continue
		}
		switch section {
		case "include":
			result.Includes = append(result.Includes, value)
		case "exclude":
			result.Excludes = append(result.Excludes, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return Allowlist{}, err
	}
	return result, nil
}

func cleanAllowlistItem(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'")
	return strings.TrimSpace(value)
}
