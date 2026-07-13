package api

import (
	"encoding/json"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func sceneDetailData(detail any, sceneID string) map[string]any {
	sanitized := sanitizeCloudData(detail)
	data := map[string]any{
		semantic.FieldDetail:      publicSceneDetail(sanitized, sceneID),
		semantic.FieldUpdateShape: sceneUpdateShape(),
	}
	if payload := editableScenePayload(detail, sceneID); len(payload) > 0 {
		data[semantic.FieldEditablePayload] = payload
	}
	return data
}

func automationDetailData(detail any, automationID string) map[string]any {
	data := map[string]any{
		semantic.FieldDetail:      publicAutomationDetail(detail, automationID),
		semantic.FieldUpdateShape: automationUpdateShape(),
	}
	if payload := editableAutomationPayload(detail, automationID); len(payload) > 0 {
		data[semantic.FieldEditablePayload] = payload
	}
	return data
}

func publicSceneDetail(value any, sceneID string) any {
	detail, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := map[string]any{}
	putFirst(result, semantic.FieldSceneID, sceneID, detail, semantic.FieldSceneID, "id", "entityId")
	putFirst(result, semantic.FieldName, "", detail, semantic.FieldName, "sceneName")
	putFirst(result, semantic.FieldDescription, "", detail, semantic.InternalField(semantic.DomainCommon, semantic.FieldDescription), semantic.FieldDescription)
	putFirst(result, semantic.FieldIcon, "", detail, semantic.FieldIcon, "img")
	if actions := editableActionList(firstNonNil(detail[semantic.FieldDetails], detail[semantic.FieldActions])); len(actions) > 0 {
		result[semantic.FieldActions] = actions
	}
	return compactEditableMap(result)
}

func publicAutomationDetail(value any, automationID string) any {
	detail, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := map[string]any{}
	putFirst(result, semantic.FieldAutomationID, automationID, detail, semantic.FieldAutomationID, "id", "entityId")
	putFirst(result, semantic.FieldName, "", detail, semantic.FieldName, "automationName")
	putAutomationSchedule(result, detail)
	putFirst(result, semantic.FieldVersion, "", detail, semantic.FieldVersion)
	if params := editableJSONValue(detail[semantic.InternalAutomationParamsField()]); params != nil {
		putAutomationConditions(result, params)
	}
	if actions := editableActionList(firstNonNil(detail[semantic.FieldActions], detail[semantic.FieldDetails])); len(actions) > 0 {
		result[semantic.FieldActions] = actions
	}
	return compactEditableMap(result)
}

func editableScenePayload(value any, sceneID string) map[string]any {
	detail, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	payload := map[string]any{}
	putFirst(payload, semantic.FieldSceneID, sceneID, detail, semantic.FieldSceneID, "id", "entityId")
	putFirst(payload, semantic.FieldName, "", detail, semantic.FieldName, "sceneName")
	putFirst(payload, semantic.FieldDescription, "", detail, semantic.InternalField(semantic.DomainCommon, semantic.FieldDescription), semantic.FieldDescription)
	putFirst(payload, semantic.FieldIcon, "", detail, semantic.FieldIcon, "img")
	if details := editableActionList(firstNonNil(detail[semantic.FieldDetails], detail[semantic.FieldActions])); len(details) > 0 {
		payload[semantic.FieldActions] = details
	}
	return compactEditableMap(payload)
}

func editableAutomationPayload(value any, automationID string) map[string]any {
	detail, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	payload := map[string]any{}
	putFirst(payload, semantic.FieldAutomationID, automationID, detail, semantic.FieldAutomationID, "id", "entityId")
	putFirst(payload, semantic.FieldName, "", detail, semantic.FieldName, "automationName")
	putAutomationSchedule(payload, detail)
	putFirst(payload, semantic.FieldVersion, "", detail, semantic.FieldVersion)
	if params := editableJSONValue(detail[semantic.InternalAutomationParamsField()]); params != nil {
		putAutomationConditions(payload, params)
	}
	if actions := editableActionList(firstNonNil(detail[semantic.FieldActions], detail[semantic.FieldDetails])); len(actions) > 0 {
		payload[semantic.FieldActions] = actions
	}
	return compactEditableMap(payload)
}

func editableActionList(value any) []any {
	rows := rowsFromData(value)
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		action := map[string]any{}
		action = semantic.ToPublicAction(item)
		for _, key := range []string{
			semantic.FieldTargetType,
			semantic.FieldTargetID,
			semantic.FieldTargetKey,
			semantic.FieldTargetName,
			semantic.FieldAction,
			semantic.FieldRank,
			semantic.FieldRoomID,
			semantic.FieldSubIndex,
			semantic.FieldSet,
		} {
			if _, already := action[key]; already {
				continue
			}
			if value, ok := item[key]; ok {
				action[key] = value
			}
		}
		if params := editableJSONValue(item[semantic.InternalActionParamsField()]); params != nil {
			semantic.MergePublicActionParams(action, params)
		}
		if len(action) > 0 {
			result = append(result, compactEditableMap(action))
		}
	}
	return result
}

