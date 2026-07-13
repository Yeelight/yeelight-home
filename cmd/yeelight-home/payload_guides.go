package main

import "github.com/yeelight/yeelight-home/internal/semantic"

func payloadGuideForIntent(intent string) map[string]any {
	switch intent {
	case "home.create":
		return homeCreatePayloadGuide()
	case "room.create":
		return roomCreatePayloadGuide()
	case "area.create":
		return areaCreatePayloadGuide()
	case "scene.create", "scene.update":
		return scenePayloadGuide(intent)
	case "automation.create", "automation.update":
		return automationPayloadGuide(intent)
	case "group.create":
		return groupCreatePayloadGuide()
	case "group.update":
		return groupUpdatePayloadGuide()
	case "group.members.update":
		return groupMembersPayloadGuide()
	case "room.rename":
		return roomRenamePayloadGuide()
	case "device.rename":
		return deviceRenamePayloadGuide()
	case "device.move":
		return deviceMovePayloadGuide()
	case "lighting.design.apply":
		return lightingDesignApplyPayloadGuide()
	case "lighting.experience.apply":
		return lightingExperienceApplyPayloadGuide()
	case "light.power.set":
		return lightPowerSetPayloadGuide()
	case "light.brightness.set":
		return lightBrightnessSetPayloadGuide()
	case "light.brightness.adjust":
		return lightBrightnessAdjustPayloadGuide()
	case "light.color_temperature.set":
		return lightColorTemperatureSetPayloadGuide()
	case "light.color_temperature.adjust":
		return lightColorTemperatureAdjustPayloadGuide()
	case "light.color.set":
		return lightColorSetPayloadGuide()
	case "device.property.set":
		return devicePropertySetPayloadGuide()
	case "node.property.set":
		return nodePropertySetPayloadGuide()
	case "node.property.toggle":
		return nodePropertyTogglePayloadGuide()
	case "node.action.execute":
		return nodeActionExecutePayloadGuide()
	case "lighting.flow.execute":
		return lightingFlowExecutePayloadGuide()
	case "node.properties.set":
		return nodePropertiesSetPayloadGuide()
	case "node.property.batch_set":
		return nodePropertyBatchSetPayloadGuide()
	case "state.batch.query":
		return stateBatchQueryPayloadGuide()
	case "home.property.set":
		return homePropertySetPayloadGuide()
	case "home.property.get":
		return homePropertyGetPayloadGuide()
	case "panel.click":
		return panelClickPayloadGuide()
	case "sensor.event.write":
		return sensorEventWritePayloadGuide()
	case "lighting.design.import", "device.slot.create":
		return lightingDesignImportPayloadGuide()
	case "panel.button.configure":
		return panelButtonConfigurePayloadGuide()
	case "panel.button.type.get":
		return panelButtonTypeGetPayloadGuide()
	case "panel.button_event.update", "panel.button_event.batch_update":
		return panelButtonEventPayloadGuide()
	case "panel.button_event.reset":
		return panelButtonEventResetPayloadGuide()
	case "knob.configure":
		return knobConfigurePayloadGuide()
	case "knob.reset":
		return knobResetPayloadGuide()
	case "operation.batch.configure":
		return operationBatchConfigurePayloadGuide()
	case "home.sort.configure":
		return homeSortConfigurePayloadGuide()
	case "home.member.invite", "home.member.accept_share", "home.member.configure", "home.member.remove", "home.member.transfer", "home.member.quit":
		return homeMemberPayloadGuide(intent)
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
	case "node.property_config.get":
		return nodePropertyConfigGetPayloadGuide()
	case "app_upgrade.latest.get":
		return appUpgradeLatestGetPayloadGuide()
	case "progress.get":
		return progressGetPayloadGuide()
	case "thing.schema.get", "thing.schema.event.list", "thing.component.get", "thing.product.info.batch_get", "thing.product.info.v3.batch_get":
		return thingModelPayloadGuide(intent)
	case "thing.product_faq.list", "thing.product_faq.detail.get", "thing.product_faq.page.list", "thing.product_faq.page_detail.list":
		return productFAQPayloadGuide(intent)
	case "product.pedia.search":
		return productPediaSearchPayloadGuide()
	case "operation.lesson.record":
		return operationLessonRecordPayloadGuide()
	case "recommendation.feedback":
		return recommendationFeedbackPayloadGuide()
	default:
		return nil
	}
}

func nodePropertyTogglePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile",
			semantic.FieldNodeType:   "required scope type: home, room, area, group, or device",
			semantic.FieldNodeID:     "required installed node id; for home, houseId can be used",
			semantic.FieldTargetType: "optional alias for nodeType",
			semantic.FieldTargetID:   "optional alias for nodeId",
			semantic.FieldProperty:   "required boolean writable property name such as power",
			semantic.FieldPayload:    "optional extra endpoint payload",
			semantic.FieldDuration:   "optional control duration when supported by the property",
			semantic.FieldDelay:      "optional delayed control value when supported by the property",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldNodeType: "room", semantic.FieldNodeID: "401398", semantic.FieldProperty: "power"},
		},
		semantic.FieldNextStep: "Use node.property.toggle only when the UI already knows the exact node and boolean property. Ordinary on/off should prefer light.power.set.",
	}
}

func nodeActionExecutePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile",
			semantic.FieldNodeType:   "required scope type: home, room, area, group, or device",
			semantic.FieldNodeID:     "required installed node id",
			semantic.FieldActionName: "required actionName from capability evidence",
			semantic.FieldPayload:    "optional action payload object",
			semantic.FieldDuration:   "optional duration when the action supports it",
			semantic.FieldDelay:      "optional delay when the action supports it",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldNodeType: "device", semantic.FieldNodeID: "50018330", semantic.FieldActionName: "set_mode", semantic.FieldPayload: map[string]any{semantic.FieldMode: "auto"}},
		},
		semantic.FieldNextStep: "Use actionName from device.complex.get, group.complex.get, entity.capabilities, or thing schema evidence. Do not invent action names from UI labels.",
	}
}

func lightingFlowExecutePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:  "required unless supplied by selected profile",
			semantic.FieldNodeType: "required scope type: home, room, area, group, or device",
			semantic.FieldNodeID:   "required installed node id",
			semantic.FieldFlow:     "required flow payload supported by the selected light capability",
			semantic.FieldPayload:  "optional alias body when the flow object must be passed as endpoint payload",
			semantic.FieldDuration: "optional duration",
			semantic.FieldDelay:    "optional delay",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldNodeType: "group", semantic.FieldNodeID: "8001", semantic.FieldFlow: map[string]any{"mode": "rainbow", semantic.FieldDuration: 30}},
		},
		semantic.FieldNextStep: "Use lighting.flow.execute as an advanced lighting effect capability only after the selected node exposes compatible flow support.",
	}
}

func nodePropertiesSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile",
			semantic.FieldNodeType:   "required scope type: home, room, area, group, or device",
			semantic.FieldNodeID:     "required installed node id",
			semantic.FieldProperties: "required object of writable propertyName:value pairs",
			semantic.FieldSet:        "accepted alias for properties",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldNodeType: "room", semantic.FieldNodeID: "401398", semantic.FieldProperties: map[string]any{semantic.FieldPower: true, semantic.FieldBrightness: 80}},
		},
		semantic.FieldNextStep: "Only include properties the target exposes as writable. Runtime filters sensitive fields but does not infer missing values.",
	}
}

func nodePropertyBatchSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:  "required unless supplied by selected profile",
			semantic.FieldNodeType: "required shared node type for all targets",
			semantic.FieldNodeIDs:  "required array of installed node ids; ids/deviceIds/roomIds/areaIds/groupIds are accepted aliases",
			semantic.FieldProperty: "required writable property name",
			semantic.FieldValue:    "required property value applied to all nodes",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldNodeType: "device", semantic.FieldNodeIDs: []any{"50018330", "50018331"}, semantic.FieldProperty: "power", semantic.FieldValue: false},
		},
		semantic.FieldNextStep: "Use batch-set when the caller has exact ids for several nodes of the same nodeType. For mixed entity types, call operation.batch.configure or several direct controls.",
	}
}

func stateBatchQueryPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required for non-device scope nodes unless supplied by selected profile",
			semantic.FieldItems:      "optional array of {nodeType,nodeId,property|properties}; preferred for mixed targets",
			semantic.FieldNodeType:   "optional shared node type when nodeIds are used",
			semantic.FieldNodeIDs:    "optional array of ids for shared nodeType",
			semantic.FieldDeviceIDs:  "optional array of device ids",
			semantic.FieldProperty:   "optional single property name",
			semantic.FieldProperties: "optional selected property list",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldItems: []any{
				map[string]any{semantic.FieldNodeType: "device", semantic.FieldNodeID: "50018330", semantic.FieldProperty: "power"},
				map[string]any{semantic.FieldNodeType: "room", semantic.FieldNodeID: "401398", semantic.FieldProperties: []any{semantic.FieldPower, semantic.FieldBrightness}},
			}},
		},
		semantic.FieldNextStep: "Use exact ids. This is a fan-out read helper; it should not be used as a write preflight when the UI already has current entity state.",
	}
}

func homePropertyGetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID: "required unless supplied by selected profile",
		},
		semantic.FieldExamples: []any{map[string]any{semantic.FieldHouseID: "200171"}},
		semantic.FieldNextStep: "Home properties are advanced family metadata. Present only user-understandable fields in generated apps.",
	}
}

func homePropertySetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile",
			semantic.FieldProperties: "required object of home metadata fields to write",
			semantic.FieldPayload:    "accepted alias for properties",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldHouseID: "200171", semantic.FieldProperties: map[string]any{semantic.FieldDisplayName: "我的家"}},
		},
		semantic.FieldNextStep: "Use only after the product meaning of each property key is known. Do not expose raw metadata editing to ordinary generated-app users.",
	}
}

func panelClickPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldPanelID:  "required panel/device resource id; deviceId/targetId/id accepted aliases",
			semantic.FieldPayload:  "optional click payload supported by the panel endpoint",
			semantic.FieldButtonID: "optional when the endpoint payload needs a button id",
			semantic.FieldEventID:  "optional when the endpoint payload needs an event id",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldPanelID: "50018330", semantic.FieldPayload: map[string]any{semantic.FieldButtonEventID: "101"}},
		},
		semantic.FieldNextStep: "panel.click simulates hardware click/test behavior. Runtime supports it, but production user templates should default to panel button configuration and scene execution instead.",
	}
}

func sensorEventWritePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldOperation: "create, update, delete/remove, or test",
			semantic.FieldEventID:   "required for update/delete",
			semantic.FieldPayload:   "required sensor event payload for create/update/test",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldOperation: "update", semantic.FieldEventID: "9001", semantic.FieldPayload: map[string]any{semantic.FieldName: "人在触发"}},
		},
		semantic.FieldNextStep: "sensor.event.write edits sensor-event configuration. Keep it separate from automation.create/update, which owns user automation rules.",
	}
}

func productFAQPayloadGuide(intent string) map[string]any {
	shape := map[string]any{}
	examples := []any{}
	nextStep := "Use product FAQ reads for product-help evidence. Search or page first when the FAQ id is unknown; do not treat FAQ content as installed-device state."

	switch intent {
	case "thing.product_faq.detail.get":
		shape[semantic.FieldFAQID] = "required FAQ id returned by product FAQ list/page reads"
		shape[semantic.FieldID] = "accepted alias for faqId when the source evidence names the row id"
		examples = append(examples, map[string]any{semantic.FieldFAQID: "773"})
		nextStep = "Pass faqId from a prior product FAQ list/page response. If the user gives only a product word or question, use a product FAQ search or detailed FAQ page read first."
	case "thing.product_faq.list":
		shape[semantic.FieldCapabilityProductID] = "optional product capability/firmware identity; this is not an exact SKU"
		shape[semantic.FieldKeyword] = "optional FAQ search text from the user question"
		shape[semantic.FieldLocale] = "optional locale such as zh-CN"
		shape[semantic.FieldLanguageCode] = "optional alias for locale/language"
		shape[semantic.FieldPageNo] = "optional page number, starting from 1"
		shape[semantic.FieldPageSize] = "optional page size"
		examples = append(examples, map[string]any{semantic.FieldKeyword: "重置", semantic.FieldPageNo: 1, semantic.FieldPageSize: 5})
		nextStep = "Use keyword for natural user questions. Add capabilityPid only when product context is already known from product pedia/schema evidence."
	case "thing.product_faq.page.list", "thing.product_faq.page_detail.list":
		shape[semantic.FieldCapabilityProductID] = "optional product capability/firmware identity; Runtime can also use it as moduleId when the FAQ endpoint expects module context"
		shape[semantic.FieldModuleID] = "optional FAQ module/product context id when returned by FAQ catalog evidence"
		shape[semantic.FieldKeyword] = "optional FAQ search text from the user question"
		shape[semantic.FieldLocale] = "optional locale such as zh-CN"
		shape[semantic.FieldLanguageCode] = "optional alias for locale/language"
		shape[semantic.FieldPageNo] = "optional page number, starting from 1"
		shape[semantic.FieldPageSize] = "optional page size"
		examples = append(examples, map[string]any{semantic.FieldKeyword: "重置", semantic.FieldPageNo: 1, semantic.FieldPageSize: 5})
		nextStep = "Use a fuller FAQ read when the user wants answer snippets and context in one step. Use a concise FAQ read for compact rows."
	}

	return map[string]any{
		semantic.FieldPayloadShape: shape,
		semantic.FieldExamples:     examples,
		semantic.FieldNextStep:     nextStep,
	}
}

func productPediaSearchPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldQuery:               "primary product search text from the user wording",
			semantic.FieldKeyword:             "accepted alias for query",
			semantic.FieldMultiField:          "accepted alias for fuzzy multi-field search",
			semantic.FieldName:                "accepted alias for product name or fuzzy wording",
			semantic.FieldProductName:         "exact or fuzzy product name",
			semantic.FieldProductShortName:    "short product name when known",
			semantic.FieldSKU:                 "SKU wording when known",
			semantic.FieldSKUCode:             "exact selected SKU/material code such as 1-000003268",
			semantic.FieldProductSKU:          "product SKU wording when Runtime or product docs use it",
			semantic.FieldSPU:                 "SPU wording when known",
			semantic.FieldProductSPU:          "product SPU wording when Runtime or product docs use it",
			semantic.FieldModel:               "model wording when known",
			semantic.FieldProductModel:        "product model wording when known",
			semantic.FieldModelNo:             "model number when known",
			semantic.FieldBarcode:             "barcode when known",
			semantic.FieldCapabilityProductID: "optional capability/firmware identity when product context already supplied it",
			semantic.FieldLimit:               "optional maximum result count",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldQuery: "青空灯说明书"},
			map[string]any{semantic.FieldSKUCode: "1-000003268"},
			map[string]any{semantic.FieldProductModel: "YP-0117"},
		},
		semantic.FieldNextStep: "Use the most specific product wording available. Product pedia is product documentation evidence only; use entity.capabilities or state.query for installed-device truth.",
	}
}

func thingModelPayloadGuide(intent string) map[string]any {
	shape := map[string]any{
		semantic.FieldHouseID: "optional context house id; product schema reads are product-context reads and do not prove installed-device capability",
	}
	examples := []any{}
	nextStep := "Use product schema reads only for product capability vocabulary. For installed-device execution, use entity.capabilities or state.query evidence."
	switch intent {
	case "thing.schema.get":
		shape[semantic.FieldCapabilityProductID] = "required product capability/firmware identity such as 198666; this is not the exact SKU"
		examples = append(examples, map[string]any{semantic.FieldCapabilityProductID: "198666"})
		nextStep = "Pass capabilityPid from Runtime product or entity evidence to read the product schema summary. Do not treat it as an exact SKU."
	case "thing.schema.event.list":
		shape[semantic.FieldCapabilityProductID] = "required product capability/firmware identity whose events should be listed"
		examples = append(examples, map[string]any{semantic.FieldCapabilityProductID: "198666"})
		nextStep = "Pass capabilityPid to list product event vocabulary for automation planning; it does not prove a specific installed device event is available."
	case "thing.component.get":
		shape[semantic.FieldID] = "optional exact component id from thing.component.list or product schema evidence"
		shape[semantic.FieldComponentName] = "optional natural component name when the user says a component word instead of an id"
		shape[semantic.FieldName] = "accepted alias of componentName"
		examples = append(examples, map[string]any{semantic.FieldComponentName: "亮度"})
		nextStep = "Use componentName/name for natural component words. Use id only when Runtime or product schema evidence already supplied an exact component id."
	case "thing.product.info.batch_get":
		shape[semantic.FieldCapabilityProductID] = "optional single product capability/firmware identity"
		shape[semantic.FieldCapabilityProductIDs] = "optional array or comma-separated list of product capability/firmware identities; use when querying multiple products"
		examples = append(examples, map[string]any{semantic.FieldCapabilityProductIDs: []any{"198666", "198661"}})
		nextStep = "Pass capabilityPid for one product or capabilityPids for several products. Use product.pedia.search when the user gives marketing/SKU wording instead of a capability id."
	case "thing.product.info.v3.batch_get":
		shape[semantic.FieldCapabilityProductID] = "optional single product capability/firmware identity"
		shape[semantic.FieldCapabilityProductIDs] = "optional array or comma-separated list of product capability/firmware identities; use when querying multiple products"
		shape[semantic.FieldVersion] = "required product schema version number or string when using v3 product info"
		shape[semantic.FieldSchemaVersion] = "optional alias for version when the source evidence names it schemaVersion"
		examples = append(examples, map[string]any{semantic.FieldCapabilityProductID: "198666", semantic.FieldVersion: 2})
		nextStep = "Pass capabilityPid/capabilityPids plus version or schemaVersion. If version is unknown, first use thing.schema.list or product.pedia.search to find product context."
	}
	return map[string]any{
		semantic.FieldPayloadShape: shape,
		semantic.FieldExamples:     examples,
		semantic.FieldNextStep:     nextStep,
	}
}

func nodePropertyConfigGetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeID:     "required unless a device target or deviceId supplies the installed node id",
			semantic.FieldDeviceID:   "optional alias for nodeId when the node is an installed device",
			semantic.FieldNodeType:   "required installed node type such as device, room, group, or scene when supported by Runtime evidence",
			semantic.FieldType:       "optional alias for nodeType",
			semantic.FieldEntityType: "optional alias for nodeType when it describes the installed node type",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldNodeID:   "50018376",
				semantic.FieldNodeType: "device",
			},
		},
		semantic.FieldNextStep: "Use node.property_config.get only for installed-node configuration evidence. Pass nodeType explicitly; if the cloud endpoint returns unavailable evidence, report partial unknowns instead of inventing properties.",
	}
}

func appUpgradeLatestGetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldAppType:      "required app type: yeelight/user/用户版 or 1, installer/师傅版 or 2, tv or 3, commercial/商照saas or 4",
			semantic.FieldOSType:       "required OS type: android/安卓 or 1, ios/苹果 or 2",
			semantic.FieldLanguageCode: "optional language code such as zh-CN",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldAppType:      "yeelight",
				semantic.FieldOSType:       "android",
				semantic.FieldLanguageCode: "zh-CN",
			},
		},
		semantic.FieldNextStep: "Use semantic appType and osType values; Runtime maps them to the cloud enum fields before reading the latest app upgrade record.",
	}
}

func progressGetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldProgressKey: "required task progress key from prior Runtime output or explicit user context",
			semantic.FieldKey:         "optional alias for progressKey",
			semantic.FieldID:          "optional alias for progressKey",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldProgressKey: "job-1"},
		},
		semantic.FieldNextStep: "Use progress.get only when a concrete progressKey/key is already known. If no progress key exists, explain that live progress cannot be read yet instead of inventing one.",
	}
}

func lightPowerSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:     "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType or entityType",
			semantic.FieldTargetID:   "optional alias for nodeId or entityId",
			semantic.FieldEntityType: "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:     "optional area id for direct area light control",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for direct light group control",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldRoomID:     "optional room id for direct room light control",
			semantic.FieldRoomName:   "optional room qualifier for duplicate device names",
			semantic.FieldPower:      "required boolean power value; true turns on, false turns off",
			semantic.FieldValue:      "optional alias for power; accepts boolean or on/off wording",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "孩子屋",
				semantic.FieldDeviceName: "吸顶灯",
				semantic.FieldPower:      true,
			},
		},
		semantic.FieldNextStep: "Use light.power.set for direct light on/off. It supports home, room, area, group, and device targets; pass nodeType+nodeId when the UI already has exact ids.",
	}
}

func lightBrightnessSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:     "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType or entityType",
			semantic.FieldTargetID:   "optional alias for nodeId or entityId",
			semantic.FieldEntityType: "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:     "optional area id for direct area light control",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for direct light group control",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldRoomID:     "optional room id for direct room light control",
			semantic.FieldRoomName:   "optional room qualifier for duplicate device names",
			semantic.FieldBrightness: "required brightness integer 1..100",
			semantic.FieldValue:      "optional alias for brightness integer 1..100",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "主灯",
				semantic.FieldBrightness: 60,
			},
		},
		semantic.FieldNextStep: "Use light.brightness.set for direct brightness changes. It supports home, room, area, group, and device targets when the target supports brightness.",
	}
}

func lightBrightnessAdjustPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:     "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType or entityType",
			semantic.FieldTargetID:   "optional alias for nodeId or entityId",
			semantic.FieldEntityType: "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:     "optional area id for direct area light adjustment",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for direct light group adjustment",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldRoomID:     "optional room id for direct room light adjustment",
			semantic.FieldRoomName:   "optional room qualifier or room scope name",
			semantic.FieldDelta:      "required signed brightness delta such as 10 or -10",
			semantic.FieldStep:       "optional alias for delta",
			semantic.FieldValue:      "optional alias for delta",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "主灯",
				semantic.FieldDelta:      -10,
			},
			map[string]any{
				semantic.FieldTargetType: "room",
				semantic.FieldTargetID:   "room-1",
				semantic.FieldDelta:      -10,
			},
		},
		semantic.FieldNextStep: "Use light.brightness.adjust for relative brightness changes. It supports home, room, area, group, and device targets when the target supports brightness. Use light.brightness.set when the user gives an absolute target value.",
	}
}

func lightColorTemperatureSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:          "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:         "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:           "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType:       "optional alias for nodeType or entityType",
			semantic.FieldTargetID:         "optional alias for nodeId or entityId",
			semantic.FieldEntityType:       "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:         "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName:       "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:           "optional area id for direct area light control",
			semantic.FieldAreaName:         "optional area name for natural area targeting",
			semantic.FieldGroupID:          "optional group id for direct light group control",
			semantic.FieldGroupName:        "optional group name for natural group targeting",
			semantic.FieldRoomID:           "optional room id for direct room light control",
			semantic.FieldRoomName:         "optional room qualifier for duplicate device names",
			semantic.FieldColorTemperature: "required color temperature integer 2700..6500",
			semantic.FieldValue:            "optional alias for color temperature integer 2700..6500",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:         "孩子屋",
				semantic.FieldDeviceName:       "吸顶灯",
				semantic.FieldColorTemperature: 3000,
			},
		},
		semantic.FieldNextStep: "Use light.color_temperature.set for direct color-temperature changes. It supports home, room, area, group, and device targets when the target supports color temperature.",
	}
}

func lightColorTemperatureAdjustPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:     "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType or entityType",
			semantic.FieldTargetID:   "optional alias for nodeId or entityId",
			semantic.FieldEntityType: "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:     "optional area id for direct area color-temperature adjustment",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for direct light group color-temperature adjustment",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldRoomID:     "optional room id for direct room color-temperature adjustment",
			semantic.FieldRoomName:   "optional room qualifier or room scope name",
			semantic.FieldDelta:      "required signed color-temperature delta such as 300 or -300",
			semantic.FieldStep:       "optional alias for delta",
			semantic.FieldValue:      "optional alias for delta",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "孩子屋",
				semantic.FieldDeviceName: "吸顶灯",
				semantic.FieldDelta:      -300,
			},
			map[string]any{
				semantic.FieldTargetType: "group",
				semantic.FieldTargetID:   "group-1",
				semantic.FieldDelta:      -300,
			},
		},
		semantic.FieldNextStep: "Use light.color_temperature.adjust for relative warmer/cooler changes. It supports home, room, area, group, and device targets when the target supports color temperature. Use light.color_temperature.set for an absolute Kelvin target.",
	}
}

func lightColorSetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "optional scope type: home, room, area, group, or device; use with nodeId for direct scope control",
			semantic.FieldNodeID:     "optional installed node id; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType or entityType",
			semantic.FieldTargetID:   "optional alias for nodeId or entityId",
			semantic.FieldEntityType: "optional target entity type: home, room, area, group, or device",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldAreaID:     "optional area id for direct area light control",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for direct light group control",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldRoomID:     "optional room id for direct room light control",
			semantic.FieldRoomName:   "optional room qualifier for duplicate device names",
			semantic.FieldColor:      "required RGB integer 0..16777215, hex string, or object with red/green/blue 0..255",
			semantic.FieldHex:        "optional rrggbb or #rrggbb color string",
			semantic.FieldValue:      "optional alias for RGB integer 0..16777215",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "氛围灯",
				semantic.FieldColor: map[string]any{
					semantic.FieldRed:   255,
					semantic.FieldGreen: 136,
					semantic.FieldBlue:  170,
				},
			},
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "氛围灯",
				semantic.FieldHex:        "#ff88aa",
			},
		},
		semantic.FieldNextStep: "Use light.color.set only for RGB-capable targets. It supports home, room, area, group, and device targets when the target supports color.",
	}
}

func devicePropertySetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldDeviceID:   "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName: "optional device name when the target is unique or qualified by roomName",
			semantic.FieldRoomName:   "optional room qualifier for duplicate device names",
			semantic.FieldProperty:   "required public Runtime property name or known alias, such as targetPosition, targetRotaryAngle, airConditionerPower, airConditionerTargetTemperature, airConditionerMode, airConditionerFanSpeed, switchPower, volume, or height",
			semantic.FieldValue:      "required explicit value to write. Use boolean for power-like properties, integer for percent/temperature/mode/fan/angle properties, and string only when the Runtime property is string-valued. Sensitive properties are blocked.",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "南向梦幻帘",
				semantic.FieldProperty:   "targetPosition",
				semantic.FieldValue:      50,
			},
			map[string]any{
				semantic.FieldRoomName:   "主卧",
				semantic.FieldDeviceName: "空调",
				semantic.FieldProperty:   "airConditionerTargetTemperature",
				semantic.FieldValue:      26,
			},
			map[string]any{
				semantic.FieldDeviceName: "玄关开关",
				semantic.FieldProperty:   "switchPower",
				semantic.FieldValue:      true,
			},
		},
		semantic.FieldNextStep: "Use device.property.set only when the UI or Runtime evidence already has a concrete writable property and explicit value. Do not use it for scenes, automations, sensitive credentials, or vague mood requests.",
	}
}

func nodePropertySetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldNodeType:   "required unless targets[] or entityType supplies the scope type: home, room, area, group, or device",
			semantic.FieldNodeID:     "required unless the target can be resolved by name; for home, houseId is used as the node id",
			semantic.FieldTargetType: "optional alias for nodeType",
			semantic.FieldTargetID:   "optional alias for nodeId",
			semantic.FieldEntityType: "optional alias for nodeType",
			semantic.FieldEntityID:   "optional alias for nodeId",
			semantic.FieldRoomID:     "optional room id for room scope",
			semantic.FieldRoomName:   "optional room name for natural room targeting",
			semantic.FieldAreaID:     "optional area id for area scope",
			semantic.FieldAreaName:   "optional area name for natural area targeting",
			semantic.FieldGroupID:    "optional group id for group scope",
			semantic.FieldGroupName:  "optional group name for natural group targeting",
			semantic.FieldDeviceID:   "optional device id for device scope",
			semantic.FieldDeviceName: "optional device name for natural device targeting",
			semantic.FieldProperty:   "required writable property id or Runtime property alias, such as p, l, ct, c, targetPosition, targetRotaryAngle, switchPower, volume, or height",
			semantic.FieldValue:      "required explicit value to write. Sensitive properties are blocked.",
			semantic.FieldDuration:   "optional transition duration when the target property supports it",
			semantic.FieldDelay:      "optional delayed execution when the target property supports it",
			semantic.FieldIndex:      "optional sub-device index when the target property supports it",
			semantic.FieldCategory:   "optional property category when the target property supports it",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldNodeType: "home",
				semantic.FieldProperty: "p",
				semantic.FieldValue:    true,
			},
			map[string]any{
				semantic.FieldNodeType: "room",
				semantic.FieldNodeID:   "room-1",
				semantic.FieldProperty: "l",
				semantic.FieldValue:    70,
			},
			map[string]any{
				semantic.FieldTargetType: "area",
				semantic.FieldTargetID:   "area-1",
				semantic.FieldProperty:   "ct",
				semantic.FieldValue:      3500,
			},
		},
		semantic.FieldNextStep: "Use node.property.set when the UI already knows it is controlling a whole home, room, area, group, or device node. Prefer light.* set intents for ordinary light p/l/ct/c controls when the UI wants semantic light wording.",
	}
}

func lightingExperienceApplyPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:          "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldDeviceID:         "optional device id from Runtime evidence; prefer natural target when unique",
			semantic.FieldDeviceName:       "optional device name when the target is unique or qualified by roomName",
			semantic.FieldRoomName:         "optional room qualifier for duplicate device names",
			semantic.FieldBrightness:       "optional explicit temporary brightness integer 1..100",
			semantic.FieldColorTemperature: "optional explicit temporary color temperature integer 2700..6500",
			semantic.FieldColor:            "optional explicit temporary RGB integer 0..16777215, hex string, or object with red/green/blue 0..255",
			semantic.FieldHex:              "optional rrggbb or #rrggbb color string alias for color",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRoomName:         "客厅",
				semantic.FieldDeviceName:       "氛围灯",
				semantic.FieldBrightness:       35,
				semantic.FieldColorTemperature: 3000,
			},
			map[string]any{
				semantic.FieldRoomName:   "客厅",
				semantic.FieldDeviceName: "氛围灯",
				semantic.FieldHex:        "#ff88aa",
			},
		},
		semantic.FieldNextStep: "Use lighting.experience.apply only when the user already specified the temporary light action. Runtime delegates to direct light control and does not invent brightness, colorTemperature, or color from mood words alone.",
	}
}

func panelButtonTypeGetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:    "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldDeviceID:   "required unless deviceName uniquely resolves the panel device",
			semantic.FieldDeviceName: "optional panel device name when unique",
			semantic.FieldButtonType: "required panel button type value from panel.get returned button rows, such as 2; this is not the click/hold button event type",
			semantic.FieldType:       "optional alias for buttonType",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldDeviceID:   "50018379",
				semantic.FieldButtonType: "2",
			},
		},
		semantic.FieldNextStep: "Call panel.get first when the button type value is unknown. Use the returned button row type for panel.button.type.get; use buttonEventId from panel.get for click/hold event update or reset.",
	}
}

func homeCreatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldName:        "required home name",
			semantic.FieldDescription: "optional home description",
			semantic.FieldIcon:        "optional home icon",
			semantic.FieldAreaCode:    "optional geographic area code from geo_area.search when needed",
			semantic.FieldAreaName:    "optional geographic area name",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldName:        "我的新家",
				semantic.FieldDescription: "用于照明设计和设备管理",
				semantic.FieldAreaName:    "上海",
			},
		},
		semantic.FieldNextStep: "Send name and optional descriptive metadata. If the user also asks for a full lighting design, prefer one lighting.design.import request instead of home.create plus separate design steps.",
	}
}

func roomCreatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:     "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldName:        "required room name",
			semantic.FieldRoomName:    "optional alias for name",
			semantic.FieldDescription: "optional room description",
			semantic.FieldIcon:        "optional room icon",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldName:        "孩子屋",
				semantic.FieldDescription: "儿童房",
			},
		},
		semantic.FieldNextStep: "Use room.create for one new room. Use room.batch_create when the user explicitly asks for multiple rooms in one request.",
	}
}

func areaCreatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:     "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldName:        "required area name",
			semantic.FieldDescription: "optional area description",
			semantic.FieldIcon:        "optional area icon",
			semantic.FieldParentID:    "optional parent area id from Runtime evidence",
			semantic.FieldRoomIDs:     "optional associated room id list from Runtime evidence",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldName:    "一楼公共区",
				semantic.FieldRoomIDs: []any{"401391", "401392"},
			},
		},
		semantic.FieldNextStep: "Resolve any room membership from Runtime evidence, then send name plus optional roomIds. Use area.update later for complete room association changes.",
	}
}

func groupCreatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldName:            "required group name",
			semantic.FieldDescription:     "optional group description",
			semantic.FieldIcon:            "optional group icon",
			semantic.FieldRoomID:          "optional room id from Runtime evidence; omit when roomName uniquely identifies the room",
			semantic.FieldRoomName:        "optional room name; Runtime resolves a unique room name",
			semantic.FieldGroupCategory:   "optional semantic category such as lighting",
			semantic.FieldGroupCapability: "required semantic capability such as light; Runtime derives the group capability model from this public value",
			semantic.FieldDeviceIDs:       "required unless deviceNames supplies at least one resolvable member device id list from Runtime evidence",
			semantic.FieldDeviceNames:     "required unless deviceIds supplies at least one member; Runtime resolves unique device names within the room when supplied",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldName:            "客厅格栅灯组",
				semantic.FieldRoomName:        "客厅",
				semantic.FieldGroupCategory:   "lighting",
				semantic.FieldGroupCapability: "light",
				semantic.FieldDeviceNames:     []any{"左侧格栅灯", "右侧格栅灯"},
			},
		},
		semantic.FieldNextStep: "Send name, roomName or roomId, groupCapability, and deviceNames or deviceIds. Prefer names from the user request; Runtime resolves and validates unique targets.",
	}
}

func groupUpdatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:        "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldGroupID:        "required unless currentName/groupName uniquely resolves the group",
			semantic.FieldCurrentName:    "optional current group name for unique target resolution",
			semantic.FieldGroupName:      "optional alias for currentName",
			semantic.FieldName:           "optional new group name",
			semantic.FieldDescription:    "optional group description",
			semantic.FieldIcon:           "optional group icon",
			semantic.FieldRoomID:         "optional target room id from Runtime evidence",
			semantic.FieldTargetRoomName: "optional target room name for unique target resolution",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldGroupID:     "4767",
				semantic.FieldName:        "客厅格栅氛围灯组",
				semantic.FieldDescription: "客厅同类型格栅灯统一控制",
			},
			map[string]any{
				semantic.FieldGroupName:      "客厅格栅灯组",
				semantic.FieldName:           "客厅格栅氛围灯组",
				semantic.FieldTargetRoomName: "客厅",
			},
		},
		semantic.FieldNextStep: "Use group.update for group name, description, icon, or room assignment only. Use group.members.update when membership must change.",
	}
}

func groupMembersPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:           "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldGroupID:           "required unless groupName/currentName uniquely resolves the group",
			semantic.FieldGroupName:         "optional current group name for unique target resolution",
			semantic.FieldDeviceIDs:         "optional complete target member device id list; Runtime computes add/remove from current group detail",
			semantic.FieldDeviceNames:       "optional complete target member device name list; Runtime resolves unique names and computes add/remove",
			semantic.FieldAddDeviceIDs:      "optional device ids to add",
			semantic.FieldAddDeviceNames:    "optional unique device names to add",
			semantic.FieldRemoveDeviceIDs:   "optional device ids to remove",
			semantic.FieldRemoveDeviceNames: "optional unique device names to remove",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldGroupID:   "4767",
				semantic.FieldDeviceIDs: []any{"50018330", "50018331", "50018332"},
			},
			map[string]any{
				semantic.FieldGroupName:         "客厅格栅灯组",
				semantic.FieldAddDeviceNames:    []any{"餐边柜灯"},
				semantic.FieldRemoveDeviceNames: []any{"旧灯带"},
			},
		},
		semantic.FieldNextStep: "For normal editing UIs, send the complete selected deviceIds or deviceNames list. Runtime will read current group detail, compute the delta, write addDeviceList/removeDeviceList, and verify by group detail.",
	}
}

func roomRenamePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:     "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldRoomID:      "required unless currentName/roomName uniquely resolves the room",
			semantic.FieldCurrentName: "optional current room name for unique target resolution",
			semantic.FieldRoomName:    "optional alias for currentName",
			semantic.FieldNewName:     "required new room name",
			semantic.FieldName:        "accepted alias for newName when roomName/currentName identifies the target",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldRoomID: "401398", semantic.FieldNewName: "影音室"},
			map[string]any{semantic.FieldRoomName: "客厅", semantic.FieldNewName: "影音室"},
		},
		semantic.FieldNextStep: "Use room.rename for one room. Prefer roomId when already known; otherwise use roomName/currentName plus newName so Runtime can resolve and verify the target.",
	}
}

func deviceRenamePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:     "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldDeviceID:    "required unless currentName/deviceName uniquely resolves the device",
			semantic.FieldCurrentName: "optional current device name for unique target resolution",
			semantic.FieldDeviceName:  "optional alias for currentName",
			semantic.FieldNewName:     "required new device name",
			semantic.FieldName:        "accepted alias for newName when deviceName/currentName identifies the target",
			semantic.FieldAlias:       "accepted alias for newName",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldDeviceID: "50018330", semantic.FieldNewName: "阅读主灯"},
			map[string]any{semantic.FieldDeviceName: "主灯", semantic.FieldNewName: "阅读主灯"},
		},
		semantic.FieldNextStep: "Use device.rename for one device. Prefer deviceId when already known; otherwise use deviceName/currentName plus newName so Runtime can resolve and verify the target.",
	}
}

func deviceMovePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:        "required unless supplied by selected profile, --house-id, or homeRef",
			semantic.FieldDeviceID:       "required unless currentName/deviceName uniquely resolves the device",
			semantic.FieldCurrentName:    "optional current device name for unique target resolution",
			semantic.FieldDeviceName:     "optional alias for currentName",
			semantic.FieldRoomID:         "required unless targetRoomName uniquely resolves the target room",
			semantic.FieldTargetRoomID:   "accepted alias for roomId",
			semantic.FieldTargetRoomName: "optional target room name for unique target resolution",
			semantic.FieldRoomName:       "optional alias for targetRoomName",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldDeviceID: "50018330", semantic.FieldRoomID: "401398"},
			map[string]any{semantic.FieldDeviceName: "主灯", semantic.FieldTargetRoomName: "客厅"},
		},
		semantic.FieldNextStep: "Use device.move for one device room assignment. Runtime resolves natural device and room names only when each name is unique in the selected home.",
	}
}

func scenePayloadGuide(intent string) map[string]any {
	sceneIDShape := "omit for scene.create"
	nameShape := "required scene name"
	extraShape := map[string]any{}
	if intent == "scene.update" {
		sceneIDShape = "optional scene id; omit when sceneName/currentName uniquely resolves the scene"
		nameShape = "optional updated scene name; omit to preserve the current name"
		extraShape[semantic.FieldSceneName] = "optional current scene name for unique target resolution"
		extraShape[semantic.FieldCurrentName] = "optional alias for current scene name"
		extraShape[semantic.FieldEntityName] = "optional alias for current scene name"
		extraShape[semantic.FieldTargetName] = "optional alias for current scene name"
		extraShape[semantic.FieldNewName] = "optional alias for updated scene name"
	}
	shape := map[string]any{
		semantic.FieldSceneID:     sceneIDShape,
		semantic.FieldName:        nameShape,
		semantic.FieldDescription: "optional",
		semantic.FieldIcon:        "optional",
		semantic.FieldActions:     []any{sceneActionItemShape()},
	}
	for key, value := range extraShape {
		shape[key] = value
	}
	return map[string]any{
		semantic.FieldPayloadShape: shape,
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldIntent: "scene.create example; omit sceneId",
				semantic.FieldName:   "孩子屋开灯",
				semantic.FieldActions: []any{
					map[string]any{
						semantic.FieldTargetType: "device",
						semantic.FieldTargetID:   "50018330",
						semantic.FieldTargetName: "孩子屋吸顶灯",
						semantic.FieldRank:       0,
						semantic.FieldSet: map[string]any{
							semantic.FieldPower:            true,
							semantic.FieldBrightness:       60,
							semantic.FieldColorTemperature: 3000,
						},
					},
				},
			},
			map[string]any{
				semantic.FieldIntent:    "scene.update example; preserve full actions[] from scene.detail.get editablePayload",
				semantic.FieldSceneName: "孩子屋开灯",
				semantic.FieldNewName:   "孩子屋开灯",
				semantic.FieldActions: []any{
					map[string]any{
						semantic.FieldTargetType: "device",
						semantic.FieldTargetID:   "50018330",
						semantic.FieldTargetName: "孩子屋吸顶灯",
						semantic.FieldRank:       0,
						semantic.FieldSet: map[string]any{
							semantic.FieldPower:            true,
							semantic.FieldBrightness:       60,
							semantic.FieldColorTemperature: 3000,
						},
					},
				},
			},
		},
		semantic.FieldNextStep: "For scene.update, use sceneId when known or sceneName/currentName when unique, keep the complete actions list when preserving an existing scene, edit only the intended action set, then send scene.update with the complete updated list. For scene.create, send name plus complete action rows.",
	}
}

