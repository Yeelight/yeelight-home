package main

import (
	"fmt"
	"io"
	"strings"
)

const rootHelpTemplate = `Yeelight Home CLI

Usage:
  yeelight-home <command> [flags]
  yeelight-home help [command]

Commands:
  auth       Login, inspect auth status, and manage local tokens
  profile    List, show, activate, and delete profiles
  config     Read and update non-secret profile configuration
  completion Generate shell completion scripts
  intent     Explain Runtime intents and complex payload contracts
  explain    Print a machine-readable schema for one Runtime intent
  invoke     Execute one Skill Runtime request from stdin
  api        Run account-scoped API smoke checks
  doctor     Print local installation and auth diagnostics
  version    Print CLI version

Resource commands:
%s

Global flags:
  -h, --help       Show help
  -v, --version    Show CLI version

Command model:
  Human-friendly operations use: yeelight-home <resource> <action> [flags]
  Skill and automation integrations use: yeelight-home invoke --stdin
  Resource commands and invoke share the same Runtime adapters, validation, redaction, and write verification.
  Reads and semantic writes execute directly by default. Use --dry-run, --preview-only, or request options.dryRun=true when the caller wants a no-write preview before asking its own user confirmation.

Configuration precedence:
  command flags > environment variables > active profile metadata/credential store > defaults

Common examples:
  yeelight-home auth login --qr --region dev
  yeelight-home auth status
  yeelight-home home list --json --region dev
  yeelight-home home select --house-id <id> --region dev
  yeelight-home help device
  yeelight-home help device detail
  yeelight-home device list --json
  yeelight-home device detail --device-id <id> --json
  yeelight-home product search --keyword 青空灯 --json
  yeelight-home product search --product-model YP-0117 --json
  yeelight-home intent explain --intent scene.update --json
  yeelight-home intent schema --intent scene.update --json
  yeelight-home explain lighting.design.import --json
  yeelight-home scene execute --scene-id <id> --json
  yeelight-home light on --device-id <id> --json
  yeelight-home automation enable --automation-id <id> --json
  yeelight-home invoke --stdin
  yeelight-home doctor
  yeelight-home completion zsh
`

var moduleCommandDescriptions = map[string]string{
	"account":        "Inspect account-scoped profile and user information",
	"ai-voice":       "Inspect AI voice account and product support",
	"area":           "List, search, create, update, or delete areas",
	"automation":     "List, explain, create, update, enable, disable, or delete automations",
	"device":         "List, inspect, diagnose, move, rename, remove, unbind devices, or create design slots",
	"entity":         "List and inspect unified home entities",
	"favorite":       "List, plan, add, update, or delete home favorites",
	"gateway":        "List, inspect, diagnose, configure, or delete gateways",
	"group":          "List, search, create, update, or delete device groups",
	"home":           "List, select, inspect, sort, invite, update, or delete homes",
	"knob":           "Inspect, configure, or reset knobs",
	"light":          "Human-friendly light controls such as on, off, brightness, ct, color",
	"lighting":       "Plan, import, and apply lighting designs, slots, and experiences",
	"memory":         "Manage local preference memory",
	"meshgroup":      "Inspect mesh group details",
	"message":        "List home messages",
	"node":           "Inspect node sorting and property configuration",
	"operation":      "Run composite helpers and inspect learned operation lessons",
	"panel":          "List, inspect, and configure panels, screens, and buttons",
	"product":        "Search Yeelight product pedia records, manuals, FAQ candidates, and attachments",
	"progress":       "Inspect async operation progress",
	"recommendation": "Record, list, and provide feedback on local recommendations",
	"room":           "List, search, create, update, rename, move, or delete rooms",
	"scene":          "List, search, execute, create, update, test, or delete scenes",
	"schedule":       "List schedule jobs",
	"screen":         "Inspect screen control capabilities",
	"sensor":         "List sensors and sensor events",
	"thing":          "Inspect thing-model categories, domains, products, FAQ, and schema",
	"upgrade":        "Inspect upgrade files, OTA files, and progress",
}

