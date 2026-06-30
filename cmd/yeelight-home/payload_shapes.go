package main

func lightActionParamsShape() map[string]any {
	return map[string]any{
		"inputType": "object is recommended; JSON string is accepted and compacted before cloud write because the Java API stores params as JSON text",
		"set": map[string]any{
			"p":  "optional power bool, true means on and false means off",
			"l":  "optional brightness integer 1..100",
			"ct": "optional color temperature integer 2700..6500",
			"c":  "optional RGB color integer 0..16777215 or supported rrggbb string in high-level lighting intents",
			"tp": "optional curtain target percentage only when Runtime validates curtain support; 0 means fully closed and 100 means fully open in current Yeelight vocabulary",
			"sp": "optional switch power for single-channel switch capability; channel-prefixed keys such as 1-sp are evidence-only",
		},
		"toggle":   "optional list of property names to toggle, such as [\"p\"]; use only when target capability/detail evidence supports toggle",
		"adjust":   "optional relative change object such as {\"l\":\"+10/100\"} or {\"ct\":\"-1/5\"}; preserve source-backed aliases such as b only when Runtime/detail evidence returns them",
		"delay":    "optional non-negative milliseconds before this action starts",
		"duration": "optional non-negative transition duration milliseconds when cloud/device supports it",
		"delayoff": "optional non-negative milliseconds for delayed off after turning on; use only when target supports it",
		"flow":     "optional dynamic light flow object returned by detail/capability evidence. Common members are count, tuples[] and ending; tuples[] rows preserve type=set/pause style phases, set values and duration milliseconds",
		"action":   "optional product-specific action object such as blink, motorAdjust, delayCancel, musicPlayerCtrl or localAudioCtrl; only use when returned by detail/capability evidence",
		"custom":   "preserve other product-specific keys returned by detail editablePayload or supported capability evidence; do not invent vendor keys",
		"keyVocabulary": map[string]any{
			"basicLight":      "p=power, l=brightness, ct=color temperature, c=RGB color, m=mode, o=online/read-only state",
			"sensors":         "mv=motion, oc=occupancy, dc=door closed, act=sensor active, alm=alarm, t=temperature, h=humidity, ll=environment light level",
			"curtain":         "cp=current covered percentage/read evidence, tp=target percentage/write evidence, tra=curtain travel/angle evidence; use only with Runtime curtain capability evidence",
			"switchChannels":  "sp=switch power and blp=backlight power; channel-prefixed keys such as 0-sp, 1-sp, 1-p or 2-p may appear in source-backed detail",
			"airConditioning": "acp/acm/acct/actt/acf/aco/acd are HVAC channel keys; examples include 1-acp or 2-actt and require Runtime evidence",
			"audio":           "mpmp/mppm/vol and action.musicPlayerCtrl/localAudioCtrl are audio/product-specific keys; preserve only from Runtime evidence",
			"deviceAttrs":     "lc/li/slisaon/slisaon_rdy/bp/rl/rd/dd/ch_num/acn are node attributes; route through supported semantic adapters instead of raw action JSON",
		},
	}
}

func sceneActionItemShape() map[string]any {
	return map[string]any{
		"typeId":  "required resource type. Runtime directly validates 2=device, 3=Runtime group/custom scope, 4=mesh group, 6=scene; design import may also map 1=room and 5=house when returned by source-backed metadata",
		"resId":   "required target resource id",
		"resName": "required target display name; Runtime may backfill from current home entities when available",
		"action":  "cloud action type; Runtime defaults to 0 when omitted",
		"rank":    "ordering integer; Runtime defaults to 0 when omitted",
		"idx":     "optional sub-device index",
		"roomId":  "optional room id for panel/event style rows; preserve when returned by detail",
		"params":  lightActionParamsShape(),
	}
}

func automationConditionParamsShape() map[string]any {
	return map[string]any{
		"inputType": "object is recommended; JSON string is accepted and compacted before cloud write because the Java API stores params as JSON text",
		"type":      "required top-level condition operator. Runtime write validation currently accepts 'and' at the root; nested event/fact groups may contain 'and' or 'or' only when returned by detail/support evidence",
		"conditions": []any{
			map[string]any{
				"type":       "alarm for clock schedule; event/fact/fact_change for source-backed device conditions; and/or for nested groups",
				"clock":      "required HH:mm:ss for alarm/timer-style conditions",
				"id":         "source-backed event/fact id from automation supported-list/detail evidence; do not invent",
				"pid":        "source-backed product id for device condition rows when evidence provides it",
				"typeId":     "required only for resource-backed condition rows, using the same group type ids as action rows",
				"resId":      "required only for referenced condition resources",
				"prop":       "optional source-backed property name for fact/fact_change conditions",
				"operation":  "optional comparison operation, defaults to eq in Java model when omitted",
				"value":      "optional comparison value for fact/fact_change conditions",
				"extArgs":    "optional source-backed arguments such as thresholds; preserve from Runtime evidence",
				"actionItem": "optional UI/source evidence object; preserve when returned by automation.detail.get",
				"conditions": "optional nested condition group; preserve from automation.detail.get editablePayload when editing",
			},
		},
	}
}

func automationActionItemShape() map[string]any {
	return map[string]any{
		"typeId":  "required resource type. General automation actions validate 2=device, 4=mesh group, 6=scene; design-import/source-backed payloads may map 1=room, 3=custom scope, or 5=house only when Runtime evidence supports them",
		"resId":   "required target resource id",
		"resName": "required by cloud for action rows; Runtime may backfill when available",
		"rank":    "ordering integer; Runtime defaults to 0 when omitted",
		"idx":     "optional sub-device index",
		"params":  lightActionParamsShape(),
	}
}