func automationPayloadGuide(intent string) map[string]any {
	automationIDShape := "omit for automation.create"
	nameShape := "required automation name"
	extraShape := map[string]any{}
	if intent == "automation.update" {
		automationIDShape = "optional automation id; omit when automationName/currentName uniquely resolves the automation"
		nameShape = "optional updated automation name; omit to preserve the current name"
		extraShape[semantic.FieldAutomationName] = "optional current automation name for unique target resolution"
		extraShape[semantic.FieldCurrentName] = "optional alias for current automation name"
		extraShape[semantic.FieldEntityName] = "optional alias for current automation name"
		extraShape[semantic.FieldTargetName] = "optional alias for current automation name"
		extraShape[semantic.FieldNewName] = "optional alias for updated automation name"
	}
	shape := map[string]any{
		semantic.FieldAutomationID: automationIDShape,
		semantic.FieldName:         nameShape,
		semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "optional HH:mm:ss; defaults to 00:00:00", semantic.FieldEnd: "optional HH:mm:ss; defaults to 23:59:59"},
		semantic.FieldRepeat:       "optional repeat preset: daily, weekdays, weekend, once, custom, legal_holiday, legal_workday. Defaults to daily.",
		semantic.FieldVersion:      "optional",
		semantic.FieldTrigger:      automationConditionParamsShape()[semantic.FieldTrigger],
		semantic.FieldConditions:   automationConditionParamsShape()[semantic.FieldConditions],
		semantic.FieldActions:      []any{automationActionItemShape()},
	}
	for key, value := range extraShape {
		shape[key] = value
	}
	return map[string]any{
		semantic.FieldPayloadShape: shape,
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldIntent:       "automation.create example; omit automationId",
				semantic.FieldName:         "主卧每天9点开灯",
				semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "00:00:00", semantic.FieldEnd: "23:59:59"},
				semantic.FieldRepeat:       "daily",
				semantic.FieldTrigger:      map[string]any{semantic.FieldConditionKind: "alarm", semantic.FieldTime: "09:00:00"},
				semantic.FieldActions:      []any{map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldTargetName: "主卧吸顶灯", semantic.FieldRank: 0, semantic.FieldSet: map[string]any{semantic.FieldPower: true, semantic.FieldBrightness: 60, semantic.FieldColorTemperature: 3000}}},
			},
			map[string]any{
				semantic.FieldIntent:       "automation.create non-timer example; use only with Runtime/product evidence",
				semantic.FieldName:         "开灯后柔和一点",
				semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "00:00:00", semantic.FieldEnd: "23:59:59"},
				semantic.FieldRepeat:       "daily",
				semantic.FieldTrigger: map[string]any{
					semantic.FieldConditionType: "and",
					semantic.FieldConditions: []any{
						map[string]any{semantic.FieldConditionKind: "fact_change", semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldCapabilityProductID: 198666, semantic.FieldProperty: semantic.FieldPower, semantic.FieldValue: true},
						map[string]any{semantic.FieldConditionKind: "fact", semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldCapabilityProductID: 198666, semantic.FieldProperty: semantic.FieldBrightness, semantic.FieldOperation: "gt", semantic.FieldValue: 10},
					},
				},
				semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldTargetName: "客厅灯", semantic.FieldRank: 0, semantic.FieldSet: map[string]any{semantic.FieldPower: true, semantic.FieldBrightness: 35, semantic.FieldColorTemperature: 3000}}},
			},
			map[string]any{
				semantic.FieldIntent:         "automation.update example; preserve full rule from automation.detail.get editablePayload",
				semantic.FieldAutomationName: "主卧每天9点开灯",
				semantic.FieldNewName:        "主卧每天9点开灯",
				semantic.FieldActiveWindow:   map[string]any{semantic.FieldStart: "00:00:00", semantic.FieldEnd: "23:59:59"},
				semantic.FieldRepeat:         "daily",
				semantic.FieldTrigger:        map[string]any{semantic.FieldConditionKind: "alarm", semantic.FieldTime: "09:00:00"},
				semantic.FieldActions: []any{
					map[string]any{
						semantic.FieldTargetType: "device",
						semantic.FieldTargetID:   "50018330",
						semantic.FieldTargetName: "主卧吸顶灯",
						semantic.FieldRank:       0,
						semantic.FieldSet: map[string]any{
							semantic.FieldPower:            true,
							semantic.FieldBrightness:       60,
							semantic.FieldColorTemperature: 3000,
						},
					},
				},
			},
		},
		semantic.FieldNextStep: "For automation.update, use automationId when known or automationName/currentName when unique, keep the complete trigger/condition/action payload when preserving an existing rule, edit only intended fields, then send automation.update. Use automation.enable or automation.disable for status changes.",
	}
}

func lightingDesignApplyPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldTargets: "device, room, group, or area targets; Runtime resolves to controllable devices",
			semantic.FieldActions: []any{
				map[string]any{
					semantic.FieldTargetType: "device",
					semantic.FieldTargetID:   "optional device id when known",
					semantic.FieldTargetName: "natural device name accepted when id is unknown",
					semantic.FieldSet: map[string]any{
						semantic.FieldPower:            "optional bool",
						semantic.FieldBrightness:       "optional 1..100",
						semantic.FieldColorTemperature: "optional 2700..6500",
						semantic.FieldColor:            "optional RGB integer or rrggbb hex",
					},
				},
			},
			semantic.FieldDirectFields: map[string]any{
				semantic.FieldPower:            "optional bool",
				semantic.FieldBrightness:       "optional 1..100",
				semantic.FieldColorTemperature: "optional 2700..6500",
				semantic.FieldColor:            "optional RGB integer or rrggbb hex",
			},
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldActions: []any{
					map[string]any{
						semantic.FieldTargetType: "device",
						semantic.FieldTargetName: "主灯",
						semantic.FieldSet: map[string]any{
							semantic.FieldPower:            true,
							semantic.FieldBrightness:       60,
							semantic.FieldColorTemperature: 3000,
						},
					},
				},
			},
		},
		semantic.FieldNextStep: "Use lighting.design.apply only for real device state changes. The caller/Skill must translate subjective mood words into explicit actions[] or direct power/brightness/colorTemperature/color fields before calling Runtime. Runtime does not infer an executable recipe from mood/design text. Use lighting.design.import for rooms, device slots, groups, scenes, or automations.",
	}
}

func lightingDesignImportPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:         "existing home only; omit for new-home import",
			semantic.FieldKey:             "optional stable design key",
			semantic.FieldName:            "required home/design name",
			semantic.FieldGatewayName:     "optional gateway display name",
			semantic.FieldGatewayDeviceID: "optional existing gateway device id when importing into a real gateway",
			semantic.FieldRooms: []any{
				map[string]any{
					semantic.FieldKey:  "required stable room key referenced by areas/actions",
					semantic.FieldName: "required room name",
					semantic.FieldIcon: "optional room icon",
					semantic.FieldDeviceSlots: []any{
						map[string]any{
							semantic.FieldKey:  "required stable slot key referenced by groups/actions",
							semantic.FieldName: "required slot display name",
							semantic.FieldProduct: map[string]any{
								semantic.FieldSKUCode:             "required concrete selected SKU number",
								semantic.FieldCapabilityProductID: "required selected product capability identity; it describes the shared controllable capability class and is not the exact SKU",
								semantic.FieldProductComponentID:  "required selected product component id when known",
								semantic.FieldProductName:         "optional selected product display name",
							},
							semantic.FieldNotes: "optional installer/design notes",
						},
					},
					semantic.FieldGroups: []any{
						map[string]any{
							semantic.FieldKey:             "required stable group key referenced by actions",
							semantic.FieldName:            "required group name",
							semantic.FieldGroupCategory:   "optional semantic category such as lighting",
							semantic.FieldGroupCapability: "required semantic capability such as light",
							semantic.FieldSlotKeys:        "required list of device slot keys in this room",
						},
					},
				},
			},
			semantic.FieldAreas: []any{
				map[string]any{
					semantic.FieldKey:      "required stable area key",
					semantic.FieldName:     "required area name",
					semantic.FieldIcon:     "optional area icon",
					semantic.FieldRoomKeys: "required list of room keys",
				},
			},
			semantic.FieldScenes: []any{
				map[string]any{
					semantic.FieldKey:     "required stable scene key",
					semantic.FieldName:    "required scene name",
					semantic.FieldIcon:    "optional scene icon",
					semantic.FieldActions: []any{lightingDesignImportActionShape()},
				},
			},
			semantic.FieldAutomations: []any{
				map[string]any{
					semantic.FieldKey:          "required stable automation key",
					semantic.FieldName:         "required automation name",
					semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "optional HH:mm:ss; defaults to 00:00:00", semantic.FieldEnd: "optional HH:mm:ss; defaults to 23:59:59"},
					semantic.FieldRepeat:       "optional repeat preset: daily, weekdays, weekend, once, custom, legal_holiday, legal_workday. Defaults to daily.",
					semantic.FieldTrigger:      automationConditionParamsShape()[semantic.FieldTrigger],
					semantic.FieldActions:      []any{lightingDesignImportActionShape()},
				},
			},
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldName: "粒粒的美丽家庭",
				semantic.FieldRooms: []any{
					map[string]any{
						semantic.FieldKey:  "living",
						semantic.FieldName: "客厅",
						semantic.FieldDeviceSlots: []any{
							map[string]any{semantic.FieldKey: "living-grille-1", semantic.FieldName: "黑色格栅灯1", semantic.FieldProduct: map[string]any{semantic.FieldSKUCode: "1-000002044", semantic.FieldCapabilityProductID: 198666, semantic.FieldProductComponentID: 4, semantic.FieldProductName: "selected product"}},
							map[string]any{semantic.FieldKey: "living-grille-2", semantic.FieldName: "黑色格栅灯2", semantic.FieldProduct: map[string]any{semantic.FieldSKUCode: "1-000002044", semantic.FieldCapabilityProductID: 198666, semantic.FieldProductComponentID: 4, semantic.FieldProductName: "selected product"}},
						},
						semantic.FieldGroups: []any{
							map[string]any{semantic.FieldKey: "living-grilles", semantic.FieldName: "客厅格栅灯组", semantic.FieldGroupCategory: "lighting", semantic.FieldGroupCapability: "light", semantic.FieldSlotKeys: []any{"living-grille-1", "living-grille-2"}},
						},
					},
				},
				semantic.FieldScenes: []any{
					map[string]any{semantic.FieldKey: "living-home", semantic.FieldName: "客厅回家模式", semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "group", semantic.FieldTargetKey: "living-grilles", semantic.FieldTargetName: "客厅格栅灯组", semantic.FieldRank: 0, semantic.FieldDelay: 0, semantic.FieldSet: map[string]any{semantic.FieldPower: true, semantic.FieldBrightness: 60, semantic.FieldColorTemperature: 3000}}}},
				},
				semantic.FieldAutomations: []any{
					map[string]any{semantic.FieldKey: "living-9am", semantic.FieldName: "客厅每天9点", semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "00:00:00", semantic.FieldEnd: "23:59:59"}, semantic.FieldRepeat: "daily", semantic.FieldTrigger: map[string]any{semantic.FieldConditionKind: "alarm", semantic.FieldTime: "09:00:00"}, semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "group", semantic.FieldTargetKey: "living-grilles", semantic.FieldTargetName: "客厅格栅灯组", semantic.FieldRank: 0, semantic.FieldDelay: 0, semantic.FieldSet: map[string]any{semantic.FieldPower: true}}}},
					map[string]any{semantic.FieldKey: "living-power-soften", semantic.FieldName: "开灯后柔和一点", semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "00:00:00", semantic.FieldEnd: "23:59:59"}, semantic.FieldRepeat: "daily", semantic.FieldTrigger: map[string]any{semantic.FieldConditionType: "and", semantic.FieldConditions: []any{map[string]any{semantic.FieldConditionKind: "fact_change", semantic.FieldTargetType: "device", semantic.FieldTargetKey: "living-grille-1", semantic.FieldCapabilityProductID: 198666, semantic.FieldProperty: semantic.FieldPower, semantic.FieldValue: true}, map[string]any{semantic.FieldConditionKind: "fact", semantic.FieldTargetType: "device", semantic.FieldTargetKey: "living-grille-1", semantic.FieldCapabilityProductID: 198666, semantic.FieldProperty: semantic.FieldBrightness, semantic.FieldOperation: "gt", semantic.FieldValue: 10}}}, semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "group", semantic.FieldTargetKey: "living-grilles", semantic.FieldTargetName: "客厅格栅灯组", semantic.FieldRank: 0, semantic.FieldSet: map[string]any{semantic.FieldPower: true, semantic.FieldBrightness: 35, semantic.FieldColorTemperature: 3000}}}},
				},
			},
		},
		semantic.FieldNextStep: "Generate the standard lighting design model. The caller owns product selection, quantity expansion, same-type grouping, and subjective lighting recipes. Runtime validates references, fills small deterministic defaults, submits the import, and verifies the resulting home topology. For device.slot.create in an existing home, use a new design-room name; Runtime blocks same-name existing rooms because the current import path would duplicate the room instead of appending slots.",
	}
}

func lightingDesignImportActionShape() map[string]any {
	return map[string]any{
		semantic.FieldTargetType: "standard target kind: room, device, group, home, or scene",
		semantic.FieldTargetKey:  "stable key of an imported room/device slot/group/scene in the same request",
		semantic.FieldTargetName: "optional target display name; Runtime can backfill from targetKey",
		semantic.FieldRank:       "optional action order integer",
		semantic.FieldSet:        lightActionParamsShape()[semantic.FieldSet],
	}
}
