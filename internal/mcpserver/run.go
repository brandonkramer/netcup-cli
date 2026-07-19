package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	cliAllowlist = map[string]bool{
		"api": true, "auth": true, "cache": true, "call": true, "describe": true,
		"disks": true, "endpoints": true, "failover": true, "firewall": true,
		"firewall-policies": true, "images": true, "iso": true, "isos": true,
		"maintenance": true, "metrics": true, "nics": true, "ping": true,
		"rdns": true, "servers": true, "snapshots": true, "spec": true,
		"ssh-keys": true, "tasks": true, "users": true, "vlans": true,
	}
	authAllow = map[string]bool{
		"status": true, "whoami": true, "refresh": true, "logout": true,
	}
	denyTop = map[string]bool{
		"tui": true, "completion": true, "update": true, "help": true, "mcp": true,
		"install-mcp": true,
	}
	secretFlags = map[string]bool{
		"--password": true, "--password-file": true, "--token": true,
		"--access-token": true, "--refresh-token": true, "--client-secret": true,
	}
	powerActions = map[string]bool{
		"start": true, "stop": true, "reboot": true, "poweroff": true,
		"reset": true, "suspend": true,
	}
)

type runError struct {
	msg string
}

func (e *runError) Error() string { return e.msg }

func netcupBin() (string, error) {
	if v := strings.TrimSpace(os.Getenv("NETCUP_BIN")); v != "" {
		return v, nil
	}
	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		return exe, nil
	}
	if p, err := exec.LookPath("netcup"); err == nil {
		return p, nil
	}
	return "", &runError{msg: "netcup binary not found; set NETCUP_BIN or install netcup"}
}

func looksSecretFlag(arg string) bool {
	base, _, _ := strings.Cut(arg, "=")
	return secretFlags[base]
}

func ensureJSONFormat(argv []string) []string {
	for _, a := range argv {
		if a == "--json" || a == "--format" || strings.HasPrefix(a, "--format=") {
			return argv
		}
	}
	return append([]string{"--format", "json"}, argv...)
}

func withServer(argv []string, server string) []string {
	if strings.TrimSpace(server) == "" {
		return argv
	}
	return append(argv, "-s", server)
}

func guardCLIArgs(args []string) error {
	if len(args) == 0 {
		return &runError{msg: "args must be non-empty (e.g. [\"servers\", \"list\"])"}
	}
	top := args[0]
	if denyTop[top] {
		return &runError{msg: fmt.Sprintf("command '%s' is not allowed via MCP", top)}
	}
	if !cliAllowlist[top] {
		return &runError{msg: fmt.Sprintf("command '%s' is not allowlisted; use curated tools or netcup_call / netcup_api", top)}
	}
	if top == "auth" {
		if len(args) < 2 || !authAllow[args[1]] {
			return &runError{msg: "auth via MCP allows only status|whoami|refresh|logout; run `netcup auth login` yourself"}
		}
	}
	for _, a := range args {
		if a == "tui" || a == "completion" || a == "mcp" {
			return &runError{msg: "tui/completion/mcp are not allowed via MCP"}
		}
	}
	return nil
}

func runCLI(ctx context.Context, argv []string, confirm bool, timeout time.Duration) (string, error) {
	for _, a := range argv {
		if looksSecretFlag(a) {
			return "", &runError{msg: "secret-bearing flags are not allowed via MCP; use netcup auth login or --password-file on a human TTY"}
		}
	}
	bin, err := netcupBin()
	if err != nil {
		return "", err
	}
	args := ensureJSONFormat(argv)
	args = append(args, "-q")
	if confirm {
		hasYes := false
		for _, a := range args {
			if a == "-y" || a == "--yes" {
				hasYes = true
				break
			}
		}
		if !hasYes {
			args = append(args, "-y")
		}
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil
	err = cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())
	if errors.Is(cctx.Err(), context.DeadlineExceeded) {
		return "", &runError{msg: fmt.Sprintf("timed out after %s", timeout)}
	}
	if err != nil {
		var ee *exec.ExitError
		code := "?"
		if errors.As(err, &ee) {
			code = fmt.Sprintf("%d", ee.ExitCode())
		}
		parts := []string{fmt.Sprintf("netcup exited %s", code)}
		if out != "" {
			parts = append(parts, out)
		}
		if errOut != "" {
			parts = append(parts, "[stderr]\n"+errOut)
		}
		return "", &runError{msg: strings.Join(parts, "\n")}
	}
	if out == "" {
		return `{"ok":true,"stdout":"","stderr":` + jsonString(errOut) + `}`, nil
	}
	if json.Valid([]byte(out)) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(out), "", "  "); err == nil {
			return buf.String(), nil
		}
		return out, nil
	}
	return out, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func errText(err error) string {
	return "error: " + err.Error()
}

func confirmRequired(msg string) string {
	return "error: confirmation required — re-call with confirm=true after explicit user intent\n" + msg
}
