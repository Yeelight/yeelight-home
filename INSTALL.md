# Installation

This document is for users installing the standalone `yeelight-home` CLI.

## Verify An Existing Install

```sh
yeelight-home version
yeelight-home version --json
yeelight-home doctor
yeelight-home doctor --json
yeelight-home doctor --json --online
```

`version --json` reports build metadata. `doctor` prints a human-readable diagnostic summary. `doctor --json` reports the same data as a machine-readable object: running CLI version, executable path, `PATH` lookup result, npm wrapper path when applicable, and local package-manager diagnostics. Homebrew diagnostics distinguish formula and cask installs.
`doctor --online` additionally checks public GitHub Release, npm registry, and Yeelight Homebrew tap latest versions. Use it when npm, Homebrew, Scoop, or a host application seems to be launching an older binary than the latest release. If one online channel reports `ok=false`, continue using the other channel results and the local warnings; public registries can rate-limit or temporarily fail independently.
Text output includes an `Install source summary` section that calls out whether `PATH` resolves an npm wrapper and which npm/Homebrew versions are installed.
If it includes `path_lookup_differs_from_running_executable`, the shell or host application is resolving a different `yeelight-home` binary than the one being inspected.
If it includes `npm_wrapper_differs_from_path_lookup`, the active npm wrapper is not the same wrapper that the current shell resolves from `PATH`; restart the host shell, check `command -v yeelight-home`, or remove the stale channel.
If it includes `npm_global_package_version_differs_from_runtime_version`, the globally installed npm wrapper version differs from the Runtime binary it launches or from the binary currently being inspected. Upgrade or reinstall through one channel and restart the host shell.
If it includes `npm_global_package_behind_latest`, the npm registry has a newer published package than the globally installed wrapper. Run `npm install -g yeelight-home@latest` and restart the shell or Skill host.
If it includes `homebrew_package_version_differs_from_runtime_version`, refresh Homebrew metadata and upgrade the formula or remove the older channel from `PATH`.
If it includes `homebrew_formula_version_differs_from_runtime_version`, refresh Homebrew metadata and upgrade the formula with `brew update && brew upgrade yeelight-home`.
If it includes `homebrew_cask_version_differs_from_runtime_version`, refresh Homebrew metadata and upgrade the cask with `brew update && brew upgrade --cask yeelight-home`.
If it includes `homebrew_formula_behind_latest`, refresh Homebrew metadata with `brew update`, then run `brew upgrade yeelight-home` or reinstall from `Yeelight/tap`.
If it includes `homebrew_cask_behind_latest`, refresh Homebrew metadata with `brew update`, then run `brew upgrade --cask yeelight-home` or reinstall from `Yeelight/tap`.
Use `install.remediations` in JSON output, or the `Suggested fixes` section in text output, for the safest next command for the current machine.

After installing, authenticate locally:

```sh
yeelight-home auth status
yeelight-home auth login --qr
```

The default region is `cn`. Use `--region sg`, `--region us`, or `--region eu` when needed.

