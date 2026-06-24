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

Homebrew:

```sh
brew install Yeelight/tap/yeelight-home
```

Scoop:

```powershell
scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket
scoop install yeelight-home
```

Winget:

- Submitted for review: https://github.com/microsoft/winget-pkgs/pull/392555
- After merge: `winget install Yeelight.yeelight-home`

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
