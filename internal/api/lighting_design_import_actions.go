package api

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func normalizeLightingDesignHouseMetaAreas(value any, index lightingDesignMetaIndex) ([]any, error) {
	items, ok := mapListFromAny(value)
	if !ok {
		return nil, nil
	}
	result := make([]any, 0, len(items))
	for areaIndex, item := range items {
		area := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(area, semantic.FieldName)) == "" {
			return nil, fmt.Errorf("areaList[].name is required")
		}
		if strings.TrimSpace(stringFromMap(area, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
			area[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("ar%d", areaIndex+1)
		}
		if strings.TrimSpace(stringFromMap(area, semantic.FieldIcon)) == "" {
			area[semantic.FieldIcon] = "area_1"
		}
		ids, ok := lightingDesignStringListFromAny(area[semantic.InternalField(semantic.DomainImport, semantic.FieldRoomKeys)])
		if !ok || len(ids) == 0 {
			return nil, fmt.Errorf("areaList[].roomTempIdList is required")
		}
		for _, roomTempID := range ids {
			if index.RoomsByTempID[roomTempID] == "" {
				return nil, fmt.Errorf("areaList[].roomTempIdList references unknown room tempId %q", roomTempID)
			}
		}
		area[semantic.InternalField(semantic.DomainImport, semantic.FieldRoomKeys)] = ids
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
		return nil, fmt.Errorf("scenes[] count exceeds limit %d", lightingDesignMaxScenes)
	}
	result := make([]any, 0, len(items))
	for sceneIndex, item := range items {
		scene := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(scene, semantic.FieldName)) == "" {
			return nil, fmt.Errorf("scenes[].name is required")
		}
		if strings.TrimSpace(stringFromMap(scene, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
			scene[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("sc%d", sceneIndex+1)
		}
		if strings.TrimSpace(stringFromMap(scene, semantic.FieldIcon)) == "" {
			scene[semantic.FieldIcon] = "scene_1"
		}
		details, ok := mapListFromAny(scene[semantic.FieldDetails])
		if !ok || len(details) == 0 {
			return nil, fmt.Errorf("scenes[].actions is required")
		}
		normalizedDetails := make([]any, 0, len(details))
		for detailIndex, detail := range details {
			normalized, err := normalizeLightingDesignHouseMetaAction(detail, detailIndex, true, *index)
			if err != nil {
				return nil, err
			}
			normalizedDetails = append(normalizedDetails, normalized)
		}
		scene[semantic.FieldName] = lightingDesignMetaName(stringFromMap(scene, semantic.FieldName))
		scene[semantic.FieldDetails] = normalizedDetails
		index.ScenesByTempID[stringFromMap(scene, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))] = stringFromMap(scene, semantic.FieldName)
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
		return nil, fmt.Errorf("automations[] count exceeds limit %d", lightingDesignMaxAutomations)
	}
	result := make([]any, 0, len(items))
	for automationIndex, item := range items {
		automation := copyLightingDesignDeepMap(item)
		if strings.TrimSpace(stringFromMap(automation, semantic.FieldName)) == "" {
			return nil, fmt.Errorf("automations[].name is required")
		}
		if strings.TrimSpace(stringFromMap(automation, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
			automation[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("at%d", automationIndex+1)
		}
		schedule, ok := semantic.AutomationScheduleFromRequest(automation)
		if !ok {
			return nil, fmt.Errorf("automations[].repeat must be daily, weekdays, weekend, once, custom, legal_holiday, or legal_workday")
		}
		automation[semantic.FieldStartTime] = schedule.StartTime
		automation[semantic.FieldEndTime] = schedule.EndTime
		automation[semantic.InternalRepeatTypeField()] = schedule.RepeatType
		if strings.TrimSpace(schedule.RepeatValue) != "" {
			automation[semantic.InternalRepeatValueField()] = schedule.RepeatValue
		}
		if _, ok := lightingDesignIntFromAny(automation[semantic.FieldVersion]); !ok {
			automation[semantic.FieldVersion] = 2
		}
		paramsSource := automation[semantic.InternalAutomationParamsField()]
		if paramsSource == nil {
			paramsSource = semantic.NormalizeAutomationParamsFromRequest(automation)
		}
		params, err := lightingDesignFormatConditionParams(paramsSource)
		if err != nil {
			return nil, err
		}
		paramsJSON, err := jsonString(params)
		if err != nil {
			return nil, fmt.Errorf("automations[].trigger or automations[].conditions must be valid JSON: %w", err)
		}
		automation[semantic.InternalAutomationParamsField()] = paramsJSON
		actions, ok := mapListFromAny(automation[semantic.FieldActions])
		if !ok || len(actions) == 0 {
			return nil, fmt.Errorf("automations[].actions is required")
		}
		normalizedActions := make([]any, 0, len(actions))
		for actionIndex, action := range actions {
			normalized, err := normalizeLightingDesignHouseMetaAction(action, actionIndex, false, index)
			if err != nil {
				return nil, err
			}
			normalizedActions = append(normalizedActions, normalized)
		}
		automation[semantic.FieldName] = lightingDesignMetaName(stringFromMap(automation, semantic.FieldName))
		automation[semantic.FieldActions] = normalizedActions
		delete(automation, semantic.FieldStatus)
		result = append(result, automation)
	}
	return result, nil
}

func normalizeLightingDesignHouseMetaAction(action map[string]any, index int, scene bool, metaIndex lightingDesignMetaIndex) (map[string]any, error) {
	result := semantic.NormalizeAction(action, semantic.ActionOptions{GroupTypeID: semantic.ResourceMeshGroup, IDField: semantic.InternalField(semantic.DomainImport, semantic.FieldKey)})
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)]); !ok {
		result = copyLightingDesignDeepMap(action)
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)]); !ok {
		return nil, fmt.Errorf("actions[].targetType is required")
	}
	typeID, _ := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType)])
	tempID := strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey)))
	if tempID == "" {
		return nil, fmt.Errorf("actions[].targetKey is required")
	}
	resName := strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)))
	if resName == "" {
		resName = lightingDesignResourceNameForTempID(typeID, tempID, metaIndex)
		if resName == "" {
			return nil, fmt.Errorf("actions[].targetName is required")
		}
		result[semantic.InternalField(semantic.DomainAction, semantic.FieldTargetName)] = resName
	}
	if !lightingDesignTempIDExists(typeID, tempID, metaIndex) {
		return nil, fmt.Errorf("actions[].targetKey %q does not match an imported resource for targetType %s", tempID, semantic.ResourceTypeName(typeID))
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.FieldRank]); !ok {
		result[semantic.FieldRank] = index
	}
	if scene {
		if _, ok := lightingDesignIntFromAny(result[semantic.FieldAction]); !ok {
			result[semantic.FieldAction] = 0
		}
	}
	params := result[semantic.InternalActionParamsField()]
	if params == nil {
		params = map[string]any{semantic.FieldDelay: 0, semantic.FieldSet: map[string]any{semantic.InternalField(semantic.DomainAction, semantic.FieldPower): true}}
	}
	formatted, err := lightingDesignFormatActionParams(params)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := jsonString(formatted)
	if err != nil {
		return nil, fmt.Errorf("actions[].set must be valid JSON: %w", err)
	}
	result[semantic.InternalActionParamsField()] = paramsJSON
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
	if existing, ok := source[semantic.FieldExtraMeta].(map[string]any); ok {
		for key, value := range existing {
			if text := lightingDesignStringFromAny(value); text != "" {
				extra[key] = text
			}
		}
	}
	for _, key := range []string{
		semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCode),
		semantic.FieldProductName,
		semantic.FieldProductBrand,
		semantic.FieldProductModel,
		semantic.FieldProductSKU,
		semantic.FieldProductSPU,
		semantic.FieldProductLine,
		semantic.FieldProductCategory,
		semantic.FieldProductLargeClass,
		semantic.FieldProductSmallClass,
		semantic.FieldProductShortName,
		semantic.FieldProductSeries,
		semantic.FieldBarcode,
		semantic.FieldBaseUnit,
		semantic.FieldProductDeclareNo,
		semantic.FieldProductDeclareName,
		semantic.FieldProductDeclareUnit,
		semantic.FieldSupportYeelightPro,
		semantic.FieldSupportHomeKit,
		semantic.FieldProductStatusName,
		semantic.FieldModelNo,
		semantic.FieldProductSaleType,
		semantic.FieldQuotationType,
		semantic.FieldProductTypeName,
		semantic.FieldCategory,
		semantic.FieldSeries,
		semantic.FieldNotes,
	} {
		value := stringFromMap(source, key)
		if value != "" {
			extra[key] = value
			delete(target, key)
		}
	}
	if len(extra) > 0 {
		target[semantic.FieldExtraMeta] = extra
	}
}

