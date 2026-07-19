// Package pluginassets embeds host plugin manifests and the netcup skill so
// install-mcp can materialize a plugin root without a git checkout (Homebrew,
// go install, Release binaries).
package pluginassets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:files
var files embed.FS

const stampName = "VERSION"

// embedPath → on-disk relative path under the plugin root.
var destByEmbed = map[string]string{
	"files/codex-plugin/plugin.json":       ".codex-plugin/plugin.json",
	"files/claude-plugin/plugin.json":      ".claude-plugin/plugin.json",
	"files/claude-plugin/marketplace.json": ".claude-plugin/marketplace.json",
	"files/cursor-plugin/plugin.json":      ".cursor-plugin/plugin.json",
	"files/cursor-plugin/mcp.json":         ".cursor-plugin/mcp.json",
	"files/skills/netcup/SKILL.md":         "skills/netcup/SKILL.md",
}

// Materialize writes the embedded plugin tree into dest when dest is missing
// or its VERSION stamp does not match binaryVersion. dest becomes a valid
// plugin root (.codex-plugin/plugin.json present).
func Materialize(dest, binaryVersion string) error {
	dest = filepath.Clean(dest)
	if dest == "" || dest == "." {
		return fmt.Errorf("pluginassets: empty dest")
	}
	ver := strings.TrimSpace(binaryVersion)
	if ver == "" {
		ver = "dev"
	}
	stampPath := filepath.Join(dest, stampName)
	if raw, err := os.ReadFile(stampPath); err == nil {
		if strings.TrimSpace(string(raw)) == ver {
			if _, err := os.Stat(filepath.Join(dest, ".codex-plugin", "plugin.json")); err == nil {
				return nil
			}
		}
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("pluginassets: mkdir %s: %w", dest, err)
	}

	for embedPath, rel := range destByEmbed {
		data, err := files.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("pluginassets: read %s: %w", embedPath, err)
		}
		out := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return fmt.Errorf("pluginassets: mkdir %s: %w", filepath.Dir(out), err)
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("pluginassets: write %s: %w", out, err)
		}
	}

	if err := os.WriteFile(stampPath, []byte(ver+"\n"), 0o644); err != nil {
		return fmt.Errorf("pluginassets: write VERSION: %w", err)
	}
	return nil
}

// SkillMarkdown returns the embedded netcup agent skill (SKILL.md).
func SkillMarkdown() ([]byte, error) {
	return files.ReadFile("files/skills/netcup/SKILL.md")
}

// ReadEmbed returns the bytes of an embedded file (path relative to package,
// e.g. files/codex-plugin/plugin.json). Used by drift tests.
func ReadEmbed(embedPath string) ([]byte, error) {
	return files.ReadFile(embedPath)
}

// EmbedPaths returns the embed paths that Materialize writes.
func EmbedPaths() []string {
	out := make([]string, 0, len(destByEmbed))
	for p := range destByEmbed {
		out = append(out, p)
	}
	return out
}

// DestRel returns the on-disk relative path for an embed path.
func DestRel(embedPath string) (string, bool) {
	rel, ok := destByEmbed[embedPath]
	return rel, ok
}

// WalkFiles walks the embed FS (for debugging/tests).
func WalkFiles(fn fs.WalkDirFunc) error {
	return fs.WalkDir(files, "files", fn)
}
