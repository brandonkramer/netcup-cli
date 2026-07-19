# AGENTS.md

**netcup-cli** — Go CLI + Bubble Tea TUI for the [netcup](https://www.netcup.com/) SCP REST API. Module: `github.com/brandonkramer/netcup-cli`. Binary: `netcup` (`./cmd/netcup`).

User docs: [README.md](./README.md). Release notes: [CHANGELOG.md](./CHANGELOG.md).

## Layout

- `cmd/netcup` — entrypoint
- `internal/cmd` — Cobra commands, coverage map, `netcup update`
- `internal/tui` — interactive ops TUI
- `internal/scpclient` — generated OpenAPI client (`make generate` from `openapi.json`)
- `internal/auth`, `config`, `cache`, `wait`, `output`, `patch` — session, config, caching, task wait, envelopes
- `internal/mcpserver` + `netcup mcp` — MCP stdio server (thin CLI facade)
- `internal/pluginassets` — embedded manifests/skill for `install-mcp` without a checkout
- `.codex-plugin` / `.cursor-plugin` / `.claude-plugin` — host plugin manifests (`netcup mcp`)
- `skills/netcup` — agent skill (prefer curated MCP tools, then `netcup_call` / `netcup_cli`)

## Commands

```bash
make build                 # bin/netcup
make test                  # go test ./...
make coverage              # OpenAPI ops mapped in internal/cmd
make generate              # regenerate scpclient from openapi.json
go test -count=1 ./...
```

Prefer typecheck/test over long-running builds unless asked. Do not start a TUI/dev server unless requested.

## Agent MCP

This repo is also a Codex/Cursor/Claude plugin. Agents should use MCP tools (not the TUI). Auth remains `netcup auth login` on a human TTY. Wire hosts with `netcup install-mcp` (`--scope user` = plugins; `--scope project|local` = `.agents/skills/netcup` + host MCP files). See README → Agent MCP and `skills/netcup/SKILL.md`.

## Rules

- Keep changes surgical; match existing Go style; avoid `any`-equivalents and drive-by refactors
- Secrets never on argv — use `--password-file` / `-` or a TTY prompt
- Nil-check API responses before `StatusCode()`; body-less HTTP 202 jobs are `UNTRACKED`, not success
- New OpenAPI ops: escape hatches work via `api`/`call`; curated verbs need a coverage-map entry (`make coverage`)
- After `netcup spec update`, run `make generate`
- Do not commit credentials or config under `~/.config/netcup`
- MCP: prefer curated tools; full coverage via `netcup_call` / `netcup_cli`; never expose `tui` or `auth login` via MCP
- After editing `.codex-plugin` / `.claude-plugin` / `.cursor-plugin` / `skills/netcup`, run `make sync-plugin-assets`
