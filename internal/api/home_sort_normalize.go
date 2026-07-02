package api

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

const (
	homeSortDeviceRoom = 1
	homeSortSceneRoom  = 2
	homeSortHomeRooms  = 3
	homeSortSubDevice  = 4
	homeSortAreaRooms  = 5

	homeSortGroupTypeRoom   = 1
	homeSortGroupTypeDevice = 2
	homeSortGroupTypeCustom = 3
	homeSortGroupTypeMesh   = 4
	homeSortGroupTypeScene  = 6
)

// NormalizeHomeSortPayload converts user-facing sort words into the backend's
// byte sort type and target fields. The backend rejects textual sort types.
func NormalizeHomeSortPayload(houseID string, source map[string]any) (map[string]any, string) {
	houseID = strings.TrimSpace(houseID)
	if source == nil {
		source = map[string]any{}
	}
	body := map[string]any{semantic.FieldHouseID: requestNumberOrStringForSort(houseID)}
	for _, key := range []string{semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType), semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID), semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldAreaID, semantic.FieldAreaName, semantic.FieldDeviceName, semantic.FieldSubIndex} {
		if value, ok := source[key]; ok {
			body[key] = value
		}
	}
	if items, ok := source[semantic.FieldItems]; ok {
		body[semantic.FieldItems] = items
	}
	sortType, typeProvided, typeValid := homeSortType(source)
	if typeProvided && !typeValid {
		return nil, "home_sort_type_invalid"
	}
	if !typeProvided {
		var ok bool
		sortType, ok = inferHomeSortType(source)
		if !ok {
			return body, ""
		}
	}
	target := homeSortTarget(houseID, sortType, source)
	if target == "" {
		return nil, "home_sort_target_missing"
	}
	body[semantic.FieldType] = sortType
	body[semantic.FieldTarget] = requestNumberOrStringForSort(target)
	if sortType == homeSortDeviceRoom || sortType == homeSortSceneRoom {
		body[semantic.FieldRoomID] = requestNumberOrStringForSort(target)
	}
	return body, ""
}

func homeSortType(source map[string]any) (int, bool, bool) {
	raw := firstHomeSortString(source, semantic.FieldType, semantic.FieldSortType)
	if raw == "" {
		return 0, false, false
	}
	if value, ok := intFromAnyForSort(raw); ok {
		return value, true, value >= homeSortDeviceRoom && value <= homeSortAreaRooms
	}
	value, ok := homeSortTypeFromAlias(raw)
	return value, true, ok
}

func inferHomeSortType(source map[string]any) (int, bool) {
	if roomID := firstHomeSortString(source, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName); roomID != "" {
		if itemType, ok := firstHomeSortItemType(source); ok && itemType == homeSortGroupTypeScene {
			return homeSortSceneRoom, true
		}
		return homeSortDeviceRoom, true
	}
	if areaID := firstHomeSortString(source, semantic.FieldAreaID, semantic.FieldAreaName); areaID != "" {
		return homeSortAreaRooms, true
	}
	if targetDeviceID := firstHomeSortString(source, semantic.FieldParentID, semantic.FieldTargetID, semantic.FieldDeviceName); targetDeviceID != "" {
		return homeSortSubDevice, true
	}
	if itemType, ok := firstHomeSortItemType(source); ok && itemType == homeSortGroupTypeRoom {
		return homeSortHomeRooms, true
	}
	return 0, false
}

func homeSortTarget(houseID string, sortType int, source map[string]any) string {
	switch sortType {
	case homeSortDeviceRoom, homeSortSceneRoom:
		return firstHomeSortString(source, semantic.FieldTarget, semantic.FieldRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName)
	case homeSortHomeRooms:
		if target := firstHomeSortString(source, semantic.FieldTarget, semantic.FieldHouseID); target != "" {
			return target
		}
		return houseID
	case homeSortSubDevice:
		return firstHomeSortString(source, semantic.FieldTarget, semantic.FieldParentID, semantic.FieldTargetID, semantic.FieldDeviceID, semantic.FieldDeviceName, semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID))
	case homeSortAreaRooms:
		return firstHomeSortString(source, semantic.FieldTarget, semantic.FieldAreaID, semantic.FieldAreaName)
	default:
		return ""
	}
}

func homeSortTypeFromAlias(value string) (int, bool) {
	compact := compactHomeSortToken(value)
	switch compact {
	case "deviceroom", "roomdevice", "roomdevices", "roomdevicegroup", "roomdevicegroups", "device", "devices", "group", "groups", "meshgroup", "meshgroups":
		return homeSortDeviceRoom, true
	case "sceneroom", "roomscene", "roomscenes", "scene", "scenes":
		return homeSortSceneRoom, true
	case "homeroom", "homerooms", "houseroom", "houserooms", "home", "house", "room", "rooms":
		return homeSortHomeRooms, true
	case "subdevice", "subdevices", "childdevice", "childdevices":
		return homeSortSubDevice, true
	case "arearoom", "arearooms", "regionroom", "regionrooms", "area", "region":
		return homeSortAreaRooms, true
	default:
		return 0, false
	}
}

func firstHomeSortItemType(source map[string]any) (int, bool) {
	if items, ok := source[semantic.FieldItems].([]any); ok {
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if typeID, ok := homeSortResourceType(item, true); ok {
				return typeID, true
			}
		}
	}
	return homeSortResourceType(source, false)
}

func homeSortResourceType(source map[string]any, allowTypeKey bool) (int, bool) {
	if value, ok := intFromAnyForSort(firstNonNilForSort(source[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)], source[semantic.FieldResourceTypeID])); ok {
		return value, true
	}
	keys := []string{semantic.FieldEntityType}
	if allowTypeKey {
		keys = append(keys, semantic.FieldType)
	}
	entityType := compactHomeSortToken(firstHomeSortString(source, keys...))
	switch entityType {
	case "room":
		return homeSortGroupTypeRoom, true
	case "device":
		return homeSortGroupTypeDevice, true
	case "group", "customgroup":
		return homeSortGroupTypeCustom, true
	case "meshgroup":
		return homeSortGroupTypeMesh, true
	case "scene":
		return homeSortGroupTypeScene, true
	default:
		return 0, false
	}
}

func firstHomeSortString(source map[string]any, keysOrValues ...string) string {
	for _, key := range keysOrValues {
		if value, ok := source[key]; ok {
			if text := strings.TrimSpace(stringFromAny(value)); text != "" {
				return text
			}
		}
	}
	return ""
}

func compactHomeSortToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "")
	return replacer.Replace(value)
}

func requestNumberOrStringForSort(value string) any {
	if parsed, ok := intFromAnyForSort(value); ok {
		return parsed
	}
	return strings.TrimSpace(value)
}

func intFromAnyForSort(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case string:
		var result int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &result); err != nil {
			return 0, false
		}
		return result, true
	default:
		return 0, false
	}
}

func firstNonNilForSort(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
