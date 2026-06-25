# Configuration

`yeelight-home` separates secret credentials from ordinary profile metadata.

- Tokens are stored in the system credential store when available, or in a protected local fallback.
- Profile metadata stores non-secret values: profile name, region, selected home, and QR device identity.
- User-facing commands do not require a client id. When QR login returns a service client id, the Runtime may keep it internally for API compatibility.

## Precedence

Runtime configuration is resolved in this order:

1. Command flags.
2. Environment variables.
3. Active profile metadata and credential store.
4. Defaults.

Defaults:

| Setting | Default |
| --- | --- |
| Profile | `default` |
| Region | `cn` |
| Home | unset; optional until a house-scoped operation needs a default home |

## Regions

Supported region values:

| Region | Meaning |
| --- | --- |
| `cn` | China cloud, default. |
| `sg` | Singapore cloud. |
| `us` | United States cloud. |
| `eu` | Europe cloud. |
| `dev` | Development cloud. Use only for internal validation. |

Examples:

```sh
yeelight-home doctor --json
yeelight-home doctor --json --region sg
YEELIGHT_CLOUD_REGION=eu yeelight-home home list --json
```

## Profiles

Profiles let one machine keep separate local metadata for different accounts, homes, or regions.

```sh
yeelight-home profile list --json
yeelight-home profile show --profile default --json
yeelight-home profile use --profile family --region cn
yeelight-home profile use --profile family --region cn --house-id <house-id>
yeelight-home profile delete --profile family
```

Selection rules:

- `--profile <name>` applies to one command.
- `YEELIGHT_HOME_PROFILE=<name>` applies to one process and overrides the active profile.
- `profile use --profile <name>` persists the active profile for later commands.
- Without any of the above, `default` is used.

## Authentication

### QR Login

```sh
yeelight-home auth login --qr --profile default
yeelight-home auth login --qr --profile default --region sg
yeelight-home auth login --qr --profile default --qr-png /tmp/yeelight-login.png
```

`--region` defaults to `cn`. `--house-id` is optional and can be passed when a specific home should be carried into the QR login payload.

### Manual Token Import

```sh
yeelight-home auth token set --profile default --token <access-token> --region cn
yeelight-home auth token set --profile default --token <access-token> --region cn --house-id <house-id>
yeelight-home auth token delete --profile default
```

Use token import only outside AI chat. Tokens are redacted from normal output and are not written to profile metadata.
Token-only setup is valid. Omit `--house-id` when you only need account-level commands or will choose a default home later.

### Status

```sh
yeelight-home auth status --json
yeelight-home auth status --json --profile family
```

The status output includes token presence and source, not token values.

## Home Selection

```sh
yeelight-home home list --json
yeelight-home home list --json --region eu
yeelight-home home select --house-id <house-id>
yeelight-home home select --profile family --house-id <house-id> --region cn
```

`home select` writes the selected home id into profile metadata. A process can temporarily override it with `--house-id` or `YEELIGHT_HOME_HOUSE_ID`.
The selected home is a default context, not an authentication requirement. Account-level commands such as `auth status`, `doctor`, `api smoke`, `home list`, home summary/search, and account info work with token-only profiles. Device, room, area, group, scene, automation, gateway, favorite, lighting, and other house-scoped operations require `houseId` at the request, environment, or profile layer.

## Config Commands

```sh
yeelight-home config get --json
yeelight-home config set --profile family --region cn --house-id <house-id> --json
yeelight-home config unset --profile family --house-id --json
```

`config set` and `config unset` update non-secret profile metadata only.

Supported metadata flags:

| Flag | Purpose |
| --- | --- |
| `--profile <name>` | Selects profile. |
| `--region <region>` | Stores region for the profile. |
| `--house-id <id>` | Stores selected home for the profile; optional until house-scoped operations need it. |
| `--qr-device <mac>` | Advanced: stores a stable QR device identity for QR login. |

## Environment Variables

| Variable | Scope | Notes |
| --- | --- | --- |
| `YEELIGHT_HOME_BIN` | Skill wrapper lookup | Absolute CLI path used by Skills. |
| `YEELIGHT_HOME_PROFILE` | Profile | Overrides active profile for this process. |
| `YEELIGHT_CLOUD_REGION` | Region | Overrides profile region for this process. |
| `YEELIGHT_HOME_HOUSE_ID` | Home | Overrides selected home for this process. |
| `YEELIGHT_HOME_ACCESS_TOKEN` | Credential | Temporary token for local smoke tests or CI. |
| `YEELIGHT_HOME_DIR` | Storage | Overrides Runtime home directory. |
| `YEELIGHT_API_BASE_URL` | Development | Overrides API base URL; do not expose in Skills. |

`YEELIGHT_HOME_ACCESS_TOKEN` has higher priority than the credential store, but it is never persisted by `config` commands.

## Storage Paths

Default paths are platform-specific:

| OS | Runtime home |
| --- | --- |
| macOS | `~/Library/Application Support/yeelight-home` plus cache/data under standard user directories |
| Linux | `~/.yeelight-home` or XDG-compatible directories depending on environment |
| Windows | `%LOCALAPPDATA%\YeelightHome` |

Run `yeelight-home doctor --json` to see the exact paths on the current machine.

## Skill Integration

Skill packages call:

```sh
yeelight-home invoke --stdin
```

Skill wrappers find the CLI in this order:

1. `YEELIGHT_HOME_BIN`
2. Development-only source checkout binary
3. `yeelight-home` on `PATH`

After installation:

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
yeelight-home home list --json
# Optional default home for house-scoped operations:
yeelight-home home select --house-id <house-id>
```

## Troubleshooting

### Runtime missing

Install the CLI from GitHub Releases or a package manager, then restart the host application if it inherited an old `PATH`.

### Auth required

Run:

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
```

### Wrong region

Use a one-time override:

```sh
yeelight-home home list --json --region sg
```

Or persist it:

```sh
yeelight-home profile use --profile default --region sg
```

### Wrong home

List homes and select the correct one:
Only do this when the failed operation is house-scoped or when you want to change the default home.

```sh
yeelight-home home list --json
yeelight-home home select --house-id <house-id>
```

### Need a diagnostic bundle

Run:

```sh
yeelight-home doctor --json
```

Share only redacted diagnostic output. Do not share token files or credential store exports.
