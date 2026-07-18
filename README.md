# netcup

Go CLI for the netcup **SCP (Server Control Panel) REST API**.

Not affiliated with netcup. CCP billing/orders are out of scope.

## Install

```bash
go build -o bin/netcup ./cmd/netcup
# optional: install onto PATH
cp bin/netcup "$(go env GOPATH)/bin/netcup"
```

Shell completions:

```bash
netcup completion zsh > "${fpath[1]}/_netcup"   # zsh example
netcup completion bash|fish|powershell
```

## Quick start

```bash
netcup auth login
netcup ping
netcup servers list
netcup servers get -s <id|name|nickname|ip>
netcup servers start -s <selector>
```

On an interactive TTY, bare `netcup` (no subcommand) opens the ops TUI (servers, tasks, metrics, media). Same via `netcup tui`. Non-TTY / piped use still shows help; scripting stays on subcommands.

Config and credentials live under `~/.config/netcup` (override with `--config-dir` / `NETCUP_CONFIG_DIR`). Profiles use `--profile` / `NETCUP_PROFILE` (default `default`).

## Global flags

| Flag / env | Meaning |
|------------|---------|
| `--format` / `NETCUP_FORMAT` | `table` (TTY default), `json`, `jsonl`, `yaml`, `brief` |
| `--json` | alias for `--format json` |
| `-y` / `--yes` | skip destructive confirmations |
| `--no-wait` | return TaskInfo immediately on HTTP 202 |
| `--wait-timeout` | default `30m` |
| `--poll-interval` | default `2s` |
| `--no-cache` | disable read cache |
| `--cache-ttl` | override cache TTL for this process |
| `--profile` / `NETCUP_PROFILE` | credential profile |
| `--user-id` | override SCP userId (from JWT `sub` by default) |
| `-s` / `--server` | server selector: id, name, nickname, or IPv4 |
| `-q` / `--quiet` | suppress non-essential stderr |
| `-v` / `--verbose` | verbose logging (tokens redacted) |
| `--full` | include full API objects in curated views |
| `--config-dir` / `NETCUP_CONFIG_DIR` | default `~/.config/netcup` |
| `NETCUP_BASE_URL` | default `https://www.servercontrolpanel.de` |
| `NO_COLOR` | disable color |

Non-TTY stdout defaults to `--format json`. Agents should parse the JSON envelope on stdout.

Destructive commands refuse non-interactive use without `-y` (exit `2`).

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API / task error |
| 2 | Usage / confirmation required |
| 3 | Auth required or refresh failed |
| 4 | Task wait timeout |
| 5 | Not found |
| 130 | Interrupted (SIGINT) |

---

## Commands

### Auth

```text
netcup auth login [--no-browser] [--no-save]
netcup auth logout [--revoke]
netcup auth status
netcup auth whoami
netcup auth refresh
```

Device-code OIDC login (`client_id=scp`, scopes `openid offline_access`). Refresh token stored at `credentials-<profile>.json` (mode `0600`).

### Misc

```text
netcup                  # interactive TUI (TTY only); alias: netcup tui
netcup ping
netcup maintenance
netcup cache stats
netcup cache clear [prefix]
netcup spec show
netcup spec update          # writes openapi.json; then: make generate
netcup spec mcp             # docs MCP proxy
netcup completion bash|zsh|fish|powershell
```

**TUI:** tabs `1` Servers · `2` Tasks · `3` Metrics · `4` Media (or `tab`).

| Tab | Keys |
|-----|------|
| Servers | `j/k` · `space` multi-select · `/` filter · `enter` detail · `s/t/r` start/stop/reboot · `P` poweroff · `x` reset · `u` suspend · `h`/`n` edit hostname/nickname · `g` guest agent · `e`/`d` rescue enable/disable · `R` refresh · `a` clear |
| Tasks | `c` cancel · `R` refresh |
| Metrics | `c/d/n/p` cpu/disk/network/packets · `h` cycle hours · `R` refresh |
| Media | `enter` attach (+boot CDROM) · `D` detach · `R` refresh |

Parallel jobs show in the bottom bar. `q` quits.

### Servers

```text
netcup servers list [--q] [--name] [--ip] [--sort] [--limit] [--offset] [--firewall-policy-id]
netcup servers get [selector]
netcup servers start|stop|poweroff|reboot|reset|suspend [selector]
netcup servers storage-optimize [--disk NAME]… [--start-after] [selector]
netcup servers gpu-driver [selector]
netcup servers guest-agent [selector]
netcup servers guest-agent status [selector]
netcup servers logs [--limit] [--offset] [selector]
netcup servers rescue status|enable|disable [selector]

netcup servers set hostname <value> [selector]
netcup servers set nickname <value> [selector]
netcup servers set uefi <true|false> [selector]
netcup servers set bootorder <CDROM,HDD,NETWORK> [selector]
netcup servers set root-password <value> [selector]
netcup servers set autostart <true|false> [selector]
netcup servers set os-optimization <LINUX|WINDOWS|BSD|LINUX_LEGACY|UNKNOWN> [selector]
netcup servers set cpu-topology <sockets>,<coresPerSocket> [selector]
netcup servers set keyboard-layout <value> [selector]
```

Server selector (`-s` or positional): numeric **id**, **name**, **nickname**, or **IPv4**. Prefer id in scripts.

Power mutations and many other writes return HTTP 202; the CLI waits for the task by default.

### Disks

```text
netcup disks list [selector]
netcup disks get <diskName> [selector]
netcup disks drivers [selector]
netcup disks set-driver <driver> [selector]
netcup disks format <diskName> [selector]    # destructive
```

### Snapshots