If QR login is not possible but you have an authorized token, import it locally without putting it in shell history:

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
```

For dev validation:

```sh
printf '%s' "$YEELIGHT_DEV_TOKEN" | yeelight-home auth token set --stdin --profile dev --region dev --json
yeelight-home home list --profile dev --region dev --json
```

## GitHub Releases

GitHub Releases are the canonical fallback channel for all platforms.

macOS and Linux:

```sh
curl -fsSL https://github.com/Yeelight/yeelight-home/releases/latest/download/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://github.com/Yeelight/yeelight-home/releases/latest/download/install.ps1 -UseB | iex
```

Installer overrides:

```sh
YEELIGHT_HOME_REPO=Yeelight/yeelight-home \
YEELIGHT_HOME_VERSION=v0.1.1 \
YEELIGHT_HOME_INSTALL_DIR="$HOME/.local/bin" \
sh install.sh
```

PowerShell:

```powershell
$env:YEELIGHT_HOME_REPO="Yeelight/yeelight-home"
$env:YEELIGHT_HOME_VERSION="v0.1.1"
$env:YEELIGHT_HOME_INSTALL_DIR="$env:LOCALAPPDATA\Programs\YeelightHome\bin"
.\install.ps1
```

The installers verify `checksums.txt` before copying the binary.

## macOS And Linux: Homebrew

```sh
brew install Yeelight/tap/yeelight-home
```

Upgrade:

```sh
brew update
brew upgrade yeelight-home
```

Uninstall:

```sh
brew uninstall yeelight-home
```

## Windows: Scoop

```powershell
scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket
scoop install yeelight-home
```

Upgrade:

```powershell
scoop update yeelight-home
```

Uninstall:

```powershell
scoop uninstall yeelight-home
```

## Windows: Winget

Winget is published through `microsoft/winget-pkgs`.

After the Yeelight package is accepted:

```powershell
winget install Yeelight.yeelight-home
winget upgrade Yeelight.yeelight-home
winget uninstall Yeelight.yeelight-home
```

If Winget cannot find the package yet, use GitHub Releases, Scoop, Homebrew on WSL, or npm.

## npm Wrapper

The npm package is a thin launcher. It downloads the matching Release binary on install or first run and verifies it against `checksums.txt`. For the official repository, the source order is GitHub, Gitee, then GitCode; a failed or timed-out source is skipped without weakening checksum verification.
When it launches the downloaded binary, it sets `YEELIGHT_HOME_NPM_WRAPPER_PATH` so `yeelight-home doctor --json` can report `install.npmWrapper` and `install.npmWrapperResolved`. This helps diagnose stale npm wrappers, Homebrew shadows, and host applications using a different `PATH`.

```sh
npm install -g yeelight-home
yeelight-home version
```

Environment overrides:

| Variable | Purpose |
| --- | --- |
| `YEELIGHT_HOME_REPO` | Release repository, default `Yeelight/yeelight-home`. |
| `YEELIGHT_HOME_VERSION` | Release tag or `latest`. |
| `YEELIGHT_HOME_DOWNLOAD_BASE_URL` | Use one explicit Release download base instead of automatic sources. |
| `YEELIGHT_HOME_DOWNLOAD_TIMEOUT_MS` | Per-request inactivity timeout, from 1000 to 120000 milliseconds. |
| `YEELIGHT_HOME_NPM_CACHE_DIR` | Binary cache directory. |
| `YEELIGHT_HOME_NPM_SKIP_INSTALL=1` | Skip binary download during npm install. |

## Linux Packages

GoReleaser produces Linux packages through nFPM:

- Debian/Ubuntu: `.deb`
- Fedora/RHEL/openSUSE: `.rpm`
- Alpine: `.apk`
- Arch package artifact

Debian/Ubuntu example:

```sh
curl -LO https://github.com/Yeelight/yeelight-home/releases/download/v0.1.1/yeelight-home_0.1.1_linux_amd64.deb
sudo apt install ./yeelight-home_0.1.1_linux_amd64.deb
```

RPM example:

```sh
curl -LO https://github.com/Yeelight/yeelight-home/releases/download/v0.1.1/yeelight-home_0.1.1_linux_amd64.rpm
sudo rpm -i ./yeelight-home_0.1.1_linux_amd64.rpm
```

Package filenames can vary by GoReleaser version. Check the release asset list when installing a pinned version.

## Arch Linux AUR

The planned package name is:

```sh
yay -S yeelight-home-bin
```

AUR publication requires an AUR package repository and deploy key. Until that is enabled, install from GitHub Releases or use the generated Arch package artifact from the release.

## Snap

The planned Snap name is:

```sh
sudo snap install yeelight-home
```

Snap publication requires Snapcraft credentials and store review. Until the Snap is visible in the store, install from GitHub Releases, Homebrew, npm, or Linux package assets.

## Docker And Container Images

Container images are intended for NAS, server, Raspberry Pi, and scheduled automation environments.

GHCR:

```sh
docker run --rm ghcr.io/yeelight/yeelight-home:latest version
```

Docker Hub:

```sh
docker run --rm yeelightdev/yeelight-home:latest version
```

Persist local config and credentials:

```sh
docker run --rm -it \
  -v "$HOME/.yeelight-home:/home/nonroot/.yeelight-home" \
  ghcr.io/yeelight/yeelight-home:latest doctor --json
```

Supported image platforms:

- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

## Go Ecosystem Discovery

For Go tooling and pkg.go.dev discovery, the public repository should use standard semantic tags such as `v0.1.1` and module path `github.com/yeelight/yeelight-home`.

Once a tag exists publicly:

```sh
GOPROXY=https://proxy.golang.org go list -m github.com/yeelight/yeelight-home@v0.1.1
```

pkg.go.dev indexes the module from the public repository and tag.

## PATH And Skill Hosts

Most package managers put `yeelight-home` on `PATH`. If a Skill host cannot find it:

```sh
export YEELIGHT_HOME_BIN="$(command -v yeelight-home)"
```

Windows PowerShell:

```powershell
$env:YEELIGHT_HOME_BIN=(Get-Command yeelight-home).Source
```

Restart the host application after changing user PATH.

## Multiple Install Sources

Use one primary install channel per machine when possible. Multiple global installers can leave an older `yeelight-home` earlier on `PATH`.

Check the active binary:

```sh
command -v yeelight-home
yeelight-home version
yeelight-home doctor --json
```

Typical ownership:

- npm global installs create a launcher under the Node prefix, for example `/opt/homebrew/bin/yeelight-home`.
- Homebrew installs under the Homebrew prefix, but it may not be active if an npm launcher with the same name is earlier on `PATH`.
- GitHub install scripts copy the binary to `YEELIGHT_HOME_INSTALL_DIR`, `$HOME/.local/bin`, or another selected directory.

Upgrade the channel that owns the active binary:

```sh
npm install -g yeelight-home@latest
brew update && brew upgrade yeelight-home
scoop update yeelight-home
```

When switching channels, uninstall or unlink the old channel first so `PATH` resolves one expected binary.

## Uninstall

Remove the binary through the same installer or package manager that installed it.

Manual GitHub installer cleanup:

```sh
rm -f /usr/local/bin/yeelight-home
rm -f "$HOME/.local/bin/yeelight-home"
```

Local Runtime data and credentials are separate. Delete them only when you intend to remove local preferences and credentials:

```sh
rm -rf ~/.yeelight-home
```

On Windows, remove `%LOCALAPPDATA%\YeelightHome` after deleting the package or binary.
