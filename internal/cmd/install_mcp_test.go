package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeMCPHosts(t *testing.T) {
	all, err := normalizeMCPHosts([]string{"all"})
	if err != nil || len(all) != 3 {
		t.Fatalf("all: %v %v", all, err)
	}
	got, err := normalizeMCPHosts([]string{"claude", "claude", "cursor"})
	if err != nil || len(got) != 2 || got[0] != "claude" || got[1] != "cursor" {
		t.Fatalf("dedupe: %v %v", got, err)
	}
	if _, err := normalizeMCPHosts([]string{"nope"}); err == nil {
		t.Fatal("expected invalid host error")
	}
}

func TestValidateAndFindPluginRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".codex-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".codex-plugin", "plugin.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "cmd")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	found, ok := findPluginRoot(nested)
	if !ok || found != root {
		t.Fatalf("findPluginRoot: got %q ok=%v want %q", found, ok, root)
	}
	if err := validatePluginRoot(root); err != nil {
		t.Fatal(err)
	}
	if err := validatePluginRoot(t.TempDir()); err == nil {
		t.Fatal("expected validation failure")
	}
}

func TestResolveMCPCommand(t *testing.T) {
	root := t.TempDir()
	distDir := filepath.Join(root, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(distDir, "netcup")
	if err := os.WriteFile(dist, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, args := resolveMCPCommand(root)
	if cmd != dist || len(args) != 1 || args[0] != "mcp" {
		t.Fatalf("got %q %v", cmd, args)
	}

	empty := t.TempDir()
	cmd, args = resolveMCPCommand(empty)
	if len(args) != 1 || args[0] != "mcp" {
		t.Fatalf("fallback args %v", args)
	}
	if cmd != "netcup" {
		if _, err := os.Stat(cmd); err != nil {
			t.Fatalf("unexpected command %q", cmd)
		}
	}
}

func TestMergeCursorMCPJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cursor", "mcp.json")
	if err := mergeCursorMCPJSON(path, "/tmp/dist/netcup", []string{"mcp"}); err != nil {
		t.Fatal(err)
	}
	// Second merge preserves other servers.
	raw := []byte(`{"mcpServers":{"other":{"command":"echo"}}}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergeCursorMCPJSON(path, "/tmp/dist/netcup", []string{"mcp"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, part := range []string{`"other"`, `"netcup"`, `/tmp/dist/netcup`, `"mcp"`} {
		if !strings.Contains(s, part) {
			t.Fatalf("merge missing %q in %s", part, s)
		}
	}
}
