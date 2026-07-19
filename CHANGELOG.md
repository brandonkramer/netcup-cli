# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.2] - 2026-07-19

### Added

- `netcup install-mcp` materializes embedded plugin files into `~/.config/netcup/plugin` when no checkout is found (Homebrew / `go install` / Releases)

### Changed

- `make build` writes `bin/netcup` again (reverted `dist/` from the dropped npm packaging)

## [0.2.1] - 2026-07-19

### Added

- `netcup mcp` — MCP stdio server in the Go binary (no Python/`uv`)

### Changed

- Agent host manifests invoke `netcup mcp` instead of the Python FastMCP launcher
- Removed `plugin/netcup_mcp.py`

## [0.2.0] - 2026-07-19

### Added

- Agent MCP plugin inside the repo (`.codex-plugin` / `.cursor-plugin` / `.claude-plugin`, `plugin/netcup_mcp.py`, `bin/netcup-mcp`, `skills/netcup`) — curated tools plus `netcup_call` / `netcup_cli` for full CLI coverage
- `netcup install-mcp` — wire/refresh the plugin for Claude, Cursor, and Codex (`--scope`, `--host`, `--root`, `--dry-run`)

## [0.1.2] - 2026-07-19

### Fixed

- Media tab no longer shows Go pointer dumps for the attached ISO; it displays the ISO name

### Changed

- TUI tip, jobs, and status share one bottom meta line (more room for lists)

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

[Unreleased]: https://github.com/brandonkramer/netcup-cli/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/brandonkramer/netcup-cli/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/brandonkramer/netcup-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/brandonkramer/netcup-cli/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/brandonkramer/netcup-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/brandonkramer/netcup-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/brandonkramer/netcup-cli/releases/tag/v0.1.0