func lightingDesignFormatActionParams(value any) (any, error) {
	item, ok := mapFromAny(value)
	if !ok {
		return value, nil
	}
	result := copyLightingDesignDeepMap(item)
	set, ok := mapFromAny(result[semantic.FieldSet])
	if !ok {
		return result, nil
	}
	index, hasIndex := lightingDesignIntFromAny(set[semantic.FieldIndex])
	if !hasIndex {
		return result, nil
	}
	formattedSet := map[string]any{}
	for key, raw := range set {
		if key == semantic.FieldIndex {
			continue
		}
		formattedSet[fmt.Sprintf("%d-%s", index, key)] = raw
	}
	result[semantic.FieldSet] = formattedSet
	return result, nil
}

func lightingDesignFormatConditionParams(value any) (any, error) {
	normalized, err := lightingDesignFormatConditionNode(value)
	if err != nil {
		return nil, err
	}
	return lightingDesignEnsureAutomationV2Params(normalized), nil
}

func lightingDesignFormatConditionNode(value any) (any, error) {
	if value == nil {
		return map[string]any{
			semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType): "and",
			semantic.FieldConditions: []any{map[string]any{
				semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionKind): "alarm",
				semantic.InternalField(semantic.DomainAutomation, semantic.FieldTime):          "09:00:00",
			}},
		}, nil
	}
	item, ok := mapFromAny(value)
	if !ok {
		return value, nil
	}
	result := copyLightingDesignDeepMap(item)
	conditions, _ := mapListFromAny(result[semantic.FieldConditions])
	hasChildren := len(conditions) > 0
	if hasChildren {
		formatted := make([]any, 0, len(conditions))
		for _, condition := range conditions {
			child, err := lightingDesignFormatConditionNode(condition)
			if err != nil {
				return nil, err
			}
			formatted = append(formatted, child)
		}
		result[semantic.FieldConditions] = formatted
	}
	lightingDesignFormatAutomationConditionFields(result, hasChildren)
	index, hasIndex := lightingDesignIntFromAny(result[semantic.FieldIndex])
	prop := strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainAutomation, semantic.FieldProperty)))
	if hasIndex && prop != "" {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldProperty)] = fmt.Sprintf("%d-%s", index, prop)
		delete(result, semantic.FieldIndex)
	}
	return result, nil
}

