package semantic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type AutomationSchedule struct {
	StartTime   string
	EndTime     string
	RepeatType  int
	RepeatValue string
}

type ActionOptions struct {
	GroupTypeID int
	IDField     string
}

func NormalizeSceneActions(value any) ([]map[string]any, bool) {
	return NormalizeActionRows(value, ActionOptions{GroupTypeID: ResourceMeshGroup})
}

func NormalizeAutomationActions(value any) ([]map[string]any, bool) {
	return NormalizeActionRows(value, ActionOptions{GroupTypeID: ResourceMeshGroup})
}

func NormalizeImportActions(value any) ([]map[string]any, bool) {
	return NormalizeActionRows(value, ActionOptions{GroupTypeID: ResourceMeshGroup, IDField: internalTempID})
}

func NormalizePanelActions(value any) ([]any, bool) {
	rows, ok := mapList(value)
	if !ok {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, NormalizePanelAction(row))
	}
	return result, true
}

func NormalizePanelAction(source map[string]any) map[string]any {
	row := NormalizeAction(source, ActionOptions{GroupTypeID: ResourceMeshGroup})
	for _, key := range []string{FieldID, FieldIndex, FieldConfigType, FieldModel, FieldAlias, FieldType, FieldProperty, FieldValue} {
		if row[key] == nil {
			if value, ok := FirstPresent(source, key); ok {
				row[key] = value
			}
		}
	}
	if row[internalResourceType] == nil {
		if typeID, ok := Int(row[internalTypeID]); ok {
			row[internalResourceType] = typeID
		}
	}
	if row[internalParam] == nil {
		if value, ok := row[internalParams]; ok {
			row[internalParam] = value
		}
	}
	if row[internalParams] != nil {
		delete(row, internalParams)
	}
	if mode := FirstString(source, FieldMode, "actionMode"); mode != "" && row[FieldMode] == nil {
		row[FieldMode] = mode
	}
	if row[internalSensitivity] == nil {
		if value, ok := FirstPresent(source, FieldSensitivity); ok {
			row[internalSensitivity] = value
		}
	}
	return row
}

func NormalizeTargetBinding(source map[string]any, groupTypeID int, typeField string) map[string]any {
	result := publicPassthrough(source, FieldID, FieldDeviceID, FieldName, FieldAlias, FieldKeyValue, FieldIndex, FieldVisible, FieldIcon, FieldSort, FieldType, FieldExtend, FieldRank, FieldStartTime, FieldEndTime, FieldAction, FieldProperty, FieldValue, FieldDelay, FieldDuration)
	if typeField == "" {
		typeField = internalTypeID
	}
	if targetType := FirstString(source, actionAliasConfig.TargetTypes...); targetType != "" {
		if typeID, ok := TargetTypeID(targetType, groupTypeID); ok {
			result[typeField] = typeID
		}
	}
	if targetID := FirstString(source, actionAliasConfig.TargetIDs...); targetID != "" {
		result[internalResourceID] = NumberOrString(targetID)
	}
	if targetName := FirstString(source, actionAliasConfig.TargetNames...); targetName != "" {
		result[internalResourceName] = targetName
	}
	removeActionAliases(result)
	return result
}

func NormalizeActionRows(value any, options ActionOptions) ([]map[string]any, bool) {
	rows, ok := mapList(value)
	if !ok {
		return nil, false
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, NormalizeAction(row, options))
	}
	return result, true
}

func NormalizeAction(source map[string]any, options ActionOptions) map[string]any {
	result := publicPassthrough(source, FieldAction, FieldRank, FieldRoomID, FieldStartTime, FieldEndTime)
	if options.GroupTypeID == 0 {
		options.GroupTypeID = ResourceMeshGroup
	}
	if options.IDField == "" {
		options.IDField = internalResourceID
	}
	if targetType := FirstString(source, actionAliasConfig.TargetTypes...); targetType != "" {
		if typeID, ok := TargetTypeID(targetType, options.GroupTypeID); ok {
			result[internalTypeID] = typeID
		}
	}
	if options.IDField == internalTempID {
		if targetKey := FirstString(source, actionAliasConfig.TargetKeys...); targetKey != "" {
			result[internalTempID] = targetKey
		}
	} else if targetID := FirstString(source, actionAliasConfig.TargetIDs...); targetID != "" {
		result[options.IDField] = NumberOrString(targetID)
	}
	if targetName := FirstString(source, actionAliasConfig.TargetNames...); targetName != "" {
		result[internalResourceName] = targetName
	}
	if value, ok := FirstPresent(source, actionAliasConfig.SubIndexes...); ok {
		result[internalSubIndex] = value
	}
	if params, ok := ActionParamsFromRow(source); ok {
		result[internalParams] = params
	}
	removeActionAliases(result)
	return result
}

