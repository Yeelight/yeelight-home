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
  Reads execute immediately. Risky writes/deletes return confirmation_required with a planId.

Common flags:
  --json                 Print the full Runtime JSON response.
  --profile <name>       Override active profile.
  --region <region>      Override profile region.
  --house-id <id>        Override selected home for house-scoped commands.

Resource flags:
%s
  --params-json <json>   Pass advanced intent parameters as a JSON object.
  --set key=value        Add advanced parameters, comma-separated.

Examples:
%s
`, resource, action, spec.Intent, spec.Utterance, moduleActionFlagHelp(resource, action, spec), moduleActionExamples(resource, action, spec)), true
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
		lines = append(lines, "  --ct <2700-6500>")
	case "light.color.set":
		lines = append(lines, "  --hex <rrggbb>")
	case "state.query":
		lines = append(lines, "  --property <name>")
	case "room.create", "area.create", "group.create", "scene.create", "automation.create", "home.create":
		lines = append(lines, "  --name <name>")
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
		return "  yeelight-home light ct --device-id <id> --ct 4000 --json\n"
	case "scene.execute":
		return "  yeelight-home scene execute --scene-id <id> --json\n"
	case "automation.enable":
		return "  yeelight-home automation enable --automation-id <id> --json\n"
	case "plan.commit":
		return "  yeelight-home plan commit --plan-id <id> --json\n"
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
	case "room.rename":
		return "  yeelight-home room rename --room-id <id> --name <new-name> --json\n"
	case "device.rename":
		return "  yeelight-home device rename --device-id <id> --name <new-name> --json\n"
	default:
		return fmt.Sprintf("  yeelight-home %s %s --json\n", resource, action)
	}
}
