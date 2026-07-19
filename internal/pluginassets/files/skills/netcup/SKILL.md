---
name: netcup
description: >-
  Operate netcup SCP (Server Control Panel) via thin MCP tools wrapping the
  netcup CLI — servers, power, tasks, snapshots, ISO, firewall, plus
  netcup_call / netcup_cli for full CLI coverage. Use when the user mentions
  netcup, SCP, VPS power, snapshots, ISO attach, or server firewall.
---

# netcup

Humans use the `netcup` CLI / TUI. Agents use **MCP tools** from `netcup mcp`. Every tool shells out to `netcup … --format json -q` (plus `-y` when `confirm=true`). Same JSON envelopes as the CLI.

Prereqs: `netcup` on `PATH` (or `NETCUP_BIN`) and a logged-in profile (`netcup auth login` — interactive; **never** via MCP).

Host install/refresh (human): `netcup install-mcp` (`--scope user|project|local`, `--host claude|cursor|codex`). Uses a checkout when present; otherwise materializes plugin files into `~/.config/netcup/plugin`.

## Routing

1. **Prefer curated tools** (typed wrappers below).
2. **Unknown OpenAPI op** → `netcup_endpoints` / `netcup_describe` → `netcup_call`.
3. **Known CLI subcommand outside curated set** → `netcup_cli` with argv (no binary name).
4. **Raw HTTP last** → `netcup_api` (non-GET needs `confirm=true`).

## Curated tools → CLI

| MCP tool | Behind it (`netcup …`) | Notes |
|----------|------------------------|-------|
| `netcup_ping` | `ping` | |
| `netcup_whoami` | `auth whoami` | |
| `netcup_auth_status` | `auth status` | |
| `netcup_servers_list` | `servers list` | optional `q`/`name`/`ip`/`limit`/`offset` |
| `netcup_servers_get` | `servers get -s <server>` | |
| `netcup_servers_power` | `servers <action> -s <server> -y` | `action`: start\|stop\|reboot\|poweroff\|reset\|suspend; **`confirm=true`** |
| `netcup_tasks_list` | `tasks list` | optional `server`/`limit`/`offset` |
| `netcup_tasks_get` | `tasks get <uuid>` | |
| `netcup_tasks_cancel` | `tasks cancel <uuid> -y` | **`confirm=true`** |
| `netcup_snapshots_list` | `snapshots list -s <server>` | other snapshot ops → `netcup_cli` |
| `netcup_iso_get` | `iso get -s <server>` | |
| `netcup_iso_available` | `iso available -s <server>` | |
| `netcup_iso_attach` | `iso attach -s <server> --iso-id\|--user-iso … -y` | exactly one of `iso_id` / `user_iso`; optional `boot_cdrom`; **`confirm=true`** |
| `netcup_iso_detach` | `iso detach -s <server> -y` | **`confirm=true`** |
| `netcup_firewall_get` | `firewall get <mac> -s <server>` | other firewall ops → `netcup_cli` |

## Coverage tools → CLI

| MCP tool | Behind it | Notes |
|----------|-----------|-------|
| `netcup_endpoints` | `endpoints [--filter] [--tag]` | discover OpenAPI ops |
| `netcup_describe` | `describe <path_or_operation>` | inspect one op |
| `netcup_call` | `call <hint> [--path k=v]… [--query]… [--body] [--dry-run] [-s]` | primary escape hatch; writes need **`confirm=true`** |
| `netcup_api` | `api <METHOD> <PATH> [--query]… [--header]… [--body]` | non-GET needs **`confirm=true`** |
| `netcup_cli` | `<args…>` with optional `-s <server>` | allowlisted top-level only; writes often need **`confirm=true`** |

### `netcup_cli` allowlist (full ops surface)

Pass `args` as the CLI after `netcup`, e.g. `["disks","list"]` + `server="899327"`.

| Top-level | Subcommands / usage (via `netcup_cli`) |
|-----------|----------------------------------------|
| `servers` | `list`, `get`, `start`…`suspend`, `set …`, `rescue …`, `logs`, `guest-agent`, `gpu-driver`, `storage-optimize` |
| `tasks` | `list`, `get`, `cancel` |
| `snapshots` | `list`, `get`, `create`, `dry-run`, `delete`, `export`, `revert` |
| `iso` | `get`, `available`, `attach`, `detach` |
| `isos` | `list`, `upload`, `delete`, `download-url`, `prepare-upload`, `part-url`, `complete-upload` |
| `images` | `list`, `flavours`, `upload`, `delete`, `setup`, `setup-user`, multipart helpers |
| `disks` | `list`, `get`, `drivers`, `set-driver`, `format` |
| `nics` | `list`, `get`, `create`, `update`, `delete` |
| `firewall` | `get`, `set`, `reapply`, `restore-copied` |
| `firewall-policies` | `list`, `get`, `create`, `update`, `delete` |
| `failover` | `ipv4\|ipv6` → `list`, `route` |
| `rdns` | `ipv4\|ipv6` → `get`, `set`, `delete` |
| `vlans` | `list`, `get`, `get-global`, `update` |
| `metrics` | `cpu`, `disk`, `network`, `packets` |
| `maintenance` | (no subcommand) |
| `users` | `get`, `logs`, `update` |
| `ssh-keys` | `list`, `add`, `delete` |
| `cache` | `stats`, `clear` |
| `spec` | `show`, `update`, `mcp` |
| `ping` | |
| `auth` | **only** `status`, `whoami`, `refresh`, `logout` |
| `call` / `api` / `endpoints` / `describe` | same as the dedicated coverage tools |

Also auto-injected: `--format json`, `-q`; `confirm=true` → `-y`; optional `server` → `-s`.

### Not available via MCP

| CLI | Why |
|-----|-----|
| `tui` / bare interactive `netcup` | TTY UI |
| `auth login` | device-code / browser — human only |
| `completion` | shell install helper |
| `update` | upgrades the CLI binary |
| `help` | use tool docs / `netcup_describe` |
| secret flags (`--password`, `--password-file`, tokens, …) | never on argv via MCP |

For `servers set root-password`, tell the user to run the CLI on a TTY with `--password-file`.

## Safety

- Parse JSON; do not invent fields.
- Destructive ops need **explicit user intent** + `confirm=true`.
- Prefer server **id** when known.
- Auth errors → user runs `netcup auth login` / `auth refresh` locally.

## Examples

```text
netcup_servers_list()
netcup_servers_get(server="899327")
netcup_servers_power(action="reboot", server="899327", confirm=true)  # user intent required
netcup_cli(args=["disks", "list"], server="899327")
netcup_cli(args=["snapshots", "create", "--name", "pre-change"], server="899327", confirm=true)
netcup_endpoints(filter="snapshot")
netcup_call(operation_hint="snapshots list", path=["serverId=899327"], dry_run=true)
```