func ActionParamsFromRow(source map[string]any) (any, bool) {
	if value, ok := FirstPresent(source, FieldCustom); ok {
		custom, valid := value.(map[string]any)
		if !valid || len(custom) == 0 {
			return nil, false
		}
		return deepMap(custom), true
	}
	if value, ok := FirstPresent(source, actionAliasConfig.ActionParams...); ok {
		return NormalizeLightParams(value), true
	}
	if value, ok := FirstPresent(source, actionAliasConfig.ActionSets...); ok {
		return NormalizeLightParams(lightParamsFromActionSource(source, value)), true
	}
	if set := lightSetFromFields(source); len(set) > 0 {
		return NormalizeLightParams(lightParamsFromActionSource(source, set)), true
	}
	return nil, false
}

func TargetTypeID(value string, groupTypeID int) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "room":
		return ResourceRoom, true
	case "device", "deviceslot", "device_slot", "slot":
		return ResourceDevice, true
	case "group", "meshgroup", "mesh_group":
		if groupTypeID == 0 {
			return ResourceMeshGroup, true
		}
		return groupTypeID, true
	case "customgroup", "custom_group":
		return ResourceCustomGroup, true
	case "home", "house":
		return ResourceHome, true
	case "scene":
		return ResourceScene, true
	case "automation":
		return ResourceAutomation, true
	default:
		return 0, false
	}
}

func NormalizeLightParams(value any) any {
	params, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := publicPassthrough(params, actionParamRowKeys()...)
	if set, ok := params[FieldSet].(map[string]any); ok {
		result[FieldSet] = NormalizeLightSet(set)
	} else if set := lightSetFromFields(params); len(set) > 0 {
		result[FieldSet] = set
	}
	return result
}

func NormalizeLightSet(source map[string]any) map[string]any {
	result := map[string]any{}
	for canonical, aliases := range map[string][]string{
		internalPower:            {FieldPower},
		internalBrightness:       {FieldBrightness},
		internalColorTemperature: {FieldColorTemperature},
		internalColor:            {FieldColor},
		internalTargetPercent:    {FieldTargetPercent},
		internalSwitchPower:      {FieldSwitchPower},
	} {
		for _, alias := range aliases {
			if value, ok := source[alias]; ok {
				result[canonical] = value
				break
			}
		}
	}
	for _, key := range actionAliasConfig.DirectLightKeys {
		delete(result, key)
	}
	return result
}

func NormalizeAutomationParamsFromRequest(parameters map[string]any) any {
	if value, ok := FirstPresent(parameters, FieldTrigger); ok {
		return NormalizeAutomationParams(value)
	}
	if conditions, ok := FirstPresent(parameters, FieldConditions); ok {
		operator := FirstString(parameters, actionAliasConfig.ConditionAliases...)
		if operator == "" {
			operator = "and"
		}
		return NormalizeAutomationParams(map[string]any{
			internalConditionType: operator,
			FieldConditions:       conditions,
		})
	}
	if time := FirstString(parameters, FieldTime); time != "" {
		return NormalizeAutomationParams(map[string]any{
			FieldConditionKind: "alarm",
			FieldTime:          time,
		})
	}
	return NormalizeAutomationParams(map[string]any{
		FieldConditionKind: "alarm",
		FieldTime:          "00:00:00",
	})
}

func NormalizeAutomationParams(value any) any {
	params, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := map[string]any{}
	if operator := FirstString(params, actionAliasConfig.ConditionAliases...); operator != "" {
		result[internalConditionType] = operator
	}
	if rows, ok := params[FieldConditions].([]any); ok {
		normalized := make([]any, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				normalized = append(normalized, NormalizeAutomationCondition(item))
			} else {
				normalized = append(normalized, row)
			}
		}
		result[FieldConditions] = normalized
	} else if trigger := triggerCondition(params); trigger != nil {
		result[FieldConditions] = []any{trigger}
	}
	if result[internalConditionType] == nil && result[FieldConditions] != nil {
		result[internalConditionType] = "and"
	}
	return result
}

