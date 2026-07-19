#!/usr/bin/env python3
"""Thin FastMCP facade over the netcup CLI — curated tools + call/api/cli coverage."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from typing import Any

from mcp.server.fastmcp import FastMCP

mcp = FastMCP("netcup")

CLI_ALLOWLIST = frozenset(
    {
        "api",
        "auth",
        "cache",
        "call",
        "describe",
        "disks",
        "endpoints",
        "failover",
        "firewall",
        "firewall-policies",
        "images",
        "iso",
        "isos",
        "maintenance",
        "metrics",
        "nics",
        "ping",
        "rdns",
        "servers",
        "snapshots",
        "spec",
        "ssh-keys",
        "tasks",
        "users",
        "vlans",
    }
)

AUTH_ALLOW = frozenset({"status", "whoami", "refresh", "logout"})
DENY_TOP = frozenset({"tui", "completion", "update", "help"})
SECRET_FLAGS = frozenset(
    {
        "--password",
        "--password-file",
        "--token",
        "--access-token",
        "--refresh-token",
        "--client-secret",
    }
)

POWER_ACTIONS = frozenset(
    {"start", "stop", "reboot", "poweroff", "reset", "suspend"}
)


class NetcupError(Exception):
    """CLI failed; message is returned to the MCP client."""


def _netcup_bin() -> str:
    override = os.environ.get("NETCUP_BIN", "").strip()
    if override:
        return override
    found = shutil.which("netcup")
    if found:
        return found
    raise NetcupError(
        "netcup binary not found on PATH; install via Homebrew/go install "
        "or set NETCUP_BIN"
    )


def _looks_secret_flag(arg: str) -> bool:
    base = arg.split("=", 1)[0]
    return base in SECRET_FLAGS


def _ensure_json_format(argv: list[str]) -> list[str]:
    out = list(argv)
    if "--json" in out or any(a == "--format" or a.startswith("--format=") for a in out):
        return out
    return ["--format", "json", *out]


def _run(
    argv: list[str],
    *,
    confirm: bool = False,
    timeout: float = 120,
) -> dict[str, Any] | str:
    if any(_looks_secret_flag(a) for a in argv):
        raise NetcupError(
            "secret-bearing flags are not allowed via MCP; use "
            "netcup auth login or --password-file on a human TTY"
        )

    cmd = [_netcup_bin(), *_ensure_json_format(argv), "-q"]
    if confirm and "-y" not in cmd and "--yes" not in cmd:
        cmd.append("-y")

    try:
        proc = subprocess.run(
            cmd,
            text=True,
            capture_output=True,
            check=False,
            timeout=timeout,
            stdin=subprocess.DEVNULL,
        )
    except subprocess.TimeoutExpired as exc:
        raise NetcupError(f"timed out after {timeout:g}s") from exc
    except FileNotFoundError as exc:
        raise NetcupError(str(exc)) from exc

    stdout = (proc.stdout or "").strip()
    stderr = (proc.stderr or "").strip()

    if proc.returncode != 0:
        parts = [f"netcup exited {proc.returncode}"]
        if stdout:
            parts.append(stdout)
        if stderr:
            parts.append(f"[stderr]\n{stderr}")
        raise NetcupError("\n".join(parts))

    if not stdout:
        return {"ok": True, "stdout": "", "stderr": stderr}

    try:
        return json.loads(stdout)
    except json.JSONDecodeError:
        return stdout


def _tool_result(value: dict[str, Any] | str) -> str:
    if isinstance(value, str):
        return value
    return json.dumps(value, indent=2, ensure_ascii=False)


def _with_server(argv: list[str], server: str | None) -> list[str]:
    if server:
        return [*argv, "-s", server]
    return argv


def _guard_cli_args(args: list[str]) -> None:
    if not args:
        raise NetcupError("args must be non-empty (e.g. ['servers', 'list'])")
    top = args[0]
    if top in DENY_TOP or top == "tui":
        raise NetcupError(f"command '{top}' is not allowed via MCP")
    if top not in CLI_ALLOWLIST:
        raise NetcupError(
            f"command '{top}' is not allowlisted; use curated tools or "
            f"netcup_call / netcup_api"
        )
    if top == "auth":
        if len(args) < 2 or args[1] not in AUTH_ALLOW:
            raise NetcupError(
                "auth via MCP allows only status|whoami|refresh|logout; "
                "run `netcup auth login` yourself"
            )
    if any(a in {"tui", "completion"} for a in args):
        raise NetcupError("tui/completion are not allowed via MCP")


# --- Layer A: curated -------------------------------------------------------


@mcp.tool()
def netcup_ping() -> str:
    """Check SCP API availability (netcup ping)."""
    try:
        return _tool_result(_run(["ping"]))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_whoami() -> str:
    """GET current user profile (netcup auth whoami)."""
    try:
        return _tool_result(_run(["auth", "whoami"]))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_auth_status() -> str:
    """Show local login status (netcup auth status)."""
    try:
        return _tool_result(_run(["auth", "status"]))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_servers_list(
    q: str | None = None,
    name: str | None = None,
    ip: str | None = None,
    limit: int | None = None,
    offset: int | None = None,
) -> str:
    """List servers (netcup servers list)."""
    argv = ["servers", "list"]
    if q:
        argv.extend(["--q", q])
    if name:
        argv.extend(["--name", name])
    if ip:
        argv.extend(["--ip", ip])
    if limit is not None:
        argv.extend(["--limit", str(limit)])
    if offset is not None:
        argv.extend(["--offset", str(offset)])
    try:
        return _tool_result(_run(argv))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_servers_get(server: str) -> str:
    """Get one server by id|name|nickname|ip (netcup servers get)."""
    try:
        return _tool_result(_run(_with_server(["servers", "get"], server)))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_servers_power(action: str, server: str, confirm: bool = False) -> str:
    """Server power action: start|stop|reboot|poweroff|reset|suspend. Requires confirm=true."""
    action = action.strip().lower()
    if action not in POWER_ACTIONS:
        return f"error: action must be one of {sorted(POWER_ACTIONS)}"
    if not confirm:
        return "error: destructive power action requires confirm=true"
    try:
        return _tool_result(
            _run(_with_server(["servers", action], server), confirm=True, timeout=600)
        )
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_tasks_list(
    server: str | None = None,
    limit: int | None = None,
    offset: int | None = None,
) -> str:
    """List async tasks (netcup tasks list)."""
    argv = ["tasks", "list"]
    if limit is not None:
        argv.extend(["--limit", str(limit)])
    if offset is not None:
        argv.extend(["--offset", str(offset)])
    try:
        return _tool_result(_run(_with_server(argv, server)))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_tasks_get(uuid: str) -> str:
    """Get one task by UUID (netcup tasks get)."""
    try:
        return _tool_result(_run(["tasks", "get", uuid]))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_tasks_cancel(uuid: str, confirm: bool = False) -> str:
    """Cancel a task (netcup tasks cancel). Requires confirm=true."""
    if not confirm:
        return "error: cancel requires confirm=true"
    try:
        return _tool_result(_run(["tasks", "cancel", uuid], confirm=True))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_snapshots_list(server: str) -> str:
    """List snapshots for a server (netcup snapshots list)."""
    try:
        return _tool_result(_run(_with_server(["snapshots", "list"], server)))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_iso_get(server: str) -> str:
    """Show attached ISO (netcup iso get)."""
    try:
        return _tool_result(_run(_with_server(["iso", "get"], server)))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_iso_available(server: str) -> str:
    """List ISOs available to attach (netcup iso available)."""
    try:
        return _tool_result(_run(_with_server(["iso", "available"], server)))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_iso_attach(
    server: str,
    confirm: bool = False,
    iso_id: int | None = None,
    user_iso: str | None = None,
    boot_cdrom: bool = False,
) -> str:
    """Attach an ISO (netcup iso attach). Pass iso_id (catalog) or user_iso (name). Requires confirm=true."""
    if not confirm:
        return "error: attach requires confirm=true"
    if (iso_id is None) == (not user_iso):
        return "error: provide exactly one of iso_id or user_iso"
    argv = ["iso", "attach"]
    if iso_id is not None:
        argv.extend(["--iso-id", str(iso_id)])
    if user_iso:
        argv.extend(["--user-iso", user_iso])
    if boot_cdrom:
        argv.append("--boot-cdrom")
    try:
        return _tool_result(
            _run(_with_server(argv, server), confirm=True, timeout=600)
        )
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_iso_detach(server: str, confirm: bool = False) -> str:
    """Detach ISO from a server (netcup iso detach). Requires confirm=true."""
    if not confirm:
        return "error: detach requires confirm=true"
    try:
        return _tool_result(
            _run(_with_server(["iso", "detach"], server), confirm=True, timeout=600)
        )
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_firewall_get(mac: str, server: str) -> str:
    """Get interface firewall for a MAC (netcup firewall get)."""
    try:
        return _tool_result(_run(_with_server(["firewall", "get", mac], server)))
    except NetcupError as exc:
        return f"error: {exc}"


# --- Layer B: discovery + full coverage -------------------------------------


@mcp.tool()
def netcup_endpoints(filter: str | None = None, tag: str | None = None) -> str:
    """List OpenAPI operations (netcup endpoints). Use before netcup_call."""
    argv = ["endpoints"]
    if filter:
        argv.extend(["--filter", filter])
    if tag:
        argv.extend(["--tag", tag])
    try:
        return _tool_result(_run(argv))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_describe(path_or_operation: str) -> str:
    """Show OpenAPI operation or path item (netcup describe)."""
    try:
        return _tool_result(_run(["describe", path_or_operation]))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_call(
    operation_hint: str,
    path: list[str] | None = None,
    query: list[str] | None = None,
    body: str | None = None,
    dry_run: bool = False,
    confirm: bool = False,
    server: str | None = None,
) -> str:
    """Resolve an OpenAPI operation and invoke it (netcup call).

    Prefer curated tools when available. path/query are k=v strings.
    Destructive calls need confirm=true. Use dry_run=true to resolve only.
    """
    argv = ["call", operation_hint]
    for item in path or []:
        argv.extend(["--path", item])
    for item in query or []:
        argv.extend(["--query", item])
    if body is not None:
        argv.extend(["--body", body])
    if dry_run:
        argv.append("--dry-run")
    argv = _with_server(argv, server)
    try:
        return _tool_result(
            _run(argv, confirm=confirm, timeout=600 if confirm else 120)
        )
    except NetcupError as exc:
        msg = str(exc)
        if "exited 2" in msg and not confirm:
            return (
                "error: confirmation required — re-call with confirm=true "
                f"after explicit user intent\n{msg}"
            )
        return f"error: {msg}"


@mcp.tool()
def netcup_api(
    method: str,
    path: str,
    query: list[str] | None = None,
    header: list[str] | None = None,
    body: str | None = None,
    confirm: bool = False,
) -> str:
    """Raw authenticated HTTP against /scp-core (netcup api METHOD PATH).

    Prefer netcup_call. Non-GET methods require confirm=true.
    """
    method_u = method.strip().upper()
    if method_u not in {"GET", "HEAD", "OPTIONS"} and not confirm:
        return "error: non-GET api calls require confirm=true"
    argv = ["api", method_u, path]
    for item in query or []:
        argv.extend(["--query", item])
    for item in header or []:
        argv.extend(["--header", item])
    if body is not None:
        argv.extend(["--body", body])
    try:
        return _tool_result(_run(argv, confirm=confirm, timeout=600))
    except NetcupError as exc:
        return f"error: {exc}"


@mcp.tool()
def netcup_cli(
    args: list[str],
    confirm: bool = False,
    server: str | None = None,
) -> str:
    """Run allowlisted netcup CLI argv (without the binary name).

    Example: args=["disks","list"], server="myvps".
    Prefer curated tools or netcup_call. Blocks tui, auth login, completion, secrets.
    Destructive commands need confirm=true (-y).
    """
    try:
        _guard_cli_args(args)
        argv = _with_server(list(args), server)
        return _tool_result(
            _run(argv, confirm=confirm, timeout=600 if confirm else 180)
        )
    except NetcupError as exc:
        msg = str(exc)
        if "exited 2" in msg and not confirm:
            return (
                "error: confirmation required — re-call with confirm=true "
                f"after explicit user intent\n{msg}"
            )
        return f"error: {msg}"


if __name__ == "__main__":
    mcp.run()
