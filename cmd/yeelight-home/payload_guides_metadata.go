package main

func panelButtonConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"deviceId": "required panel device id",
			"buttons": []any{
				map[string]any{
					"id":       "recommended existing button id; index/keyValue/name/alias can also locate current button",
					"alias":    "optional display alias",
					"keyValue": "optional physical key value",
					"index":    "optional button index",
					"resId":    "optional bound resource id",
					"resType":  "optional bound resource type, usually 2=device, 6=scene",
					"visible":  "optional 0/1",
					"icon":     "optional icon value",
					"sort":     "optional sort order",
					"type":     "optional button type; Runtime backfills from current panel button when possible",
					"extend":   "optional extension object/string accepted by cloud",
				},
			},
		},
		"examples": []any{
			map[string]any{
				"deviceId": "panel-1",
				"buttons": []any{
					map[string]any{"id": "btn-1", "alias": "回家", "resId": "scene-1", "resType": 6, "visible": 1},
				},
			},
		},
		"nextStep": "Call panel.get or panel button info first when possible, then send only the button rows to change. Runtime merges each row with the current cloud button before writing.",
	}
}

func panelButtonEventPayloadGuide() map[string]any {
	eventShape := map[string]any{
		"buttonEventId": "required existing button event id",
		"alias":         "optional event alias such as 单击/双击/长按",
		"details": []any{
			map[string]any{
				"roomId":      "optional room id for target resource",
				"resId":       "required controlled resource id",
				"typeId":      "required controlled resource type: 2=device, 3=Runtime group, 4=mesh group, 6=scene",
				"idx":         "optional sub-device index",
				"rank":        "optional order, default cloud behavior when omitted",
				"resName":     "optional controlled resource name",
				"startTime":   "optional HH:mm:ss active start",
				"endTime":     "optional HH:mm:ss active end",
				"repeatType":  "optional 1..7",
				"repeatValue": "optional custom repeat hex value such as 0x7f",
				"params":      lightActionParamsShape(),
			},
		},
	}
	return map[string]any{
		"payloadShape": map[string]any{
			"deviceId":       "required panel device id",
			"buttonEvent":    eventShape,
			"buttonEvents[]": eventShape,
		},
		"examples": []any{
			map[string]any{
				"deviceId":      "panel-1",
				"buttonEventId": "101",
				"alias":         "单击",
				"details": []any{
					map[string]any{"typeId": 6, "resId": "scene-1", "resName": "回家模式", "rank": 0},
				},
			},
			map[string]any{
				"deviceId": "panel-1",
				"buttonEvents": []any{
					map[string]any{"buttonEventId": "101", "details": []any{map[string]any{"typeId": 6, "resId": "scene-1"}}},
					map[string]any{"buttonEventId": "102", "details": []any{map[string]any{"typeId": 2, "resId": "50018330", "params": map[string]any{"set": map[string]any{"p": true, "ct": 3000}}}}},
				},
			},
		},
		"nextStep": "Use panel.button_event.update for one event and panel.button_event.batch_update for multiple events. Each event must carry its complete details list.",
	}
}

func knobConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"deviceId": "required knob device id",
			"details": []any{
				map[string]any{
					"index":      "required physical knob key index, 1-based",
					"configType": "required cloud knob config type when known",
					"resId":      "required target resource id for bound target modes",
					"typeId":     "required target resource type, 1..6 when bound to a resource",
					"resIndex":   "optional target sub-device index",
					"resName":    "optional target display name",
					"model":      "optional knob model value",
					"param":      "optional eventCode-to-parameter map",
					"sens":       "optional sensitivity",
					"mode":       "accepted alias/legacy mode field when cloud detail uses it",
					"details":    "optional nested detail object preserved when returned by knob.get",
				},
			},
		},
		"examples": []any{
			map[string]any{
				"deviceId": "knob-1",
				"details": []any{
					map[string]any{"index": 1, "configType": "scene", "resId": "scene-1", "typeId": 6, "resName": "回家模式"},
				},
			},
		},
		"nextStep": "Call knob.get first when editing an existing knob. Preserve the current detail row and change only the intended index binding or parameter map.",
	}
}

func operationBatchConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"operations": []any{
				map[string]any{
					"intent":     "required allowlisted add/update/configure intent",
					"parameters": "required object accepted by that intent",
					"targets":    "optional natural or id targets for that step",
				},
			},
		},
		"allowedIntents": []any{
			"home.update", "home.sort.configure", "favorite.add/update/batch_add/batch_update",
			"room.create/rename/update/batch_create/batch_update/area.configure",
			"area.create/update", "device.rename/move/move_room.batch", "entity.rename.batch",
			"group.create/update", "scene.create/update", "automation.create/update/enable/disable",
			"gateway.configure", "panel.button.configure", "panel.button_event.update/batch_update",
			"knob.configure", "lighting.design.import", "device.slot.create",
		},
		"examples": []any{
			map[string]any{
				"operations": []any{
					map[string]any{"intent": "room.create", "parameters": map[string]any{"name": "书房"}},
					map[string]any{"intent": "device.rename", "parameters": map[string]any{"deviceId": "50018330", "name": "书房主灯"}},
				},
			},
		},
		"nextStep": "Use one batch for one user request with multiple reversible add/update/configure steps. Keep delete, unbind, member transfer/remove, home create/delete, panel reset, and knob reset as separate semantic requests.",
	}
}

func homeSortConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"type":   "required sort surface/type. Runtime may normalize from target/roomId when available",
			"target": "required sort target, such as home or room id depending on type",
			"items": []any{
				map[string]any{
					"entityType": "recommended: room, device, group, scene, automation",
					"typeId":     "accepted explicit resource type id when known",
					"resId":      "required resource id; aliases include entityId/deviceId/sceneId/groupId/roomId",
					"rank":       "required integer order rank",
					"subIndex":   "optional sub-index for multi-channel targets",
				},
			},
		},
		"examples": []any{
			map[string]any{
				"type":   0,
				"target": "200171",
				"items": []any{
					map[string]any{"entityType": "room", "resId": "401398", "rank": 1},
					map[string]any{"entityType": "scene", "resId": "700001", "rank": 2},
				},
			},
		},
		"nextStep": "Read home.sort.list first when preserving unspecified items. Send only explicit ordered items; Runtime verifies resources belong to the current home.",
	}
}

func favoritePayloadGuide(intent string) map[string]any {
	itemShape := map[string]any{
		"favoriteId": "required for favorite.update and accepted for delete; omitted for add",
		"entityType": "recommended resource type: room, device, group, meshgroup, scene, automation",
		"typeId":     "accepted explicit resource type id when known",
		"resId":      "required resource id; aliases include entityId/deviceId/sceneId/groupId/roomId",
		"rank":       "optional order rank",
		"valid":      "optional bool enabled flag",
	}
	return map[string]any{
		"payloadShape": map[string]any{
			"single": itemShape,
			"items":  []any{itemShape},
		},
		"examples": []any{
			map[string]any{"entityType": "device", "resId": "50018330", "rank": 1},
			map[string]any{"items": []any{
				map[string]any{"entityType": "device", "resId": "50018330", "rank": 1},
				map[string]any{"entityType": "scene", "resId": "700001", "rank": 2},
			}},
		},
		"nextStep": "For batch intents, send items[] with 1..20 explicit favorite targets. Delete can use favoriteId or an unambiguous entityType plus resource id resolved from favorite.list.",
		"intent":   intent,
	}
}

func roomBatchPayloadGuide(intent string) map[string]any {
	roomShape := map[string]any{
		"roomId":            "required for room.batch_update; omit for room.batch_create",
		"name":              "required for create; optional update name",
		"desc":              "optional description",
		"icon":              "optional icon",
		"img":               "optional image for update",
		"gatewayDeviceId":   "optional gateway device id",
		"gatewayIds":        "optional gateway id list",
		"defaultGatewayIds": "optional default gateway id list",
		"seq":               "optional sequence",
		"capability":        "optional capability object/value accepted by cloud",
	}
	return map[string]any{
		"payloadShape": map[string]any{"rooms": []any{roomShape}},
		"examples": []any{
			map[string]any{"rooms": []any{map[string]any{"name": "书房"}, map[string]any{"name": "茶室"}}},
			map[string]any{"rooms": []any{map[string]any{"roomId": "401398", "name": "会客厅"}}},
		},
		"nextStep": "Use rooms[] or items[] with 1..20 room objects. Runtime rejects duplicate names, unknown update ids, and gateway references outside the home.",
		"intent":   intent,
	}
}

func roomAreaConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"roomId":         "required room id",
			"addAreaList":    "optional list of area ids to add",
			"removeAreaList": "optional list of area ids to remove",
		},
		"examples": []any{map[string]any{"roomId": "401398", "addAreaList": []any{"300001"}, "removeAreaList": []any{"300002"}}},
		"nextStep": "Provide at least one add or remove area id. Runtime validates all room and area ids in the current home.",
	}
}

func areaUpdatePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"areaId":   "required area id",
			"name":     "optional new name",
			"desc":     "optional description",
			"icon":     "optional icon",
			"parentId": "optional parent area id, cannot be itself",
			"roomIds":  "optional complete associated room id list",
		},
		"examples": []any{map[string]any{"areaId": "300001", "name": "公共区", "roomIds": []any{"401398", "401399"}}},
		"nextStep": "Use area.detail.get or entity.list first when replacing roomIds; roomIds is a complete association list, not an add/remove patch.",
	}
}

func deviceMoveRoomBatchPayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"items": []any{
				map[string]any{"deviceId": "required device id", "roomId": "required target room id"},
			},
			"itemsAsMap": "accepted alternative object: {\"deviceId\":\"roomId\"}",
		},
		"examples": []any{
			map[string]any{"items": []any{map[string]any{"deviceId": "50018330", "roomId": "401398"}}},
			map[string]any{"items": map[string]any{"50018330": "401398", "50018430": "401398"}},
		},
		"nextStep": "Send 1..20 explicit device-to-room moves. Runtime verifies every device and target room belong to the current home.",
	}
}

func entityRenameBatchPayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"items": []any{
				map[string]any{
					"entityType":  "required: device or scene",
					"id":          "required unless currentName uniquely resolves the target",
					"currentName": "optional current name for unique target resolution",
					"newName":     "required new display name",
					"typeId":      "accepted explicit type id: 2=device, 6=scene",
				},
			},
		},
		"examples": []any{map[string]any{"items": []any{
			map[string]any{"entityType": "device", "id": "50018330", "newName": "阅读主灯"},
			map[string]any{"entityType": "scene", "currentName": "已有情景", "newName": "睡前晚安"},
		}}},
		"nextStep": "Use entity.rename.batch only for devices and scenes. Use room.rename, group.update, or area.update for other entity types.",
	}
}

func gatewayConfigurePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"gatewayId": "required gateway device id",
			"name":      "optional gateway name",
			"desc":      "optional description",
			"icon":      "optional icon",
			"mac":       "optional mac value when cloud accepts it",
			"roomIds":   "optional associated room id list",
		},
		"examples": []any{map[string]any{"gatewayId": "gw-1", "name": "客厅网关", "roomIds": []any{"401398"}}},
		"nextStep": "Call gateway.detail.get first when preserving current metadata. Runtime validates roomIds in the current home.",
	}
}

func metadataBatchDeletePayloadGuide(intent string) map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"items": []any{
				map[string]any{
					"roomId/areaId/groupId/sceneId/automationId": "one id field matching the delete intent",
					"name": "accepted when it uniquely resolves within the current home",
				},
			},
			"ids":   "accepted list of target ids",
			"names": "accepted list of unique target names",
		},
		"examples": []any{map[string]any{"items": []any{map[string]any{"sceneId": "700001"}, map[string]any{"name": "睡前晚安"}}}},
		"nextStep": "Delete batches are direct semantic requests after caller-side confirmation. Send 1..20 explicit targets; Runtime resolves and verifies each target.",
		"intent":   intent,
	}
}
