package main

import (
	"fmt"
	"strings"
)

func moduleActionHelpText(topic string) (string, bool) {
	parts := strings.Fields(topic)
	if len(parts) != 2 {
		return "", false
	}
	resource := parts[0]
	action := parts[1]
	if resource == "home" && action == "select" {
		if text, ok := commandHelpText["home select"]; ok {
			return text, true
		}
	}
	spec, ok := moduleCommands[resource][action]
	if !ok {
		return "", false
	}
	return fmt.Sprintf(`Usage:
  yeelight-home %s %s [--json] [--profile <name>] [--region <region>] [--house-id <id>] [resource flags]

Intent:
  %s

Description:
  %s

Execution model:
  This shortcut builds the same Runtime request as invoke --stdin.
  Reads and supported writes execute directly. Use --dry-run or --preview-only when a caller wants a no-write preview before asking for its own confirmation.

Common flags:
  --json                 Print the full Runtime JSON response.
  --profile <name>       Override active profile.
  --region <region>      Override profile region.
  --house-id <id>        Override selected home for house-scoped commands.

Resource flags:
%s
  --params-json <json>   Pass advanced intent parameters as a JSON object.
  --set key=value        Add advanced parameters, comma-separated.
%s

Examples:
%s
`, resource, action, spec.Intent, spec.Utterance, moduleActionFlagHelp(resource, action, spec), moduleActionAdvancedHelp(spec.Intent), moduleActionExamples(resource, action, spec)), true
}