func lightActionParamsShape() map[string]any {
	return map[string]any{
		semantic.FieldInputType: "object",
		semantic.FieldSet: map[string]any{
			semantic.FieldPower:            "power bool",
			semantic.FieldBrightness:       "brightness integer 1..100",
			semantic.FieldColorTemperature: "color temperature integer 2700..6500",
			semantic.FieldColor:            "RGB color integer 0..16777215 when supported",
			semantic.FieldTargetPercent:    "curtain target percentage only when Runtime validates curtain support",
			semantic.FieldSwitchPower:      "switch power only when Runtime validates switch support",
		},
		semantic.FieldToggle:   "optional semantic property toggle list such as [\"power\"]; preserve only when returned or supported",
		semantic.FieldAdjust:   "optional relative adjustment object such as {\"brightness\":\"+10/100\"} or {\"colorTemperature\":\"-1/5\"}",
		semantic.FieldDelay:    "optional non-negative milliseconds before action",
		semantic.FieldDuration: "optional non-negative transition duration milliseconds when supported",
		semantic.FieldDelayOff: "optional non-negative milliseconds for delayed off when supported",
		semantic.FieldFlow:     "optional dynamic light flow object returned by detail/capability evidence, usually count, tuples[], ending and tuple duration/type fields",
		semantic.FieldAction:   "optional product-specific action object returned by detail/capability evidence, such as blink, motorAdjust, delayCancel, musicPlayerCtrl or localAudioCtrl",
		semantic.FieldCustom:   "preserve product-specific keys returned by detail if not changing them; do not invent effect/vendor keys without detail/capability evidence",
	}
}

func sceneActionItemShape() map[string]any {
	return map[string]any{
		semantic.FieldTargetType: "required target kind: device, group, meshGroup, scene, room, or home when the intent supports it",
		semantic.FieldTargetID:   "required target resource id",
		semantic.FieldTargetName: "required display name",
		semantic.FieldAction:     "optional action mode; preserve on update when present",
		semantic.FieldRank:       "ordering integer; Runtime defaults when omitted",
		semantic.FieldSubIndex:   "optional sub-device index",
		semantic.FieldRoomID:     "optional room id, preserve when returned",
		semantic.FieldSet:        lightActionParamsShape()[semantic.FieldSet],
	}
}

func automationConditionParamsShape() map[string]any {
	conditionShape := map[string]any{
		semantic.FieldConditionKind:       "alarm, event, fact, or fact_change",
		semantic.FieldTime:                "HH:mm:ss for alarm-style conditions",
		semantic.FieldTargetType:          "target kind for evidence-backed conditions",
		semantic.FieldTargetID:            "target resource id from Runtime evidence",
		semantic.FieldTargetKey:           "design import target key when editing a design-derived rule",
		semantic.FieldCapabilityProductID: "capability pid from Runtime/product evidence",
		semantic.FieldEventID:             "event id from Runtime/product evidence",
		semantic.FieldEventArgs:           "event arguments from Runtime/product evidence",
		semantic.FieldProperty:            "standard property name from Runtime evidence",
		semantic.FieldOperation:           "comparison operation from Runtime evidence",
		semantic.FieldValue:               "comparison value for fact/fact_change conditions",
	}
	return map[string]any{
		semantic.FieldTrigger:    conditionShape,
		semantic.FieldConditions: []any{conditionShape},
	}
}

func automationActionItemShape() map[string]any {
	return map[string]any{
		semantic.FieldTargetType: "required target kind: device, group, meshGroup, or scene",
		semantic.FieldTargetID:   "required target resource id",
		semantic.FieldTargetName: "required display name",
		semantic.FieldRank:       "ordering integer; Runtime defaults when omitted",
		semantic.FieldSubIndex:   "optional sub-device index",
		semantic.FieldSet:        lightActionParamsShape()[semantic.FieldSet],
	}
}

