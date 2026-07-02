package main

func moduleActionAdvancedHelp(intent string) string {
	switch intent {
	case "scene.create", "scene.update":
		return `
Advanced payload shape:
  scene.create creates a saved action bundle with a complete actions[] list.
  scene.update is a full scene replacement, not a patch. For updates, call
  scene.detail.get first, use editablePayload when present, keep every existing
  actions[] row, edit only the intended set field, then send the
  complete updated actions[] list.

  Required fields:
    create: name, actions
    update: sceneId or unique sceneName/currentName, plus complete actions[]
            name/newName is optional and preserves the current scene name when omitted

  actions[] item fields:
    targetType=device|group|meshGroup|scene|room|home when supported
    targetId    optional target resource id for existing resources
    targetName  target display name; Runtime resolves unique names within the selected home
    action      optional action mode; Runtime defaults to 0 when omitted
    rank        order integer; Runtime defaults to 0 when omitted
    subIndex    optional sub-device index
    roomId      optional, preserve if returned by scene.detail.get
    set         required light/action set object

  Light action set object:
    {"power":true,"brightness":60,"colorTemperature":3000,"color":16755200}
    power is bool, brightness is 1..100, colorTemperature is usually 2700..6500,
    color is RGB integer 0..16777215. Optional delay, duration, and delayoff are
    non-negative milliseconds at the action row level. toggle and adjust are capability-dependent.
    adjust values are relative strings such as {"brightness":"+10/100"} or
    {"colorTemperature":"-1/5"}. flow is a source-backed dynamic sequence with tuples and
    ending; preserve tuple type, set/pause values, and duration from evidence.
    action may carry product-specific blink, motorAdjust, delayCancel,
    musicPlayerCtrl, or localAudioCtrl objects only when Runtime detail/capability
    evidence returned them.
`
	case "automation.create", "automation.update":
		return `
Advanced payload shape:
  automation.create creates a complete trigger-action rule. automation.update is
  a complete rule replacement, not a patch. For updates, call automation.detail.get
  first, use editablePayload when present, keep the complete condition/action payload,
  edit only the intended field, then send automation.update. Use automation.enable
  or automation.disable for status changes.

  Required fields:
    create: name, activeWindow, repeat, trigger or conditions, actions
    update: automationId or unique automationName/currentName, plus complete trigger/conditions and actions
            name/newName is optional and preserves the current automation name when omitted
  Time and repeat:
    activeWindow.start/end use HH:mm:ss and default to all day when omitted.
    repeat uses daily, weekdays, weekend, once, custom, legal_holiday, or legal_workday.
  trigger object:
    {"conditionKind":"alarm","time":"09:00:00"}
    Conditions may also be sent as conditions[]. Condition kind may be alarm,
    event, fact, fact_change, or a nested and/or group. event/fact rows may
    include targetType, targetId, property, operation, value, extra, and
    actionItem, but those source-backed fields must come from Runtime
    supported-list/detail evidence, not natural-language guessing.
  actions[] item fields:
    targetType=device|group|meshGroup|scene, targetId, targetName, set
    rank is optional and subIndex is optional. Direct automation writes validate
    device, meshGroup, and scene targets.
    set uses the same light action set object as scene actions.
`
	case "lighting.design.apply":
		return `
Advanced payload shape:
  lighting.design.apply changes real device state only. It does not create rooms,
  slots, groups, scenes, or automations. Use lighting.design.import for topology.

  actions[] item fields:
    deviceId      required when more than one device is in scope
    property      one of power, brightness, colorTemperature, color
    value         power=bool, brightness=1..100, colorTemperature=2700..6500, color=0..16777215 or rrggbb

  Direct fields:
    power, brightness, colorTemperature, and color. The caller or Skill
    must translate subjective mood words into these objective values first;
    Runtime does not choose a recipe from mood, scene name, or design prose.
`
	case "lighting.design.import", "device.slot.create":
		return `
Advanced payload shape:
  lighting.design.import accepts the standard lighting design model. Use it for a new
  home, an empty or lightly configured home, or an explicit full metadata import.
  For a busy existing home, prefer dedicated room/group/scene/automation update
  intents unless the caller explicitly wants a full import/replace.

  The caller must select products, expand quantities into explicit deviceSlots,
  decide same-type groups, and author scene/automation recipes before calling.
  Runtime validates references, fills small deterministic defaults, submits the
  model, and verifies the result.

  Top-level model:
    houseId                              existing home only; omit for new-home import
    key/name                             optional stable design key and required home/design name
    gatewayName/gatewayDeviceId          optional gateway metadata
    rooms[]                              required room list
    areas[]                              optional imported areas
    scenes[]                             optional imported scenes
    automations[]                        optional imported automations

  rooms[]:
    key/name/icon                        stable room key, display name, optional icon
    deviceSlots[]                        explicit design slots; each needs key, name, product
    deviceSlots[].product                selected product fields: skuCode, capabilityPid, productComponentId; skuCode is the exact selected SKU number, capabilityPid is the capability/firmware identity, not the concrete SKU
    groups[]                             explicit groups: key, name, groupCategory/groupCapability, slotKeys[]

  scenes[].actions[] and automations[].actions[]:
    targetType                           room, device, group, home, or scene when supported
    targetKey                            key of a room/device slot/group/scene in the same request
    targetName                           optional display name
    rank/action/set                      order, optional scene action mode, light/action set

  automations[]:
    activeWindow                         optional start/end object
    repeat                               daily, weekdays, weekend, once, custom, legal_holiday, or legal_workday
    trigger                              object such as {"conditionKind":"alarm","time":"09:00:00"}

  Do not send autoGroup, quantity-only slots, clearAll, overwrite, or targetId-based
  design-import actions. New-home import omits --house-id; existing-home import
  must be explicit.
`
	case "panel.button.configure":
		return `
Advanced payload shape:
  buttons[] contains current panel button rows to change. Provide id, index,
  keyValue, name, or alias to locate the existing button. Runtime reads current
  panel button detail and merges submitted fields before writing.

  buttons[] item fields:
    id, alias, keyValue, index, targetType, targetId, visible, icon, sort, type, extend
`
	case "panel.button_event.update", "panel.button_event.batch_update":
		return `
Advanced payload shape:
  Button event updates replace the target event's complete action list. For
  batch updates, send buttonEvents[] where each row has buttonEventId and actions[] rows.

  Event fields:
    deviceId, buttonEventId, alias, actions
  action row fields:
    roomId, targetType, targetId, targetName, subIndex, set, rank,
    activeWindow, repeat
  set light example:
    {"power":true,"brightness":60,"colorTemperature":3000}
  Product-specific panel event action objects such as {"action":{"motorAdjust":{...}}},
  musicPlayerCtrl, localAudioCtrl, channel-prefixed switch keys, or curtain keys
  must come from panel/detail/capability evidence.
`
	case "knob.configure":
		return `
Advanced payload shape:
  actions contains one row per knob key/index to configure. Prefer knob.get
  first, preserve the row, and edit only the intended binding fields.

  action row fields:
    index, configType, targetType, targetId, targetName, subIndex, model,
    set, sensitivity, mode, custom
  set is a light action set when applicable. Product-specific eventCode maps
  must come from returned detail evidence. Known evidence words
  include rotate, press_rotate, click, double_click, and hold, but event codes
  and parameter maps are product-specific; preserve unknown existing entries.
  Product-specific nested objects should be carried in custom only when Runtime
  detail returns them.
`
	case "operation.batch.configure":
		return `
Advanced payload shape:
  operations[] is an ordered list of reversible add/update/configure steps.
  Each step has intent, parameters, and optional targets. Nested parameters use
  the same shape as calling that intent directly.

  Excluded from batch:
    delete, unbind, member remove/transfer/quit, home create/delete, lock/unlock,
    panel reset, knob reset, and clear-all overwrite design imports.
`
	case "home.sort.configure":
		return `
Advanced payload shape:
  items[] defines explicit ordered resources. Read home.sort.list first when
  preserving existing unspecified items.

  items[] item fields:
    targetType, targetId, rank, optional subIndex
`
	case "favorite.add", "favorite.update", "favorite.delete", "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		return `
Advanced payload shape:
  Single favorite intents use one resource identity. Batch favorite intents use
  items[] with one to twenty resource rows. Delete may use favoriteId, or a
  unique resource identity that Runtime resolves from favorite.list. Resource
  identity can use a Runtime-returned id or a unique targetName/entityName with
  explicit targetType.

  item fields:
    favoriteId, targetType, targetId, targetName, entityName, rank, valid
`
	case "room.batch_create", "room.batch_update":
		return `
Advanced payload shape:
  rooms[] or items[] contains one to twenty room rows. Create rows require name.
  Update rows require roomId and may include metadata fields.

  room fields:
    roomId, name, description, icon, image, gatewayDeviceId, gatewayIds,
    defaultGatewayIds, sequence, capability
`
	case "room.batch_delete", "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		return `
Advanced payload shape:
  Batch delete intents remove one to twenty explicit targets after caller-side
  confirmation. Runtime resolves every target and verifies it belongs to the
  selected home before writing.

  Accepted target forms:
    items[] rows with the id field matching the intent, such as roomId, areaId,
      groupId, sceneId, or automationId
    ids[] list of target ids
    names[] list of unique target names
`
	case "room.area.configure":
		return `
Advanced payload shape:
  room.area.configure mutates area membership for one room.

  fields:
    roomId or unique roomName/currentName, addAreaIds[] or addAreaNames[],
    removeAreaIds[] or removeAreaNames[]
`
	case "area.update":
		return `
Advanced payload shape:
  area.update changes area metadata. roomIds is a complete association list, not
  an add/remove patch; use room.area.configure for incremental membership changes.

  fields:
    areaId or unique areaName/currentName, name, description, icon, parentId,
    roomIds[]
`
	case "device.move_room.batch":
		return `
Advanced payload shape:
  items may be an array of {deviceId, roomId} objects or an object map
  {"deviceId":"roomId"}. Runtime caps the batch at 20 moves.
`
	case "entity.rename.batch":
		return `
Advanced payload shape:
  entity.rename.batch currently supports devices and scenes only. Each row needs
  entityType=device|scene and newName; id is preferred, currentName may be used
  when it resolves uniquely.
`
	case "gateway.configure":
		return `
Advanced payload shape:
  gateway.configure changes gateway metadata and associated rooms. Call
  gateway.detail.get first when preserving current fields.

  fields:
    gatewayId or gatewayName/deviceName, name, description, icon, mac,
    roomIds[] or roomNames[]
`
	default:
		return ""
	}
}
