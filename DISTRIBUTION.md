# Distribution

`yeelight-home` is distributed as a standalone CLI. The public source and release repository is `Yeelight/yeelight-home`; Skill packages should depend on the installed CLI through `YEELIGHT_HOME_BIN` or `PATH`.

## Channels

| Channel | Status | User install path |
| --- | --- | --- |
| GitHub Releases | Published | `curl -fsSL https://github.com/yeelight/yeelight-home/releases/latest/download/install.sh \| sh` |
| Homebrew | Published | `brew install Yeelight/tap/yeelight-home` |
| Scoop | Published | `scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket && scoop install yeelight-home` |
| Debian package | Published | Download `yeelight-home_0.1.0_amd64.deb` or `yeelight-home_0.1.0_arm64.deb` from GitHub Releases |
| Winget | Submitted | `winget install Yeelight.yeelight-home` after microsoft/winget-pkgs PR 392555 is merged |
| npm | Prepared | `npm install -g yeelight-home` after npm registry publication |

## Repository Layout Policy

- Keep `Yeelight/yeelight-home` as the public Runtime source and release repository.
- Keep `Yeelight/homebrew-tap` as a shared Homebrew tap for all Yeelight formulas. This is conventional because GitHub taps use the `homebrew-` prefix for the short `brew tap Yeelight/tap` form.
- Scoop does not require a Yeelight organization repository or a dedicated repository. It only needs a Git bucket repository containing JSON manifests. `Yeelight/scoop-bucket` is valid and already published; if repository count becomes a problem, future Scoop manifests can move into a consolidated distribution repository, but the existing bucket should remain as a compatibility pointer.
- Winget does not require a Yeelight repository. Official publication happens through `microsoft/winget-pkgs`; any personal fork is only a PR workspace.
- npm does not require a GitHub repository. It requires an npm account or automation token with permission to publish `yeelight-home` or the chosen scoped package.

## npm Package Model

The npm package is a thin installer and launcher:

1. `postinstall` downloads the matching GitHub Release asset for the user's OS and CPU.
2. The asset checksum is verified against `checksums.txt`.
3. The binary is cached under the user's local cache directory.
4. The npm `yeelight-home` binary delegates all arguments to the cached Go Runtime binary.

Environment overrides:

- `YEELIGHT_HOME_REPO=owner/repo`
- `YEELIGHT_HOME_VERSION=yeelight-home-v0.1.0` or `latest`
- `YEELIGHT_HOME_NPM_CACHE_DIR=/custom/cache`
- `YEELIGHT_HOME_NPM_SKIP_INSTALL=1`