```text
netcup snapshots list [selector]
netcup snapshots get <name> [selector]
netcup snapshots create --name <name> [selector]
netcup snapshots dry-run [selector]
netcup snapshots delete <name> [selector]    # destructive
netcup snapshots export <name> [selector]
netcup snapshots revert <name> [selector]    # destructive
```

### ISO (attached) & ISOs (library)

```text
netcup iso get|detach [selector]
netcup iso attach [--iso-id N] [--user-iso NAME] [--boot-cdrom] [selector]
netcup iso available [selector]

netcup isos list
netcup isos upload <file> [--key NAME] [--part-size BYTES]
netcup isos delete <key>                     # destructive
netcup isos download-url <key>
netcup isos prepare-upload <key> [--multipart]
netcup isos part-url <key> <uploadId> <partNumber>
netcup isos complete-upload <key> <uploadId> --body '[{...}]'
```

### Images

```text
netcup images flavours [selector]
netcup images list
netcup images setup [selector] --flavour-id N [flags]   # destructive
netcup images setup-user [selector] --name <key>
netcup images upload <file> [--key NAME]
netcup images delete <key>                   # destructive
netcup images download-url <key>
netcup images prepare-upload|part-url|complete-upload …
```

`images setup` flags include `--hostname`, `--disk`, `--locale`, `--timezone`, `--user`, `--password`, `--ssh-key-id`, `--ssh-password-auth`, `--root-full-disk`, `--email`, `--custom-script`, or `--body @file.json`.

### Networking

```text
netcup nics list|get <mac>|delete <mac> [selector]
netcup nics create --vlan-id N [--driver VIRTIO] [selector]
netcup nics update <mac> --driver <DRIVER> [selector]

netcup rdns ipv4 get|set|delete …
netcup rdns ipv6 get|set|delete …

netcup failover ipv4 list
netcup failover ipv4 route <id> <serverId>
netcup failover ipv6 list
netcup failover ipv6 route <id> <serverId>

netcup vlans list|get <vlanId>|get-global <vlanId>
netcup vlans update <vlanId> --name <name>
```

### Firewall

```text
netcup firewall get <mac> [selector]
netcup firewall set <mac> [selector] [--active] [--user-policy ID]… [--copied-policy ID]… [--body]
netcup firewall reapply <mac> [selector]
netcup firewall restore-copied <mac> [selector]

netcup firewall-policies list|get <id>|delete <id>
netcup firewall-policies create --name NAME [--description] [--body]
netcup firewall-policies update <id> --name NAME [--description] [--body]
```

### Metrics, tasks, users, SSH keys

```text
netcup metrics cpu|disk|network|packets [--hours N] [selector]   # never cached

netcup tasks list [--q] [--state] [--server-id] [--limit] [--offset]
netcup tasks get <uuid>
netcup tasks cancel <uuid>

netcup users get
netcup users logs [--limit] [--offset]
netcup users update --language … --timezone … [flags] | --body

netcup ssh-keys list
netcup ssh-keys add --name NAME --key '…' | --key-file PATH
netcup ssh-keys delete <id>
```

### Escape hatches (full API)

Every OpenAPI operation is reachable even without a curated verb:

```text
netcup api METHOD PATH [--query k=v]… [--header 'Name: value']… [--body JSON|@file|-]
netcup call OPERATION_HINT [--path k=v]… [--query k=v]… [--body] [--dry-run]
netcup endpoints [--filter TEXT] [--tag TAG]
netcup describe PATH_OR_OPERATION
```

Examples:

```bash
netcup call "GET /api/v1/servers" --dry-run
netcup call "snapshot create" --path serverId=123 --body '{"name":"x"}' -y
netcup api GET /api/v1/servers --query q=web
netcup endpoints --filter snapshot
netcup describe "iso attach"
```

`call` resolves against the pinned/local OpenAPI (path, `METHOD path`, or text hint). `--server` can fill `{serverId}`; `{userId}` is filled from the token when needed.

---

## Common workflows

**Power cycle**

```bash
netcup servers stop -s myvps
netcup servers start -s myvps
# hard poweroff / reset require confirmation or -y
netcup servers poweroff -s myvps -y
```

**Snapshot before risk**

```bash
netcup snapshots dry-run -s myvps
netcup snapshots create --name before-change -s myvps
```

**Attach Windows ISO**

```bash
netcup isos upload ./Win11.iso --key win11.iso
netcup iso attach --user-iso win11.iso --boot-cdrom -s myvps
netcup servers reboot -s myvps
```

**Reimage**

```bash
netcup images flavours -s myvps
netcup images setup --flavour-id <id> --hostname myhost -s myvps -y
```

**Agent / CI**

```bash
netcup servers list --format json
netcup servers start -s 123456 -y --format json
# non-TTY already defaults to json
```

**Rich writes with `--body`**

Curated flags cover common fields; for full request payloads use `--body` (inline JSON, `@file`, or `-` for stdin):

```bash
netcup firewall set aa:bb:cc:dd:ee:ff -s myvps --body @rules.json -y
netcup users update --body '{"language":"en","timezone":"Europe/Berlin"}'
netcup call "firewall-policies create" --body @policy.json -y
```

---

## Develop

```bash
make generate   # regenerate client from openapi.json
make build
make coverage   # assert all OpenAPI ops are mapped
go test ./...
```

Pinned OpenAPI: [`openapi.json`](./openapi.json).

### Keeping up with API drift

After `netcup spec update` (refreshes `openapi.json`), run `make generate` so the typed client matches the new spec.

New or changed operations work immediately via `api` / `call` / `endpoints` / `describe`. Curated verbs are added separately; until then, use the escape hatches. `make coverage` / `TestCoverageGate` fails if the coverage map drifts after regenerate (new ops need a map entry — curated command or `api`/`call`).
