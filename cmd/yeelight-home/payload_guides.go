package main

func payloadGuideForIntent(intent string) map[string]any {
	switch intent {
	case "scene.create", "scene.update":
		return scenePayloadGuide()
	case "automation.create", "automation.update":
		return automationPayloadGuide()
	case "lighting.design.apply":
		return lightingDesignApplyPayloadGuide()
	case "lighting.design.import", "device.slot.create":
		return lightingDesignImportPayloadGuide()
	case "panel.button.configure":
		return panelButtonConfigurePayloadGuide()
	case "panel.button_event.update", "panel.button_event.batch_update":
		return panelButtonEventPayloadGuide()
	case "knob.configure":
		return knobConfigurePayloadGuide()
	case "operation.batch.configure":
		return operationBatchConfigurePayloadGuide()
	case "home.sort.configure":
		return homeSortConfigurePayloadGuide()
	case "favorite.add", "favorite.update", "favorite.batch_add", "favorite.batch_update", "favorite.delete", "favorite.batch_delete":
		return favoritePayloadGuide(intent)
	case "room.batch_create", "room.batch_update":
		return roomBatchPayloadGuide(intent)
	case "room.area.configure":
		return roomAreaConfigurePayloadGuide()
	case "area.update":
		return areaUpdatePayloadGuide()
	case "device.move_room.batch":
		return deviceMoveRoomBatchPayloadGuide()
	case "entity.rename.batch":
		return entityRenameBatchPayloadGuide()
	case "gateway.configure":
		return gatewayConfigurePayloadGuide()
	case "room.batch_delete", "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		return metadataBatchDeletePayloadGuide(intent)
	default:
		return nil
	}
}

func scenePayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"sceneId":     "required for scene.update; omit for scene.create",
			"name":        "required scene name",
			"description": "optional",
			"icon":        "optional",
			"details":     []any{sceneActionItemShape()},
		},
		"examples": []any{
			map[string]any{
				"intent": "scene.create example; omit sceneId",
				"name":   "孩子屋开灯",
				"details": []any{
					map[string]any{
						"typeId":  2,
						"resId":   "50018330",
						"resName": "孩子屋吸顶灯",
						"action":  0,
						"rank":    0,
						"params": map[string]any{
							"set": map[string]any{
								"p":  true,
								"l":  60,
								"ct": 3000,
							},
						},
					},
				},
			},
			map[string]any{
				"intent":  "scene.update example; preserve full details[] from scene.detail.get editablePayload",
				"sceneId": "scene-1",
				"name":    "孩子屋开灯",
				"details": []any{
					map[string]any{
						"typeId":  2,
						"resId":   "50018330",
						"resName": "孩子屋吸顶灯",
						"action":  0,
						"rank":    0,
						"params": map[string]any{
							"set": map[string]any{
								"p":  true,
								"l":  60,
								"ct": 3000,
							},
						},
					},
				},
			},
		},
		"nextStep": "For scene.update, call scene.detail.get first, keep the complete details list, edit only the intended action params, then send scene.update with the complete updated list. For scene.create, send name plus the desired complete details list.",
	}
}

func automationPayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"automationId": "required for automation.update; omit for automation.create",
			"name":         "required automation name",
			"startTime":    "required HH:mm:ss active window start",
			"endTime":      "required HH:mm:ss active window end",
			"repeatType":   "required 1..7; 2 means daily",
			"repeatValue":  "optional; custom repeat uses a hex bitmask such as 0x7f",
			"version":      "optional",
			"params":       automationConditionParamsShape(),
			"actions":      []any{automationActionItemShape()},
		},
		"examples": []any{
			map[string]any{
				"intent":      "automation.create example; omit automationId",
				"name":        "主卧每天9点开灯",
				"startTime":   "00:00:00",
				"endTime":     "23:59:59",
				"repeatType":  2,
				"repeatValue": "0x7f",
				"params":      map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "alarm", "clock": "09:00:00"}}},
				"actions":     []any{map[string]any{"typeId": 2, "resId": "50018330", "resName": "主卧吸顶灯", "rank": 0, "params": map[string]any{"set": map[string]any{"p": true, "l": 60, "ct": 3000}}}},
			},
			map[string]any{
				"intent":       "automation.update example; preserve full rule from automation.detail.get editablePayload",
				"automationId": "auto-1",
				"name":         "主卧每天9点开灯",
				"startTime":    "00:00:00",
				"endTime":      "23:59:59",
				"repeatType":   2,
				"repeatValue":  "0x7f",
				"params": map[string]any{
					"type": "and",
					"conditions": []any{
						map[string]any{
							"type":  "alarm",
							"clock": "09:00:00",
						},
					},
				},
				"actions": []any{
					map[string]any{
						"typeId":  2,
						"resId":   "50018330",
						"resName": "主卧吸顶灯",
						"rank":    0,
						"params": map[string]any{
							"set": map[string]any{
								"p":  true,
								"l":  60,
								"ct": 3000,
							},
						},
					},
				},
			},
		},
		"nextStep": "For automation.update, call automation.detail.get first, keep the complete rule payload, edit only the intended condition/action fields, then send automation.update. Use automation.enable or automation.disable for status changes.",
	}
}

func lightingDesignApplyPayloadGuide() map[string]any {
	return map[string]any{
		"payloadShape": map[string]any{
			"targets": "device, room, group, or area targets; Runtime resolves to controllable devices",
			"actions": []any{
				map[string]any{
					"deviceId":     "required when multiple target devices are present",
					"propertyName": "one of p, l, ct, c",
					"value":        "p=bool, l=1..100, ct=2700..6500, c=0..16777215 or rrggbb",
				},
			},
			"directFields": map[string]any{
				"power":            "optional bool; alias p/on",
				"brightness":       "optional 1..100; alias l/level",
				"colorTemperature": "optional 2700..6500; aliases ct/color_temperature",
				"color":            "optional RGB integer or rrggbb hex; alias hex",
			},
		},
		"examples": []any{
			map[string]any{
				"actions": []any{
					map[string]any{"deviceId": "50018330", "propertyName": "p", "value": true},
					map[string]any{"deviceId": "50018330", "propertyName": "ct", "value": 3000},
					map[string]any{"deviceId": "50018330", "propertyName": "l", "value": 60},
				},
			},
		},
		"nextStep": "Use lighting.design.apply only for real device state changes. The caller/Skill must translate subjective mood words into explicit actions[] or direct power/brightness/colorTemperature/color fields before calling Runtime. Runtime does not infer an executable recipe from mood/design text. Use lighting.design.import for rooms, device slots, groups, scenes, or automations.",
	}
}

