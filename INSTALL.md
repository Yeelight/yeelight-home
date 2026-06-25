# Installation

This document is for users installing the standalone `yeelight-home` CLI.

## Verify An Existing Install

```sh
yeelight-home version
yeelight-home doctor --json
```

After installing, authenticate locally:

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
```

The default region is `cn`. Use `--region sg`, `--region us`, or `--region eu` when needed.

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

The npm package is a thin launcher. It downloads the matching GitHub Release binary on install or first run and verifies the checksum.

```sh
npm install -g yeelight-home
yeelight-home version
```

Environment overrides:

| Variable | Purpose |
| --- | --- |
| `YEELIGHT_HOME_REPO` | Release repository, default `Yeelight/yeelight-home`. |
| `YEELIGHT_HOME_VERSION` | Release tag or `latest`. |
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
