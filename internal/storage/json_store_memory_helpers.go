package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func safeStorageSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	segment := strings.Trim(builder.String(), ".")
	if segment == "" {
		return "default"
	}
	if len(segment) > 80 {
		return segment[:80]
	}
	return segment
}

func normalizeStorageRegion(region string) string {
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		return "default"
	}
	return region
}

func sameStorageRegion(left string, right string) bool {
	left = normalizeStorageRegion(left)
	right = normalizeStorageRegion(right)
	return left == right
}

func storageNamespace(profile string, region string, houseID string, dataType string) StorageNamespace {
	profile = strings.TrimSpace(profile)
	region = normalizeStorageRegion(region)
	houseID = strings.TrimSpace(houseID)
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		dataType = "memory"
	}
	return StorageNamespace{
		AccountProfile: profile,
		Profile:        profile,
		Region:         region,
		HouseID:        houseID,
		DataType:       dataType,
	}
}

func (namespace StorageNamespace) matches(profile string, region string, houseID string, dataType string) bool {
	if strings.TrimSpace(namespace.Profile) == "" {
		return true
	}
	return strings.TrimSpace(namespace.Profile) == strings.TrimSpace(profile) &&
		sameStorageRegion(namespace.Region, region) &&
		strings.TrimSpace(namespace.HouseID) == strings.TrimSpace(houseID) &&
		(strings.TrimSpace(namespace.DataType) == "" || strings.TrimSpace(namespace.DataType) == strings.TrimSpace(dataType))
}

func containsSensitiveKey(value string) bool {
	normalized := strings.ToLower(value)
	for _, forbidden := range []string{"token", "secret", "authorization", "cookie"} {
		if strings.Contains(normalized, forbidden) {
			return true
		}
	}
	return false
}

func normalizeMemoryKind(value string) string {
	switch strings.TrimSpace(value) {
	case "explicit":
		return "explicit"
	default:
		return "explicit"
	}
}

func normalizeMemoryStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "candidate", "rejected", "confirmed":
		return strings.TrimSpace(value)
	default:
		return "confirmed"
	}
}

func preferenceEquivalent(left PreferenceRecord, right PreferenceRecord) bool {
	if strings.TrimSpace(left.Profile) != strings.TrimSpace(right.Profile) ||
		!sameStorageRegion(left.Region, right.Region) ||
		strings.TrimSpace(left.HouseID) != strings.TrimSpace(right.HouseID) {
		return false
	}
	if normalizeMemoryText(left.ScopeType) != normalizeMemoryText(right.ScopeType) {
		return false
	}
	if normalizeMemoryText(left.ScopeRef) != normalizeMemoryText(right.ScopeRef) {
		return false
	}
	if normalizeMemoryText(left.PreferenceType) != normalizeMemoryText(right.PreferenceType) {
		return false
	}
	return normalizePreferenceValue(left.PreferenceValue) == normalizePreferenceValue(right.PreferenceValue)
}

func normalizeMemoryText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"，", " ", "。", " ", "、", " ", "；", " ", "：", " ",
		",", " ", ".", " ", ";", " ", ":", " ", "!", " ", "！", " ",
		"?", " ", "？", " ", "（", " ", "）", " ", "(", " ", ")", " ",
		"“", " ", "”", " ", "\"", " ", "'", " ",
	)
	normalized = replacer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func normalizePreferenceValue(value string) string {
	return normalizeMemoryText(value)
}

func mergeEvidence(existing string, incoming string) string {
	parts := dedupeEvidenceParts(existing, incoming)
	if len(parts) == 0 {
		return ""
	}
	merged := strings.Join(parts, "；")
	runes := []rune(merged)
	if len(runes) <= maxMergedEvidenceRunes {
		return merged
	}
	return string(runes[:maxMergedEvidenceRunes])
}

func dedupeEvidenceParts(values ...string) []string {
	parts := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, "；") {
			part = strings.TrimSpace(part)
			if part == "" || evidencePartExists(parts, part) {
				continue
			}
			parts = append(parts, part)
		}
	}
	return parts
}

func evidencePartExists(parts []string, incoming string) bool {
	normalizedIncoming := normalizeMemoryText(incoming)
	for _, existing := range parts {
		normalizedExisting := normalizeMemoryText(existing)
		if normalizedExisting == normalizedIncoming {
			return true
		}
		if strings.Contains(normalizedExisting, normalizedIncoming) || strings.Contains(normalizedIncoming, normalizedExisting) {
			return true
		}
	}
	return false
}

func preferenceStableID(record PreferenceRecord) string {
	key := strings.Join([]string{
		record.Profile,
		normalizeStorageRegion(record.Region),
		record.HouseID,
		record.ScopeType,
		record.ScopeRef,
		record.PreferenceType,
		record.PreferenceValue,
	}, "|")
	sum := sha1.Sum([]byte(key))
	return "mem-" + hex.EncodeToString(sum[:])[:16]
}