var moduleCommandExamples = map[string][]string{
	"account":        {"yeelight-home account info --json"},
	"ai-voice":       {"yeelight-home ai-voice products --json", "yeelight-home ai-voice list --json"},
	"area":           {"yeelight-home area detail --area-id <id> --json", "yeelight-home area search --name <keyword> --json"},
	"automation":     {"yeelight-home automation list --json", "yeelight-home automation detail --automation-id <id> --json", "yeelight-home automation enable --automation-id <id> --json"},
	"device":         {"yeelight-home device list --json", "yeelight-home device detail --device-id <id> --json", "yeelight-home device slot-create --house-id <id> --params-json '{\"name\":\"灯位设计\",\"gateway\":{\"tempId\":\"gw1\",\"name\":\"默认网关\",\"roomList\":[{\"tempId\":\"rm1\",\"name\":\"客厅\",\"deviceList\":[{\"tempId\":\"dv1\",\"name\":\"黑色格栅灯1\",\"pid\":198666,\"materialCode\":\"1-000002044\"}]}]}}' --json"},
	"entity":         {"yeelight-home entity list --json", "yeelight-home entity get --entity-id <id> --json"},
	"favorite":       {"yeelight-home favorite list --json", "yeelight-home favorite add --set typeId=2,resId=<id>,rank=1 --json"},
	"gateway":        {"yeelight-home gateway list --json", "yeelight-home gateway detail --gateway-id <id> --json", "yeelight-home gateway diagnose --gateway-id <id> --json"},
	"group":          {"yeelight-home group list --json", "yeelight-home group detail --group-id <id> --json"},
	"home":           {"yeelight-home home list --json", "yeelight-home home summary --house-id <id> --json", "yeelight-home home sort --house-id <id> --json"},
	"knob":           {"yeelight-home knob detail --knob-id <id> --json", "yeelight-home knob configure --knob-id <id> --params-json '<json>' --json"},
	"light":          {"yeelight-home light on --device-id <id> --json", "yeelight-home light brightness --device-id <id> --brightness 60 --json", "yeelight-home light ct --device-id <id> --ct 4000 --json"},
	"lighting":       {"yeelight-home lighting plan --house-id <id> --params-json '<json>' --json", "yeelight-home lighting import --house-id <id> --params-json '{\"name\":\"全屋照明设计\",\"gateway\":{\"tempId\":\"gw1\",\"name\":\"默认网关\",\"roomList\":[{\"tempId\":\"rm1\",\"name\":\"客厅\",\"deviceList\":[{\"tempId\":\"dv1\",\"name\":\"黑色格栅灯1\",\"pid\":198666,\"materialCode\":\"1-000002044\"}],\"groupList\":[{\"tempId\":\"gp1\",\"name\":\"客厅格栅灯组\",\"componentId\":4,\"deviceTempIdList\":[\"dv1\"]}]}]}}' --json", "yeelight-home lighting apply --params-json '{\"actions\":[{\"deviceId\":\"<id>\",\"propertyName\":\"power\",\"value\":true}]}' --json"},
	"memory":         {"yeelight-home memory list --json", "yeelight-home memory remember --set key=value --json"},
	"meshgroup":      {"yeelight-home meshgroup detail --meshgroup-id <id> --json"},
	"message":        {"yeelight-home message list --json"},
	"node":           {"yeelight-home node sorted-devices --node-id <id> --json", "yeelight-home node property-config --node-id <id> --json"},
	"operation":      {"yeelight-home operation batch-configure --params-json '<json>' --json", "yeelight-home operation lesson-list --set intent=scene.update --json", "yeelight-home operation lesson-record --params-json '<json>' --json"},
	"panel":          {"yeelight-home panel list --json", "yeelight-home panel detail --panel-id <id> --json", "yeelight-home panel button-configure --panel-id <id> --params-json '<json>' --json"},
	"product":        {"yeelight-home product search --keyword 青空灯 --json", "yeelight-home product search --product-model YP-0117 --json", "yeelight-home product pedia --material-code 1-000003268 --json"},
	"progress":       {"yeelight-home progress get --progress-id <id> --json"},
	"recommendation": {"yeelight-home recommendation record --params-json '<json>' --json", "yeelight-home recommendation list --json", "yeelight-home recommendation feedback --params-json '<json>' --json"},
	"room":           {"yeelight-home room list --json", "yeelight-home room detail --room-id <id> --json", "yeelight-home room rename --room-id <id> --name <new-name> --json"},
	"scene":          {"yeelight-home scene list --json", "yeelight-home scene detail --scene-id <id> --json", "yeelight-home scene execute --scene-id <id> --json"},
	"schedule":       {"yeelight-home schedule jobs --json"},
	"screen":         {"yeelight-home screen controls --device-id <id> --json"},
	"sensor":         {"yeelight-home sensor list --json", "yeelight-home sensor events --sensor-id <id> --json"},
	"thing":          {"yeelight-home thing domains --json", "yeelight-home thing schema-get --schema-id <id> --json"},
	"upgrade":        {"yeelight-home upgrade files --params-json '<json>' --json", "yeelight-home upgrade progress --progress-id <id> --json"},
}

