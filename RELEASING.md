# Releasing yeelight-home

This document is for maintainers.

## Release Model

`yeelight-home` is developed under the monorepo path:

```text
yeelight-home
```

The public repository is:

```text
https://github.com/Yeelight/yeelight-home
```

Do not maintain two independent source trees. Export the runtime-only source from the monorepo, then release from the public repository. The monorepo workflow validates and mirrors source only; it does not build or publish release binaries.
The mirror step is implemented by `scripts/mirror-runtime-public.sh` and can be run from GitHub Actions or from GitLab CI using the same `YEELIGHT_HOME_RELEASE_TOKEN`.

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
node tools/runtime-production-acceptance.js --version X.Y.Z --online
node tools/runtime-release-readiness.js --version X.Y.Z --online
go test ./...
node tools/runtime-release-readiness.js --version X.Y.Z
node tools/npm-wrapper-smoke.js
node tools/host-wrapper-smoke.js
node tools/skill-structure-validate.js
sh tools/phase0-validate.sh
```

`runtime-production-acceptance.js` is the preferred local release-candidate
acceptance entry point. It aggregates the offline Runtime, Skill, wrapper,
token-only, public-export and release-readiness gates, then reports the
confirmation-gated checks that it intentionally skips: dev live smoke, global
package installation and public release execution.

`runtime-release-readiness.js --online` checks that the target npm package
version and GitHub Release tag do not already exist. Run it before tagging;
the offline readiness check is also part of `phase0-validate.sh` and verifies
that `runtime/package.json`, GoReleaser version ldflags, the exported public
workflow and npm launcher packaging stay aligned.

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

- Missing Homebrew token skips Homebrew Formula and Cask updates.
- Missing Scoop token skips Scoop manifest updates.
- Missing Winget token skips Winget manifest publication.
- Missing AUR SSH key skips AUR publication.
- Missing Snapcraft credentials skips Snap packaging/publication.
- Docker Hub is enabled only when Docker Hub credentials exist; GHCR uses the repository `GITHUB_TOKEN`.

This keeps the core GitHub Release, checksums, install scripts, npm tarball asset and Linux package assets publishable even when a review-gated or credential-gated channel is not ready.

## Required Release Settings

No manual secret is required for the core GitHub Release and GHCR path. GitHub Actions provides `GITHUB_TOKEN` automatically for the public `Yeelight/yeelight-home` workflow. Do not add it as a repository secret.

Optional:

- `NPM_TOKEN` for publishing the npm launcher package.

Monorepo mirror to the public runtime repository:

- `YEELIGHT_HOME_RELEASE_TOKEN`

Configure this secret in the CI environment that runs the mirror step:

- GitHub-hosted monorepo mirror: repository secret `YEELIGHT_HOME_RELEASE_TOKEN`.
- GitLab-hosted monorepo mirror: protected and masked CI/CD variable `YEELIGHT_HOME_RELEASE_TOKEN`.

Do not put this token in the public `Yeelight/yeelight-home` source tree or in any checked-in configuration file.

## Docker Registry Visibility

The release workflow publishes `ghcr.io/yeelight/yeelight-home` and `yeelightdev/yeelight-home` images.

- Docker Hub image visibility can be verified with a registry pull using package credentials.
- GHCR is published from the public release workflow and should be treated as part of the normal release surface.

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

Reusable GitHub credential pattern:

- One GitHub PAT can be reused for `YEELIGHT_HOME_RELEASE_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`, `SCOOP_BUCKET_GITHUB_TOKEN`, and `WINGET_GITHUB_TOKEN` if it has write access to the target repos.
- `YEELIGHT_HOME_RELEASE_TOKEN` must at least be able to push to `Yeelight/yeelight-home`.
- The package-manager tokens only need write access to their own target repositories or PR workspace.

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

GoReleaser v2.16 marks formula generation as deprecated, but this release pipeline still updates the Formula compatibility path because `brew install Yeelight/tap/yeelight-home` resolves through Formula in existing user installs. It also updates the Cask path for the recommended newer Homebrew metadata model.
Keep this repository. It is a package-manager tap, not a Runtime source repository, and GoReleaser needs a writable target for Formula and Cask metadata.

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

The GitHub account that opens the Winget PR must have accepted the Microsoft CLA. If Microsoft comments with `msftbot/needsCLA` or adds the `Needs-CLA` label, the PR author must comment on that PR:

```text
@microsoft-github-policy-service agree
```

When contributing on behalf of a company, use:

```text
@microsoft-github-policy-service agree company="<company>"
```

After accepting, request a re-check if needed:

```text
@microsoft-github-policy-service rerun
```

Do this once for the publishing identity used by `WINGET_GITHUB_TOKEN`; otherwise future Winget PRs from that identity may remain blocked even when the manifests are correct.

### AUR

The planned package name is `yeelight-home-bin`. AUR requires an AUR account, an uploaded SSH public key, and an unencrypted SSH private key available to the release workflow.

Setup:

1. Create or log in to an AUR account.
2. Generate a dedicated unencrypted deploy key for this package.
3. Upload the public key to the AUR account.
4. Keep `AUR_GIT_URL` as `ssh://aur@aur.archlinux.org/yeelight-home-bin.git` unless the package base changes.
5. Store the private key contents in the public runtime repository secret `AUR_KEY`.

