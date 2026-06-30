package api

import (
	"fmt"
	"strings"
)

func normalizeLightingDesignHouseMetaAreas(value any, index lightingDesignMetaIndex) ([]any, error) {
	items, ok := mapListFromAny(value)
	if !ok {
		return nil, nil
	}
	result := make([]any, 0, len(items))
	for areaIndex, item := range items {
		area := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(area, "name")) == "" {
			return nil, fmt.Errorf("areaList[].name is required")
		}
		if strings.TrimSpace(stringFromMap(area, "tempId")) == "" {
			area["tempId"] = fmt.Sprintf("ar%d", areaIndex+1)
		}
		if strings.TrimSpace(stringFromMap(area, "icon")) == "" {
			area["icon"] = "area_1"
		}
		ids, ok := lightingDesignStringListFromAny(area["roomTempIdList"])
		if !ok || len(ids) == 0 {
			return nil, fmt.Errorf("areaList[].roomTempIdList is required")
		}
		for _, roomTempID := range ids {
			if index.RoomsByTempID[roomTempID] == "" {
				return nil, fmt.Errorf("areaList[].roomTempIdList references unknown room tempId %q", roomTempID)
			}
		}
		area["roomTempIdList"] = ids
		result = append(result, area)
	}
	return result, nil
}

func normalizeLightingDesignHouseMetaScenes(value any, index *lightingDesignMetaIndex) ([]any, error) {
	items, ok := mapListFromAny(value)
	if !ok {
		return nil, nil
	}
	if len(items) > lightingDesignMaxScenes {
		return nil, fmt.Errorf("sceneList count exceeds limit %d", lightingDesignMaxScenes)
	}
	result := make([]any, 0, len(items))
	for sceneIndex, item := range items {
		scene := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(scene, "name")) == "" {
			return nil, fmt.Errorf("sceneList[].name is required")
		}
		if strings.TrimSpace(stringFromMap(scene, "tempId")) == "" {
			scene["tempId"] = fmt.Sprintf("sc%d", sceneIndex+1)
		}
		if strings.TrimSpace(stringFromMap(scene, "icon")) == "" {
			scene["icon"] = "scene_1"
		}
		details, ok := mapListFromAny(scene["details"])
		if !ok || len(details) == 0 {
			return nil, fmt.Errorf("sceneList[].details is required")
		}
		normalizedDetails := make([]any, 0, len(details))
		for detailIndex, detail := range details {
			normalized, err := normalizeLightingDesignHouseMetaAction(detail, detailIndex, true, *index)
			if err != nil {
				return nil, err
			}
			normalizedDetails = append(normalizedDetails, normalized)
		}
		scene["name"] = lightingDesignMetaName(stringFromMap(scene, "name"))
		scene["details"] = normalizedDetails
		index.ScenesByTempID[stringFromMap(scene, "tempId")] = stringFromMap(scene, "name")
		result = append(result, scene)
	}
	return result, nil
}

func normalizeLightingDesignHouseMetaAutomations(value any, index lightingDesignMetaIndex) ([]any, error) {
	items, ok := mapListFromAny(value)
	if !ok {
		return nil, nil
	}
	if len(items) > lightingDesignMaxAutomations {
		return nil, fmt.Errorf("automationList count exceeds limit %d", lightingDesignMaxAutomations)
	}
	result := make([]any, 0, len(items))
	for automationIndex, item := range items {
		automation := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(automation, "name")) == "" {
			return nil, fmt.Errorf("automationList[].name is required")
		}
		if strings.TrimSpace(stringFromMap(automation, "tempId")) == "" {
			automation["tempId"] = fmt.Sprintf("at%d", automationIndex+1)
		}
		if strings.TrimSpace(stringFromMap(automation, "startTime")) == "" {
			automation["startTime"] = "00:00:00"
		}
		if strings.TrimSpace(stringFromMap(automation, "endTime")) == "" {
			automation["endTime"] = "23:59:59"
		}
		if _, ok := lightingDesignIntFromAny(automation["repeatType"]); !ok {
			automation["repeatType"] = 2
		}
		if strings.TrimSpace(stringFromMap(automation, "repeatValue")) == "" {
			automation["repeatValue"] = "0x7f"
		}
		if _, ok := lightingDesignIntFromAny(automation["version"]); !ok {
			automation["version"] = 2
		}
		params, err := lightingDesignFormatConditionParams(automation["params"])
		if err != nil {
			return nil, err
		}
		paramsJSON, err := jsonString(params)
		if err != nil {
			return nil, fmt.Errorf("automationList[].params must be JSON: %w", err)
		}
		automation["params"] = paramsJSON
		actions, ok := mapListFromAny(automation["actions"])
		if !ok || len(actions) == 0 {
			return nil, fmt.Errorf("automationList[].actions is required")
		}
		normalizedActions := make([]any, 0, len(actions))
		for actionIndex, action := range actions {
			normalized, err := normalizeLightingDesignHouseMetaAction(action, actionIndex, false, index)
			if err != nil {
				return nil, err
			}
			normalizedActions = append(normalizedActions, normalized)
		}
		automation["name"] = lightingDesignMetaName(stringFromMap(automation, "name"))
		automation["actions"] = normalizedActions
		delete(automation, "status")
		result = append(result, automation)
	}
	return result, nil
}