var commandHelpText = map[string]string{
	"api": `Usage:
  yeelight-home api smoke [--json] [--profile <name>] [--region <region>] [--house-id <id>]

Runs account and home-list smoke checks with the active local token.
`,
	"api smoke": `Usage:
  yeelight-home api smoke [--json] [--profile <name>] [--region <region>] [--house-id <id>]

Runs account, home-list, and optional house-context smoke checks with the active local token.
`,
	"auth": `Usage:
  yeelight-home auth status [--json] [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home auth login --qr [--json] [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home auth token set (--token <access-token>|--stdin) [--profile <name>] [--region <region>] [--house-id <id>] [--json]
  yeelight-home auth token delete [--profile <name>] [--json]

Tokens are stored in the system credential store when available. Profile files keep only non-secret metadata.
`,
	"auth login": `Usage:
  yeelight-home auth login --qr [--json] [--profile <name>] [--region <region>] [--house-id <id>] [--qr-png <path>]

Starts QR login for the selected region. houseId is optional profile context, not an authentication requirement.
`,
	"auth qr-check": `Usage:
  yeelight-home auth qr-check --qr-code-id <id> --json [--profile <name>] [--region <region>]

Checks a QR login request and saves credentials locally when the QR status reaches LOGIN.
`,
	"auth status": `Usage:
  yeelight-home auth status [--json] [--profile <name>] [--region <region>] [--house-id <id>]

Reports local credential presence and resolved profile context without printing token values.
`,
	"auth token": `Usage:
  yeelight-home auth token set (--token <access-token>|--stdin) [--profile <name>] [--region <region>] [--house-id <id>] [--json]
  yeelight-home auth token delete [--profile <name>] [--json]

Imports or deletes a token in the local credential store. Use --stdin to avoid putting secrets in shell history.
Tokens are never written to profile metadata.
`,
	"auth token delete": `Usage:
  yeelight-home auth token delete [--profile <name>] [--json]

Deletes the selected profile token from the local credential store and protected fallback.
`,
	"auth token set": `Usage:
  yeelight-home auth token set (--token <access-token>|--stdin) [--profile <name>] [--region <region>] [--house-id <id>] [--json]

Imports a token into local credential storage. Omit houseId for token-only account-scoped use.
For interactive use, prefer: printf '%s' "$TOKEN" | yeelight-home auth token set --stdin --profile dev --region dev
`,
	"config": `Usage:
  yeelight-home config get [--json] [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home config list [--json] [--profile <name>]
  yeelight-home config set [--profile <name>] [--region <region>] [--house-id <id>] [--qr-device <mac>] [--json]
  yeelight-home config unset [--profile <name>] [--region] [--house-id] [--qr-device] [--json]
`,
	"config get": `Usage:
  yeelight-home config get [--json] [--profile <name>] [--region <region>] [--house-id <id>]

Shows resolved non-secret configuration and credential presence using standard precedence.
`,
	"config list": `Usage:
  yeelight-home config list [--json] [--profile <name>]

Alias of config get for resolved local configuration.
`,
	"config set": `Usage:
  yeelight-home config set [--profile <name>] [--region <region>] [--house-id <id>] [--qr-device <mac>] [--json]

Updates non-secret profile metadata only. It never stores token values.
`,
	"config unset": `Usage:
  yeelight-home config unset [--profile <name>] [--region] [--house-id] [--qr-device] [--json]

Clears selected non-secret metadata fields from a profile.
`,
	"completion": `Usage:
  yeelight-home completion <bash|zsh|fish|powershell>

Prints a shell completion script to stdout.
`,
	"explain": `Usage:
  yeelight-home explain <intent> [--json]

Shortcut for yeelight-home intent schema --intent <intent>. Prints the machine-readable SkillRequest schema, nested payload shape, examples, and nextStep for one semantic intent.
`,
	"dev": `Usage:
  yeelight-home dev <seed-house|seed-room|seed-scene|seed-automation> --json --region dev --allow-write-dev ...

Development-only fixture commands. Writes require explicit dev region and --allow-write-dev.
`,
	"dev seed-automation": `Usage:
  yeelight-home dev seed-automation --json --region dev --house-id <id> --device-id <id> --name <name> --allow-write-dev
`,
	"dev seed-house": `Usage:
  yeelight-home dev seed-house --json --region dev --name <name> --allow-write-dev
`,
	"dev seed-room": `Usage:
  yeelight-home dev seed-room --json --region dev --house-id <id> --name <name> --allow-write-dev
`,
	"dev seed-scene": `Usage:
  yeelight-home dev seed-scene --json --region dev --house-id <id> --device-id <id> --name <name> --allow-write-dev
`,
	"doctor": `Usage:
  yeelight-home doctor [--json] [--online] [--profile <name>] [--region <region>] [--house-id <id>]

Prints local paths, active profile, token presence, region, houseId, and install diagnostics.
Pass --online to also check public GitHub, npm, and Homebrew latest versions.
`,
	"home list": `Usage:
  yeelight-home home list [--json] [--profile <name>] [--region <region>]

Lists homes visible to the selected account. It is account-scoped and does not require houseId.
`,
	"home select": `Usage:
  yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]

Stores a default home id for later house-scoped commands. It does not change the token.
`,
	"intent": `Usage:
  yeelight-home intent explain --intent <intent> [--json]
  yeelight-home intent schema --intent <intent> [--json]

Prints the local Runtime contract for one semantic intent. This is offline and does not require a token.
`,
	"intent explain": `Usage:
  yeelight-home intent explain --intent <intent> [--json]

Returns accepted parameter fields, nested payloadShape, examples, and nextStep when the intent accepts a complex JSON payload.
`,
	"intent schema": `Usage:
  yeelight-home intent schema --intent <intent> [--json]

Returns the machine-readable SkillRequest schema for one Runtime intent, including parameters, nested payload fields, examples, and nextStep. This is offline and does not require a token.
`,
	"area":       moduleHelpText("area"),
	"automation": moduleHelpText("automation"),
	"device":     moduleHelpText("device"),
	"favorite":   moduleHelpText("favorite"),
	"gateway":    moduleHelpText("gateway"),
	"group":      moduleHelpText("group"),
	"home": `Usage:
  yeelight-home home list [--json] [--profile <name>] [--region <region>]
  yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]
  yeelight-home home <summary|search|detail|stat|members|current-member|sort|sort-configure|create|update|delete|invite|accept-share|member-configure|member-remove|member-transfer|quit|lock-all|unlock-all> [flags]

home list is account-scoped and requires only a token. houseId is optional until a house-scoped command is used.
Use yeelight-home help home <action> for resource action flags.
`,
	"lighting":       moduleHelpText("lighting"),
	"light":          moduleHelpText("light"),
	"memory":         moduleHelpText("memory"),
	"panel":          moduleHelpText("panel"),
	"product":        moduleHelpText("product"),
	"recommendation": moduleHelpText("recommendation"),
	"room":           moduleHelpText("room"),
	"scene":          moduleHelpText("scene"),
	"thing":          moduleHelpText("thing"),
	"invoke": `Usage:
  yeelight-home invoke --stdin [--profile <name>] [--region <region>] [--house-id <id>]

Reads one Skill Runtime JSON request from stdin and returns one JSON response.
Flags override environment/profile defaults before request parameters are resolved.
`,
	"profile": `Usage:
  yeelight-home profile list [--json]
  yeelight-home profile show [--json] [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home profile use --profile <name> [--region <region>] [--house-id <id>] [--json]
  yeelight-home profile delete --profile <name> [--json]
`,
	"profile delete": `Usage:
  yeelight-home profile delete --profile <name> [--json]

Deletes profile metadata and local credentials for the selected profile.
`,
	"profile list": `Usage:
  yeelight-home profile list [--json]

Lists saved profiles, active profile marker, region, selected houseId, and token presence.
`,
	"profile show": `Usage:
  yeelight-home profile show [--json] [--profile <name>] [--region <region>] [--house-id <id>]

Shows the resolved profile context without exposing token values.
`,
	"profile use": `Usage:
  yeelight-home profile use --profile <name> [--region <region>] [--house-id <id>] [--json]

Persists the active profile and optional non-secret profile metadata.
`,
	"version": `Usage:
  yeelight-home version [--json]
  yeelight-home --version [--json]
`,
}

