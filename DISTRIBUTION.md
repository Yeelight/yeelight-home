# Distribution

`yeelight-home` is distributed as a standalone CLI. Skill packages should not carry Runtime binaries. They depend on `YEELIGHT_HOME_BIN` or `yeelight-home` on `PATH`.

## Channel Model

| Channel | GoReleaser role | User install path |
| --- | --- | --- |
| GitHub Releases | Primary release target for archives, installers, checksums, SBOMs, packages, and metadata. | `curl -fsSL https://github.com/Yeelight/yeelight-home/releases/latest/download/install.sh \| sh` |
| Homebrew tap | Updates Formula compatibility metadata and Cask metadata in `Yeelight/homebrew-tap`. | `brew install Yeelight/tap/yeelight-home` |
| Scoop bucket | Updates manifest metadata in `Yeelight/scoop-bucket`. | `scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket && scoop install yeelight-home` |
| npm wrapper | Publishes the launcher package after GoReleaser assets exist. | `npm install -g yeelight-home` |
| Linux packages | Builds `.deb`, `.rpm`, `.apk`, and Arch package artifacts through nFPM. | Download package assets from GitHub Releases. |
| Winget | Generates or supports the Microsoft registry submission path. | `winget install Yeelight.yeelight-home` after registry acceptance. |
| AUR | Publishes `yeelight-home-bin` only when AUR SSH is configured. | `yay -S yeelight-home-bin` after publication. |
| Snap | Builds/publishes only when Snapcraft credentials and store review are ready. | `sudo snap install yeelight-home` after store visibility. |
| Docker GHCR | Publishes multi-arch images when registry credentials are available. | `docker run --rm ghcr.io/yeelight/yeelight-home:latest version` |
| Docker Hub | Publishes multi-arch images when Docker Hub credentials are available. | `docker run --rm yeelightdev/yeelight-home:latest version` |
| pkg.go.dev | Indexes public `v*` module tags. | `https://pkg.go.dev/github.com/yeelight/yeelight-home` |

## Repository Layout Policy

- `Yeelight/yeelight-home` is the public Runtime source and release repository.
- The monorepo keeps the same source at the top-level `yeelight-home/` directory; no `yeelight-smart-home/runtime` compatibility directory or source copy exists.
- `yeelight-home/` may be initialized as a nested Git working tree for GitHub while the outer GitLab repository continues to track its files. Nested `.git` metadata remains local and is ignored by the outer repository.
- `yeelight-smart-home/scripts/export-runtime-public.sh` remains as a compatibility entry point for validation and mirroring.
- `Yeelight/homebrew-tap` is a conventional shared Homebrew tap. It is not a runtime source repo, should stay small, and should contain only package manager metadata generated or reviewed for release.
- `Yeelight/scoop-bucket` is a conventional shared Scoop bucket. Scoop can live in one consolidated Yeelight bucket repository; it does not need one repository per app.
- Winget does not need a Yeelight organization repository. Publication happens through `microsoft/winget-pkgs`; any fork is only a PR workspace.
- AUR does require an AUR Git repository per package, but that repository is not in the GitHub organization.
- Snap is managed through Snapcraft, not a GitHub distribution repo.
- Docker Hub and GHCR are image registries, not source repositories.

## GoReleaser Decision

Use GoReleaser for the public runtime release pipeline.

Reasoning:

- GoReleaser standardizes cross-platform Go builds, archives, checksums, Linux packages, Homebrew, Scoop, Docker images, SBOMs, AUR, Snap, and Winget workflows.
- It replaces the old monorepo hand-written cross-compile/package/release workflow. The monorepo now mirrors source only; public release builds happen in `Yeelight/yeelight-home`.
- It aligns with Go CLI user expectations and improves discoverability through package managers.
- It still does not remove external gates: Winget review, AUR account/SSH, Snapcraft credentials, Docker Hub credentials, and Homebrew/Scoop write tokens are still required.
- The public workflow automatically skips channels whose secrets are not configured, so optional channels do not block core GitHub Release assets, checksums, install scripts, npm wrapper assets and Linux package assets.
- GoReleaser v2.16 marks Homebrew formula generation as deprecated. New automation still updates the Formula compatibility path because many users install with `brew install Yeelight/tap/yeelight-home`; it also publishes the Cask path for the recommended newer Homebrew metadata model.
- While Formula compatibility remains supported, workflows pin GoReleaser `v2.15.2`; GoReleaser 2.17 treats the retained `brews` configuration as a failed check. Upgrade only when the Formula channel is intentionally removed or migrated.

Scope:

- Use GoReleaser in the public `Yeelight/yeelight-home` repository.
- Keep the monorepo export workflow responsible only for validating and pushing runtime-only public repo content.
- Use standard public runtime tags such as `v0.1.1` for Go ecosystem compatibility.
- Keep previous `yeelight-home-v*` release tags installable where they already exist, but do not use that prefix for future public runtime tags.

## Tap And Bucket Retention

Keep `Yeelight/homebrew-tap` and `Yeelight/scoop-bucket`.

They remain useful and standard even after GoReleaser adoption:

- Homebrew and Scoop install flows require a tap or bucket repository unless the package is accepted into broader upstream registries.
- GoReleaser updates those repositories automatically; maintainers should not hand-edit generated version URLs and checksums except for emergency repair.
- A shared `homebrew-tap` and `scoop-bucket` is cleaner than one repository per CLI. These repositories should contain only package metadata for Yeelight public tools.
- They must not contain Runtime source, Skill assets, raw docs, credentials, generated release archives, or development governance material.