func lightingDesignFormatAutomationConditionFields(result map[string]any, hasChildren bool) {
	if hasChildren {
		if conditionType := strings.TrimSpace(stringFromMap(result, semantic.FieldConditionType)); conditionType != "" {
			result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType)] = conditionType
		}
		delete(result, semantic.FieldConditionType)
	} else {
		if conditionKind := strings.TrimSpace(stringFromMap(result, semantic.FieldConditionKind)); conditionKind != "" {
			result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionKind)] = conditionKind
		}
		delete(result, semantic.FieldConditionKind)
	}
	lightingDesignMoveConditionValue(result, semantic.FieldTime, semantic.InternalField(semantic.DomainAutomation, semantic.FieldTime))
	lightingDesignMoveConditionValue(result, semantic.FieldTargetKey, semantic.InternalField(semantic.DomainAutomation, semantic.FieldTargetKey))
	if targetID := strings.TrimSpace(stringFromMap(result, semantic.FieldTargetID)); targetID != "" {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldTargetID)] = semantic.NumberOrString(targetID)
		delete(result, semantic.FieldTargetID)
	}
	if targetType := strings.TrimSpace(stringFromMap(result, semantic.FieldTargetType)); targetType != "" {
		if typeID, ok := semantic.TargetTypeID(targetType, semantic.ResourceMeshGroup); ok {
			result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldTargetType)] = typeID
		}
		delete(result, semantic.FieldTargetType)
	}
	lightingDesignMoveConditionValue(result, semantic.FieldCapabilityProductID, semantic.InternalField(semantic.DomainAutomation, semantic.FieldCapabilityProductID))
	lightingDesignMoveConditionValue(result, semantic.FieldEventID, semantic.InternalField(semantic.DomainAutomation, semantic.FieldEventID))
	lightingDesignMoveConditionValue(result, semantic.FieldEventArgs, semantic.InternalField(semantic.DomainAutomation, semantic.FieldEventArgs))
	if property := strings.TrimSpace(stringFromMap(result, semantic.FieldProperty)); property != "" {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldProperty)] = lightingDesignAutomationPropertyID(property)
		delete(result, semantic.FieldProperty)
	} else if property := strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainAutomation, semantic.FieldProperty))); property != "" {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldProperty)] = lightingDesignAutomationPropertyID(property)
	}
	delete(result, semantic.FieldTargetName)
}

