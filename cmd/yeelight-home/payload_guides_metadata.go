package main

import "github.com/yeelight/yeelight-home/internal/semantic"

func panelButtonConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldDeviceID:   "panel device id when known",
			semantic.FieldDeviceName: "natural panel device name accepted when id is unknown",
			semantic.FieldButtons: []any{
				map[string]any{
					semantic.FieldID:         "recommended existing button id; index/keyValue/name/alias can also locate current button",
					semantic.FieldAlias:      "optional display alias",
					semantic.FieldKeyValue:   "optional physical key value",
					semantic.FieldIndex:      "optional button index",
					semantic.FieldTargetID:   "optional bound resource id",
					semantic.FieldTargetType: "optional bound resource type: device, group, meshGroup, or scene",
					semantic.FieldVisible:    "optional 0/1",
					semantic.FieldIcon:       "optional icon value",
					semantic.FieldSort:       "optional sort order",
					semantic.FieldType:       "optional button display type; Runtime backfills from current panel button when possible",
					semantic.FieldExtend:     "optional extension object/string accepted by cloud",
				},
			},
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldDeviceName: "客厅面板",
				semantic.FieldButtons: []any{
					map[string]any{semantic.FieldID: "btn-1", semantic.FieldAlias: "回家", semantic.FieldTargetType: "scene", semantic.FieldTargetID: "scene-1", semantic.FieldVisible: 1},
				},
			},
		},
		semantic.FieldNextStep: "Call panel.get or panel button info first when preserving existing rows. Use deviceName for natural user wording; Runtime resolves the panel device and merges each row with the current cloud button before writing.",
	}
}

func panelButtonEventPayloadGuide() map[string]any {
	eventShape := map[string]any{
		semantic.FieldButtonEventID: "required existing button event id",
		semantic.FieldAlias:         "optional event alias such as 单击/双击/长按",
		semantic.FieldActions: []any{
			map[string]any{
				semantic.FieldRoomID:       "optional room id for target resource",
				semantic.FieldTargetType:   "target kind: device, group, meshGroup, or scene",
				semantic.FieldTargetID:     "controlled resource id",
				semantic.FieldTargetName:   "controlled resource display name",
				semantic.FieldSubIndex:     "optional sub-device index",
				semantic.FieldRank:         "optional order, default cloud behavior when omitted",
				semantic.FieldActiveWindow: map[string]any{semantic.FieldStart: "optional HH:mm:ss active start", semantic.FieldEnd: "optional HH:mm:ss active end"},
				semantic.FieldRepeat:       "optional repeat preset such as daily, weekdays, weekend, once, or custom",
				semantic.FieldSet:          lightActionParamsShape()[semantic.FieldSet],
			},
		},
	}
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldDeviceID:                          "panel device id when known",
			semantic.FieldDeviceName:                        "natural panel device name accepted when id is unknown",
			semantic.FieldButtonEvent:                       eventShape,
			semantic.ArrayField(semantic.FieldButtonEvents): eventShape,
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldDeviceName:    "客厅面板",
				semantic.FieldButtonEventID: "101",
				semantic.FieldAlias:         "单击",
				semantic.FieldActions: []any{
					map[string]any{semantic.FieldTargetType: "scene", semantic.FieldTargetID: "scene-1", semantic.FieldTargetName: "回家模式", semantic.FieldRank: 0},
				},
			},
			map[string]any{
				semantic.FieldDeviceName: "客厅面板",
				semantic.FieldButtonEvents: []any{
					map[string]any{semantic.FieldButtonEventID: "101", semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "scene", semantic.FieldTargetID: "scene-1"}}},
					map[string]any{semantic.FieldButtonEventID: "102", semantic.FieldActions: []any{map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldSet: map[string]any{semantic.FieldPower: true, semantic.FieldColorTemperature: 3000}}}},
				},
			},
		},
		semantic.FieldNextStep: "Use panel.button_event.update for one event and panel.button_event.batch_update for multiple events. Use deviceName for natural user wording. Each event must carry its complete actions list.",
	}
}

func panelButtonEventResetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldDeviceID:      "panel device id when known",
			semantic.FieldDeviceName:    "natural panel device name accepted when id is unknown",
			semantic.FieldButtonEventID: "required existing button event id from panel.get/button event evidence",
			semantic.FieldIndex:         "optional human-facing index only when Runtime evidence maps it to a button event id",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldDeviceName: "客厅面板", semantic.FieldButtonEventID: "101"},
		},
		semantic.FieldNextStep: "Use panel.button_event.reset only after the user explicitly asks to clear one existing event binding. Prefer buttonEventId from panel.get evidence; deviceName may identify the panel.",
	}
}

func knobConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldDeviceID:   "knob device id when known",
			semantic.FieldDeviceName: "natural knob device name accepted when id is unknown",
			semantic.FieldActions: []any{
				map[string]any{
					semantic.FieldIndex:       "required physical knob key index, 1-based",
					semantic.FieldConfigType:  "required knob config type when known",
					semantic.FieldTargetType:  "target kind: device, group, meshGroup, scene, or room",
					semantic.FieldTargetID:    "target resource id for bound target modes",
					semantic.FieldTargetName:  "target display name",
					semantic.FieldSet:         "light action set when applicable",
					semantic.FieldSubIndex:    "optional target sub-device index",
					semantic.FieldModel:       "optional knob model value",
					semantic.FieldSensitivity: "optional sensitivity",
					semantic.FieldMode:        "optional mode field returned by knob.get",
					semantic.FieldCustom:      "optional evidence-backed product-specific object preserved when returned by knob.get",
				},
			},
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldDeviceName: "客厅旋钮",
				semantic.FieldActions: []any{
					map[string]any{semantic.FieldIndex: 1, semantic.FieldConfigType: "scene", semantic.FieldTargetType: "scene", semantic.FieldTargetID: "scene-1", semantic.FieldTargetName: "回家模式"},
				},
			},
		},
		semantic.FieldNextStep: "Call knob.get first when editing an existing knob. Use deviceName for natural user wording; Runtime resolves the knob device. Preserve the current detail row and change only the intended index binding or parameter map.",
	}
}

func knobResetPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldDeviceID:   "knob device id when known",
			semantic.FieldDeviceName: "natural knob device name accepted when id is unknown",
			semantic.FieldIndex:      "required physical knob key index to reset",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldDeviceName: "客厅旋钮", semantic.FieldIndex: 1},
		},
		semantic.FieldNextStep: "Use knob.reset only after the user explicitly asks to clear one knob sub-key binding. deviceName may identify the knob; index selects the sub-key.",
	}
}

func operationBatchConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldOperations: []any{
				map[string]any{
					semantic.FieldIntent:     "required allowlisted add/update/configure intent",
					semantic.FieldParameters: "required object accepted by that intent",
					semantic.FieldTargets: []any{
						map[string]any{
							semantic.FieldEntityType: "optional target entity type",
							semantic.FieldID:         "optional target id",
							semantic.FieldName:       "optional natural target name",
						},
					},
				},
			},
		},
		semantic.FieldAllowedIntents: []any{
			"home.update", "home.sort.configure", "favorite.add/update/batch_add/batch_update",
			"room.create/rename/update/batch_create/batch_update/area.configure",
			"area.create/update", "device.rename/move/move_room.batch", "entity.rename.batch",
			"group.create/update", "scene.create/update", "automation.create/update/enable/disable",
			"gateway.configure", "panel.button.configure", "panel.button_event.update/batch_update",
			"knob.configure", "lighting.design.import", "device.slot.create",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldOperations: []any{
					map[string]any{semantic.FieldIntent: "room.create", semantic.FieldParameters: map[string]any{semantic.FieldName: "书房"}},
					map[string]any{semantic.FieldIntent: "device.rename", semantic.FieldParameters: map[string]any{semantic.FieldDeviceID: "50018330", semantic.FieldName: "书房主灯"}},
				},
			},
		},
		semantic.FieldNextStep: "Use one batch for one user request with multiple reversible add/update/configure steps. Keep delete, unbind, member transfer/remove, home create/delete, panel reset, and knob reset as separate semantic requests.",
	}
}

