# yeelight-home

`yeelight-home` is the local Runtime CLI for Yeelight smart-home Skills. It keeps credentials on the user's machine, resolves semantic Skill requests, calls Yeelight cloud APIs directly, and returns redacted structured results.

## Install

Install from the public runtime release repository:

```sh
curl -fsSL https://github.com/yeelight/yeelight-home/releases/latest/download/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://github.com/yeelight/yeelight-home/releases/latest/download/install.ps1 -UseB | iex
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

Debian/Ubuntu users can download the `yeelight-home_0.1.0_amd64.deb` or `yeelight-home_0.1.0_arm64.deb` asset from the GitHub Release and install it with `apt` or `dpkg`.

Npm:

```sh
npm install -g yeelight-home
```

Winget publication is submitted for review at https://github.com/microsoft/winget-pkgs/pull/392555. Until it is merged, use GitHub Releases, Homebrew, Scoop, or set `YEELIGHT_HOME_BIN` to an absolute `yeelight-home` executable path.

## Quick Start

```sh
yeelight-home version
yeelight-home auth status --json
yeelight-home auth login --qr --region dev
yeelight-home home list --json
yeelight-home home select --house-id <house-id>
yeelight-home doctor --json
```

For non-interactive local setup, import a token outside chat:

```sh
yeelight-home auth token set --token <access-token> --region cn --client-id <client-id> --house-id <house-id>
```

Do not paste tokens into AI chat. The CLI stores tokens in the system credential store when available, and otherwise in a protected local credential fallback under the Runtime config directory.

## Skill Integration

Skills should call only:

```sh
yeelight-home invoke --stdin
```

The Yeelight Smart Home Skill wrapper resolves the Runtime in this order:

1. `YEELIGHT_HOME_BIN`
2. development-only bundled binary when present in a source checkout
3. `yeelight-home` on `PATH`

No Skill should call raw URLs, headers, curl, MCP compatibility services, or token-bearing commands.
