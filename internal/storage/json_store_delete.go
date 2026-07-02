package storage

import (
	"errors"
	"strings"
)

func (store JSONStore) DeletePreferencesByID(profile string, region string, houseID string, ids []string) ([]PreferenceRecord, error) {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(houseID) == "" {
		return nil, errors.New("profile and houseId are required")
	}
	wanted := idSet(ids)
	if len(wanted) == 0 {
		return nil, nil
	}
	region = normalizeStorageRegion(region)
	deleted := []PreferenceRecord{}
	err := store.mutateScope(profile, region, houseID, func(document *jsonDocument) error {
		kept := document.Preferences[:0]
		for _, record := range document.Preferences {
			if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID && wanted[record.ID] {
				deleted = append(deleted, record)
				continue
			}
			kept = append(kept, record)
		}
		document.Preferences = kept
		return nil
	})
	return deleted, err
}

func (store JSONStore) DeleteRecommendationsByID(profile string, region string, houseID string, ids []string) ([]RecommendationRecord, error) {
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(houseID) == "" {
		return nil, errors.New("profile and houseId are required")
	}
	wanted := idSet(ids)
	if len(wanted) == 0 {
		return nil, nil
	}
	region = normalizeStorageRegion(region)
	deleted := []RecommendationRecord{}
	err := store.mutateScope(profile, region, houseID, func(document *jsonDocument) error {
		kept := document.Recommendations[:0]
		for _, record := range document.Recommendations {
			if record.Profile == profile && sameStorageRegion(record.Region, region) && record.HouseID == houseID && wanted[record.ID] {
				deleted = append(deleted, record)
				continue
			}
			kept = append(kept, record)
		}
		document.Recommendations = kept
		return nil
	})
	return deleted, err
}

func idSet(ids []string) map[string]bool {
	result := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			result[id] = true
		}
	}
	return result
}