func putAutomationSchedule(target map[string]any, detail map[string]any) {
	start := firstStringFrom(detail, semantic.FieldStartTime, "start_time")
	end := firstStringFrom(detail, semantic.FieldEndTime, "end_time")
	if start != "" || end != "" {
		target[semantic.FieldActiveWindow] = map[string]any{
			semantic.FieldStart: start,
			semantic.FieldEnd:   end,
		}
	}
	if repeat := publicRepeat(detail); repeat != nil {
		target[semantic.FieldRepeat] = repeat
	}
}

func firstStringFrom(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := strings.TrimSpace(stringFromAny(values[key])); text != "" {
			return text
		}
	}
	return ""
}

func putAutomationConditions(target map[string]any, value any) {
	params := semantic.ToPublicConditionParams(value)
	mapped, ok := params.(map[string]any)
	if !ok {
		target[semantic.FieldConditions] = params
		return
	}
	conditions, _ := mapped[semantic.FieldConditions].([]any)
	if len(conditions) == 1 {
		if first, ok := conditions[0].(map[string]any); ok && first[semantic.FieldConditions] == nil {
			target[semantic.FieldTrigger] = first
			return
		}
	}
	if len(conditions) > 0 {
		conditionType := strings.ToLower(strings.TrimSpace(stringFromAny(mapped[semantic.FieldConditionType])))
		if conditionType == "and" || conditionType == "or" {
			target[semantic.FieldConditionType] = conditionType
		}
		target[semantic.FieldConditions] = conditions
	}
}

func publicRepeat(detail map[string]any) any {
	repeatType := intFromAny(firstNonNil(detail[semantic.InternalRepeatTypeField()], detail["repeat_type"]))
	if repeatType == 0 {
		return nil
	}
	repeatValue := strings.TrimSpace(stringFromAny(firstNonNil(detail[semantic.InternalRepeatValueField()], detail["repeat_value"])))
	switch repeatType {
	case 1:
		return "once"
	case 2:
		return "daily"
	case 3:
		return "weekdays"
	case 5:
		return "weekend"
	case 6:
		return "legal_holiday"
	case 7:
		return "legal_workday"
	default:
		if repeatValue != "" {
			return map[string]any{semantic.FieldType: "custom", semantic.FieldRepeatDays: repeatValue}
		}
		return map[string]any{semantic.FieldType: "custom"}
	}
}

func editableJSONValue(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return value
	}
	return parsed
}

func putFirst(target map[string]any, targetKey string, fallback string, source map[string]any, keys ...string) {
	if strings.TrimSpace(fallback) != "" {
		target[targetKey] = strings.TrimSpace(fallback)
		return
	}
	for _, key := range keys {
		value, ok := source[key]
		if !ok || isEmptyEditableValue(value) {
			continue
		}
		target[targetKey] = value
		return
	}
}

func compactEditableMap(source map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range source {
		if isEmptyEditableValue(value) {
			continue
		}
		result[key] = value
	}
	return result
}

func isEmptyEditableValue(value any) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false
}

func sceneUpdateShape() map[string]any {
	return map[string]any{
		semantic.FieldIntent:       "scene.update",
		semantic.FieldCompleteList: true,
		semantic.FieldRequired:     []string{"sceneId or unique sceneName/currentName", semantic.FieldActions},
		semantic.FieldFlow:         []string{"call scene.detail.get", "copy editablePayload", "preserve every actions[] row", "change only intended actions[].set keys", "send scene.update"},
		semantic.FieldActions:      []any{sceneActionItemShape()},
		semantic.FieldEditFlow:     "read scene.detail.get, modify editablePayload.actions[].set or other intended fields, then send the complete actions list to scene.update",
	}
}

func automationUpdateShape() map[string]any {
	return map[string]any{
		semantic.FieldIntent:       "automation.update",
		semantic.FieldCompleteRule: true,
		semantic.FieldRequired:     []string{"automationId or unique automationName/currentName", semantic.FieldTrigger, semantic.FieldActions},
		semantic.FieldFlow:         []string{"call automation.detail.get", "copy editablePayload", "preserve trigger/conditions and actions[] unless explicitly replacing", "change only intended schedule/action keys", "send automation.update"},
		semantic.FieldTrigger:      automationConditionParamsShape()[semantic.FieldTrigger],
		semantic.FieldConditions:   automationConditionParamsShape()[semantic.FieldConditions],
		semantic.FieldActions:      []any{automationActionItemShape()},
		semantic.FieldStatusChange: "Use automation.enable or automation.disable.",
		semantic.FieldEditFlow:     "read automation.detail.get, modify editablePayload trigger/conditions/actions or schedule fields, then send the complete rule to automation.update",
	}
}
