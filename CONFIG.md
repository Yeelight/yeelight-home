# Configuration

## Precedence

Runtime configuration is resolved in this order:

1. Command flags: `--profile`, `--region`, `--client-id`, `--house-id`
2. Environment variables: `YEELIGHT_HOME_PROFILE`, `YEELIGHT_CLOUD_REGION`, `YEELIGHT_HOME_CLIENT_ID`, `YEELIGHT_HOME_HOUSE_ID`, `YEELIGHT_HOME_ACCESS_TOKEN`
3. Active profile, profile metadata, and credential store
4. Defaults: profile `default`, region `dev`

`YEELIGHT_API_BASE_URL` is a developer override for local testing. Do not expose it in Skill responses or user-facing automation.

## Profiles

```sh
yeelight-home profile list --json
yeelight-home profile show --profile default --json
yeelight-home profile use --profile family --region cn --client-id <client-id> --house-id <house-id>
yeelight-home profile delete --profile family
```

`profile use` saves metadata and sets the active local profile. Profile metadata is stored in `~/.yeelight-home/config/profiles.json` by default. It contains active profile, region, client id, house id, and QR device metadata. It must not contain access tokens.

## Credentials

Preferred login:

```sh
yeelight-home auth login --qr --region cn --profile default
```

Manual token import:

```sh
yeelight-home auth token set --profile default --token <access-token> --region cn --client-id <client-id> --house-id <house-id>
```

Status:

```sh
yeelight-home auth status --json
```

Tokens are loaded from `YEELIGHT_HOME_ACCESS_TOKEN` first, then from the local credential store. Status and doctor output only report whether a token is present.

## Home Selection

```sh
yeelight-home home list --json
yeelight-home home select --house-id <house-id> --profile default
```

`home select` changes only local profile metadata. It does not modify Yeelight cloud home data.

## Diagnostics

```sh
yeelight-home doctor --json
yeelight-home api smoke --json --region cn --profile default
```

`doctor` checks local installation, paths, profile, region, token presence, and Runtime storage status. `api smoke` performs read-only cloud checks when credentials are configured.
