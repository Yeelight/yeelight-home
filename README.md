# yeelight-home

`yeelight-home` is the standalone local Runtime CLI for Yeelight smart-home Skills and automation scripts. It runs on the user's machine, keeps credentials local, resolves semantic smart-home requests, calls Yeelight cloud APIs directly, and returns redacted structured results.

The Runtime is intentionally not bundled inside Skills. A Skill finds `yeelight-home` through `YEELIGHT_HOME_BIN` or `PATH` and sends one JSON request to `yeelight-home invoke --stdin`.

## Features

- Direct Yeelight cloud API access for homes, rooms, areas, devices, groups, gateways, scenes, automations, diagnostics, lighting design, memory, and personalization.
- Guarded write model for persistent changes: risky changes create a local pending plan first, and `plan.commit` executes only a stored `planId`.
- Local credential handling: access tokens are stored in the system credential store when available, with a protected local fallback.
- Multiple profiles for different accounts, regions, or homes.
- Region-aware cloud endpoints with default region `cn`.
- Redacted JSON output for Skill hosts and diagnostics.
- Cross-platform distribution through the GoReleaser-backed GitHub Releases pipeline, with Homebrew, Scoop, npm, Linux packages, and optional container/package-manager channels.
- Optional Docker/GHCR and Docker Hub images for NAS, server, and scheduled automation use.

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
yeelight-home invoke --stdin
```

Reads a SkillRequest JSON object from stdin and writes a SkillResponse JSON object to stdout. This is the only command Skills should call for smart-home operations.

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

If QR login is unavailable and the user already has an approved token, import it locally outside chat:

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
```

Skills must not call raw URLs, raw headers, curl, compatibility services, or token-bearing commands.

## Release And Packaging

`yeelight-home` uses a runtime-only public repository at `Yeelight/yeelight-home`. The source-of-truth code remains under `yeelight-smart-home/runtime` and is exported by automation.

The public runtime release pipeline uses GoReleaser from `Yeelight/yeelight-home`. The monorepo mirror workflow only validates and exports runtime source; it no longer builds or publishes CLI binaries.

One tagged public `v*` release can produce:

- macOS, Linux, Windows archives for `amd64`, `arm64`, and Linux `armv7`.
- Checksums and release metadata.
- Homebrew tap Formula compatibility and Cask automation.
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
- Persistent writes use the Runtime pending-plan model; the model cannot execute arbitrary raw API payloads.
