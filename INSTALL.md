# Installation

## GitHub Releases

macOS and Linux:

```sh
curl -fsSL https://github.com/yeelight/yeelight-home/releases/latest/download/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://github.com/yeelight/yeelight-home/releases/latest/download/install.ps1 -UseB | iex
```

Override the install source when testing a fork:

```sh
YEELIGHT_HOME_REPO=owner/repo YEELIGHT_HOME_VERSION=yeelight-home-v1.0.0 sh install.sh
```

PowerShell:

```powershell
$env:YEELIGHT_HOME_REPO="owner/repo"
$env:YEELIGHT_HOME_VERSION="yeelight-home-v1.0.0"
.\install.ps1
```

## Package Managers

Planned public channels:

- Homebrew: `brew install yeelight/tap/yeelight-home`
- Scoop: `scoop install yeelight-home`
- Winget: `winget install Yeelight.yeelight-home`

Use the GitHub release installer until the corresponding package is published.

## Verify

```sh
yeelight-home version
yeelight-home doctor --json
```

## Uninstall

Remove the binary from the install directory and delete local Runtime data only when you intend to remove credentials and local preferences:

```sh
rm -f /usr/local/bin/yeelight-home
rm -rf ~/.yeelight-home
```

On Windows, remove `%LOCALAPPDATA%\Programs\YeelightHome\bin\yeelight-home.exe` and remove that directory from the user `Path` if it was added by the installer.

