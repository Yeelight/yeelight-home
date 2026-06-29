package api

import (
	"encoding/json"
	"strings"
)

func sceneDetailData(detail any, sceneID string) map[string]any {
	sanitized := sanitizeCloudData(detail)
	data := map[string]any{
		"detail":      sanitized,
		"updateShape": sceneUpdateShape(),
	}
	if payload := editableScenePayload(sanitized, sceneID); len(payload) > 0 {
		data["editablePayload"] = payload
	}
	return data
}

func automationDetailData(detail any, automationID string) map[string]any {
	sanitized := sanitizeCloudData(detail)
	data := map[string]any{
		"detail":      sanitized,
		"updateShape": automationUpdateShape(),
	}
	if payload := editableAutomationPayload(sanitized, automationID); len(payload) > 0 {
		data["editablePayload"] = payload
	}
	return data
}

func editableScenePayload(value any, sceneID string) map[string]any {
	detail, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	payload := map[string]any{}
	putFirst(payload, "sceneId", sceneID, detail, "sceneId", "id", "entityId")
	putFirst(payload, "name", "", detail, "name", "sceneName")
	putFirst(payload, "description", "", detail, "desc", "description")
	putFirst(payload, "icon", "", detail, "icon", "img")
	if details := editableActionList(firstNonNil(detail["details"], detail["actions"])); len(details) > 0 {
		payload["details"] = details
	}
	return compactEditableMap(payload)
}

func editableAutomationPayload(value any, automationID string) map[string]any {
	detail, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	payload := map[string]any{}
	putFirst(payload, "automationId", automationID, detail, "automationId", "id", "entityId")
	putFirst(payload, "name", "", detail, "name", "automationName")
	putFirst(payload, "startTime", "", detail, "startTime", "start_time")
	putFirst(payload, "endTime", "", detail, "endTime", "end_time")
	putFirst(payload, "repeatType", "", detail, "repeatType", "repeat_type")
	putFirst(payload, "repeatValue", "", detail, "repeatValue", "repeat_value")
	putFirst(payload, "version", "", detail, "version")
	if params := editableJSONValue(detail["params"]); params != nil {
		payload["params"] = params
	}
	if actions := editableActionList(firstNonNil(detail["actions"], detail["details"])); len(actions) > 0 {
		payload["actions"] = actions
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
		for _, key := range []string{"typeId", "resId", "resName", "action", "rank", "idx", "roomId"} {
			if value, ok := item[key]; ok {
				action[key] = value
			}
		}
		if params := editableJSONValue(item["params"]); params != nil {
			action["params"] = params
		}
		if len(action) > 0 {
			result = append(result, compactEditableMap(action))
		}
	}
	return result
}

func lightActionParamsShape() map[string]any {
	return map[string]any{
		"inputType": "object is returned in editablePayload; Runtime also accepts JSON string for write payloads",
		"set": map[string]any{
			"p":  "power bool",
			"l":  "brightness integer 1..100",
			"ct": "color temperature integer 2700..6500",
			"c":  "RGB color integer 0..16777215 when supported",
			"tp": "curtain target percentage only when Runtime validates curtain support",
			"sp": "switch power for supported single-channel switch targets; channel-prefixed variants require evidence",
		},
		"toggle":   "optional property toggle list such as [\"p\"]; preserve only when returned or supported",
		"adjust":   "optional relative adjustment object such as {\"l\":\"+10/100\"} or {\"ct\":\"-1/5\"}; preserve aliases such as b only when returned or supported",
		"delay":    "optional non-negative milliseconds before action",
		"duration": "optional non-negative transition duration milliseconds when supported",
		"delayoff": "optional non-negative milliseconds for delayed off when supported",
		"flow":     "optional dynamic light flow object returned by detail/capability evidence, usually count, tuples[], ending and tuple duration/type fields",
		"action":   "optional product-specific action object returned by detail/capability evidence, such as blink, motorAdjust, delayCancel, musicPlayerCtrl or localAudioCtrl",
		"custom":   "preserve product-specific keys returned by detail if not changing them; do not invent effect/vendor keys without detail/capability evidence",
		"keyVocabulary": map[string]any{
			"basicLight":      "p=power, l=brightness, ct=color temperature, c=RGB color, m=mode, o=online/read-only state",
			"sensors":         "mv=motion, oc=occupancy, dc=door closed, act=sensor active, alm=alarm, t=temperature, h=humidity, ll=environment light level",
			"curtain":         "cp=current covered percentage/read evidence, tp=target percentage/write evidence, tra=curtain travel/angle evidence; use only with Runtime curtain capability evidence",
			"switchChannels":  "sp=switch power and blp=backlight power; channel-prefixed keys such as 0-sp, 1-sp, 1-p or 2-p may appear in source-backed detail",
			"airConditioning": "acp/acm/acct/actt/acf/aco/acd are HVAC channel keys; examples include 1-acp or 2-actt and require Runtime evidence",
			"audio":           "mpmp/mppm/vol and action.musicPlayerCtrl/localAudioCtrl are audio/product-specific keys; preserve only from Runtime evidence",
			"deviceAttrs":     "lc/li/slisaon/slisaon_rdy/bp/rl/rd/dd/ch_num/acn are node attributes; use semantic adapters instead of raw action JSON",
		},
	}
}