func AutomationScheduleFromRequest(parameters map[string]any) (AutomationSchedule, bool) {
	start, end := activeWindowFromRequest(parameters)
	repeatType, repeatValue, ok := repeatFromRequest(parameters)
	return AutomationSchedule{
		StartTime:   firstNonEmpty(start, "00:00:00"),
		EndTime:     firstNonEmpty(end, "23:59:59"),
		RepeatType:  repeatType,
		RepeatValue: repeatValue,
	}, ok
}

func activeWindowFromRequest(parameters map[string]any) (string, string) {
	start := FirstString(parameters, FieldStartTime)
	end := FirstString(parameters, FieldEndTime)
	if window, ok := parameters[FieldActiveWindow].(map[string]any); ok {
		if start == "" {
			start = FirstString(window, FieldStart, FieldStartTime)
		}
		if end == "" {
			end = FirstString(window, FieldEnd, FieldEndTime)
		}
	}
	return start, end
}

func repeatFromRequest(parameters map[string]any) (int, string, bool) {
	if repeat, ok := parameters[FieldRepeat].(map[string]any); ok {
		return repeatFromObject(repeat)
	}
	if repeat := FirstString(parameters, FieldRepeat); repeat != "" {
		return repeatPreset(repeat)
	}
	if days := stringList(parameters[FieldRepeatDays]); len(days) > 0 {
		return 4, repeatDaysValue(days), true
	}
	return 2, "0x7f", true
}

func repeatFromObject(repeat map[string]any) (int, string, bool) {
	if days := stringList(repeat[FieldRepeatDays]); len(days) > 0 {
		return 4, repeatDaysValue(days), true
	}
	if value := FirstString(repeat, FieldType, FieldName); value != "" {
		return repeatPreset(value)
	}
	return 2, "0x7f", true
}

func repeatPreset(value string) (int, string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "daily", "everyday", "every_day", "每天", "每日":
		return 2, "0x7f", true
	case "weekday", "weekdays", "workday", "workdays", "工作日", "周一到周五":
		return 3, "0x3e", true
	case "weekend", "weekends", "周末":
		return 5, "0x41", true
	case "once", "one_time", "仅一次", "一次":
		return 1, "", true
	case "custom", "自定义":
		return 4, "", true
	case "legal_holiday", "legal_holidays", "holiday", "holidays", "法定节假日":
		return 6, "", true
	case "legal_workday", "legal_workdays", "法定工作日":
		return 7, "", true
	default:
		return 0, "", false
	}
}

func repeatDaysValue(days []string) string {
	mask := 0
	for _, day := range days {
		switch strings.ToLower(strings.TrimSpace(day)) {
		case "sun", "sunday", "0", "7", "周日", "星期日", "星期天", "礼拜日", "礼拜天":
			mask |= 1 << 0
		case "mon", "monday", "1", "周一", "星期一", "礼拜一":
			mask |= 1 << 1
		case "tue", "tuesday", "2", "周二", "星期二", "礼拜二":
			mask |= 1 << 2
		case "wed", "wednesday", "3", "周三", "星期三", "礼拜三":
			mask |= 1 << 3
		case "thu", "thursday", "4", "周四", "星期四", "礼拜四":
			mask |= 1 << 4
		case "fri", "friday", "5", "周五", "星期五", "礼拜五":
			mask |= 1 << 5
		case "sat", "saturday", "6", "周六", "星期六", "礼拜六":
			mask |= 1 << 6
		}
	}
	if mask == 0 {
		return ""
	}
	return fmt.Sprintf("0x%x", mask)
}