func lightingDesignMoveConditionValue(result map[string]any, publicKey string, internalKey string) {
	value, ok := result[publicKey]
	if !ok {
		return
	}
	result[internalKey] = value
	if publicKey != internalKey {
		delete(result, publicKey)
	}
}

func lightingDesignAutomationPropertyID(value string) string {
	if id, ok := semantic.PropertyID(value); ok {
		return id
	}
	return strings.TrimSpace(value)
}

func lightingDesignEnsureAutomationV2Params(value any) any {
	item, ok := mapFromAny(value)
	if !ok || lightingDesignIsAutomationV2Params(item) {
		return value
	}
	conditions, ok := mapListFromAny(item[semantic.FieldConditions])
	if !ok || len(conditions) == 0 {
		return value
	}
	groups := make([]any, 0, 2)
	eventLike := make([]any, 0, len(conditions))
	facts := make([]any, 0, len(conditions))
	for _, condition := range conditions {
		if lightingDesignConditionHasChildren(condition) {
			groups = append(groups, lightingDesignNormalizeV2ConditionGroup(condition))
			continue
		}
		if lightingDesignConditionIsFact(condition) {
			facts = append(facts, condition)
			continue
		}
		eventLike = append(eventLike, condition)
	}
	if len(eventLike) > 0 {
		groups = append(groups, map[string]any{
			semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType): "or",
			semantic.FieldConditions: eventLike,
		})
	}
	if len(facts) > 0 {
		groups = append(groups, map[string]any{
			semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType): "and",
			semantic.FieldConditions: facts,
		})
	}
	if len(groups) == 0 {
		return value
	}
	return map[string]any{
		semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType): "and",
		semantic.FieldConditions: groups,
	}
}

func lightingDesignIsAutomationV2Params(item map[string]any) bool {
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(item, semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType))), "and") {
		return false
	}
	conditions, ok := mapListFromAny(item[semantic.FieldConditions])
	if !ok || len(conditions) == 0 {
		return false
	}
	for _, condition := range conditions {
		if !lightingDesignConditionHasChildren(condition) {
			return false
		}
		conditionType := strings.ToLower(strings.TrimSpace(stringFromMap(condition, semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType))))
		if conditionType != "and" && conditionType != "or" {
			return false
		}
	}
	return true
}

func lightingDesignNormalizeV2ConditionGroup(condition map[string]any) map[string]any {
	result := copyLightingDesignDeepMap(condition)
	conditions, _ := mapListFromAny(result[semantic.FieldConditions])
	if len(conditions) == 0 {
		return result
	}
	conditionType := strings.ToLower(strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType))))
	if lightingDesignConditionsContainEventLike(conditions) {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType)] = "or"
		return result
	}
	if conditionType != "and" && conditionType != "or" {
		result[semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType)] = "and"
	}
	return result
}

func lightingDesignConditionHasChildren(condition map[string]any) bool {
	conditions, ok := mapListFromAny(condition[semantic.FieldConditions])
	return ok && len(conditions) > 0
}

func lightingDesignConditionsContainEventLike(conditions []map[string]any) bool {
	for _, condition := range conditions {
		if lightingDesignConditionHasChildren(condition) {
			childConditions, _ := mapListFromAny(condition[semantic.FieldConditions])
			if lightingDesignConditionsContainEventLike(childConditions) {
				return true
			}
			continue
		}
		if !lightingDesignConditionIsFact(condition) {
			return true
		}
	}
	return false
}

func lightingDesignConditionIsFact(condition map[string]any) bool {
	conditionType := strings.ToLower(strings.TrimSpace(stringFromMap(condition, semantic.InternalField(semantic.DomainAutomation, semantic.FieldConditionType))))
	return conditionType == "fact"
}