func operationLessonRecordPayloadGuide() map[string]any {
	lessonShape := map[string]any{
		semantic.FieldIntent:          "required target Runtime intent that the lesson is about, not operation.lesson.record",
		semantic.FieldLessonType:      "required one of fast_path, resource_resolution, parameter_shape, failure_pattern, fallback, capability_gap",
		semantic.FieldSymptom:         "required short symptom or repeated problem",
		semantic.FieldRecommendedPath: "required concise path future agents should use",
		semantic.FieldCause:           "optional known cause",
		semantic.FieldAvoid:           "optional anti-pattern to avoid",
		semantic.FieldParametersHint:  "optional compact payload hint",
		semantic.FieldFallbackIntent:  "optional fallback Runtime intent",
		semantic.FieldEvidence:        "optional sanitized evidence such as an error code or response summary",
		semantic.FieldSource:          "optional source tag such as ai_skill, runtime_response, or validated_cli",
		semantic.FieldConfidence:      "optional low, medium, or high",
		semantic.FieldStatus:          "optional candidate, confirmed, deprecated, or rejected",
		semantic.FieldStale:           "optional boolean",
		semantic.FieldLastValidatedAt: "optional Unix seconds",
	}
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID: "optional; omit for reusable profile-global lessons and include only when the lesson depends on one home's topology",
			semantic.FieldLesson:  lessonShape,
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldLesson: map[string]any{
					semantic.FieldIntent:          "scene.update",
					semantic.FieldLessonType:      "parameter_shape",
					semantic.FieldSymptom:         "invalid_scene_update_payload when guessing nested scene actions",
					semantic.FieldRecommendedPath: "Call scene.detail.get first, use editablePayload/updateShape, then send the complete updated action list.",
					semantic.FieldAvoid:           "Do not invent nested action rows from acceptedFields alone.",
					semantic.FieldSource:          "runtime_response",
					semantic.FieldConfidence:      "high",
					semantic.FieldStatus:          "confirmed",
				},
			},
		},
		semantic.FieldNextStep: "Record only reusable Runtime behavior, stable cloud boundaries, payload-shape rules, fallbacks, or fast paths that cannot be fixed in the current Skill/CLI flow. Do not record fixable CLI bugs, stale Skill rules, user preferences, or one-off topology snapshots.",
	}
}

func recommendationFeedbackPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldHouseID:          "required home context for the local recommendation store",
			semantic.FieldRecommendationID: "required exact recommendation id returned by recommendation.record or recommendation.list",
			semantic.FieldFeedback:         "required feedback value: accepted, rejected, dismissed, or cooldown; aliases such as accept/reject/hide/ignore are normalized",
			semantic.FieldCooldownHours:    "optional integer 1-720; only used when feedback is cooldown",
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldRecommendationID: "rec-123",
				semantic.FieldFeedback:         "dismissed",
			},
			map[string]any{
				semantic.FieldRecommendationID: "rec-123",
				semantic.FieldFeedback:         "cooldown",
				semantic.FieldCooldownHours:    24,
			},
		},
		semantic.FieldNextStep: "Use only a concrete recommendationId from Runtime output. For 'hide' or 'do not remind me again' wording, send feedback=dismissed; for explicit rejection, send feedback=rejected; for later reminders, send feedback=cooldown with cooldownHours when known.",
	}
}

func homeSortConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldSortType:       "required sort surface, such as device_room or home",
			semantic.FieldRoomID:         "room id when sortType targets resources inside a room; omit when roomName uniquely identifies the room",
			semantic.FieldRoomName:       "room name accepted for room-scoped sort targets",
			semantic.FieldTargetRoomName: "accepted alias for roomName",
			semantic.FieldItems: []any{
				map[string]any{
					semantic.FieldTargetType: "preferred standard target kind: room, device, group, meshGroup, scene, automation",
					semantic.FieldTargetID:   "resource id; omit when targetName/entityName uniquely identifies the resource",
					semantic.FieldTargetName: "resource name accepted when unique in the current home or room qualifier",
					semantic.FieldRank:       "required integer order rank",
					semantic.FieldSubIndex:   "optional sub-index for multi-channel targets",
				},
			},
		},
		semantic.FieldExamples: []any{
			map[string]any{
				semantic.FieldSortType: "device_room",
				semantic.FieldRoomName: "客厅",
				semantic.FieldItems: []any{
					map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetName: "主灯", semantic.FieldRank: 1},
					map[string]any{semantic.FieldTargetType: "scene", semantic.FieldTargetName: "回家模式", semantic.FieldRank: 2},
				},
			},
		},
		semantic.FieldNextStep: "Read home.sort.list first when preserving unspecified items. Send only explicit ordered items; Runtime verifies resources belong to the current home.",
	}
}

func homeMemberPayloadGuide(intent string) map[string]any {
	shape := map[string]any{
		semantic.FieldHouseID: "required for member operations except accept_share when the share payload already carries a target house id",
	}
	examples := []any{}
	nextStep := "Read home.member.list first when choosing an existing member. Runtime uses the current local account for sensitive caller identity and returns masked member evidence."
	switch intent {
	case "home.member.invite":
		shape[semantic.FieldExpiresAt] = "required Unix seconds when the invite expires"
		shape[semantic.FieldUserRole] = "optional invite role: normal/member/0 or admin/2; defaults to normal"
		shape[semantic.FieldReuseBarcode] = "optional boolean; defaults to true"
		examples = append(examples, map[string]any{
			semantic.FieldExpiresAt:    1783000000,
			semantic.FieldUserRole:     "normal",
			semantic.FieldReuseBarcode: true,
		})
		nextStep = "Generate an invite only after the user explicitly asks to share a home. Use an expiry time and normal/admin role; Runtime redacts returned share evidence."
	case "home.member.accept_share":
		shape[semantic.FieldShareID] = "required share id from a structured home-share invitation"
		shape[semantic.FieldCreateTime] = "required share create time from the same invitation"
		shape[semantic.FieldHouseID] = "required expected target home id from the invitation when available"
		examples = append(examples, map[string]any{
			semantic.FieldShareID:    "7001",
			semantic.FieldCreateTime: 1710000000,
			semantic.FieldHouseID:    "200171",
		})
		nextStep = "Accept only a structured share invitation with shareId and createTime. Do not ask the user for tokens; Runtime derives the recipient from the local login."
	case "home.member.configure":
		shape[semantic.FieldMemberID] = "optional member id when already known"
		shape[semantic.FieldMemberName] = "optional unique member display name; Runtime resolves it from the current member list without exposing raw ids"
		shape[semantic.FieldUserRole] = "required target role: normal/member/0 or admin/2; owner transfer is not supported here"
		examples = append(examples, map[string]any{
			semantic.FieldMemberName: "张三",
			semantic.FieldUserRole:   "admin",
		})
		nextStep = "Use configure only for normal/admin role changes. Use home.member.transfer for ownership transfer and apply R3 confirmation rules."
	case "home.member.remove", "home.member.transfer":
		shape[semantic.FieldMemberID] = "optional member id when already known; owner/master members are rejected"
		shape[semantic.FieldMemberName] = "optional unique member display name; Runtime resolves it from the current member list without exposing raw ids"
		shape[semantic.FieldConfirmed] = "required true only after explicit chat confirmation for execution"
		examples = append(examples, map[string]any{
			semantic.FieldMemberName: "张三",
			semantic.FieldConfirmed:  true,
		})
		nextStep = "This is an R3 permission-sensitive operation. Ask for explicit natural-language confirmation before execution; dry-run or missing confirmed must not mutate cloud state."
	case "home.member.quit":
		shape[semantic.FieldMemberID] = "optional member id; omit to use the current local account when Runtime can verify it is a non-owner shared-home member"
		shape[semantic.FieldConfirmed] = "required true only after explicit chat confirmation for execution"
		examples = append(examples, map[string]any{
			semantic.FieldConfirmed: true,
		})
		nextStep = "This is an R3 operation for leaving a shared home. Ask for explicit confirmation first; Runtime rejects owner/master quit through this path."
	}
	return map[string]any{
		semantic.FieldPayloadShape: shape,
		semantic.FieldExamples:     examples,
		semantic.FieldNextStep:     nextStep,
	}
}

