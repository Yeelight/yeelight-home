package main

func moduleActionAdvancedHelp(intent string) string {
	switch intent {
	case "scene.create", "scene.update":
		return `
Advanced payload shape:
  scene.create creates a saved action bundle with a complete details[] list.
  scene.update is a full scene replacement, not a patch. For updates, call
  scene.detail.get first, use editablePayload when present, keep every existing
  details[] row, edit only the intended params.set field, then send the
  complete updated details[] list.

  Required fields:
    create: name, details
    update: sceneId plus create fields

  details[] item fields:
    typeId   required. Direct scene writes validate 2=device,
             3=Runtime group/custom scope, 4=mesh group, 6=scene.
             Source-backed design metadata may also map 1=room or 5=house.
    resId    required target resource id
    resName  required target display name; Runtime may backfill from entities
    action   cloud action type; Runtime defaults to 0 when omitted
    rank     order integer; Runtime defaults to 0 when omitted
    idx      optional sub-device index
    roomId   optional, preserve if returned by scene.detail.get
    params   required object or JSON string; Runtime compacts it for cloud write

  Light action params object:
    {"set":{"p":true,"l":60,"ct":3000,"c":16755200},"delay":0,"duration":500}
    p=power bool, l=brightness 1..100, ct=color temperature 2700..6500,
    c=RGB integer 0..16777215. delay, duration, and delayoff are
    non-negative milliseconds. toggle and adjust are capability-dependent.
    adjust values are relative strings such as {"l":"+10/100"} or
    {"ct":"-1/5"}. flow is a source-backed dynamic sequence with tuples and
    ending; preserve tuple type, set/pause values, and duration from evidence.
    action may carry product-specific blink, motorAdjust, delayCancel,
    musicPlayerCtrl, or localAudioCtrl objects. Preserve curtain keys such as
    tp/tra, HVAC keys such as 1-acp, switch keys such as 0-sp/1-sp, and
    multi-channel light keys such as 1-p/2-p only when Runtime detail/capability
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
    create: name, startTime, endTime, repeatType, params, actions
    update: automationId plus create fields
  Time and repeat:
    startTime/endTime use HH:mm:ss. repeatType: 1 once, 2 daily, 3 weekdays,
    4 custom, 5 weekend, 6 legal holidays, 7 legal workdays.
    repeatValue is optional; custom repeats use hex values such as 0x7f.
  params condition object:
    {"type":"and","conditions":[{"type":"alarm","clock":"09:00:00"}]}
    Top-level type is and for direct writes. Condition type may be alarm,
    event, fact, fact_change, or a nested and/or group. event/fact rows may
    include id, pid, typeId, resId, prop, operation, value, extArgs, and
    actionItem, but those source-backed fields must come from automation
    supported-list/detail evidence, not natural-language guessing.
  actions[] item fields:
    typeId, resId, resName, rank, params are required by cloud; idx is optional.
    Direct automation writes validate 2=device, 4=mesh group, 6=scene.
    Action params use the same light action params object as scene details.
`
	case "lighting.design.apply":
		return `
Advanced payload shape:
  lighting.design.apply changes real device state only. It does not create rooms,
  slots, groups, scenes, or automations. Use lighting.design.import for topology.

  actions[] item fields:
    deviceId      required when more than one device is in scope
    propertyName  one of p, l, ct, c
    value         p=bool, l=1..100, ct=2700..6500, c=0..16777215 or rrggbb

  Direct fields:
    power/p/on, brightness/l/level, colorTemperature/ct/color_temperature,
    and color/hex are accepted as deterministic aliases. The caller or Skill
    must translate subjective mood words into these objective values first;
    Runtime does not choose a recipe from mood, scene name, or design prose.
`
	case "lighting.design.import", "device.slot.create":
		return `
Advanced payload shape:
  lighting.design.import materializes design metadata: rooms, future-device slots,
  caller-authored groups, scene metadata, and automation metadata. Design slots
  are not paired online devices. Runtime validates and adapts the payload; it
  does not decide whether same-type slots should be grouped.

  Natural topology:
    rooms[].name                          required room name
    rooms[].items[]                       slot families; aliases slots/devices
    rooms[].items[].name                  required slot display name
    rooms[].items[].quantity              optional, default 1
    rooms[].items[].category/color/installStyle/beamAngle/series
    rooms[].items[].materialCode/pid/pcId/productName/productSku/productSpu/modelNo
    rooms[].items[].connectType/namePattern/groupKey/notes
    groups[]                              optional caller-authored alias for named groups
    groups[].name                         desired group name
    groups[].roomName                     imported room name to match
    groups[].match                        category/name/series/productName/materialCode/groupKey

  scenes[] rows:
    name, optional localId, optional details[] using scene.create details item
    shape. If executable targets are not known, use params as design metadata
    and do not claim the scene can run.
  automations[] rows:
    name, startTime, endTime, repeatType, repeatValue, optional localId,
    params using automation.create condition shape, actions[] using automation
    action item shape. If targets are future slots, treat as design metadata.

  Normalized topology alternative:
    gateways[]: localId, localName, optional id, pid, pcId, mac, role
    rooms[]: localId, localName, gatewayIds[], optional id, areaId, cloudAreaId
    devices[]: localId, localName, gatewayDeviceId or cloudGatewayDeviceId,
      roomId or cloudRoomId, addr, pid, optional pcId, mac, connectType, attrs
    deviceGroups[]: localId, localName, optional id, roomId, deviceIds[]
    source-backed operation words: add, modify, remove, bind, unbind. Use them
      only when Runtime payloadShape accepts them; design slots are metadata,
      not paired online devices.
`
	case "panel.button.configure":
		return `
Advanced payload shape:
  buttons[] contains current panel button rows to change. Provide id, index,
  keyValue, name, or alias to locate the existing button. Runtime reads current
  panel button detail and merges submitted fields before writing.

  buttons[] item fields:
    id, alias, keyValue, index, resId, resType, visible, icon, sort, type, extend
`
	case "panel.button_event.update", "panel.button_event.batch_update":
		return `
Advanced payload shape:
  Button event updates replace the target event's complete details[] list. For
  batch updates, send buttonEvents[] where each row has buttonEventId and details[].

  Event fields:
    deviceId, buttonEventId, alias, details[]
  details[] item fields:
    roomId, resId, typeId, idx, params, rank, resName, startTime, endTime,
    repeatType, repeatValue
  params light example:
    {"set":{"p":true,"l":60,"ct":3000},"delay":0}
  Product-specific panel event params such as {"action":{"motorAdjust":{...}}},
  musicPlayerCtrl, localAudioCtrl, channel-prefixed switch keys, or curtain keys
  must come from panel/detail/capability evidence.
`
	case "knob.configure":
		return `
Advanced payload shape:
  details[] contains one row per knob key/index to configure. Prefer knob.get
  first, preserve the row, and edit only the intended binding fields.

  details[] item fields:
    index, configType, resId, typeId, resIndex, resName, model, param, sens,
    mode, details
  param/actionParamMap is an eventCode-to-parameter map. Known evidence words
  include rotate, press_rotate, click, double_click, and hold, but event codes
  and parameter maps are product-specific; preserve unknown existing entries.
  Light-style details may include lightDetail fields; curtain-style details may
  include curtainDetail res1/res2 and propertyName when Runtime detail returns it.
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
    entityType or typeId, resId/entityId/deviceId/sceneId/groupId/roomId, rank,
    optional subIndex
`
	case "favorite.add", "favorite.update", "favorite.delete", "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		return `
Advanced payload shape:
  Single favorite intents use one resource identity. Batch favorite intents use
  items[] with one to twenty resource rows. Delete may use favoriteId, or a
  unique resource identity that Runtime resolves from favorite.list.

  item fields:
    favoriteId, entityType or typeId, resId/entityId/deviceId/sceneId/groupId/roomId,
    rank, valid
`
	case "room.batch_create", "room.batch_update":
		return `
Advanced payload shape:
  rooms[] or items[] contains one to twenty room rows. Create rows require name.
  Update rows require roomId and may include metadata fields.

  room fields:
    roomId, name, desc, icon, img, gatewayDeviceId, gatewayIds,
    defaultGatewayIds, seq, capability
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
    roomId, addAreaList[], removeAreaList[]
`
	case "area.update":
		return `
Advanced payload shape:
  area.update changes area metadata. roomIds is a complete association list, not
  an add/remove patch; use room.area.configure for incremental membership changes.

  fields:
    areaId, name, desc, icon, parentId, roomIds[]
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
    gatewayId, name, desc, icon, mac, roomIds[]
`
	default:
		return ""
	}
}