func moduleHelpText(resource string) string {
	actions := moduleCommandNames(resource)
	return fmt.Sprintf(`Usage:
  yeelight-home %s <%s> [--json] [--profile <name>] [--region <region>] [--house-id <id>] [resource flags]

Human-friendly shortcut commands for Runtime intents. They use the same direct adapter model as invoke:
reads call Yeelight cloud APIs directly, and semantic writes validate, execute, and verify immediately by default. Use --dry-run or --preview-only for a no-write preview when a caller wants to handle confirmation itself.

Actions:
  %s

Common flags:
  --json                 Print the full Skill Runtime JSON response.
  --profile <name>       Override active profile for this command.
  --region <region>      Override profile region.
  --house-id <id>        Override selected home for house-scoped commands.
  --params-json <json>   Pass advanced intent parameters as a JSON object.
  --set key=value        Add one or more advanced parameters, comma-separated.

Examples:
%s
  yeelight-home help %s %s
  yeelight-home invoke --stdin
`, resource, strings.Join(actions, "|"), strings.Join(actions, ", "), moduleExamples(resource), resource, actions[0])
}

func moduleExamples(resource string) string {
	examples := moduleCommandExamples[resource]
	if len(examples) == 0 {
		actions := moduleCommandNames(resource)
		if len(actions) > 0 {
			examples = []string{fmt.Sprintf("yeelight-home %s %s --json", resource, actions[0])}
		}
	}
	lines := make([]string, 0, len(examples))
	for _, example := range examples {
		lines = append(lines, "  "+example)
	}
	return strings.Join(lines, "\n")
}

