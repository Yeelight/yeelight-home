# yeelight-home

Default language: English. Chinese documentation is available in [README.zh-CN.md](README.zh-CN.md).

`yeelight-home` is the standalone local Runtime CLI for Yeelight smart-home Skills and automation scripts. It runs on the user's machine, keeps credentials local, resolves smart-home requests, executes supported Yeelight home capabilities, and returns redacted structured results.

The Runtime is intentionally not bundled inside Skills. A Skill finds `yeelight-home` through `YEELIGHT_HOME_BIN` or `PATH` and sends one JSON request to `yeelight-home invoke --stdin`.

## Features

- Yeelight home capabilities for homes, rooms, areas, devices, groups, gateways, scenes, automations, diagnostics, lighting design, product knowledge, memory, and personalization.
- Product pedia search for fuzzy product lookup, product codes, product metadata, attachment records, and candidate manual or FAQ resource URLs.
- Thin execution model for persistent changes: supported writes execute directly after Runtime validation; callers own any user confirmation and can use dry-run previews when needed.
- Local credential handling: access tokens are stored in the system credential store when available, with a protected local fallback.
- Multiple profiles for different accounts, regions, or homes.
- Region-aware cloud endpoints with default region `cn`.
- Redacted JSON output for Skill hosts and diagnostics.
- Local preference memory and recommendation feedback stored under the Runtime data directory, not in Skill prompts.
- Human-friendly resource commands plus a stable `invoke --stdin` contract for Skills and generated apps.
- Cross-platform distribution through the GoReleaser-backed GitHub Releases pipeline, with Homebrew, Scoop, npm, Linux packages, and optional container/package-manager channels.
- Optional Docker/GHCR and Docker Hub images for NAS, server, and scheduled automation use.

## Capability Map

| Area | Examples |
| --- | --- |
| Home topology | homes, rooms, areas, groups, gateways, panels, knobs, sensors, unified entities |
| Device and scope control | light power, brightness, relative brightness/color-temperature adjustment, RGB color, node property set/read for home, room, area, group, and device targets |
| Organization writes | room/device naming, room movement, favorites, home sorting, panel/knob configuration |
| Scenes and automations | list, detail, execute, create/update/delete, enable/disable, verification |
| Product knowledge | `product.pedia.search`, manuals and FAQ candidates, thing-model schema and product definitions |
| Diagnostics | gateway/device diagnostics, upgrade files, operation progress, install and credential checks |
| Local intelligence | local preference memory, recommendation list, recommendation feedback and cooldown |

Reads execute immediately. Persistent writes and deletes also execute after Runtime validation; use `--dry-run`, `--preview-only`, or `options.dryRun=true` only when the caller wants a no-write preview before its own user confirmation.

## Install

macOS and Linux:

```sh
curl -fsSL https://github.com/Yeelight/yeelight-home/releases/latest/download/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://github.com/Yeelight/yeelight-home/releases/latest/download/install.ps1 -UseB | iex
```

Homebrew:

```sh
brew install Yeelight/tap/yeelight-home
```

Scoop:

```powershell
scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket
scoop install yeelight-home
```

npm wrapper:

```sh
npm install -g yeelight-home
```

Debian, Ubuntu, Fedora, Arch, AUR, Snap, Docker, GHCR, Docker Hub, and Winget channel details are maintained in [INSTALL.md](INSTALL.md) and [DISTRIBUTION.md](DISTRIBUTION.md).

## AI-Assisted Install

If you use a local AI assistant that can run terminal commands, paste this single request:

```text
Install the official Yeelight Home Runtime CLI for my operating system from Yeelight's GitHub Release or supported package manager, then install the Yeelight Smart Home Skill from the official Yeelight Skill release or ClawHub source. Verify the CLI with `yeelight-home doctor --json`, and guide me through `yeelight-home auth login --qr`; do not ask me to paste tokens, passwords, or cookies into chat.
```

## Quick Start

