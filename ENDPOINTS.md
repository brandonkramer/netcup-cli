# SCP API endpoint inventory

Generated from pinned `openapi.json` — **SCP (Server Control Panel) REST API** `2026.0703.095128`.

**Total operations:** 87

## Miscellaneous

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `GET` | `/api/ping` |  | Check if application is available |
| `GET` | `/api/v1/maintenance` |  | Get maintenance information for system |
| `GET` | `/api/v1/openapi` |  | Get openapi spec |
| `POST` | `/api/v1/openapi/mcp` |  | SCP OpenAPI Spec MCP Server |

## Server Disks

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `PATCH` | `/api/v1/servers/{serverId}/disks` | 202 | Patch disk driver of a server. |
| `GET` | `/api/v1/servers/{serverId}/disks` |  | Get disks of a server |
| `GET` | `/api/v1/servers/{serverId}/disks/supported-drivers` |  | Get a list of supported storage drivers for this server |
| `GET` | `/api/v1/servers/{serverId}/disks/{diskName}` |  | Get a disk of a server |
| `POST` | `/api/v1/servers/{serverId}/disks/{diskName}:format` | 202 | Format disk of a server. Attention: All data will be lost during formatting! |

## Server Firewalls

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `PUT` | `/api/v1/servers/{serverId}/interfaces/{mac}/firewall` | 202 | Configure firewall |
| `GET` | `/api/v1/servers/{serverId}/interfaces/{mac}/firewall` |  | Get firewall |
| `POST` | `/api/v1/servers/{serverId}/interfaces/{mac}/firewall:reapply` | 202 | Reapply firewall. Necessary if policy update timed out due to long running write operation on server (f.e. storage optimization) |
| `POST` | `/api/v1/servers/{serverId}/interfaces/{mac}/firewall:restore-copied-policies` | 202 | Restore copied firewall policies. |
| `POST` | `/api/v1/users/{userId}/firewall-policies` |  | Create firewall policy |
| `GET` | `/api/v1/users/{userId}/firewall-policies` |  | Get firewall policies |
| `PUT` | `/api/v1/users/{userId}/firewall-policies/{id}` | 202 | Update firewall policy |
| `DELETE` | `/api/v1/users/{userId}/firewall-policies/{id}` |  | Delete firewall policy |
| `GET` | `/api/v1/users/{userId}/firewall-policies/{id}` |  | Get firewall policy |

## Server ISO

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `POST` | `/api/v1/servers/{serverId}/iso` | 202 | Attach an ISO to a server. |
| `DELETE` | `/api/v1/servers/{serverId}/iso` |  | Detach an ISO from a server. |
| `GET` | `/api/v1/servers/{serverId}/iso` |  | Get attached ISO of a server. |
| `GET` | `/api/v1/servers/{serverId}/isoimages` |  | Get available ISO images for server |
| `GET` | `/api/v1/users/{userId}/isos` |  | Get all available ISOs |
| `POST` | `/api/v1/users/{userId}/isos/{key}` |  | Prepares an upload for an ISO |
| `DELETE` | `/api/v1/users/{userId}/isos/{key}` |  | Delete an ISO |
| `GET` | `/api/v1/users/{userId}/isos/{key}` |  | Get presigned URL for an ISO |
| `PUT` | `/api/v1/users/{userId}/isos/{key}/{uploadId}` |  | Completes a multipart upload for an ISO |
| `GET` | `/api/v1/users/{userId}/isos/{key}/{uploadId}/parts/{partNumber}` |  | Get a presigned upload URL for a single part |

## Server Images

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `POST` | `/api/v1/servers/{serverId}/image` | 202 | Setup image for a server. Attention: All data will be lost during formatting on selected disk! |
| `GET` | `/api/v1/servers/{serverId}/imageflavours` |  | Get available image flavours for server image setup. Images whose storage driver is not supported by the server's machine type are not shown. |
| `POST` | `/api/v1/servers/{serverId}/user-image` | 202 | Setup user image for a server. |
| `GET` | `/api/v1/users/{userId}/images` |  | Get all available user images |
| `POST` | `/api/v1/users/{userId}/images/{key}` |  | Prepares an upload for an image |
| `DELETE` | `/api/v1/users/{userId}/images/{key}` |  | Delete an image |
| `GET` | `/api/v1/users/{userId}/images/{key}` |  | Get download informations for an image |
| `PUT` | `/api/v1/users/{userId}/images/{key}/{uploadId}` |  | Completes a multipart upload for an image |
| `GET` | `/api/v1/users/{userId}/images/{key}/{uploadId}/parts/{partNumber}` |  | Get a presigned upload URL for a single part |

## Server Metrics

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `GET` | `/api/v1/servers/{serverId}/metrics/cpu` |  | Get CPU metrics of a server. |
| `GET` | `/api/v1/servers/{serverId}/metrics/disk` |  | Get disk metrics of a server. |
| `GET` | `/api/v1/servers/{serverId}/metrics/network` |  | Get network metrics of a server. |
| `GET` | `/api/v1/servers/{serverId}/metrics/network/packet` |  | Get network packet metrics of a server. |