func normalizeLightingDesignHouseMetaAction(action map[string]any, index int, scene bool, metaIndex lightingDesignMetaIndex) (map[string]any, error) {
	result := copyLightingDesignDeepMap(action)
	if _, ok := lightingDesignIntFromAny(result["typeId"]); !ok {
		return nil, fmt.Errorf("action typeId is required")
	}
	typeID, _ := lightingDesignIntFromAny(result["typeId"])
	tempID := strings.TrimSpace(stringFromMap(result, "tempId"))
	if tempID == "" {
		return nil, fmt.Errorf("action tempId is required")
	}
	resName := strings.TrimSpace(stringFromMap(result, "resName"))
	if resName == "" {
		resName = lightingDesignResourceNameForTempID(typeID, tempID, metaIndex)
		if resName == "" {
			return nil, fmt.Errorf("action resName is required")
		}
		result["resName"] = resName
	}
	if !lightingDesignTempIDExists(typeID, tempID, metaIndex) {
		return nil, fmt.Errorf("action tempId %q does not match an imported resource for typeId %d", tempID, typeID)
	}
	if _, ok := lightingDesignIntFromAny(result["rank"]); !ok {
		result["rank"] = index
	}
	if scene {
		if _, ok := lightingDesignIntFromAny(result["action"]); !ok {
			result["action"] = 0
		}
	}
	params := result["params"]
	if params == nil {
		params = map[string]any{"delay": 0, "set": map[string]any{"p": true}}
	}
	formatted, err := lightingDesignFormatActionParams(params)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := jsonString(formatted)
	if err != nil {
		return nil, fmt.Errorf("action params must be JSON: %w", err)
	}
	result["params"] = paramsJSON
	return result, nil
}

func lightingDesignTempIDExists(typeID int, tempID string, index lightingDesignMetaIndex) bool {
	switch typeID {
	case 1:
		return index.RoomsByTempID[tempID] != ""
	case 2:
		return index.DevicesByTempID[tempID] != ""
	case 4:
		return index.GroupsByTempID[tempID] != ""
	case 5:
		return tempID != ""
	case 6:
		return index.ScenesByTempID[tempID] != ""
	default:
		return tempID != ""
	}
}

func lightingDesignResourceNameForTempID(typeID int, tempID string, index lightingDesignMetaIndex) string {
	switch typeID {
	case 1:
		return index.RoomsByTempID[tempID]
	case 2:
		return index.DevicesByTempID[tempID]
	case 4:
		return index.GroupsByTempID[tempID]
	case 5:
		return "家庭"
	case 6:
		return index.ScenesByTempID[tempID]
	default:
		return ""
	}
}

func lightingDesignMergeExtraMeta(target map[string]any, source map[string]any) {
	extra := map[string]string{}
	if existing, ok := source["extraMeta"].(map[string]any); ok {
		for key, value := range existing {
			if text := lightingDesignStringFromAny(value); text != "" {
				extra[key] = text
			}
		}
	}
	for _, key := range []string{
		"materialCode",
		"productName",
		"productBrand",
		"productModel",
		"productSku",
		"productSpu",
		"productLine",
		"productCategoryName",
		"productLargeClass",
		"productSmallClass",
		"productShortName",
		"productSeries",
		"barcode",
		"baseUnit",
		"productDeclareNo",
		"productDeclareName",
		"productDeclareUnit",
		"isSupportYeelightPro",
		"isSupportHomekit",
		"productStatusName",
		"modelNo",
		"productSaleTypeName",
		"quotationTypeDesc",
		"productTypeName",
		"category",
		"series",
		"notes",
	} {
		value := stringFromMap(source, key)
		if value != "" {
			extra[key] = value
			delete(target, key)
		}
	}
	if len(extra) > 0 {
		target["extraMeta"] = extra
	}
}

func lightingDesignFormatActionParams(value any) (any, error) {
	item, ok := mapFromAny(value)
	if !ok {
		return value, nil
	}
	result := copyLightingDesignDeepMap(item)
	set, ok := mapFromAny(result["set"])
	if !ok {
		return result, nil
	}
	index, hasIndex := lightingDesignIntFromAny(set["index"])
	if !hasIndex {
		return result, nil
	}
	formattedSet := map[string]any{}
	for key, raw := range set {
		if key == "index" {
			continue
		}
		formattedSet[fmt.Sprintf("%d-%s", index, key)] = raw
	}
	result["set"] = formattedSet
	return result, nil
}

func lightingDesignFormatConditionParams(value any) (any, error) {
	if value == nil {
		return map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "alarm", "clock": "09:00:00"}}}, nil
	}
	item, ok := mapFromAny(value)
	if !ok {
		return value, nil
	}
	result := copyLightingDesignDeepMap(item)
	conditions, _ := mapListFromAny(result["conditions"])
	if len(conditions) > 0 {
		formatted := make([]any, 0, len(conditions))
		for _, condition := range conditions {
			child, err := lightingDesignFormatConditionParams(condition)
			if err != nil {
				return nil, err
			}
			formatted = append(formatted, child)
		}
		result["conditions"] = formatted
	}
	index, hasIndex := lightingDesignIntFromAny(result["index"])
	prop := strings.TrimSpace(stringFromMap(result, "prop"))
	if hasIndex && prop != "" {
		result["prop"] = fmt.Sprintf("%d-%s", index, prop)
		delete(result, "index")
	}
	return result, nil
}