func favoritePayloadGuide(intent string) map[string]any {
	itemShape := map[string]any{
		semantic.FieldFavoriteID: "needed when updating/deleting by favorite row id; omit for favorite.add",
		semantic.FieldTargetType: "required standard favorite target kind: device, group, meshGroup, or scene",
		semantic.FieldEntityType: "accepted alias for targetType when the request is phrased as an entity selection",
		semantic.FieldTargetID:   "resource id; omit when targetName/entityName uniquely identifies the resource",
		semantic.FieldEntityID:   "accepted alias for targetId when the request is phrased as an entity selection",
		semantic.FieldTargetName: "resource name accepted when it uniquely identifies the targetType inside the current home",
		semantic.FieldEntityName: "accepted alias for targetName",
		semantic.FieldRoomID:     "optional room qualifier for device or room-scoped scene names",
		semantic.FieldRoomName:   "optional room name qualifier for device or room-scoped scene names",
		semantic.FieldRank:       "optional order rank",
		semantic.FieldValid:      "optional bool enabled flag",
	}
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldFavoriteID: itemShape[semantic.FieldFavoriteID],
			semantic.FieldTargetType: itemShape[semantic.FieldTargetType],
			semantic.FieldEntityType: itemShape[semantic.FieldEntityType],
			semantic.FieldTargetID:   itemShape[semantic.FieldTargetID],
			semantic.FieldEntityID:   itemShape[semantic.FieldEntityID],
			semantic.FieldTargetName: itemShape[semantic.FieldTargetName],
			semantic.FieldEntityName: itemShape[semantic.FieldEntityName],
			semantic.FieldRoomID:     itemShape[semantic.FieldRoomID],
			semantic.FieldRoomName:   itemShape[semantic.FieldRoomName],
			semantic.FieldRank:       itemShape[semantic.FieldRank],
			semantic.FieldValid:      itemShape[semantic.FieldValid],
			semantic.FieldItems:      []any{itemShape},
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldRank: 1},
			map[string]any{semantic.FieldTargetType: "device", semantic.FieldRoomName: "客厅", semantic.FieldTargetName: "主灯", semantic.FieldRank: 1},
			map[string]any{semantic.FieldItems: []any{
				map[string]any{semantic.FieldTargetType: "device", semantic.FieldTargetID: "50018330", semantic.FieldRank: 1},
				map[string]any{semantic.FieldTargetType: "scene", semantic.FieldTargetName: "回家模式", semantic.FieldRank: 2},
			}},
		},
		semantic.FieldNextStep: "For batch intents, send items[] with 1..20 explicit favorite targets. Prefer targetName/entityName for user-facing requests when the targetType and name are unique; Runtime resolves and validates ids before writing. Cloud-verifiable favorites are devices, groups/mesh groups, and scenes. Delete can use favoriteId or an unambiguous targetType plus target id/name resolved from favorite.list.",
		semantic.FieldIntent:   intent,
	}
}