## Server Networking

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `POST` | `/api/v1/rdns/ipv4` |  | Set an rDNS entry for an IPv4. |
| `DELETE` | `/api/v1/rdns/ipv4/{ip}` |  | Delete an rDNS entry of an IPv4. |
| `GET` | `/api/v1/rdns/ipv4/{ip}` |  | Get rDNS for an IPv4. |
| `POST` | `/api/v1/rdns/ipv6` |  | Set an rDNS entry for an IPv6. |
| `DELETE` | `/api/v1/rdns/ipv6/{ip}` |  | Delete an rDNS entry of an IPv6. |
| `GET` | `/api/v1/rdns/ipv6/{ip}` |  | Get rDNS for an IPv6. |
| `POST` | `/api/v1/servers/{serverId}/interfaces` | 202 | Create an interface in a server. |
| `GET` | `/api/v1/servers/{serverId}/interfaces` |  | Get all interfaces and IPs of a server including routed IPs and rDNS entries. |
| `PUT` | `/api/v1/servers/{serverId}/interfaces/{mac}` | 202 | Update interface attributes. |
| `DELETE` | `/api/v1/servers/{serverId}/interfaces/{mac}` | 202 |  |
| `GET` | `/api/v1/servers/{serverId}/interfaces/{mac}` |  | Get an interface and IPs of a server including routed IPs and rDNS entries. |
| `GET` | `/api/v1/users/{userId}/failoverips/v4` |  | Get all failover IPv4s of this user. |
| `PATCH` | `/api/v1/users/{userId}/failoverips/v4/{id}` | 202 | Route a failover IPv4. |
| `GET` | `/api/v1/users/{userId}/failoverips/v6` |  | Get all failover IPv6s of this user. |
| `PATCH` | `/api/v1/users/{userId}/failoverips/v6/{id}` | 202 | Route a failover IPv6. |
| `GET` | `/api/v1/users/{userId}/vlans` |  | Get VLans of a user |
| `PUT` | `/api/v1/users/{userId}/vlans/{vlanId}` | 202 | Update a VLan |
| `GET` | `/api/v1/users/{userId}/vlans/{vlanId}` |  | Get a VLan of a user |
| `GET` | `/api/v1/vlans/{vlanId}` |  | Get a VLan |

## Server Snapshots

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `POST` | `/api/v1/servers/{serverId}/snapshots` | 202 | Create a snapshot |
| `GET` | `/api/v1/servers/{serverId}/snapshots` |  | Get all snapshots of a server. |
| `DELETE` | `/api/v1/servers/{serverId}/snapshots/{name}` | 202 | Delete a snapshot of a server |
| `GET` | `/api/v1/servers/{serverId}/snapshots/{name}` |  | Get a snapshot of a server. |
| `POST` | `/api/v1/servers/{serverId}/snapshots/{name}/export` | 202 | Export a snapshot of a server |
| `POST` | `/api/v1/servers/{serverId}/snapshots/{name}/revert` | 202 | Revert a snapshot of a server |
| `POST` | `/api/v1/servers/{serverId}/snapshots:dryrun` |  | Check if creating a snapshot is possible. |

## Servers

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `GET` | `/api/v1/servers` |  | Get servers |
| `PATCH` | `/api/v1/servers/{serverId}` | 202 | Start - stop server or update attributes like hostname, nickname, uefi, bootorder, ... |
| `GET` | `/api/v1/servers/{serverId}` |  | Get one server |
| `GET` | `/api/v1/servers/{serverId}/gpu-driver` |  | Generate presigned download URL for GPU driver if available |
| `GET` | `/api/v1/servers/{serverId}/guest-agent` |  | Get guest agent data for server |
| `GET` | `/api/v1/servers/{serverId}/guest-agent/status` |  | Get guest agent status for server |
| `GET` | `/api/v1/servers/{serverId}/logs` |  | Get server logs  |
| `POST` | `/api/v1/servers/{serverId}/rescuesystem` | 202 | Activate rescue system for a server. |
| `DELETE` | `/api/v1/servers/{serverId}/rescuesystem` | 202 | Deactivate rescue system for a server. |
| `GET` | `/api/v1/servers/{serverId}/rescuesystem` |  | Get rescue system status for a server. |
| `POST` | `/api/v1/servers/{serverId}/storageoptimization` | 202 | Optimize storage of a server. |

## Tasks

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `GET` | `/api/v1/tasks` |  | Get all tasks |
| `GET` | `/api/v1/tasks/{uuid}` |  | Get one task |
| `PUT` | `/api/v1/tasks/{uuid}:cancel` | 202 | Cancel a running task |

## Users

| Method | Path | Async | Summary |
|--------|------|-------|---------|
| `PUT` | `/api/v1/users/{userId}` |  | Update a user |
| `GET` | `/api/v1/users/{userId}` |  | Get one user |
| `GET` | `/api/v1/users/{userId}/logs` |  | Get user logs  |
| `POST` | `/api/v1/users/{userId}/ssh-keys` |  | Create SSH key |
| `GET` | `/api/v1/users/{userId}/ssh-keys` |  | Get SSH keys |
| `DELETE` | `/api/v1/users/{userId}/ssh-keys/{id}` |  | Delete SSH key |

