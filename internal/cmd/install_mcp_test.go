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
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		".codex-plugin/plugin.json",
		"bin/netcup-mcp",
		"plugin/netcup_mcp.py",
	} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
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

func TestMergeCursorMCPJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cursor", "mcp.json")
	launcher := "/tmp/netcup-cli/bin/netcup-mcp"
	if err := mergeCursorMCPJSON(path, launcher); err != nil {
		t.Fatal(err)
	}
	// Second merge preserves other servers.
	raw := []byte(`{"mcpServers":{"other":{"command":"echo"}}}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergeCursorMCPJSON(path, launcher); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, part := range []string{`"other"`, `"netcup"`, launcher} {
		if !strings.Contains(s, part) {
			t.Fatalf("merge missing %q in %s", part, s)
		}
	}
}
