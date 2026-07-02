package main

import "github.com/yeelight/yeelight-home/internal/semantic"

func lightActionParamsShape() map[string]any {
	return map[string]any{
		semantic.FieldInputType: "object is recommended; Runtime handles any required internal serialization",
		semantic.FieldSet: map[string]any{
			semantic.FieldPower:            "optional power bool, true means on and false means off",
			semantic.FieldBrightness:       "optional brightness integer 1..100",
			semantic.FieldColorTemperature: "optional color temperature integer 2700..6500",
			semantic.FieldColor:            "optional RGB color integer 0..16777215 or supported rrggbb string in high-level lighting intents",
			semantic.FieldTargetPercent:    "optional curtain target percentage only when Runtime validates curtain support",
			semantic.FieldSwitchPower:      "optional switch power only when Runtime validates switch support",
		},
		semantic.FieldToggle:   "optional list of standard property names to toggle, such as [\"power\"]; use only when target capability/detail evidence supports toggle",
		semantic.FieldAdjust:   "optional relative change object such as {\"brightness\":\"+10/100\"} or {\"colorTemperature\":\"-1/5\"}",
		semantic.FieldDelay:    "optional non-negative milliseconds before this action starts",
		semantic.FieldDuration: "optional non-negative transition duration milliseconds when cloud/device supports it",
		semantic.FieldDelayOff: "optional non-negative milliseconds for delayed off after turning on; use only when target supports it",
		semantic.FieldFlow:     "optional dynamic light flow object returned by detail/capability evidence. Common members are count, tuples[] and ending; tuples[] rows preserve type=set/pause style phases, set values and duration milliseconds",
		semantic.FieldAction:   "optional product-specific action object such as blink, motorAdjust, delayCancel, musicPlayerCtrl or localAudioCtrl; only use when returned by detail/capability evidence",
		semantic.FieldCustom:   "preserve other product-specific keys returned by detail editablePayload or supported capability evidence; do not invent vendor keys",
	}
}

func sceneActionItemShape() map[string]any {
	return map[string]any{
		semantic.FieldTargetType: "required standard target kind: device, group, meshGroup, scene, or room when the intent supports it",
		semantic.FieldTargetID:   "optional target id from Runtime evidence; omit when targetName uniquely resolves the target",
		semantic.FieldTargetName: "optional target display name; Runtime resolves unique names within the selected home",
		semantic.FieldSet:        lightActionParamsShape()[semantic.FieldSet],
		semantic.FieldAction:     "optional action mode returned by detail evidence; preserve on update when present",
		semantic.FieldRank:       "ordering integer; Runtime defaults to 0 when omitted",
		semantic.FieldSubIndex:   "optional sub-device index",
		semantic.FieldRoomID:     "optional room id for panel/event style rows; preserve when returned by detail",
	}
}

func automationConditionParamsShape() map[string]any {
	conditionShape := map[string]any{
		semantic.FieldConditionKind:       "alarm for clock schedule; event for device events; fact for state checks; fact_change for property-change triggers. Use non-alarm kinds only when Runtime/product evidence provides the source condition",
		semantic.FieldTime:                "HH:mm:ss only for alarm/timer-style conditions",
		semantic.FieldTargetType:          "target kind for evidence-backed device/group/scene conditions",
		semantic.FieldTargetID:            "target id from Runtime evidence",
		semantic.FieldTargetKey:           "design import target key when referencing a slot, group, scene, room, or home in the same design model",
		semantic.FieldCapabilityProductID: "capability pid from Runtime/product evidence for event/fact/fact_change conditions",
		semantic.FieldEventID:             "event id from Runtime/product evidence for event conditions",
		semantic.FieldEventArgs:           "event arguments from Runtime/product evidence for event conditions",
		semantic.FieldProperty:            "standard property name from Runtime evidence, such as power, brightness, colorTemperature, motionDetected, occupancyDetected, doorClosed, or alarm",
		semantic.FieldOperation:           "comparison operation from Runtime evidence for fact conditions",
		semantic.FieldValue:               "comparison value for fact/fact_change conditions",
		semantic.FieldConditions:          "optional nested condition group; preserve from automation.detail.get editablePayload when editing",
	}
	return map[string]any{
		semantic.FieldConditionType: "top-level condition operator; defaults to and",
		semantic.FieldTrigger:       conditionShape,
		semantic.FieldConditions:    []any{conditionShape},
	}
}

func automationActionItemShape() map[string]any {
	return map[string]any{
		semantic.FieldTargetType: "required standard target kind: device, group, meshGroup, or scene",
		semantic.FieldTargetID:   "optional target id from Runtime evidence; omit when targetName uniquely resolves the target",
		semantic.FieldTargetName: "optional target display name; Runtime resolves unique names within the selected home",
		semantic.FieldSet:        lightActionParamsShape()[semantic.FieldSet],
		semantic.FieldRank:       "ordering integer; Runtime defaults to 0 when omitted",
		semantic.FieldSubIndex:   "optional sub-device index",
	}
}
