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
	return map[string]any{
		"payloadShape": map[string]any{
			"houseId": "optional in parameters when already selected by --house-id/profile; required by house-scoped execution",
			"tempId":  "optional temporary home id, default hm1",
			"name":    "required-ish home/design name; Runtime defaults to AI照明设计 when omitted",
			"version": "optional metadata version; Runtime defaults to 2",
			"gateway": map[string]any{
				"tempId":          "required when gatewayDeviceId is not provided; Runtime defaults to gw1",
				"name":            "required-ish gateway display name; Runtime defaults to 默认网关",
				"pid":             "required by Java GatewayMeta; Runtime defaults to a conservative gateway placeholder when omitted",
				"gatewayDeviceId": "optional existing gateway device id when importing into a real gateway",
				"roomList": []any{
					map[string]any{
						"tempId": "required room temporary id referenced by devices, areas, and action rows",
						"name":   "required room name",
						"icon":   "optional room icon; Runtime defaults to room_1",
						"deviceList": []any{
							map[string]any{
								"tempId":     "required device-slot temporary id referenced by groups and action rows",
								"name":       "required slot/device display name",
								"pid":        "required product id. Skill/caller must select the product before import; Runtime may only backfill from explicit materialCode",
								"roomTempId": "optional when nested under a room; Runtime fills the parent room tempId",
								"extraMeta":  "optional string map for selected product and installer notes",
							},
						},
						"groupList": []any{
							map[string]any{
								"tempId":           "required group temporary id referenced by scene/automation action rows",
								"name":             "required group name",
								"componentId":      "required group component id from selected products; all member slots should be compatible",
								"deviceTempIdList": "required list of imported device tempIds in this group",
							},
						},
					},
				},
			},
			"areaList": []any{
				map[string]any{
					"tempId":         "required area temporary id",
					"name":           "required area name",
					"icon":           "optional area icon; Runtime defaults to area_1",
					"roomTempIdList": "required list of imported room tempIds",
				},
			},
			"sceneList": []any{
				map[string]any{
					"tempId":  "required scene temporary id",
					"name":    "required scene name; Runtime truncates to the backend 14-character limit",
					"icon":    "optional scene icon; Runtime defaults to scene_1",
					"details": []any{houseMetaImportActionShape(true)},
				},
			},
			"automationList": []any{
				map[string]any{
					"tempId":      "required automation temporary id",
					"name":        "required automation name; Runtime truncates to the backend 14-character limit",
					"startTime":   "optional HH:mm:ss active window start; Runtime defaults to 00:00:00",
					"endTime":     "optional HH:mm:ss active window end; Runtime defaults to 23:59:59",
					"repeatType":  "optional 1..7; Runtime defaults to 2 daily",
					"repeatValue": "optional repeat bitmask such as 0x7f; Runtime defaults to daily",
					"version":     "optional; Runtime defaults to 2",
					"params":      automationConditionParamsShape(),
					"actions":     []any{houseMetaImportActionShape(false)},
				},
			},
			"shortKeyCompatibility": map[string]any{
				"core":    "tid->tempId, n->name, rl->roomList, dl->deviceList, gl->groupList, al->areaList, sl->sceneList, atl->automationList",
				"refs":    "rtids->roomTempIdList, dtids->deviceTempIdList, cid->componentId, tpid->typeId, rn->resName",
				"actions": "as->actions, ds->details, ap->params, rk->rank, ps->params, tp->type, cs->conditions, c->clock, s->set, i->index, v->value",
			},
			"sceneActionContract": sceneShape["details"],
			"automationContract":  map[string]any{"params": automationShape["params"], "actions": automationShape["actions"]},
		},
		"examples": []any{
			map[string]any{
				"name": "粒粒的美丽家庭",
				"gateway": map[string]any{
					"tempId": "gw1",
					"name":   "默认网关",
					"roomList": []any{map[string]any{
						"tempId": "rm1",
						"name":   "客厅",
						"deviceList": []any{
							map[string]any{"tempId": "dv1", "name": "黑色格栅灯1", "pid": 198666, "materialCode": "1-000002044", "productName": "Skill selected product"},
							map[string]any{"tempId": "dv2", "name": "黑色格栅灯2", "pid": 198666, "materialCode": "1-000002044", "productName": "Skill selected product"},
						},
						"groupList": []any{
							map[string]any{"tempId": "gp1", "name": "客厅格栅灯组", "componentId": 4, "deviceTempIdList": []any{"dv1", "dv2"}},
						},
					}},
				},
				"sceneList": []any{
					map[string]any{"tempId": "sc1", "name": "客厅回家模式", "details": []any{map[string]any{"typeId": 4, "tempId": "gp1", "resName": "客厅格栅灯组", "rank": 0, "params": map[string]any{"delay": 0, "set": map[string]any{"p": true, "l": 60, "ct": 3000}}}}},
				},
				"automationList": []any{
					map[string]any{"tempId": "at1", "name": "客厅每天9点", "startTime": "00:00:00", "endTime": "23:59:59", "repeatType": 2, "repeatValue": "0x7f", "params": map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "alarm", "clock": "09:00:00"}}}, "actions": []any{map[string]any{"typeId": 4, "tempId": "gp1", "resName": "客厅格栅灯组", "rank": 0, "params": map[string]any{"delay": 0, "set": map[string]any{"p": true}}}}},
				},
			},
			map[string]any{
				"tid": "hm1",
				"n":   "紧凑短键示例",
				"gateway": map[string]any{
					"tid": "gw1",
					"n":   "默认网关",
					"rl": []any{map[string]any{
						"tid": "rm1",
						"n":   "主卧",
						"dl": []any{
							map[string]any{"tid": "dv1", "n": "36°射灯1", "pid": 198666, "mc": "1-000004714"},
							map[string]any{"tid": "dv2", "n": "36°射灯2", "pid": 198666, "mc": "1-000004714"},
						},
						"gl": []any{map[string]any{"tid": "gp1", "n": "主卧36°射灯组", "cid": 4, "dtids": []any{"dv1", "dv2"}}},
					}},
				},
				"atl": []any{
					map[string]any{"tid": "at1", "n": "主卧每天9点", "st": "00:00:00", "et": "23:59:59", "rt": 2, "rv": "0x7f", "ps": map[string]any{"tp": "and", "cs": []any{map[string]any{"tp": "alarm", "c": "09:00:00"}}}, "as": []any{map[string]any{"tpid": 4, "tid": "gp1", "rn": "主卧36°射灯组", "ap": map[string]any{"s": map[string]any{"p": true}}}}},
				},
			},
		},
		"nextStep": "Generate HouseMeta for /v1/meta/import. The caller/Skill owns product selection, same-type grouping, and subjective lighting recipes. Runtime only validates references, fills small deterministic defaults, expands short keys, compacts params JSON, and posts meta.import. Natural rooms/items/groups/scenes/automations payloads are not executable import payloads.",
	}
}

func houseMetaImportActionShape(scene bool) map[string]any {
	shape := map[string]any{
		"typeId":  "required imported target type. Use 1=room, 2=device slot, 4=group, 5=home, 6=imported scene",
		"tempId":  "required tempId of an imported room/device/group/scene, not a cloud resId and not a future-* placeholder",
		"resName": "optional target display name; Runtime backfills it from tempId when possible",
		"rank":    "optional order integer; Runtime defaults to the row index",
		"params":  lightActionParamsShape(),
	}
	if scene {
		shape["action"] = "optional cloud action code; Runtime defaults to 0"
	}
	return shape
}
