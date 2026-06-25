# Releasing yeelight-home

This document is for maintainers.

## Release Model

`yeelight-home` is developed under the monorepo path:

```text
yeelight-smart-home/runtime
```

The public repository is:

```text
https://github.com/Yeelight/yeelight-home
```

Do not maintain two independent source trees. Export the runtime-only source from the monorepo, then release from the public repository. The monorepo workflow validates and mirrors source only; it does not build or publish release binaries.

## Versioning

Use standard semantic version tags in the public repository:

```text
v0.1.1
v0.2.0
v1.0.0
```

This is important for Go module tooling and pkg.go.dev. Older `yeelight-home-v*` tags can remain for compatibility, but new public runtime releases should use `v*`.

The npm package version must match the release version without the leading `v`.

## Preflight

From the monorepo:

```sh
cd yeelight-smart-home
go test ./...
node tools/host-wrapper-smoke.js
node tools/skill-structure-validate.js
sh tools/phase0-validate.sh
```

Export a public snapshot:

```sh
sh scripts/export-runtime-public.sh /tmp/yeelight-home-public
cd /tmp/yeelight-home-public
go test ./...
npm pack --dry-run
```

Optional local smoke when GoReleaser and Syft are installed:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

Local smoke is not the required release environment. The authoritative release run is the public `Yeelight/yeelight-home` GitHub Actions workflow on `ubuntu-latest`, which installs Go, Node, QEMU/buildx and Syft before invoking GoReleaser. Do not bypass Go module checksum verification. If local GoReleaser installation fails, use the GitHub Actions workflow instead of weakening checksum checks.

## Public Repository Release

1. Export runtime-only source.
2. Push it to `Yeelight/yeelight-home`.
3. Tag the public repository with `vX.Y.Z`.
4. Run the public repository release workflow.

The workflow runs:

```sh
go test ./...
goreleaser release --clean
npm publish --access public
```

GoReleaser produces archives, checksums, installer script assets, npm package tarball assets, Linux packages, Homebrew/Scoop updates, Docker images, and optional AUR/Snap/Winget outputs depending on configured secrets.

Do not add a second `softprops/action-gh-release` publish step for Runtime assets in the public workflow. GoReleaser is the single release asset publisher.

The generated public repository workflow computes a `--skip` list before running GoReleaser:

- Missing Homebrew token skips Homebrew cask updates.
- Missing Scoop token skips Scoop manifest updates.
- Missing Winget token skips Winget manifest publication.
- Missing AUR SSH key skips AUR publication.
- Missing Snapcraft credentials skips Snap packaging/publication.
- Docker Hub is enabled only when Docker Hub credentials exist; GHCR uses the repository `GITHUB_TOKEN`.

This keeps the core GitHub Release, checksums, install scripts, npm tarball asset and Linux package assets publishable even when a review-gated or credential-gated channel is not ready.

## Required Release Settings

Minimum:

- `NPM_TOKEN`

`GITHUB_TOKEN` is provided by GitHub Actions automatically for GitHub Releases and GHCR. Do not add it as a repository secret.

Package manager automation:

- `HOMEBREW_TAP_GITHUB_TOKEN`
- `SCOOP_BUCKET_GITHUB_TOKEN`
- `WINGET_GITHUB_TOKEN`

Container automation:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Linux ecosystem automation:

- `AUR_KEY`
- `SNAPCRAFT_STORE_CREDENTIALS`
- optional repository variable `AUR_GIT_URL` when the default `ssh://aur@aur.archlinux.org/yeelight-home-bin.git` is not correct

Configure only the secrets for channels that are ready.

Winget needs both a token and a PR workspace fork:

- `WINGET_GITHUB_TOKEN`
- repository variables `WINGET_REPOSITORY_OWNER` and `WINGET_REPOSITORY_NAME`
- optional repository variable `WINGET_REPOSITORY_BRANCH`

When these are absent, the workflow skips Winget. Do not create a permanent Yeelight organization repository only for Winget; a fork of `microsoft/winget-pkgs` is a review workspace, not Runtime source.

## Channel Notes

### GitHub Releases

Always publish archives, checksums, installers, npm package tarball, and metadata.

### Homebrew

Cask updates go to:

```text
Yeelight/homebrew-tap
```

Use:

```sh
brew install Yeelight/tap/yeelight-home
```

An existing Formula may remain available as a compatibility path. New GoReleaser automation uses Homebrew Casks because GoReleaser v2.16 marks formula generation as deprecated.
Keep this repository. It is a package-manager tap, not a Runtime source repository, and GoReleaser needs a writable target for cask metadata.

### Scoop

Manifest updates go to:

```text
Yeelight/scoop-bucket
```

Use:

```powershell
scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket
scoop install yeelight-home
```

Keep this repository. It is the shared Yeelight Scoop bucket and should contain only package metadata, not Runtime source or release archives.

### Winget

Winget publication is review-gated through `microsoft/winget-pkgs`. The workflow may create a manifest or PR, but Microsoft review still controls final availability.
Do not create a permanent Yeelight organization repository for Winget. A fork is only a temporary PR workspace if needed.

### AUR

The planned package name is `yeelight-home-bin`. AUR requires a package Git repository and an SSH deploy key.

### Snap

Snap publication requires Snapcraft credentials and store review. Snapcraft cannot always build inside containerized CI runners, so treat it as optional until the release runner is prepared.

### Docker

Publish multi-arch images:

- `ghcr.io/yeelight/yeelight-home`
- `yeelight/yeelight-home`

Required platforms:

- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

## pkg.go.dev

After a public `v*` tag exists, trigger Go proxy discovery:

```sh
GOPROXY=https://proxy.golang.org go list -m github.com/yeelight/yeelight-home@vX.Y.Z
```

Then check:

```text
https://pkg.go.dev/github.com/yeelight/yeelight-home
```

## Post-Release Smoke

Install from every published channel available for the version:

```sh
yeelight-home version
yeelight-home doctor --json
yeelight-home auth status --json
```

For Skill wrapper smoke:

```sh
cd yeelight-smart-home/skill/yeelight-smart-home
printf '{"contractVersion":"1.0","requestId":"host-smoke","locale":"zh-CN","utterance":"列出家庭","intent":"entity.list"}' | scripts/invoke.sh
```

When intentionally testing Runtime missing behavior, temporarily remove `yeelight-home` from `PATH` and unset `YEELIGHT_HOME_BIN`.
