package storage

const secondsPerDay int64 = 24 * 60 * 60

func compactScopedDocument(document jsonDocument, now int64) jsonDocument {
	document = normalizeDocument(document)
	if now <= 0 {
		now = maxDocumentTimestamp(document)
	}
	if now <= 0 {
		return document
	}
	document.Recommendations = compactRecommendations(document.Recommendations, now)
	document.Signals = compactInteractionSignals(document.Signals, now)
	return document
}

func compactRecommendations(records []RecommendationRecord, now int64) []RecommendationRecord {
	cutoff := now - int64(DefaultRecommendationRetentionDays)*secondsPerDay
	result := make([]RecommendationRecord, 0, len(records))
	for _, record := range records {
		if isExpiredRecommendation(record, cutoff) {
			continue
		}
		result = append(result, record)
	}
	return result
}

func compactInteractionSignals(records []InteractionSignalRecord, now int64) []InteractionSignalRecord {
	cutoff := now - int64(DefaultInteractionRetentionDays)*secondsPerDay
	result := make([]InteractionSignalRecord, 0, len(records))
	for _, record := range records {
		if record.LastSeenAt > 0 && record.LastSeenAt < cutoff {
			continue
		}
		result = append(result, record)
	}
	return result
}

func isExpiredRecommendation(record RecommendationRecord, cutoff int64) bool {
	switch record.Status {
	case "accepted", "dismissed", "rejected":
		timestamp := firstPositiveInt64Storage(record.UpdatedAt, record.CreatedAt)
		return timestamp > 0 && timestamp < cutoff
	default:
		return false
	}
}

func maxDocumentTimestamp(document jsonDocument) int64 {
	var max int64
	for _, record := range document.Consents {
		max = maxInt64(max, record.UpdatedAt)
	}
	for _, record := range document.Preferences {
		max = maxInt64(max, record.UpdatedAt, record.CreatedAt)
	}
	for _, record := range document.Recommendations {
		max = maxInt64(max, record.UpdatedAt, record.CreatedAt, record.CooldownUntil, record.LastShownAt)
	}
	for _, record := range document.Signals {
		max = maxInt64(max, record.LastSeenAt, record.FirstSeenAt)
	}
	for _, record := range document.Lessons {
		max = maxInt64(max, record.UpdatedAt, record.CreatedAt, record.LastValidatedAt)
	}
	return max
}

func maxInt64(current int64, values ...int64) int64 {
	for _, value := range values {
		if value > current {
			current = value
		}
	}
	return current
}

func firstPositiveInt64Storage(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
