# Changelog

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Initial CLI surface: `auth`, `context`, `repo`, `pr`, `branch`,
  `compare`, `issue`, `webhook`, `pipeline`, `api`.
- Backend drivers for Bitbucket Cloud (v2) and Data Center (v1).
- First-class diff rendering: colourised unified diff, diffstat, pager
  passthrough (delta / less), `NO_COLOR` / `--color` honouring.
- Multi-context config with OS-keyring credentials and env-var
  fallback for CI.
- Release pipeline: GoReleaser, cosign keyless, GHCR images, Homebrew
  tap, Scoop bucket, SBOMs.
