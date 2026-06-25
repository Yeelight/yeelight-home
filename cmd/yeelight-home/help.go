package main

import (
	"fmt"
	"io"
	"strings"
)

const rootHelpText = `Yeelight Home CLI

Usage:
  yeelight-home <command> [flags]
  yeelight-home help [command]

Commands:
  auth       Login, inspect auth status, and manage local tokens
  profile    List, show, activate, and delete profiles
  config     Read and update non-secret profile configuration
  home       List and select homes
  invoke     Execute one Skill Runtime request from stdin
  approve    Commit a guarded pending plan
  api        Run account-scoped API smoke checks
  doctor     Print local installation and auth diagnostics
  version    Print CLI version

Global flags:
  -h, --help       Show help
  -v, --version    Show CLI version

Configuration precedence:
  command flags > environment variables > active profile metadata/credential store > defaults

Common examples:
  yeelight-home auth login --qr --region dev
  yeelight-home auth status --json
  yeelight-home home list --json --region dev
  yeelight-home home select --house-id <id> --region dev
  yeelight-home invoke --stdin
  yeelight-home doctor --json
`

var commandHelpText = map[string]string{
	"api": `Usage:
  yeelight-home api smoke --json [--profile <name>] [--region <region>] [--house-id <id>]

Runs account and home-list smoke checks with the active local token.
`,
	"api smoke": `Usage:
  yeelight-home api smoke --json [--profile <name>] [--region <region>] [--house-id <id>]

Runs account, home-list, and optional house-context smoke checks with the active local token.
`,
	"approve": `Usage:
  yeelight-home approve --json --plan-id <id> --challenge <text>

Commits a guarded pending plan created by yeelight-home invoke.
`,
	"auth": `Usage:
  yeelight-home auth status --json [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home auth login --qr [--json] [--profile <name>] [--region <region>] [--house-id <id>]
  yeelight-home auth token set --token <access-token> [--profile <name>] [--region <region>] [--house-id <id>] [--json]
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
  yeelight-home auth status --json [--profile <name>] [--region <region>] [--house-id <id>]

Reports local credential presence and resolved profile context without printing token values.
`,
	"auth token": `Usage:
  yeelight-home auth token set --token <access-token> [--profile <name>] [--region <region>] [--house-id <id>] [--json]
  yeelight-home auth token delete [--profile <name>] [--json]

Imports or deletes a token in the local credential store. Tokens are never written to profile metadata.
`,
	"auth token delete": `Usage:
  yeelight-home auth token delete [--profile <name>] [--json]

Deletes the selected profile token from the local credential store and protected fallback.
`,
	"auth token set": `Usage:
  yeelight-home auth token set --token <access-token> [--profile <name>] [--region <region>] [--house-id <id>] [--json]

Imports a token into local credential storage. Omit houseId for token-only account-scoped use.
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
  yeelight-home doctor --json [--profile <name>] [--region <region>] [--house-id <id>]

Prints local paths, active profile, token presence, region, houseId, and install diagnostics.
`,
	"home": `Usage:
  yeelight-home home list [--json] [--profile <name>] [--region <region>]
  yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]

home list is account-scoped and requires only a token. houseId is optional until a house-scoped command is used.
`,
	"home list": `Usage:
  yeelight-home home list [--json] [--profile <name>] [--region <region>]

Lists homes visible to the selected account. It is account-scoped and does not require houseId.
`,
	"home select": `Usage:
  yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]

Stores a default home id for later house-scoped commands. It does not change the token.
`,
	"invoke": `Usage:
  yeelight-home invoke --stdin

Reads one Skill Runtime JSON request from stdin and returns one JSON response.
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
  yeelight-home version
  yeelight-home --version
`,
}

func printRootHelp(stdout io.Writer) int {
	_, _ = fmt.Fprint(stdout, rootHelpText)
	return exitOK
}

func printCommandHelp(stdout io.Writer, stderr io.Writer, command string) int {
	if text, ok := commandHelpText[command]; ok {
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

func printVersion(stdout io.Writer) int {
	_, _ = fmt.Fprintf(stdout, "yeelight-home %s\n", version)
	return exitOK
}