Do not create extra Yeelight organization repositories for Winget, Snap, Docker, or pkg.go.dev. Those channels use external registries or the existing public Runtime repository.

## Release Automation Targets

GoReleaser configuration lives at:

```text
.goreleaser.yaml
```

The exported public repo includes:

```text
.goreleaser.yaml
Dockerfile
README.md
INSTALL.md
CONFIG.md
DISTRIBUTION.md
RELEASING.md
cmd/
internal/
npm/
scripts/
go.mod
go.sum
```

Target artifacts:

- Archives:
  - `darwin/amd64`
  - `darwin/arm64`
  - `linux/amd64`
  - `linux/arm64`
  - `linux/arm/v7`
  - `windows/amd64`
  - `windows/arm64`
- Checksums:
  - `checksums.txt`
- Linux packages:
  - `.deb`
  - `.rpm`
  - `.apk`
  - Arch package artifact
- Package-manager manifests:
  - Homebrew Formula and Cask in `Yeelight/homebrew-tap`
  - Scoop manifest in `Yeelight/scoop-bucket`
  - Winget manifest or PR flow once enabled
  - AUR `yeelight-home-bin` once AUR SSH is configured
  - Snap package once Snapcraft is configured
- Container images:
  - `ghcr.io/yeelight/yeelight-home`
  - `yeelightdev/yeelight-home`

## Required Release Settings

GitHub Actions provides `GITHUB_TOKEN` automatically for GitHub Releases and GHCR publishing. Do not create a repository secret named `GITHUB_TOKEN`.

| Secret | Purpose |
| --- | --- |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Push Homebrew cask to `Yeelight/homebrew-tap`. |
| `SCOOP_BUCKET_GITHUB_TOKEN` | Push Scoop manifest to `Yeelight/scoop-bucket`. |
| `NPM_TOKEN` | Publish npm wrapper package. |
| `DOCKERHUB_USERNAME` | Docker Hub login. |
| `DOCKERHUB_TOKEN` | Docker Hub publish token. |
| `AUR_KEY` | AUR SSH private key for `yeelight-home-bin`. |
| `SNAPCRAFT_STORE_CREDENTIALS` | Snap store publish credentials. |
| `WINGET_GITHUB_TOKEN` | Winget manifest PR token. |
| `YEELIGHT_HOME_RELEASE_TOKEN` | Push runtime mirror snapshots to `Yeelight/yeelight-home`. |

| Repository Variable | Purpose |
| --- | --- |
| `WINGET_REPOSITORY_OWNER` | Winget PR workspace fork owner. |
| `WINGET_REPOSITORY_NAME` | Winget PR workspace fork name. |
| `WINGET_REPOSITORY_BRANCH` | Optional Winget PR branch. Defaults to a release-specific branch. |
| `AUR_GIT_URL` | Optional AUR Git URL override. Defaults to `ssh://aur@aur.archlinux.org/yeelight-home-bin.git`. |
| `DOCKERHUB_IMAGE` | Optional Docker Hub image override. Defaults to `<DOCKERHUB_USERNAME>/yeelight-home`. |

Only configure settings for channels that are ready to publish.
Without Winget token and PR workspace variables, the public release workflow skips Winget and still publishes core GitHub Release artifacts.
You can self-solve Winget by forking `microsoft/winget-pkgs` as the PR workspace and pointing the repository variables at that fork; no permanent Yeelight organization repository is required for Winget publication.
Without `YEELIGHT_HOME_RELEASE_TOKEN`, the monorepo mirror workflow cannot push runtime-only source snapshots into the public repository.
AUR is optional until the package repository and SSH key are available.
Snapcraft is optional until a Linux or Ubuntu runner can execute the login/export flow and store publication.

Reusable GitHub credential pattern:

- One GitHub PAT can be reused for `YEELIGHT_HOME_RELEASE_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`, `SCOOP_BUCKET_GITHUB_TOKEN`, and `WINGET_GITHUB_TOKEN` if it has write access to the target repos or PR workspace.
- `YEELIGHT_HOME_RELEASE_TOKEN` only needs write access to `Yeelight/yeelight-home`.
- The package-manager tokens only need write access to their own target repositories or PR workspace.

## npm Package Model

The npm package is a thin installer and launcher:

1. `postinstall` downloads the matching GitHub Release asset for the user's OS and CPU.
2. The asset checksum is verified against `checksums.txt`.
3. The binary is cached under the user's local cache directory.
4. The npm `yeelight-home` binary delegates all arguments to the cached Go Runtime binary.

Environment overrides:

- `YEELIGHT_HOME_REPO=owner/repo`
- `YEELIGHT_HOME_VERSION=v0.1.1` or `latest`
- `YEELIGHT_HOME_NPM_CACHE_DIR=/custom/cache`
- `YEELIGHT_HOME_NPM_SKIP_INSTALL=1`

## Skill Distribution Contract

Skill packages include only:

- Skill instructions.
- Skill references and schemas.
- Wrapper scripts that call `yeelight-home invoke --stdin`.

Skill packages must not include:

- Runtime source.
- Runtime binaries.
- Installer scripts.
- Development docs.
- Raw API docs or compatibility-service references.

When Runtime is missing, the Skill should guide users to install `yeelight-home` from a public channel and then run:

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
```