func triggerCondition(params map[string]any) map[string]any {
	condition := NormalizeAutomationCondition(params)
	if len(condition) == 0 {
		return nil
	}
	if condition[internalConditionKind] == nil {
		condition[internalConditionKind] = "alarm"
	}
	return condition
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func NormalizeAutomationCondition(source map[string]any) map[string]any {
	result := map[string]any{}
	rows, hasRows := source[FieldConditions].([]any)
	if hasRows {
		if operator := FirstString(source, actionAliasConfig.ConditionAliases...); operator != "" {
			result[internalConditionType] = operator
		} else if kind := FirstString(source, actionAliasConfig.ConditionKinds...); kind == "and" || kind == "or" {
			result[internalConditionType] = kind
		}
	} else if kind := FirstString(source, actionAliasConfig.ConditionKinds...); kind != "" {
		result[internalConditionKind] = kind
	}
	if value, ok := FirstPresent(source, actionAliasConfig.TimeAliases...); ok {
		result[internalClock] = value
	}
	if targetID := FirstString(source, FieldTargetID); targetID != "" {
		result[internalResourceID] = NumberOrString(targetID)
	}
	if targetKey := FirstString(source, FieldTargetKey); targetKey != "" {
		result[internalTempID] = targetKey
	}
	if targetType := FirstString(source, FieldTargetType); targetType != "" {
		if typeID, ok := TargetTypeID(targetType, ResourceMeshGroup); ok {
			result[internalTypeID] = typeID
		}
	}
	if value, ok := FirstPresent(source, FieldCapabilityProductID, internalProductID); ok {
		result[internalProductID] = deepValue(value)
	}
	if value, ok := FirstPresent(source, FieldEventID, FieldID); ok {
		result[FieldID] = deepValue(value)
	}
	if value, ok := FirstPresent(source, FieldEventArgs, internalEventArgs); ok {
		result[internalEventArgs] = deepValue(value)
	}
	if property := FirstString(source, FieldProperty); property != "" {
		result[internalProperty] = automationPropertyID(property)
	} else if property := FirstString(source, internalProperty); property != "" {
		result[internalProperty] = automationPropertyID(property)
	}
	if operation := FirstString(source, FieldOperation); operation != "" {
		result[FieldOperation] = operation
	}
	if value, ok := source[FieldValue]; ok {
		result[FieldValue] = deepValue(value)
	}
	if hasRows {
		normalized := make([]any, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				normalized = append(normalized, NormalizeAutomationCondition(item))
			} else {
				normalized = append(normalized, row)
			}
		}
		result[FieldConditions] = normalized
	}
	return result
}

func automationPropertyID(value string) string {
	if id, ok := PropertyID(value); ok {
		return id
	}
	return strings.TrimSpace(value)
}

func LightPropertyID(value string) (string, bool) {
	id := lightPropertyIDs[strings.TrimSpace(value)]
	if id != "" {
		return id, true
	}
	id = lightPropertyIDs[strings.ToLower(strings.TrimSpace(value))]
	return id, id != ""
}

func LightPropertyName(value string) string {
	if id, ok := LightPropertyID(value); ok {
		if name := lightPropertyNames[id]; name != "" {
			return name
		}
		return id
	}
	return strings.TrimSpace(value)
}

func FirstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text := String(value); text != "" {
				return text
			}
		}
	}
	return ""
}

func FirstPresent(values map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != nil {
			return value, true
		}
	}
	return nil, false
}

func String(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	}
	return ""
}

