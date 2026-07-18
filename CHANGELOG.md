# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-07-18

### Added

- `netcup update` — detect Homebrew / `go install` / manual installs and run the matching upgrade path (`--dry-run`, `--method`)
- Install docs for Homebrew, `go install`, GitHub Releases, and `netcup update`

### Changed

- README title set to `netcup-cli` with a link to [netcup.com](https://www.netcup.com/)
- README aligned with current CLI/TUI surface (password-file secrets, metrics sub-view, release notes)

## [0.1.0] - 2026-07-18

### Added

- Initial public release of the netcup SCP CLI and ops TUI
- Auth gate, health chip, and resource tabs (servers, tasks, snapshots, disks, firewall, media, network, account)
- GoReleaser + GitHub Actions release workflow on `v*` tags
- Homebrew formula in [brandonkramer/homebrew-tap](https://github.com/brandonkramer/homebrew-tap)

### Security / hardening

- No passwords on argv (`--password-file` / TTY prompt)
- Profile name validation and `0700` config/cache dirs
- Nil-safe API response handling across CLI and TUI
- Body-less HTTP 202 jobs reported as `UNTRACKED` instead of success
- Signal-aware CLI exit (`130`) and cancelable TUI retries

[Unreleased]: https://github.com/brandonkramer/netcup-cli/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/brandonkramer/netcup-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/brandonkramer/netcup-cli/releases/tag/v0.1.0
