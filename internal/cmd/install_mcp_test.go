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

func TestResolvePluginRootMaterialize(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("NETCUP_CONFIG_DIR", cfg)
	t.Setenv("NETCUP_PLUGIN_ROOT", "")

	// Empty cwd that is not under a checkout with manifests.
	empty := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(empty); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	root, err := resolvePluginRoot("", "1.2.3-test")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cfg, "plugin")
	if root != want {
		t.Fatalf("root=%q want %q", root, want)
	}
	if err := validatePluginRoot(root); err != nil {
		t.Fatal(err)
	}
}

func TestResolveMCPCommand(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(binDir, "netcup")
	if err := os.WriteFile(local, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, args := resolveMCPCommand(root)
	if cmd != local || len(args) != 1 || args[0] != "mcp" {
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
	if err := mergeCursorMCPJSON(path, "/tmp/bin/netcup", []string{"mcp"}); err != nil {
		t.Fatal(err)
	}
	// Second merge preserves other servers.
	raw := []byte(`{"mcpServers":{"other":{"command":"echo"}}}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergeCursorMCPJSON(path, "/tmp/bin/netcup", []string{"mcp"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, part := range []string{`"other"`, `"netcup"`, `/tmp/bin/netcup`, `"mcp"`} {
		if !strings.Contains(s, part) {
			t.Fatalf("merge missing %q in %s", part, s)
		}
	}
}
