# AGENTS.md

**netcup-cli** — Go CLI + Bubble Tea TUI for the [netcup](https://www.netcup.com/) SCP REST API. Module: `github.com/brandonkramer/netcup-cli`. Binary: `netcup` (`./cmd/netcup`).

User docs: [README.md](./README.md). Release notes: [CHANGELOG.md](./CHANGELOG.md).

## Layout

- `cmd/netcup` — entrypoint
- `internal/cmd` — Cobra commands, coverage map, `netcup update`
- `internal/tui` — interactive ops TUI
- `internal/scpclient` — generated OpenAPI client (`make generate` from `openapi.json`)
- `internal/auth`, `config`, `cache`, `wait`, `output`, `patch` — session, config, caching, task wait, envelopes

## Commands

```bash
make build                 # bin/netcup
make test                  # go test ./...
make coverage              # OpenAPI ops mapped in internal/cmd
make generate              # regenerate scpclient from openapi.json
go test -count=1 ./...
```

Prefer typecheck/test over long-running builds unless asked. Do not start a TUI/dev server unless requested.

## Rules

- Keep changes surgical; match existing Go style; avoid `any`-equivalents and drive-by refactors
- Secrets never on argv — use `--password-file` / `-` or a TTY prompt
- Nil-check API responses before `StatusCode()`; body-less HTTP 202 jobs are `UNTRACKED`, not success
- New OpenAPI ops: escape hatches work via `api`/`call`; curated verbs need a coverage-map entry (`make coverage`)
- After `netcup spec update`, run `make generate`
- Do not commit credentials or config under `~/.config/netcup`

## Releases

1. Move `[Unreleased]` notes into a dated section in `CHANGELOG.md`, update compare links, commit
2. Annotated tag `vX.Y.Z` and push — `.github/workflows/release.yml` runs GoReleaser
3. Bump `brandonkramer/homebrew-tap` `Formula/netcup.rb` URLs + sha256 from `checksums.txt`
4. Verify with `brew upgrade brandonkramer/tap/netcup` and `netcup --version`

Details: README → Develop → Releases. Never force-push `main` or retag an existing release.