func Int(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func NumberOrString(value string) any {
	if parsed, ok := Int(value); ok {
		return parsed
	}
	return value
}

func lightSetFromFields(source map[string]any) map[string]any {
	set := map[string]any{}
	for canonical, aliases := range map[string][]string{
		internalPower:            {FieldPower},
		internalBrightness:       {FieldBrightness},
		internalColorTemperature: {FieldColorTemperature},
		internalColor:            {FieldColor},
		internalTargetPercent:    {FieldTargetPercent},
		internalSwitchPower:      {FieldSwitchPower},
	} {
		for _, alias := range aliases {
			if value, ok := source[alias]; ok {
				set[canonical] = value
				break
			}
		}
	}
	return set
}

func ToPublicAction(source map[string]any) map[string]any {
	result := map[string]any{}
	if targetType := ResourceTypeName(source[internalTypeID]); targetType != "" {
		result[FieldTargetType] = targetType
	}
	if value, ok := source[internalResourceID]; ok {
		result[FieldTargetID] = value
	}
	if value, ok := source[internalTempID]; ok {
		result[FieldTargetKey] = value
	}
	if value, ok := source[internalResourceName]; ok {
		result[FieldTargetName] = value
	}
	for _, key := range []string{FieldAction, FieldRank, FieldRoomID, FieldStartTime, FieldEndTime} {
		if value, ok := source[key]; ok {
			result[key] = deepValue(value)
		}
	}
	if value, ok := source[internalSubIndex]; ok {
		result[FieldSubIndex] = deepValue(value)
	}
	if params, ok := source[internalParams]; ok {
		MergePublicActionParams(result, params)
	}
	return result
}

func MergePublicActionParams(target map[string]any, value any) {
	if encoded, ok := value.(string); ok {
		var decoded map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(encoded)), &decoded) != nil {
			return
		}
		value = decoded
	}
	params, ok := ToPublicLightParams(value).(map[string]any)
	if !ok {
		return
	}
	if set, ok := params[FieldSet]; ok {
		target[FieldSet] = set
		delete(params, FieldSet)
	}
	for _, key := range actionParamRowKeys() {
		if item, ok := params[key]; ok {
			target[key] = deepValue(item)
			delete(params, key)
		}
	}
	for _, key := range []string{FieldAction, FieldToggle, FieldAdjust, FieldFlow, FieldCustom} {
		if item, ok := params[key]; ok {
			target[key] = deepValue(item)
			delete(params, key)
		}
	}
	if len(params) > 0 {
		target[FieldCustom] = deepMap(params)
	}
}

func ToPublicLightParams(value any) any {
	params, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := deepMap(params)
	if set, ok := result[FieldSet].(map[string]any); ok {
		result[FieldSet] = ToPublicLightSet(set)
	}
	return result
}

func ToPublicLightSet(source map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range source {
		publicKey, ok := publicSetPropertyName(key)
		if !ok {
			continue
		}
		result[publicKey] = deepValue(value)
	}
	return result
}

func publicSetPropertyName(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" || PropertySensitive(key) {
		return "", false
	}
	if propertyID, ok := PropertyID(key); ok {
		return PropertyName(propertyID), true
	}
	if index, property, ok := splitIndexedProperty(key); ok {
		if PropertySensitive(property) {
			return "", false
		}
		if propertyID, ok := PropertyID(property); ok {
			return index + "-" + PropertyName(propertyID), true
		}
	}
	if isInternalPropertyLikeKey(key) {
		return "", false
	}
	return key, true
}

func splitIndexedProperty(key string) (string, string, bool) {
	before, after, ok := strings.Cut(key, "-")
	if !ok {
		return "", "", false
	}
	before = strings.TrimSpace(before)
	after = strings.TrimSpace(after)
	if before == "" || after == "" {
		return "", "", false
	}
	if _, err := strconv.Atoi(before); err != nil {
		return "", "", false
	}
	return before, after, true
}

func isInternalPropertyLikeKey(key string) bool {
	compact := strings.ToLower(strings.TrimSpace(key))
	if compact == "" {
		return true
	}
	if len(compact) <= 3 {
		return true
	}
	if strings.ContainsAny(compact, "_-.") {
		return true
	}
	return false
}

func ToPublicConditionParams(value any) any {
	params, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := deepMap(params)
	if result[FieldConditionType] == nil {
		if value, ok := result[internalConditionType]; ok {
			result[FieldConditionType] = value
		}
	}
	delete(result, internalConditionType)
	if rows, ok := result[FieldConditions].([]any); ok {
		normalized := make([]any, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				normalized = append(normalized, ToPublicCondition(item))
			} else {
				normalized = append(normalized, row)
			}
		}
		result[FieldConditions] = normalized
	}
	return result
}