func moduleActionFlagHelp(resource string, action string, spec moduleCommandSpec) string {
	lines := []string{}
	if len(spec.TargetIDKeys) > 0 {
		lines = append(lines, fmt.Sprintf("  --%s <id>", spec.TargetIDKeys[0]))
		for _, alias := range spec.TargetIDKeys[1:] {
			lines = append(lines, fmt.Sprintf("    alias: --%s", alias))
		}
	}
	if spec.TargetName {
		lines = append(lines, "  --name <name>")
	}
	switch spec.Intent {
	case "light.brightness.set":
		lines = append(lines, "  --brightness <1-100>")
	case "light.color_temperature.set":
		lines = append(lines, "  --color-temperature <2700-6500>")
	case "light.color.set":
		lines = append(lines, "  --hex <rrggbb>")
	case "state.query":
		lines = append(lines, "  --property <name>")
	case "room.create", "area.create", "group.create", "home.create":
		lines = append(lines, "  --name <name>")
	case "scene.create":
		lines = append(lines, "  --name <name>")
		lines = append(lines, "  --params-json <json>   Requires complete actions[]. Use action rows with targetType, targetId, targetName, and set.")
	case "automation.create":
		lines = append(lines, "  --name <name>")
		lines = append(lines, "  --params-json <json>   Requires activeWindow, repeat, trigger or conditions[], and actions[].")
	case "device.slot.create":
		lines = append(lines, "  --name <slot-name>")
		lines = append(lines, "  --room-id <id>")
		lines = append(lines, "  --room-name <name>")
		lines = append(lines, "  --params-json <json>   Requires standard lighting design model rooms[].deviceSlots[]; caller expands quantities and selects products first.")
	case "lighting.design.import":
		lines = append(lines, "  --params-json <json>   Requires standard lighting design model: rooms, deviceSlots, groups, scenes, automations. Omit --house-id for new-home import; pass --house-id only for explicit existing-home import.")
	case "scene.update":
		lines = append(lines, "  --params-json <json>   Requires sceneId, name, and complete actions[]. Use scene detail first for editablePayload.")
	case "automation.update":
		lines = append(lines, "  --params-json <json>   Use automationId or unique automationName/currentName, plus activeWindow, repeat, trigger or conditions[], and complete actions[].")
	case "lighting.design.apply":
		lines = append(lines, "  --params-json <json>   Supports actions[] with deviceId, property, value. Properties: power, brightness, colorTemperature, color.")
	case "operation.batch.configure":
		lines = append(lines, "  --params-json <json>   Requires operations[].intent and operations[].parameters. Only allowlisted add/update/configure steps are accepted; delete/unbind/member/reset actions stay separate.")
	case "panel.button.configure":
		lines = append(lines, "  --params-json <json>   Requires deviceId or deviceName, plus buttons[]. Runtime merges each button row with current panel button detail before writing.")
	case "panel.button_event.update":
		lines = append(lines, "  --params-json <json>   Requires deviceId or deviceName, buttonEventId, and a complete actions[] list.")
	case "panel.button_event.batch_update":
		lines = append(lines, "  --params-json <json>   Requires deviceId or deviceName and buttonEvents[]. Each event requires buttonEventId and a complete actions[] list.")
	case "knob.configure":
		lines = append(lines, "  --params-json <json>   Requires deviceId or deviceName and actions[] rows with index plus knob binding fields such as configType, targetType, targetId, set, sensitivity.")
	case "home.sort.configure":
		lines = append(lines, "  --params-json <json>   Requires type, target, and items[] with targetType, targetId, rank.")
	case "favorite.add", "favorite.update", "favorite.delete", "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		lines = append(lines, "  --params-json <json>   Single favorite uses targetType plus targetId or unique targetName; batch uses items[] with favorite/resource identity, rank, and valid when applicable.")
	case "room.batch_create", "room.batch_update":
		lines = append(lines, "  --params-json <json>   Requires rooms[] or items[]; update rows require roomId.")
	case "room.batch_delete", "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		lines = append(lines, "  --params-json <json>   Requires items[], ids[], or names[] matching the delete target type.")
	case "room.area.configure":
		lines = append(lines, "  --params-json <json>   Use roomId or unique roomName/currentName, plus addAreaIds/addAreaNames or removeAreaIds/removeAreaNames.")
	case "area.update":
		lines = append(lines, "  --params-json <json>   Use areaId or unique areaName/currentName, plus mutable fields such as name, parentId, or complete roomIds[].")
	case "device.move_room.batch":
		lines = append(lines, "  --params-json <json>   Requires items[] with deviceId and roomId, or an object map of deviceId to roomId.")
	case "entity.rename.batch":
		lines = append(lines, "  --params-json <json>   Requires items[] with entityType=device|scene and newName.")
	case "gateway.configure":
		lines = append(lines, "  --params-json <json>   Requires gatewayId plus one or more mutable metadata fields such as name or roomIds.")
	case "device.move":
		lines = append(lines, "  --room-id <id>")
	case "room.search", "group.search", "scene.search", "geo_area.search":
		lines = append(lines, "  --name <keyword>")
	}
	if len(lines) == 0 {
		return "  (none required for the basic form)\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

func moduleActionExamples(resource string, action string, spec moduleCommandSpec) string {
	switch spec.Intent {
	case "light.power.set":
		return fmt.Sprintf("  yeelight-home %s %s --device-id <id> --json\n", resource, action)
	case "light.brightness.set":
		return "  yeelight-home light brightness --device-id <id> --brightness 60 --json\n"
	case "light.color_temperature.set":
		return "  yeelight-home light color-temperature --device-id <id> --color-temperature 4000 --json\n"
	case "scene.execute":
		return "  yeelight-home scene execute --scene-id <id> --json\n"
	case "automation.enable":
		return "  yeelight-home automation enable --automation-id <id> --json\n"
	case "device.detail.get":
		return "  yeelight-home device detail --device-id <id> --json\n"
	case "room.detail.get":
		return "  yeelight-home room detail --room-id <id> --json\n"
	case "group.detail.get":
		return "  yeelight-home group detail --group-id <id> --json\n"
	case "gateway.detail.get":
		return "  yeelight-home gateway detail --gateway-id <id> --json\n"
	case "scene.detail.get":
		return "  yeelight-home scene detail --scene-id <id> --json\n"
	case "automation.detail.get":
		return "  yeelight-home automation detail --automation-id <id> --json\n"
	case "scene.update":
		return "  yeelight-home scene update --params-json '{\"sceneName\":\"孩子屋开灯\",\"newName\":\"孩子屋开灯\",\"actions\":[{\"targetType\":\"device\",\"targetName\":\"孩子屋吸顶灯\",\"action\":0,\"rank\":0,\"set\":{\"power\":true,\"brightness\":60,\"colorTemperature\":3000}}]}' --json\n"
	case "scene.create":
		return "  yeelight-home scene create --house-id <id> --name 孩子屋开灯 --params-json '{\"actions\":[{\"targetType\":\"device\",\"targetId\":\"50018330\",\"targetName\":\"孩子屋吸顶灯\",\"action\":0,\"rank\":0,\"set\":{\"power\":true,\"brightness\":60,\"colorTemperature\":3000}}]}' --json\n"
	case "automation.update":
		return "  yeelight-home automation update --params-json '{\"automationName\":\"主卧每天9点开灯\",\"newName\":\"主卧每天9点开灯\",\"activeWindow\":{\"start\":\"00:00:00\",\"end\":\"23:59:59\"},\"repeat\":\"daily\",\"trigger\":{\"conditionKind\":\"alarm\",\"time\":\"09:00:00\"},\"actions\":[{\"targetType\":\"device\",\"targetName\":\"主卧吸顶灯\",\"rank\":0,\"set\":{\"power\":true,\"brightness\":60,\"colorTemperature\":3000}}]}' --json\n"
	case "automation.create":
		return "  yeelight-home automation create --house-id <id> --name 主卧每天9点开灯 --params-json '{\"activeWindow\":{\"start\":\"00:00:00\",\"end\":\"23:59:59\"},\"repeat\":\"daily\",\"trigger\":{\"conditionKind\":\"alarm\",\"time\":\"09:00:00\"},\"actions\":[{\"targetType\":\"device\",\"targetId\":\"50018330\",\"targetName\":\"主卧吸顶灯\",\"rank\":0,\"set\":{\"power\":true,\"brightness\":60,\"colorTemperature\":3000}}]}' --json\n"
	case "room.rename":
		return "  yeelight-home room rename --room-id <id> --name <new-name> --json\n"
	case "device.rename":
		return "  yeelight-home device rename --device-id <id> --name <new-name> --json\n"
	case "device.slot.create":
		return "  yeelight-home device slot-create --house-id <id> --params-json '{\"name\":\"灯位设计\",\"rooms\":[{\"key\":\"living\",\"name\":\"客厅\",\"deviceSlots\":[{\"key\":\"living-grille-1\",\"name\":\"黑色格栅灯1\",\"product\":{\"skuCode\":\"1-000002044\",\"capabilityPid\":198666,\"productComponentId\":4}},{\"key\":\"living-grille-2\",\"name\":\"黑色格栅灯2\",\"product\":{\"skuCode\":\"1-000002044\",\"capabilityPid\":198666,\"productComponentId\":4}}]}]}' --json\n"
	case "lighting.design.import":
		return "  yeelight-home lighting import --params-json '{\"name\":\"新家照明设计\",\"rooms\":[{\"key\":\"living\",\"name\":\"客厅\",\"deviceSlots\":[{\"key\":\"living-grille-1\",\"name\":\"黑色格栅灯1\",\"product\":{\"skuCode\":\"1-000002044\",\"capabilityPid\":198666,\"productComponentId\":4}},{\"key\":\"living-grille-2\",\"name\":\"黑色格栅灯2\",\"product\":{\"skuCode\":\"1-000002044\",\"capabilityPid\":198666,\"productComponentId\":4}}],\"groups\":[{\"key\":\"living-grilles\",\"name\":\"客厅格栅灯组\",\"groupCategory\":\"lighting\",\"groupCapability\":\"light\",\"slotKeys\":[\"living-grille-1\",\"living-grille-2\"]}]}],\"scenes\":[{\"key\":\"living-home\",\"name\":\"客厅回家模式\",\"actions\":[{\"targetType\":\"group\",\"targetKey\":\"living-grilles\",\"targetName\":\"客厅格栅灯组\",\"set\":{\"power\":true,\"brightness\":60,\"colorTemperature\":3000}}]}]}' --json\n  yeelight-home lighting import --house-id <existing-id> --params-json '{\"name\":\"已有家庭照明设计\",\"rooms\":[]}' --json\n"
	case "lighting.design.apply":
		return "  yeelight-home lighting apply --house-id <id> --params-json '{\"actions\":[{\"deviceId\":\"50018330\",\"property\":\"power\",\"value\":true},{\"deviceId\":\"50018330\",\"property\":\"colorTemperature\",\"value\":3000},{\"deviceId\":\"50018330\",\"property\":\"brightness\",\"value\":60}]}' --json\n"
	case "operation.batch.configure":
		return "  yeelight-home operation batch-configure --house-id <id> --params-json '{\"operations\":[{\"intent\":\"room.create\",\"parameters\":{\"name\":\"书房\"}},{\"intent\":\"device.rename\",\"parameters\":{\"deviceId\":\"50018330\",\"name\":\"书房主灯\"}}]}' --json\n"
	case "panel.button.configure":
		return "  yeelight-home panel button-configure --panel-id <id> --params-json '{\"buttons\":[{\"id\":\"btn-1\",\"alias\":\"回家\",\"targetType\":\"scene\",\"targetId\":\"700001\",\"visible\":1}]}' --json\n"
	case "panel.button_event.update":
		return "  yeelight-home panel button-event-update --panel-id <id> --params-json '{\"buttonEventId\":\"101\",\"alias\":\"单击\",\"actions\":[{\"targetType\":\"scene\",\"targetId\":\"700001\",\"targetName\":\"回家模式\",\"rank\":0}]}' --json\n"
	case "panel.button_event.batch_update":
		return "  yeelight-home panel button-events-update --panel-id <id> --params-json '{\"buttonEvents\":[{\"buttonEventId\":\"101\",\"actions\":[{\"targetType\":\"scene\",\"targetId\":\"700001\"}]},{\"buttonEventId\":\"102\",\"actions\":[{\"targetType\":\"device\",\"targetId\":\"50018330\",\"set\":{\"power\":true,\"colorTemperature\":3000}}]}]}' --json\n"
	case "knob.configure":
		return "  yeelight-home knob configure --knob-id <id> --params-json '{\"actions\":[{\"index\":1,\"configType\":\"scene\",\"targetType\":\"scene\",\"targetId\":\"700001\",\"targetName\":\"回家模式\"}]}' --json\n"
	case "home.sort.configure":
		return "  yeelight-home home sort-configure --house-id <id> --params-json '{\"type\":0,\"target\":\"<house-id>\",\"items\":[{\"targetType\":\"room\",\"targetId\":\"401398\",\"rank\":1},{\"targetType\":\"scene\",\"targetId\":\"700001\",\"rank\":2}]}' --json\n"
	case "favorite.batch_add":
		return "  yeelight-home favorite batch-add --house-id <id> --params-json '{\"items\":[{\"targetType\":\"device\",\"targetName\":\"主灯\",\"rank\":1},{\"targetType\":\"scene\",\"targetName\":\"回家模式\",\"rank\":2}]}' --json\n"
	case "favorite.add":
		return "  yeelight-home favorite add --house-id <id> --params-json '{\"targetType\":\"device\",\"targetName\":\"主灯\",\"rank\":1}' --json\n"
	case "favorite.update":
		return "  yeelight-home favorite update --favorite-id <id> --params-json '{\"favoriteId\":\"<id>\",\"rank\":2,\"valid\":true}' --json\n"
	case "favorite.delete":
		return "  yeelight-home favorite delete --favorite-id <id> --json\n"
	case "room.batch_create":
		return "  yeelight-home room batch-create --house-id <id> --params-json '{\"rooms\":[{\"name\":\"书房\"},{\"name\":\"茶室\"}]}' --json\n"
	case "room.batch_update":
		return "  yeelight-home room batch-update --house-id <id> --params-json '{\"rooms\":[{\"roomId\":\"401398\",\"name\":\"会客厅\"}]}' --json\n"
	case "room.batch_delete":
		return "  yeelight-home room batch-delete --house-id <id> --params-json '{\"items\":[{\"roomId\":\"401398\"},{\"name\":\"临时房间\"}]}' --json\n"
	case "group.create":
		return "  yeelight-home group create --house-id <id> --params-json '{\"name\":\"客厅格栅灯组\",\"roomName\":\"客厅\",\"groupCapability\":\"light\",\"deviceNames\":[\"左侧格栅灯\",\"右侧格栅灯\"]}' --json\n"
	case "area.update":
		return "  yeelight-home area update --area-id <id> --params-json '{\"name\":\"公共区\",\"roomIds\":[\"401398\",\"401399\"]}' --json\n"
	case "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		return fmt.Sprintf("  yeelight-home %s %s --house-id <id> --params-json '{\"items\":[{\"id\":\"<target-id>\"},{\"name\":\"<unique-name>\"}]}' --json\n", resource, action)
	case "device.move_room.batch":
		return "  yeelight-home device move-room-batch --house-id <id> --params-json '{\"items\":[{\"deviceId\":\"50018330\",\"roomId\":\"401398\"},{\"deviceId\":\"50018430\",\"roomId\":\"401398\"}]}' --json\n"
	case "entity.rename.batch":
		return "  yeelight-home entity rename-batch --house-id <id> --params-json '{\"items\":[{\"entityType\":\"device\",\"id\":\"50018330\",\"newName\":\"阅读主灯\"},{\"entityType\":\"scene\",\"currentName\":\"已有情景\",\"newName\":\"睡前晚安\"}]}' --json\n"
	default:
		return fmt.Sprintf("  yeelight-home %s %s --json\n", resource, action)
	}
}