Example key generation on a maintainer machine:

```sh
ssh-keygen -t ed25519 -C "yeelight-home-bin aur release" -f ./yeelight-home-bin-aur -N ""
```

Upload `yeelight-home-bin-aur.pub` to the AUR account, then configure the release repository:

```sh
gh secret set AUR_KEY -R Yeelight/yeelight-home < ./yeelight-home-bin-aur
gh variable set AUR_GIT_URL -R Yeelight/yeelight-home --body "ssh://aur@aur.archlinux.org/yeelight-home-bin.git"
```

Delete the local private key after it is stored securely, unless the maintainer has a deliberate offline backup policy.

Do not put the private key in `.goreleaser.yaml`, shell scripts, documentation, or local config files. GoReleaser reads it from `AUR_KEY` at release time and skips AUR publication when the secret is absent.

### Snap

Snap publication requires a registered Snap name and exported Snapcraft store credentials.

Setup from an Ubuntu/Linux environment with Snapcraft available:

```sh
snapcraft login
snapcraft register yeelight-home
snapcraft export-login --snaps=yeelight-home \
  --acls package_access,package_push,package_update,package_release \
  snapcraft-login.txt
```

Store the full contents of `snapcraft-login.txt` in the public runtime repository secret `SNAPCRAFT_STORE_CREDENTIALS`, then delete the local file after the secret is configured.

Snapcraft cannot always build or authenticate cleanly on macOS or inside unavailable Docker daemons. If local Snapcraft is unavailable, generate this credential on Ubuntu/Linux or a prepared CI runner. GoReleaser skips Snap publication when `SNAPCRAFT_STORE_CREDENTIALS` is absent.

### Docker

Publish multi-arch images:

- `ghcr.io/yeelight/yeelight-home`
- `yeelightdev/yeelight-home`

The Docker Hub image can be overridden with repository variable `DOCKERHUB_IMAGE`.

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

Run the read-only channel verifier:

```sh
node scripts/verify-runtime-public-release.mjs X.Y.Z
```

The verifier checks GitHub Releases, npm, Homebrew Formula/Cask, Scoop, Docker Hub, GHCR visibility, Winget PR state, AUR `yeelight-home-bin`, Snapcraft `yeelight-home`, pkg.go.dev indexing and the local `PATH` version without installing or upgrading anything globally. GitHub Release validation requires `metadata.json`, `install.sh`, `install.ps1`, the npm tarball, checksums, desktop `.sbom.json` files, all desktop archives for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `linux/arm/v7`, `windows/amd64` and `windows/arm64`, plus nFPM Linux packages for `.deb`, `.rpm`, `.apk` and Arch package formats across `amd64`, `arm64` and `armv7`. Docker validation requires `linux/amd64`, `linux/arm64` and `linux/arm/v7` manifests. GHCR private visibility, Winget review gating, AUR/Snap optional publication and pkg.go.dev indexing latency are reported as warnings rather than release blockers when core install channels are healthy.

Install from every published channel available for the version:

```sh
yeelight-home version
yeelight-home version --json
yeelight-home doctor
yeelight-home doctor --json
yeelight-home auth status --json
```

For Skill wrapper smoke:

```sh
printf '{"contractVersion":"1.0","requestId":"host-smoke","locale":"zh-CN","utterance":"列出家庭","intent":"entity.list"}' | /path/to/skill/scripts/invoke.sh
```

When intentionally testing Runtime missing behavior, temporarily remove `yeelight-home` from `PATH` and unset `YEELIGHT_HOME_BIN`.