func sceneActionItemShape() map[string]any {
	return map[string]any{
		"typeId":  "required resource type. Runtime validates 2=device, 3=Runtime group/custom scope, 4=mesh group, 6=scene for direct scene writes; design metadata may carry 1=room or 5=house when source-backed",
		"resId":   "required target resource id",
		"resName": "required display name",
		"action":  "required by cloud; default 0 if omitted through Runtime",
		"rank":    "required ordering integer; default 0 if omitted through Runtime",
		"idx":     "optional sub-device index",
		"roomId":  "optional room id, preserve when returned",
		"params":  lightActionParamsShape(),
	}
}

func automationConditionParamsShape() map[string]any {
	return map[string]any{
		"inputType": "object is returned in editablePayload; Runtime also accepts JSON string for write payloads",
		"type":      "required top-level operator. Runtime write validation accepts 'and' at root; nested groups may preserve and/or from evidence",
		"conditions": []any{
			map[string]any{
				"type":       "alarm, event, fact, fact_change, or nested and/or group",
				"clock":      "HH:mm:ss for alarm/timer-style conditions",
				"id":         "source-backed event/fact id; preserve from Runtime evidence",
				"pid":        "source-backed product id for device conditions",
				"typeId":     "required only for resource-backed condition rows",
				"resId":      "required only for resource-backed condition rows",
				"prop":       "optional source-backed property name for fact/fact_change rows",
				"operation":  "optional comparison operation; Java model defaults to eq",
				"value":      "optional comparison value",
				"extArgs":    "optional source-backed arguments such as thresholds",
				"actionItem": "optional evidence object returned by detail/support APIs",
				"conditions": "optional nested condition group; preserve from editablePayload unless replacing",
			},
		},
	}
}

func automationActionItemShape() map[string]any {
	return map[string]any{
		"typeId":  "required resource type. Direct automation writes validate 2=device, 4=mesh group, 6=scene; source-backed design metadata may map other group types",
		"resId":   "required target resource id",
		"resName": "required display name",
		"rank":    "required ordering integer; default 0 if omitted through Runtime",
		"idx":     "optional sub-device index",
		"params":  lightActionParamsShape(),
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
		"intent":       "scene.update",
		"completeList": true,
		"required":     []string{"sceneId", "name", "details"},
		"flow":         []string{"call scene.detail.get", "copy editablePayload", "preserve every details[] row", "change only intended details[].params.set keys", "send scene.update"},
		"details":      []any{sceneActionItemShape()},
		"editFlow":     "read scene.detail.get, modify editablePayload.details[].params.set or other intended fields, then send the complete details list to scene.update",
	}
}

func automationUpdateShape() map[string]any {
	return map[string]any{
		"intent":       "automation.update",
		"completeRule": true,
		"required":     []string{"automationId", "name", "startTime", "endTime", "repeatType", "params", "actions"},
		"flow":         []string{"call automation.detail.get", "copy editablePayload", "preserve params and actions[] unless explicitly replacing", "change only intended schedule/action keys", "send automation.update"},
		"params":       automationConditionParamsShape(),
		"actions":      []any{automationActionItemShape()},
		"statusChange": "Use automation.enable or automation.disable.",
		"editFlow":     "read automation.detail.get, modify editablePayload params/actions or schedule fields, then send the complete rule to automation.update",
	}
}