func roomBatchPayloadGuide(intent string) map[string]any {
	roomShape := map[string]any{
		semantic.FieldRoomID:            "required for room.batch_update; omit for room.batch_create",
		semantic.FieldName:              "required for create; optional update name",
		semantic.FieldDescription:       "optional description",
		semantic.FieldIcon:              "optional icon",
		semantic.FieldImage:             "optional image for update",
		semantic.FieldGatewayDeviceID:   "optional gateway device id",
		semantic.FieldGatewayIDs:        "optional gateway id list",
		semantic.FieldDefaultGatewayIDs: "optional default gateway id list",
		semantic.FieldSequence:          "optional sequence",
		semantic.FieldCapability:        "optional capability object/value accepted by cloud",
	}
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{semantic.FieldRooms: []any{roomShape}},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldRooms: []any{map[string]any{semantic.FieldName: "书房"}, map[string]any{semantic.FieldName: "茶室"}}},
			map[string]any{semantic.FieldRooms: []any{map[string]any{semantic.FieldRoomID: "401398", semantic.FieldName: "会客厅"}}},
		},
		semantic.FieldNextStep: "Use rooms[] or items[] with 1..20 room objects. Runtime rejects duplicate names, unknown update ids, and gateway references outside the home.",
		semantic.FieldIntent:   intent,
	}
}

func roomAreaConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldRoomID:          "room id; omit when roomName/currentName uniquely identifies the room",
			semantic.FieldRoomName:        "room name accepted when it uniquely identifies the room",
			semantic.FieldAddAreaIDs:      "optional list of area ids to add",
			semantic.FieldAddAreaNames:    "optional list of area names to add; Runtime resolves unique names",
			semantic.FieldRemoveAreaIDs:   "optional list of area ids to remove",
			semantic.FieldRemoveAreaNames: "optional list of area names to remove; Runtime resolves unique names",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldRoomID: "401398", semantic.FieldAddAreaIDs: []any{"300001"}, semantic.FieldRemoveAreaIDs: []any{"300002"}},
			map[string]any{semantic.FieldRoomName: "主卧", semantic.FieldAddAreaNames: []any{"休息区"}},
		},
		semantic.FieldNextStep: "Provide at least one add/remove area id or unique area name. Prefer natural names for user-facing requests; Runtime resolves and validates the current home before writing.",
	}
}

func areaUpdatePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldAreaID:      "area id; omit when areaName/currentName uniquely identifies the area",
			semantic.FieldAreaName:    "current area name accepted when it uniquely identifies the area",
			semantic.FieldName:        "optional new name",
			semantic.FieldDescription: "optional description",
			semantic.FieldIcon:        "optional icon",
			semantic.FieldParentID:    "optional parent area id, cannot be itself",
			semantic.FieldRoomIDs:     "optional complete associated room id list",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldAreaID: "300001", semantic.FieldName: "公共区", semantic.FieldRoomIDs: []any{"401398", "401399"}},
			map[string]any{semantic.FieldAreaName: "南区", semantic.FieldName: "公共区"},
		},
		semantic.FieldNextStep: "Use area.detail.get or entity.list first when replacing roomIds; roomIds is a complete association list, not an add/remove patch. For rename-only requests, areaName/currentName can avoid a separate id lookup.",
	}
}

func deviceMoveRoomBatchPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldItems: []any{
				map[string]any{
					semantic.FieldDeviceID:       "device id when known; deviceName is accepted when the user names the device",
					semantic.FieldDeviceName:     "natural device name accepted when id is unknown",
					semantic.FieldRoomID:         "target room id when known",
					semantic.FieldTargetRoomName: "natural target room name accepted when id is unknown",
				},
			},
			semantic.FieldItemsAsMap:     "accepted alternative object: {\"deviceIdOrName\":\"roomIdOrName\"}",
			semantic.FieldDeviceNames:    "accepted shortcut when multiple named devices move to the same target room",
			semantic.FieldTargetRoomName: "accepted shortcut target room when deviceNames is used",
			semantic.FieldRoomID:         "accepted shortcut target room id when deviceNames is used",
		},
		semantic.FieldExamples: []any{
			map[string]any{semantic.FieldItems: []any{map[string]any{semantic.FieldDeviceID: "50018330", semantic.FieldRoomID: "401398"}}},
			map[string]any{semantic.FieldItems: []any{map[string]any{semantic.FieldDeviceName: "主灯", semantic.FieldTargetRoomName: "客厅"}}},
			map[string]any{semantic.FieldDeviceNames: []string{"主灯", "筒灯"}, semantic.FieldTargetRoomName: "客厅"},
			map[string]any{semantic.FieldItems: map[string]any{"50018330": "401398", "50018430": "401398"}},
		},
		semantic.FieldNextStep: "Send 1..20 explicit device-to-room moves. When the user moves several devices to one room, use deviceNames plus targetRoomName. Runtime resolves and verifies every device and target room belong to the current home.",
	}
}

func entityRenameBatchPayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldItems: []any{
				map[string]any{
					semantic.FieldEntityType:  "required: device or scene",
					semantic.FieldID:          "target id when known; currentName is accepted when the user names the target",
					semantic.FieldCurrentName: "optional current name for unique target resolution",
					semantic.FieldNewName:     "required new display name",
				},
			},
		},
		semantic.FieldExamples: []any{map[string]any{semantic.FieldItems: []any{
			map[string]any{semantic.FieldEntityType: "device", semantic.FieldID: "50018330", semantic.FieldNewName: "阅读主灯"},
			map[string]any{semantic.FieldEntityType: "scene", semantic.FieldCurrentName: "已有情景", semantic.FieldNewName: "睡前晚安"},
		}}},
		semantic.FieldNextStep: "Use entity.rename.batch only for devices and scenes. Use room.rename, group.update, or area.update for other entity types.",
	}
}

func gatewayConfigurePayloadGuide() map[string]any {
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldGatewayID:   "gateway device id when known",
			semantic.FieldGatewayName: "natural gateway name accepted when id is unknown",
			semantic.FieldDeviceName:  "accepted natural gateway device name",
			semantic.FieldName:        "optional gateway name",
			semantic.FieldDescription: "optional description",
			semantic.FieldIcon:        "optional icon",
			semantic.FieldMAC:         "optional mac value when cloud accepts it",
			semantic.FieldRoomIDs:     "optional associated room id list",
			semantic.FieldRoomNames:   "optional associated room names accepted when ids are unknown",
		},
		semantic.FieldExamples: []any{map[string]any{semantic.FieldGatewayName: "客厅网关", semantic.FieldName: "客厅主网关", semantic.FieldRoomNames: []any{"客厅"}}},
		semantic.FieldNextStep: "Call gateway.detail.get first when preserving current metadata. Use gatewayName/deviceName and roomNames for natural user wording; Runtime resolves and validates ids in the current home.",
	}
}

func metadataBatchDeletePayloadGuide(intent string) map[string]any {
	targetType, idField, _, ok := metadataBatchDeleteIntentSpec(intent)
	if !ok {
		idField = "roomId/areaId/groupId/sceneId/automationId"
	}
	itemShape := map[string]any{
		idField:                   "one id field matching the delete intent",
		semantic.FieldName:        "accepted when it uniquely resolves within the current home",
		semantic.FieldEntityName:  "accepted alias for name",
		semantic.FieldCurrentName: "accepted alias for name",
		semantic.FieldTargetName:  "accepted alias for name",
	}
	for _, nameField := range metadataDeleteNameFields(targetType) {
		if _, exists := itemShape[nameField]; !exists {
			itemShape[nameField] = "accepted type-specific target name"
		}
	}
	return map[string]any{
		semantic.FieldPayloadShape: map[string]any{
			semantic.FieldItems: []any{
				itemShape,
			},
			semantic.FieldIDs:         "accepted list of target ids",
			semantic.FieldNames:       "accepted list of unique target names",
			semantic.FieldEntityNames: "accepted alias for names",
		},
		semantic.FieldExamples: []any{map[string]any{semantic.FieldItems: []any{map[string]any{idField: "target-1"}, map[string]any{semantic.FieldName: "睡前晚安"}}}},
		semantic.FieldNextStep: "After caller-side confirmation, send 1..20 explicit targets with confirmed=true; Runtime resolves and verifies each target.",
		semantic.FieldIntent:   intent,
	}
}
