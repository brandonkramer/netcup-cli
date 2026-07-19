package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

var expectedTools = []string{
	"netcup_ping",
	"netcup_whoami",
	"netcup_auth_status",
	"netcup_servers_list",
	"netcup_servers_get",
	"netcup_servers_power",
	"netcup_tasks_list",
	"netcup_tasks_get",
	"netcup_tasks_cancel",
	"netcup_snapshots_list",
	"netcup_iso_get",
	"netcup_iso_available",
	"netcup_iso_attach",
	"netcup_iso_detach",
	"netcup_firewall_get",
	"netcup_endpoints",
	"netcup_describe",
	"netcup_call",
	"netcup_api",
	"netcup_cli",
}

func TestToolsList(t *testing.T) {
	s := NewServer("test")
	resp := s.HandleMessage(context.Background(), []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list"
	}`))
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("response type %T", resp)
	}
	result, ok := jr.Result.(mcp.ListToolsResult)
	if !ok {
		t.Fatalf("result type %T", jr.Result)
	}
	got := map[string]bool{}
	for _, tool := range result.Tools {
		got[tool.Name] = true
	}
	if len(got) != len(expectedTools) {
		t.Fatalf("tool count %d want %d: %v", len(got), len(expectedTools), got)
	}
	for _, name := range expectedTools {
		if !got[name] {
			t.Errorf("missing tool %s", name)
		}
	}
}

func TestToolCallPowerRequiresConfirm(t *testing.T) {
	s := NewServer("test")
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "netcup_servers_power",
			"arguments": map[string]any{
				"action":  "reboot",
				"server":  "1",
				"confirm": false,
			},
		},
	})
	resp := s.HandleMessage(context.Background(), raw)
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("type %T", resp)
	}
	result, ok := jr.Result.(*mcp.CallToolResult)
	if !ok {
		// some versions return value type
		if r2, ok2 := jr.Result.(mcp.CallToolResult); ok2 {
			result = &r2
		} else {
			t.Fatalf("result type %T", jr.Result)
		}
	}
	text := toolText(result)
	if !strings.Contains(text, "confirm=true") {
		t.Fatalf("text=%q", text)
	}
}

func TestToolCallCLIBlocksTUI(t *testing.T) {
	s := NewServer("test")
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "netcup_cli",
			"arguments": map[string]any{
				"args": []string{"tui"},
			},
		},
	})
	resp := s.HandleMessage(context.Background(), raw)
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("type %T", resp)
	}
	result := asCallResult(t, jr.Result)
	text := toolText(result)
	if !strings.Contains(text, "not allowed") {
		t.Fatalf("text=%q", text)
	}
}

func TestToolCallAPINonGETRequiresConfirm(t *testing.T) {
	s := NewServer("test")
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "netcup_api",
			"arguments": map[string]any{
				"method":  "DELETE",
				"path":    "/api/v1/x",
				"confirm": false,
			},
		},
	})
	resp := s.HandleMessage(context.Background(), raw)
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("type %T", resp)
	}
	text := toolText(asCallResult(t, jr.Result))
	if !strings.Contains(text, "confirm=true") {
		t.Fatalf("text=%q", text)
	}
}

func TestToolCallPingWithFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake binary")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-netcup")
	argsFile := filepath.Join(dir, "args.txt")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + argsFile + "'\necho '{\"ok\":true,\"command\":\"ping\"}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETCUP_BIN", script)

	s := NewServer("test")
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "netcup_ping",
			"arguments": map[string]any{},
		},
	})
	resp := s.HandleMessage(context.Background(), raw)
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("type %T", resp)
	}
	text := toolText(asCallResult(t, jr.Result))
	if !strings.Contains(text, `"ok"`) {
		t.Fatalf("text=%q", text)
	}
	argBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := string(argBytes)
	if !strings.Contains(args, "--format") || !strings.Contains(args, "json") || !strings.Contains(args, "ping") {
		t.Fatalf("argv file=%q", args)
	}
}

func TestToolCallISOAttachXor(t *testing.T) {
	s := NewServer("test")
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "netcup_iso_attach",
			"arguments": map[string]any{
				"server":  "1",
				"confirm": true,
			},
		},
	})
	resp := s.HandleMessage(context.Background(), raw)
	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("type %T", resp)
	}
	text := toolText(asCallResult(t, jr.Result))
	if !strings.Contains(text, "exactly one of iso_id or user_iso") {
		t.Fatalf("text=%q", text)
	}
}

func asCallResult(t *testing.T, result any) *mcp.CallToolResult {
	t.Helper()
	if r, ok := result.(*mcp.CallToolResult); ok {
		return r
	}
	if r, ok := result.(mcp.CallToolResult); ok {
		return &r
	}
	t.Fatalf("result type %T", result)
	return nil
}

func toolText(r *mcp.CallToolResult) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// Ensure runCLI timeout path is exercised quickly with a sleeping fake.
func TestRunCLITimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake binary")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-netcup")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETCUP_BIN", script)
	_, err := runCLI(context.Background(), []string{"ping"}, false, 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("got %v", err)
	}
}