```sh
yeelight-home version
yeelight-home doctor
yeelight-home doctor --json
yeelight-home auth status --json
yeelight-home auth login --qr
yeelight-home home list --json
# Optional: choose a default home before house-scoped device, room, scene, or automation operations.
yeelight-home home select --house-id <house-id>
yeelight-home device list --json
yeelight-home product search --multi-field 青空灯 --json
yeelight-home scene execute --scene-id <scene-id> --json
yeelight-home light on --device-id <device-id> --json
yeelight-home automation enable --automation-id <automation-id> --json
```

The default region is `cn`. Pass `--region sg`, `--region us`, or `--region eu` when your Yeelight account belongs to another cloud region.

For non-interactive local setup, import a token outside chat. Prefer `--stdin` in real shells so the token is not saved in shell history:

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
printf '%s' "$YEELIGHT_DEV_TOKEN" | yeelight-home auth token set --stdin --profile dev --region dev --json
```

Token-only setup is valid. `houseId` is optional profile metadata for the default home. In other words, houseId is optional until you run a house-scoped command. Account-level commands such as `auth status`, `doctor`, `api smoke`, `home list`, `home.summary`, `home.search`, and `account.info` do not require it. House-scoped operations such as device, room, scene, group, gateway, favorite, and automation actions require a `houseId` from the request, `YEELIGHT_HOME_HOUSE_ID`, or the selected profile.

Do not paste tokens into AI chat. The CLI stores tokens locally and never prints token values in normal status or doctor output.

## Configuration Model

Runtime settings are resolved in this order:

1. Command flags.
2. Environment variables.
3. Active profile metadata and credential store.
4. Defaults.

Default values:

- Profile: `default`
- Region: `cn`
- Home: unset until selected, and only required for house-scoped operations

Common environment variables:

| Variable | Purpose |
| --- | --- |
| `YEELIGHT_HOME_BIN` | Absolute path used by Skills to find the CLI. |
| `YEELIGHT_HOME_PROFILE` | Selects a profile for this process. |
| `YEELIGHT_CLOUD_REGION` | Overrides region for this process: `cn`, `sg`, `us`, `eu`, or `dev` for development. |
| `YEELIGHT_HOME_HOUSE_ID` | Temporarily overrides selected home. |
| `YEELIGHT_HOME_ACCESS_TOKEN` | Temporary token for local smoke tests or CI; not written to profile metadata. |
| `YEELIGHT_HOME_DIR` | Overrides Runtime home directory. |
| `YEELIGHT_API_BASE_URL` | Developer-only API base URL override. Do not use in Skill prompts or user automation. |

See [CONFIG.md](CONFIG.md) for full command and precedence details.

## Command Reference

### Human Commands Versus `invoke`

`invoke --stdin` is the stable machine contract for Skills, generated apps, and automation hosts. It accepts one SkillRequest JSON object and returns one SkillResponse JSON object. It also accepts `--profile`, `--region`, and `--house-id` for one-shot context overrides.

Human operators should usually use resource commands:

```sh
yeelight-home device list --json
yeelight-home room list --json
yeelight-home scene execute --scene-id <scene-id> --json
yeelight-home light brightness --device-id <device-id> --brightness 60 --json
yeelight-home automation enable --automation-id <automation-id> --json
```

The resource commands are thin wrappers around Runtime intents. They keep the same profile, region, credential, redaction, preflight, direct execution, dry-run preview, and verification rules as `invoke`.

The command shape is intentionally conventional:

```text
yeelight-home <resource> <action> [--json] [--profile <name>] [--region <region>] [--house-id <id>] [resource flags]
```

Common resources include `home`, `room`, `area`, `device`, `entity`, `gateway`, `group`, `scene`, `automation`, `light`, `lighting`, `favorite`, `panel`, `knob`, `sensor`, `thing`, `upgrade`, `memory`, `recommendation`, and `account`. Run `yeelight-home --help` for the full resource list.

Use `yeelight-home help <resource>` to list actions, and `yeelight-home help <resource> <action>` for action-specific flags. Examples:

```sh
yeelight-home help device
yeelight-home help scene execute
yeelight-home help light brightness
```

For machine-readable intent contracts, use `intent schema` or the `explain` shortcut:

```sh
yeelight-home intent explain --intent scene.update --json
yeelight-home intent schema --intent lighting.design.import --json
yeelight-home explain lighting.design.import --json
```

These commands are offline. They print the SkillRequest envelope, accepted parameter keys, nested payload shape, examples, and nextStep hints so Skills and traditional programs do not need to guess large JSON fields such as lighting design models, scene `actions[]`, automation `trigger` / `conditions` / `actions[]`, panel button events, or batch operations.

For uncommon fields, pass advanced parameters through the documented request contract:

```sh
yeelight-home room search --name 客厅 --json
yeelight-home scene create --params-json '{"name":"回家灯光","actions":[{"targetType":"device","targetId":"50018330","set":{"power":true}}]}' --json
yeelight-home favorite add --set targetType=device,targetId=50018330,rank=1 --json
yeelight-home product search --multi-field 青空灯 --json
yeelight-home product search --product-model YP-0117 --json
yeelight-home thing schema-get --schema-id <schema-id> --json
yeelight-home upgrade files --json
yeelight-home panel button-configure --device-id <panel-id> --params-json '<json>' --json
```

### Product Knowledge

```sh
yeelight-home product search --multi-field 青空灯 --json
yeelight-home product search --product-code 1-000003268 --json
yeelight-home product search --product-model YP-0117 --json
```

`product search` returns redacted product metadata such as product name, brand, model, SKU/SPU, category/class fields, product code, support markers, status, attachments, and candidate manual or FAQ resource URLs when they can be derived safely. Product knowledge explains what a product is; it does not prove that a matching device is installed in a user's home. Use `entity capabilities`, `device detail`, or `state query` for installed-device truth.

### Local Memory And Recommendations

```sh
yeelight-home memory remember --house-id <house-id> --set scopeType=room,scopeRef=客厅,preferenceType=brightness,preferenceValue=45 --json
yeelight-home memory remember --house-id <house-id> --params-json '{"preferences":[{"scopeType":"profile","preferenceType":"ambience","preferenceValue":"prefer_romantic_warm","evidence":"user explicitly asked to remember romantic ambience"},{"scopeType":"profile","preferenceType":"product_preference","preferenceValue":"prefer_premium_luxury","evidence":"user explicitly asked to remember premium product positioning"}]}' --json
yeelight-home recommendation record --house-id <house-id> --params-json '{"type":"automation","source":"ai_skill","targetIntent":"automation.create","scopeType":"room","scopeRef":"主卧","explanation":"Create a warm evening automation from the saved romantic ambience preference.","evidence":"Saved memory ambience=prefer_romantic_warm"}' --json
yeelight-home recommendation list --house-id <house-id> --json
yeelight-home recommendation feedback --params-json '{"recommendationId":"<id>","feedback":"cooldown","cooldownHours":24}' --json
```

Local memory and recommendations are enabled by default for every `profile + region + houseId` scope. `memory pause` is the explicit opt-out switch, and `memory resume` turns local learning back on. `memory remember` directly upserts one structured local preference or multiple structured preferences in `parameters.preferences[]`. `recommendation record` directly upserts a caller-authored structured candidate; the Runtime only validates, stores, deduplicates, ranks, lists, and records feedback. Recommendation judgment belongs to the caller/Skill, not the Runtime. Feedback such as `accepted`, `dismissed`, `rejected`, or `cooldown` is stored locally and respected by later recommendation reads.

The Runtime does not store full conversation logs as memory and does not interpret subjective natural-language preferences. Callers such as Skills must pass structured fields such as `scopeType`, `scopeRef`, `preferenceType`, `preferenceValue`, and `evidence`. If callers want "warm soft light" and "warmer" to be the same memory, they should pass the same canonical `preferenceValue`; Runtime then merges exact same structured preferences and evidence instead of duplicating them.

The JSON store is sharded under `~/.yeelight-home/data/memory/<profile>/<region>/<houseId>.json`. Each shard and export carries namespace metadata with `accountProfile`, `profile`, `region`, `houseId`, and `dataType`. Interaction signals are coarse counters with `intent` and response `status` evidence only; user utterances are not stored as signal evidence. Accepted, dismissed, or rejected recommendation evidence and interaction signals are compacted after the local retention window, while explicit preferences remain until the user forgets them.

### `doctor`

```sh
yeelight-home doctor [--json] [--online] [--profile <name>] [--region <region>] [--house-id <id>]
```

Reports installation, config directories, selected profile, selected region, selected home, token presence, and warnings. Without `--json`, it prints a human-readable diagnostic summary. With `--json`, it prints the machine-readable diagnostic object. Token values are never printed.
The selected home may be empty; that is healthy for token-only account-level use.
The `install` object includes the running CLI version, executable path, `PATH` lookup result, OS, architecture, and npm wrapper path when launched through the npm package. If `path_lookup_differs_from_running_executable` appears, a shell, package manager, or Skill host is resolving a different `yeelight-home` binary than the one currently being inspected.
It also includes `packageManagers.npm` and `packageManagers.homebrew` when those tools are available. Homebrew diagnostics include separate `formula` and `cask` entries so PATH drift can be traced to the exact channel. Use those fields to find stale npm wrappers, Homebrew formula installs, or Homebrew cask installs that differ from the Runtime binary on `PATH`.
When launched by the npm wrapper, `doctor --json` reports `install.npmWrapper` and `install.npmWrapperResolved`. `npm_wrapper_differs_from_path_lookup` means the running wrapper is not the same file that the current shell would resolve from `PATH`; restart the host shell or remove the stale channel.
Text output includes `Install source summary` to make the active PATH channel and installed npm/Homebrew versions visible without parsing JSON.
Pass `--online` when you want `doctor` to query the public GitHub Release, npm registry, and Yeelight Homebrew tap latest versions. This is intentionally opt-in so default diagnostics stay fast and offline-friendly. A single online channel with `ok=false` means that channel could not be checked; other successful channel results and local warnings remain useful.
The `install.remediations` array and the text-mode `Suggested fixes` section provide safe next commands for the detected local install shape.

### `version`

```sh
yeelight-home version
yeelight-home version --json
```

`version --json` reports build metadata: version, commit, build date, OS, and architecture.

### `auth status`

```sh
yeelight-home auth status [--json] [--profile <name>] [--region <region>] [--house-id <id>]
```

Reports whether the selected profile has a usable local credential.
Without `--json`, it prints a human-readable summary. With `--json`, it prints the machine-readable status object.

### `auth login`

```sh
yeelight-home auth login --qr [--profile <name>] [--region <region>] [--house-id <id>] [--json] [--qr-png <path>]
```

Starts the local QR login flow. If `--region` is omitted, `cn` is used.
`--house-id` is optional and should be used only when a home context must be carried into the login payload.

### `auth token set`

```sh
yeelight-home auth token set (--token <access-token>|--stdin) [--profile <name>] [--region <region>] [--house-id <id>] [--json]
```

Imports a token into the local credential store. It never writes the token into the profile metadata file.
Prefer `--stdin` in real shells to avoid saving secrets in command history.
`--house-id` is optional. Omit it when you only need account-level commands or plan to select a home later.

### `profile`

```sh
yeelight-home profile list [--json]
yeelight-home profile show [--json] [--profile <name>] [--region <region>] [--house-id <id>]
yeelight-home profile use --profile <name> [--region <region>] [--house-id <id>] [--json]
yeelight-home profile delete --profile <name> [--json]
```

Profiles isolate account metadata and selected home. Use `YEELIGHT_HOME_PROFILE` or `--profile` for temporary selection.
The selected home can be empty in a profile.

### `config`

```sh
yeelight-home config get [--profile <name>] [--region <region>] [--house-id <id>] [--json]
yeelight-home config list [--profile <name>] [--json]
yeelight-home config set [--profile <name>] [--region <region>] [--house-id <id>] [--qr-device <mac>] [--json]
yeelight-home config unset [--profile <name>] [--region] [--house-id] [--qr-device] [--json]
```

`config` changes non-secret profile metadata only.

### `completion`

```sh
yeelight-home completion <bash|zsh|fish|powershell>
```

Prints a shell completion script to stdout. Install it using the standard mechanism for your shell.

### `home`

```sh
yeelight-home home list [--profile <name>] [--region <region>] [--json]
yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]
```

Lists homes available to the selected credential and stores the default home for later Skill calls.
Run `home select` only when you want future house-scoped commands to use a default home. You can also pass a one-time `--house-id` or `YEELIGHT_HOME_HOUSE_ID`.

### `invoke`

```sh
yeelight-home invoke --stdin [--profile <name>] [--region <region>] [--house-id <id>]
```

Reads a SkillRequest JSON object from stdin and writes a SkillResponse JSON object to stdout. This is the only command Skills should call for smart-home operations.
Flag overrides are applied before request parameters are resolved; request `parameters.region` and `parameters.houseId` still work when the corresponding flag is omitted.

Interactive users do not need to hand-write SkillRequest JSON for common operations. Prefer resource commands such as `device list`, `scene execute`, `light on`, `room create`, and `automation enable`.

### `api smoke`

```sh
yeelight-home api smoke [--json] [--profile <name>] [--region <region>] [--house-id <id>]
```

Runs a local cloud smoke check using the selected credential. This is intended for installation and support diagnostics.
Without `--json`, it prints a human-readable summary. With `--json`, it prints account and home-list check details for support automation.

## Skill Integration

Skill wrapper lookup order:

1. `YEELIGHT_HOME_BIN`
2. `yeelight-home` on `PATH`

Published Skill packages do not carry or auto-discover Runtime source-tree
binaries. Use `YEELIGHT_HOME_BIN` for a deliberate local override, or install
the public CLI so it is available on `PATH`.

When the Runtime is missing, install it from a published public channel, then run:

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
yeelight-home home list --json
```