func printRootHelp(stdout io.Writer) int {
	_, _ = fmt.Fprintf(stdout, rootHelpTemplate, rootModuleHelpLines())
	return exitOK
}

func rootModuleHelpLines() string {
	lines := []string{}
	for _, resource := range moduleResourceNames() {
		description := moduleCommandDescriptions[resource]
		if description == "" {
			description = "Runtime resource commands"
		}
		lines = append(lines, fmt.Sprintf("  %-14s %s", resource, description))
	}
	return strings.Join(lines, "\n")
}

func printCommandHelp(stdout io.Writer, stderr io.Writer, command string) int {
	if text, ok := commandHelpText[command]; ok {
		_, _ = fmt.Fprint(stdout, text)
		return exitOK
	}
	if text, ok := moduleActionHelpText(command); ok {
		_, _ = fmt.Fprint(stdout, text)
		return exitOK
	}
	_, _ = fmt.Fprintf(stderr, "unknown help topic %q\n", command)
	_ = printRootHelp(stdout)
	return exitInvalidInput
}

func printHelpForArgs(stdout io.Writer, stderr io.Writer, args []string) (int, bool) {
	topic, ok := helpTopic(args)
	if !ok {
		return 0, false
	}
	if topic == "" {
		return printRootHelp(stdout), true
	}
	return printCommandHelp(stdout, stderr, topic), true
}

func helpTopic(args []string) (string, bool) {
	if len(args) == 0 {
		return "", true
	}
	if isHelpArg(args[0]) {
		if len(args) == 1 {
			return "", true
		}
		return strings.Join(args[1:], " "), true
	}
	if isHelpArg(args[len(args)-1]) {
		return strings.Join(args[:len(args)-1], " "), true
	}
	return "", false
}

func isHelpArg(value string) bool {
	return value == "help" || value == "--help" || value == "-h"
}

func isVersionArg(value string) bool {
	return value == "version" || value == "--version" || value == "-v"
}
