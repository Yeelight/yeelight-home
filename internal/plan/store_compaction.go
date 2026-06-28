package plan

import (
	"sort"
	"time"
)

func (store Store) compact(records []Record) []Record {
	if len(records) == 0 {
		return records
	}
	now := store.clock()
	nowUnix := now.Unix()
	terminalCutoff := now.Add(-store.terminalRetentionDuration()).Unix()
	expiredPendingCutoff := now.Add(-store.expiredPendingRetentionDuration()).Unix()

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		switch record.Status {
		case StatusPending:
			if record.ExpiresAt > 0 && record.ExpiresAt < nowUnix && record.ExpiresAt < expiredPendingCutoff {
				continue
			}
		case StatusCommitted, StatusCanceled:
			if recordActivityAt(record) < terminalCutoff {
				continue
			}
		}
		filtered = append(filtered, record)
	}

	maxRecords := store.maxPlans()
	if maxRecords <= 0 || len(filtered) <= maxRecords {
		return filtered
	}

	ranked := append([]Record(nil), filtered...)
	sort.SliceStable(ranked, func(i int, j int) bool {
		leftActive := isActionablePending(ranked[i], nowUnix)
		rightActive := isActionablePending(ranked[j], nowUnix)
		if leftActive != rightActive {
			return leftActive
		}
		leftAt := recordActivityAt(ranked[i])
		rightAt := recordActivityAt(ranked[j])
		if leftAt != rightAt {
			return leftAt > rightAt
		}
		return ranked[i].ID > ranked[j].ID
	})

	keep := map[string]bool{}
	for index, record := range ranked {
		if index >= maxRecords {
			break
		}
		keep[record.ID] = true
	}

	result := make([]Record, 0, len(keep))
	for _, record := range filtered {
		if keep[record.ID] {
			result = append(result, record)
		}
	}
	return result
}

func (store Store) runBeforeCompact(records []Record) error {
	if store.beforeCompact == nil {
		return nil
	}
	snapshot := make([]Record, len(records))
	copy(snapshot, records)
	return store.beforeCompact(snapshot)
}

func (store Store) terminalRetentionDuration() time.Duration {
	if store.terminalRetention <= 0 {
		return DefaultTerminalRetention
	}
	return store.terminalRetention
}

func (store Store) expiredPendingRetentionDuration() time.Duration {
	if store.expiredPendingRetention <= 0 {
		return DefaultExpiredPendingRetention
	}
	return store.expiredPendingRetention
}

func (store Store) maxPlans() int {
	if store.maxStoredPlans == 0 {
		return DefaultMaxStoredPlans
	}
	return store.maxStoredPlans
}

func isActionablePending(record Record, nowUnix int64) bool {
	return record.Status == StatusPending && (record.ExpiresAt == 0 || record.ExpiresAt >= nowUnix)
}

func recordActivityAt(record Record) int64 {
	for _, value := range []int64{record.CommittedAt, record.CanceledAt, record.ApprovedAt, record.CreatedAt, record.ExpiresAt} {
		if value > 0 {
			return value
		}
	}
	return 0
}