func lightingDesignImportPayloadGuide() map[string]any {
	sceneShape := scenePayloadGuide()["payloadShape"].(map[string]any)
	automationShape := automationPayloadGuide()["payloadShape"].(map[string]any)
	normalizedShape := normalizedLightingDesignTopologyShape()
	return map[string]any{
		"payloadShape": map[string]any{
			"houseId": "optional in parameters when already selected by --house-id/profile; required by house-scoped execution",
			"rooms": []any{
				map[string]any{
					"name":     "required room name for natural topology",
					"roomName": "accepted alias for name",
					"localId":  "optional local id; Runtime generates one when omitted",
					"items": []any{
						map[string]any{
							"name":         "required slot display name",
							"quantity":     "optional, default 1",
							"count":        "accepted alias for quantity",
							"category":     "optional product family",
							"color":        "optional",
							"installStyle": "optional",
							"beamAngle":    "optional",
							"series":       "optional",
							"materialCode": "recommended explicit product identity",
							"pid":          "optional product id",
							"pcId":         "optional product category/component id",
							"productName":  "optional product name",
							"productSku":   "optional product SKU name",
							"productSpu":   "optional product SPU name",
							"modelNo":      "optional product model number",
							"connectType":  "optional device connect type when known",
							"namePattern":  "optional slot naming pattern; use {n} for expanded quantity index",
							"groupKey":     "optional same-type grouping key; defaults to category/product/name",
							"notes":        "optional design assumption",
						},
					},
					"slots/devices": "accepted aliases for items[] in natural topology",
				},
			},
			"groups": []any{
				map[string]any{
					"name":     "caller-authored natural group alias; Runtime converts to deviceGroups[]",
					"roomName": "required imported room name",
					"match": map[string]any{
						"category":     "optional slot category matcher",
						"name":         "optional slot display-name contains matcher",
						"series":       "optional slot/product series matcher",
						"productName":  "optional exact product-name matcher",
						"materialCode": "optional exact material-code matcher",
						"groupKey":     "optional exact grouping-key matcher",
					},
				},
			},
			"scenes": []any{
				map[string]any{
					"name":    "required design scene name",
					"localId": "optional local id used by syncMetadata mapping",
					"details": []any{sceneActionItemShape()},
					"params":  "optional design metadata object when executable action rows are not known; use details[] for executable rows",
				},
			},
			"automations": []any{
				map[string]any{
					"name":        "required design automation name",
					"startTime":   "optional HH:mm:ss active window start",
					"endTime":     "optional HH:mm:ss active window end",
					"repeatType":  "optional 1..7",
					"repeatValue": "optional custom repeat value such as 0x7f",
					"localId":     "optional local id used by syncMetadata mapping",
					"params":      automationConditionParamsShape(),
					"actions":     []any{automationActionItemShape()},
				},
			},
			"normalizedAlternative": normalizedShape,
			"clearAll":              "optional high-impact overwrite flag",
			"overwrite":             "alias for clearAll",
			"async":                 "optional server-side async metadata sync flag",
			"sceneActionContract":   sceneShape["details"],
			"automationContract":    map[string]any{"params": automationShape["params"], "actions": automationShape["actions"]},
		},
		"examples": []any{
			map[string]any{
				"rooms": []any{
					map[string]any{
						"name": "客厅",
						"items": []any{
							map[string]any{"name": "吸顶灯", "quantity": 1, "category": "吸顶灯"},
							map[string]any{"name": "黑色格栅灯", "quantity": 2, "category": "格栅灯", "color": "黑色"},
						},
					},
				},
				"groups": []any{
					map[string]any{"name": "客厅格栅灯组", "roomName": "客厅", "match": map[string]any{"category": "格栅灯"}},
				},
			},
			map[string]any{
				"rooms": []any{
					map[string]any{"name": "主卧", "items": []any{map[string]any{"name": "36°射灯", "quantity": 4, "category": "射灯", "beamAngle": "36°", "materialCode": "1-000004714", "notes": "caller-selected focused accent slot candidate"}}},
				},
				"groups": []any{
					map[string]any{"name": "主卧36°射灯组", "roomName": "主卧", "match": map[string]any{"name": "36°射灯"}},
				},
				"scenes": []any{
					map[string]any{"name": "主卧暖光", "details": []any{map[string]any{"typeId": 2, "resId": "future-or-real-target-id", "resName": "主卧射灯组", "rank": 0, "params": map[string]any{"set": map[string]any{"p": true, "l": 50, "ct": 3000}}}}},
				},
				"automations": []any{
					map[string]any{"name": "主卧每天9点开灯", "startTime": "00:00:00", "endTime": "23:59:59", "repeatType": 2, "repeatValue": "0x7f", "params": map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "alarm", "clock": "09:00:00"}}}, "actions": []any{map[string]any{"typeId": 2, "resId": "future-or-real-target-id", "resName": "主卧射灯组", "rank": 0, "params": map[string]any{"set": map[string]any{"p": true, "l": 60, "ct": 3000}}}}},
				},
			},
		},
		"nextStep": "Use product selection evidence when possible, decide grouping in the caller/Skill layer, then send lighting.design.import once for the complete topology. Natural groups[] is accepted for caller-authored room-local group names and is converted to source-backed deviceGroups[].",
	}
}