If QR login is unavailable and the user already has an authorized token, import it locally outside chat:

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
```

Skills must use `yeelight-home` commands instead of URLs, headers, curl, third-party services, or token-bearing commands.

## Release And Packaging

`yeelight-home` uses a runtime-only public repository at `Yeelight/yeelight-home`. The source-of-truth code remains under `yeelight-smart-home/runtime` and is exported by automation.

The public runtime release pipeline uses GoReleaser from `Yeelight/yeelight-home`. The monorepo mirror workflow only validates and exports runtime source; it no longer builds or publishes CLI binaries.

One tagged public `v*` release can produce:

- macOS, Linux, Windows archives for `amd64`, `arm64`, and Linux `armv7`.
- Checksums and release metadata.
- Homebrew tap Formula and Cask automation.
- Scoop bucket manifest.
- Linux packages through nFPM: `.deb`, `.rpm`, `.apk`, and Arch package artifacts.
- Docker/GHCR and Docker Hub multi-arch images.
- Snap and AUR artifacts or publication when required credentials are configured.
- Winget manifest or PR flow when the Windows package route is enabled.

`Yeelight/homebrew-tap` and `Yeelight/scoop-bucket` remain standard package-manager metadata repositories. They should be updated by GoReleaser, not used as Runtime source repositories.

See [DISTRIBUTION.md](DISTRIBUTION.md) and [RELEASING.md](RELEASING.md).

## Security Notes

- Do not paste tokens, passwords, or account secrets into AI chat.
- `auth status`, `doctor`, and `invoke` responses are redacted.
- Profile metadata contains non-secret values such as profile name, region, selected home, and QR device identity. Tokens stay in credential storage.
- Persistent writes use supported Runtime intents; the model cannot execute arbitrary low-level payloads. Callers own user confirmation for high-impact operations.