func ToPublicCondition(source map[string]any) map[string]any {
	result := deepMap(source)
	hasRows := false
	if rows, ok := result[FieldConditions].([]any); ok {
		hasRows = true
		normalized := make([]any, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				normalized = append(normalized, ToPublicCondition(item))
			} else {
				normalized = append(normalized, row)
			}
		}
		result[FieldConditions] = normalized
	}
	if hasRows && result[FieldConditionType] == nil {
		if value, ok := result[internalConditionType]; ok {
			result[FieldConditionType] = value
		}
	}
	if !hasRows && result[FieldConditionKind] == nil {
		if value, ok := result[internalConditionKind]; ok {
			result[FieldConditionKind] = value
		}
	}
	if result[FieldTime] == nil {
		if value, ok := result[internalClock]; ok {
			result[FieldTime] = value
		}
	}
	if result[FieldTargetID] == nil {
		if value, ok := result[internalResourceID]; ok {
			result[FieldTargetID] = value
		}
	}
	if result[FieldTargetKey] == nil {
		if value, ok := result[internalTempID]; ok {
			result[FieldTargetKey] = value
		}
	}
	if result[FieldTargetName] == nil {
		if value, ok := result[internalResourceName]; ok {
			result[FieldTargetName] = value
		}
	}
	if result[FieldTargetType] == nil {
		if name := ResourceTypeName(result[internalTypeID]); name != "" {
			result[FieldTargetType] = name
		}
	}
	if value, ok := result[FieldProperty]; ok {
		result[FieldProperty] = PropertyName(String(value))
	} else if value, ok := result[internalProperty]; ok {
		result[FieldProperty] = PropertyName(String(value))
	}
	if result[FieldCapabilityProductID] == nil {
		if value, ok := result[internalProductID]; ok {
			result[FieldCapabilityProductID] = value
		}
	}
	if result[FieldEventID] == nil {
		if value, ok := result[FieldID]; ok {
			result[FieldEventID] = value
		}
	}
	if result[FieldEventArgs] == nil {
		if value, ok := result[internalEventArgs]; ok {
			result[FieldEventArgs] = value
		}
	}
	delete(result, internalConditionKind)
	delete(result, internalClock)
	delete(result, internalResourceID)
	delete(result, internalResourceName)
	delete(result, internalTypeID)
	delete(result, internalProperty)
	delete(result, internalTempID)
	delete(result, internalProductID)
	delete(result, internalEventArgs)
	delete(result, FieldID)
	return result
}

func ResourceTypeName(value any) string {
	typeID, ok := Int(value)
	if !ok {
		return ""
	}
	switch typeID {
	case ResourceRoom:
		return "room"
	case ResourceDevice:
		return "device"
	case ResourceCustomGroup:
		return "group"
	case ResourceMeshGroup:
		return "meshGroup"
	case ResourceHome:
		return "home"
	case ResourceScene:
		return "scene"
	case ResourceAutomation:
		return "automation"
	default:
		return ""
	}
}

func removeActionAliases(result map[string]any) {
	for _, key := range append(append(append(append([]string{},
		actionAliasConfig.TargetTypes...),
		actionAliasConfig.TargetIDs...),
		actionAliasConfig.TargetNames...),
		actionAliasConfig.TargetKeys...) {
		delete(result, key)
	}
	for _, key := range append(actionAliasConfig.ActionParams, actionAliasConfig.DirectLightKeys...) {
		delete(result, key)
	}
	for _, key := range actionAliasConfig.ActionSets {
		delete(result, key)
	}
	for _, key := range actionAliasConfig.SubIndexes {
		delete(result, key)
	}
	for _, key := range actionParamRowKeys() {
		delete(result, key)
	}
}

func RemoveActionRowAliases(result map[string]any) {
	removeActionAliases(result)
}

func lightParamsFromActionSource(source map[string]any, set any) map[string]any {
	params := map[string]any{FieldSet: set}
	for _, key := range actionParamRowKeys() {
		if value, ok := source[key]; ok {
			params[key] = value
		}
	}
	return params
}

func actionParamRowKeys() []string {
	return []string{FieldDelay, FieldDuration, FieldDelayOff, FieldToggle, FieldAdjust, FieldFlow, FieldCustom}
}

func publicPassthrough(source map[string]any, keys ...string) map[string]any {
	result := map[string]any{}
	for _, key := range keys {
		if value, ok := source[key]; ok {
			result[key] = deepValue(value)
		}
	}
	return result
}

func mapList(value any) ([]map[string]any, bool) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		result = append(result, typed)
	}
	return result, true
}

func deepMap(source map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range source {
		result[key] = deepValue(value)
	}
	return result
}

func deepValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepMap(typed)
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, deepValue(item))
		}
		return result
	default:
		return typed
	}
}

func DebugConfigSummary() string {
	return fmt.Sprintf("targetTypes=%d targetIds=%d targetNames=%d", len(actionAliasConfig.TargetTypes), len(actionAliasConfig.TargetIDs), len(actionAliasConfig.TargetNames))
}
