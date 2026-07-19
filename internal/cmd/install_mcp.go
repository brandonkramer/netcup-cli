package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/config"
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/pluginassets"
	"github.com/spf13/cobra"
)

const (
	mcpHostAll    = "all"
	mcpHostClaude = "claude"
	mcpHostCursor = "cursor"
	mcpHostCodex  = "codex"

	mcpScopeUser    = "user"
	mcpScopeProject = "project"
	mcpScopeLocal   = "local"

	claudeMarketplaceName = "netcup-local"
	codexMarketplaceName  = "netcup"

	projectSkillRel = ".agents/skills/netcup/SKILL.md"
)

func newInstallMCPCmd() *cobra.Command {
	var (
		hosts      []string
		scope      string
		pluginRoot string
		projectDir string
		dryRun     bool
	)
	c := &cobra.Command{
		Use:   "install-mcp",
		Short: "Wire netcup MCP + skill into Claude, Cursor, and/or Codex",
		Long: `Install or refresh netcup for agent hosts.

  --scope user
    Full plugin wiring. Resolves plugin root (--root, NETCUP_PLUGIN_ROOT,
    checkout, or materialize into ~/.config/netcup/plugin), then:
      Claude  marketplace + plugin install
      Cursor  ~/.cursor/mcp.json
      Codex   plugin marketplace add + plugin add

  --scope project|local
    Lightweight project files only (no plugin/marketplace CLIs):
      skill   .agents/skills/netcup/SKILL.md  (once; Cursor+Codex discover it)
      Cursor  .cursor/mcp.json
      Claude  .mcp.json
      Codex   .codex/config.toml  ([mcp_servers.netcup])
    local uses the same paths; gitignore if personal-only.

Hosts launch netcup mcp (PATH, or bin/netcup under --root / checkout).
Re-run after upgrading netcup to refresh.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "install-mcp"

			scope = strings.ToLower(strings.TrimSpace(scope))
			switch scope {
			case mcpScopeUser, mcpScopeProject, mcpScopeLocal:
			default:
				return output.Exit(output.ExitUsage, "invalid --scope (want user|project|local)")
			}

			selected, err := normalizeMCPHosts(hosts)
			if err != nil {
				return err
			}

			proj := strings.TrimSpace(projectDir)
			if proj == "" {
				proj, err = os.Getwd()
				if err != nil {
					return output.Exit(output.ExitUsage, "resolve project dir: "+err.Error())
				}
			}
			proj, err = filepath.Abs(proj)
			if err != nil {
				return output.Exit(output.ExitUsage, "resolve project dir: "+err.Error())
			}

			projectLike := scope == mcpScopeProject || scope == mcpScopeLocal
			var root string
			if projectLike {
				root, err = resolveMCPBinaryRoot(pluginRoot)
				if err != nil {
					return err
				}
			} else {
				root, err = resolvePluginRoot(pluginRoot, version)
				if err != nil {
					return err
				}
			}

			results := make([]map[string]any, 0, len(selected))
			if root != "" {
				app.Out.Info(fmt.Sprintf("plugin root: %s", root))
			}
			app.Out.Info(fmt.Sprintf("scope:       %s", scope))
			app.Out.Info(fmt.Sprintf("hosts:       %s", strings.Join(selected, ",")))
			if dryRun {
				app.Out.Info("dry-run; no changes will be written")
			}

			skillPath := filepath.Join(proj, filepath.FromSlash(projectSkillRel))
			if projectLike {
				app.Out.Info(fmt.Sprintf("skill:       %s", skillPath))
				if !dryRun {
					if err := writeProjectSkill(proj); err != nil {
						return err
					}
				}
			}

			for _, host := range selected {
				var (
					res map[string]any
					err error
				)
				if projectLike {
					res, err = installMCPProjectHost(host, root, scope, proj, dryRun)
				} else {
					switch host {
					case mcpHostClaude:
						res, err = installMCPClaudeUser(cmd, root, scope, dryRun)
					case mcpHostCursor:
						res, err = installMCPCursorUser(root, dryRun)
					case mcpHostCodex:
						res, err = installMCPCodexUser(cmd, root, dryRun)
					}
				}
				if err != nil {
					return err
				}
				results = append(results, res)
				status := fmt.Sprint(res["status"])
				app.Out.Info(fmt.Sprintf("%s: %s", host, status))
				if note, ok := res["note"].(string); ok && note != "" {
					for _, line := range strings.Split(note, "\n") {
						app.Out.Info("  " + line)
					}
				}
			}

			data := map[string]any{
				"scope":       scope,
				"project_dir": proj,
				"dry_run":     dryRun,
				"hosts":       results,
				"version":     version,
			}
			if root != "" {
				data["plugin_root"] = root
			}
			if projectLike {
				data["skill"] = skillPath
			}
			return emitInstallMCPResult(data)
		},
	}
	c.Flags().StringSliceVar(&hosts, "host", []string{mcpHostAll}, "hosts: all|claude|cursor|codex (repeatable)")
	c.Flags().StringVar(&scope, "scope", mcpScopeUser, "install scope: user|project|local")
	c.Flags().StringVar(&pluginRoot, "root", "", "plugin/binary root (user: required plugin tree; project: optional bin/netcup)")
	c.Flags().StringVar(&projectDir, "project-dir", "", "project directory for project/local scope (default: cwd)")
	c.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print actions without changing host config")
	return c
}

func emitInstallMCPResult(data map[string]any) error {
	if app.Out.Format == output.FormatTable {
		hosts, _ := data["hosts"].([]map[string]any)
		rows := make([][]string, 0, len(hosts))
		for _, h := range hosts {
			rows = append(rows, []string{
				fmt.Sprint(h["host"]),
				fmt.Sprint(h["status"]),
				fmt.Sprint(data["scope"]),
			})
		}
		return app.Out.Success(output.TableData{
			Headers: []string{"host", "status", "scope"},
			Rows:    rows,
			Raw:     data,
		})
	}
	return app.Out.Success(data)
}

func normalizeMCPHosts(hosts []string) ([]string, error) {
	if len(hosts) == 0 {
		hosts = []string{mcpHostAll}
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if h == mcpHostAll {
			return []string{mcpHostClaude, mcpHostCursor, mcpHostCodex}, nil
		}
		switch h {
		case mcpHostClaude, mcpHostCursor, mcpHostCodex:
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		default:
			return nil, output.Exit(output.ExitUsage, "invalid --host (want all|claude|cursor|codex)")
		}
	}
	if len(out) == 0 {
		return nil, output.Exit(output.ExitUsage, "no hosts selected")
	}
	return out, nil
}

func resolvePluginRoot(explicit, binaryVersion string) (string, error) {
	if v := strings.TrimSpace(explicit); v != "" {
		return requirePluginRoot(v)
	}
	if v := strings.TrimSpace(os.Getenv("NETCUP_PLUGIN_ROOT")); v != "" {
		return requirePluginRoot(v)
	}
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := findPluginRoot(cwd); ok {
			return root, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		resolved := exe
		if r, err := filepath.EvalSymlinks(exe); err == nil {
			resolved = r
		}
		if root, ok := findPluginRoot(filepath.Dir(resolved)); ok {
			return root, nil
		}
	}
	dest := filepath.Join(config.DefaultConfigDir(), "plugin")
	if err := pluginassets.Materialize(dest, binaryVersion); err != nil {
		return "", output.Exit(output.ExitAPI, "materialize plugin root: "+err.Error())
	}
	return requirePluginRoot(dest)
}

// resolveMCPBinaryRoot is for project/local: optional --root / env / checkout
// only to prefer bin/netcup; empty means use PATH.
func resolveMCPBinaryRoot(explicit string) (string, error) {
	if v := strings.TrimSpace(explicit); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", output.Exit(output.ExitUsage, "resolve --root: "+err.Error())
		}
		return abs, nil
	}
	if v := strings.TrimSpace(os.Getenv("NETCUP_PLUGIN_ROOT")); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", output.Exit(output.ExitUsage, "resolve NETCUP_PLUGIN_ROOT: "+err.Error())
		}
		return abs, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := findPluginRoot(cwd); ok {
			return root, nil
		}
	}
	return "", nil
}

func findPluginRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if err := validatePluginRoot(dir); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func requirePluginRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", output.Exit(output.ExitUsage, "resolve plugin root: "+err.Error())
	}
	if err := validatePluginRoot(abs); err != nil {
		return "", output.Exit(output.ExitUsage, err.Error())
	}
	return abs, nil
}

func validatePluginRoot(root string) error {
	manifest := filepath.Join(root, ".codex-plugin", "plugin.json")
	if st, err := os.Stat(manifest); err != nil || st.IsDir() {
		return fmt.Errorf("not a netcup plugin root (%s missing)", manifest)
	}
	return nil
}

// resolveMCPCommand returns command+args to start netcup mcp for host configs.
func resolveMCPCommand(pluginRoot string) (command string, args []string) {
	if pluginRoot != "" {
		local := filepath.Join(pluginRoot, "bin", "netcup")
		if st, err := os.Stat(local); err == nil && !st.IsDir() {
			return local, []string{"mcp"}
		}
	}
	if p, err := exec.LookPath("netcup"); err == nil {
		return p, []string{"mcp"}
	}
	return "netcup", []string{"mcp"}
}

func writeProjectSkill(projectDir string) error {
	data, err := pluginassets.SkillMarkdown()
	if err != nil {
		return output.Exit(output.ExitAPI, "read embedded skill: "+err.Error())
	}
	dest := filepath.Join(projectDir, filepath.FromSlash(projectSkillRel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir skill dir: "+err.Error())
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return output.Exit(output.ExitAPI, "write skill: "+err.Error())
	}
	return nil
}

func installMCPProjectHost(host, binaryRoot, scope, projectDir string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": host, "scope": scope}
	command, mcpArgs := resolveMCPCommand(binaryRoot)
	skillPath := filepath.Join(projectDir, filepath.FromSlash(projectSkillRel))
	res["skill"] = skillPath

	var mcpPath string
	var merge func() error
	switch host {
	case mcpHostCursor:
		mcpPath = filepath.Join(projectDir, ".cursor", "mcp.json")
		merge = func() error { return mergeMCPServersJSON(mcpPath, command, mcpArgs) }
	case mcpHostClaude:
		mcpPath = filepath.Join(projectDir, ".mcp.json")
		merge = func() error { return mergeMCPServersJSON(mcpPath, command, mcpArgs) }
	case mcpHostCodex:
		mcpPath = filepath.Join(projectDir, ".codex", "config.toml")
		merge = func() error { return mergeCodexProjectMCP(mcpPath, command, mcpArgs) }
	default:
		return nil, output.Exit(output.ExitUsage, "invalid host")
	}
	res["mcp_config"] = mcpPath
	steps := []string{
		fmt.Sprintf("write %s", skillPath),
		fmt.Sprintf("merge netcup into %s", mcpPath),
	}
	if scope == mcpScopeLocal {
		steps = append(steps, "consider gitignoring these files if personal-only")
	}
	res["steps"] = steps

	if dryRun {
		res["status"] = "dry-run"
		res["note"] = fmt.Sprintf("would write skill + mcp command=%s args=%v", command, mcpArgs)
		return res, nil
	}
	if err := merge(); err != nil {
		return nil, err
	}
	res["status"] = "ok"
	res["note"] = "reload host / restart to pick up MCP and skill"
	return res, nil
}

func installMCPClaudeUser(cmd *cobra.Command, pluginRoot, scope string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": mcpHostClaude, "scope": scope}
	if _, err := exec.LookPath("claude"); err != nil {
		res["status"] = "skipped"
		res["note"] = "claude not found on PATH"
		return res, nil
	}

	marketDir, err := claudeMarketplaceDir(scope, "")
	if err != nil {
		return nil, err
	}
	res["marketplace_dir"] = marketDir

	steps := []string{
		fmt.Sprintf("ensure marketplace at %s (symlink netcup -> plugin root)", marketDir),
		fmt.Sprintf("claude plugin marketplace add %s --scope %s", marketDir, scope),
		fmt.Sprintf("claude plugin install netcup@%s --scope %s", claudeMarketplaceName, scope),
		fmt.Sprintf("claude plugin update netcup@%s (if already installed)", claudeMarketplaceName),
	}
	res["steps"] = steps

	if dryRun {
		res["status"] = "dry-run"
		res["note"] = strings.Join(steps, "\n")
		return res, nil
	}

	if err := ensureClaudeMarketplace(marketDir, pluginRoot); err != nil {
		return nil, err
	}

	_ = runHostCommand(cmd, []string{"claude", "plugin", "marketplace", "add", marketDir, "--scope", scope}, true)
	installErr := runHostCommand(cmd, []string{
		"claude", "plugin", "install", "netcup@" + claudeMarketplaceName, "--scope", scope,
	}, true)
	_ = runHostCommand(cmd, []string{"claude", "plugin", "marketplace", "update", claudeMarketplaceName}, true)
	updateErr := runHostCommand(cmd, []string{"claude", "plugin", "update", "netcup@" + claudeMarketplaceName}, true)

	if installErr != nil && updateErr != nil {
		return nil, output.Exit(output.ExitAPI,
			fmt.Sprintf("claude plugin install/update failed: install=%v update=%v", installErr, updateErr))
	}
	res["status"] = "ok"
	res["note"] = "reload Claude plugins (/reload-plugins) or restart Claude Code"
	return res, nil
}

func claudeMarketplaceDir(scope, _ string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", output.Exit(output.ExitUsage, "home dir: "+err.Error())
	}
	// User-scope only (project uses .mcp.json + .agents/skills).
	if scope != mcpScopeUser {
		return "", output.Exit(output.ExitUsage, "claude marketplace is user-scope only")
	}
	return filepath.Join(home, ".claude", "netcup-marketplace"), nil
}

func ensureClaudeMarketplace(marketDir, pluginRoot string) error {
	if err := os.MkdirAll(filepath.Join(marketDir, ".claude-plugin"), 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir marketplace: "+err.Error())
	}
	manifest := map[string]any{
		"name": claudeMarketplaceName,
		"owner": map[string]string{
			"name": "brandonkramer",
		},
		"metadata": map[string]string{
			"description": "Local netcup-cli plugin managed by netcup install-mcp",
		},
		"plugins": []map[string]any{
			{
				"name":        "netcup",
				"description": "netcup SCP CLI + MCP tools",
				"source":      "./netcup",
				"category":    "developer-tools",
			},
		},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	manifestPath := filepath.Join(marketDir, ".claude-plugin", "marketplace.json")
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		return output.Exit(output.ExitAPI, "write marketplace.json: "+err.Error())
	}

	link := filepath.Join(marketDir, "netcup")
	_ = os.Remove(link)
	if err := os.Symlink(pluginRoot, link); err != nil {
		return output.Exit(output.ExitAPI, "symlink plugin into marketplace: "+err.Error())
	}
	return nil
}

func installMCPCursorUser(pluginRoot string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": mcpHostCursor, "scope": mcpScopeUser}
	command, mcpArgs := resolveMCPCommand(pluginRoot)
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, output.Exit(output.ExitUsage, "home dir: "+err.Error())
	}
	mcpPath := filepath.Join(home, ".cursor", "mcp.json")
	res["mcp_json"] = mcpPath
	res["steps"] = []string{fmt.Sprintf("merge netcup server into %s", mcpPath)}

	if dryRun {
		res["status"] = "dry-run"
		res["note"] = fmt.Sprintf("would merge mcp server command=%s args=%v", command, mcpArgs)
		return res, nil
	}

	if err := mergeMCPServersJSON(mcpPath, command, mcpArgs); err != nil {
		return nil, err
	}
	res["status"] = "ok"
	res["note"] = "restart Cursor or reload MCP servers"
	return res, nil
}

func mergeMCPServersJSON(path, command string, args []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir mcp config: "+err.Error())
	}

	root := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return output.Exit(output.ExitAPI, "parse "+path+": "+err.Error())
		}
	}

	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		root["mcpServers"] = servers
	}
	servers["netcup"] = map[string]any{
		"command": command,
		"args":    args,
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return output.Exit(output.ExitAPI, "write "+path+": "+err.Error())
	}
	return nil
}

// mergeCursorMCPJSON is kept for tests; same as mergeMCPServersJSON.
func mergeCursorMCPJSON(path, command string, args []string) error {
	return mergeMCPServersJSON(path, command, args)
}

func installMCPCodexUser(cmd *cobra.Command, pluginRoot string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": mcpHostCodex, "scope": mcpScopeUser}
	if _, err := exec.LookPath("codex"); err != nil {
		res["status"] = "skipped"
		res["note"] = "codex not found on PATH"
		return res, nil
	}

	steps := []string{
		"write .agents/plugins/marketplace.json (Codex catalog; local/gitignored)",
		fmt.Sprintf("codex plugin marketplace add %s", pluginRoot),
		fmt.Sprintf("codex plugin add netcup@%s", codexMarketplaceName),
	}
	res["steps"] = steps

	if dryRun {
		res["status"] = "dry-run"
		res["note"] = strings.Join(steps, "\n")
		return res, nil
	}

	if err := ensureCodexMarketplaceFile(pluginRoot); err != nil {
		return nil, err
	}

	if err := runHostCommand(cmd, []string{"codex", "plugin", "marketplace", "add", pluginRoot}, false); err != nil {
		app.Out.Info(fmt.Sprintf("  marketplace add: %v (continuing)", err))
	}
	if err := runHostCommand(cmd, []string{"codex", "plugin", "add", "netcup@" + codexMarketplaceName, "--json"}, false); err != nil {
		return nil, output.Exit(output.ExitAPI, "codex plugin add failed: "+err.Error())
	}
	res["status"] = "ok"
	res["note"] = "restart Codex / ChatGPT Work mode to pick up plugin changes"
	return res, nil
}

func ensureCodexMarketplaceFile(pluginRoot string) error {
	dir := filepath.Join(pluginRoot, ".agents", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir .agents/plugins: "+err.Error())
	}
	manifest := map[string]any{
		"name": codexMarketplaceName,
		"interface": map[string]string{
			"displayName": "netcup",
		},
		"plugins": []map[string]any{
			{
				"name": "netcup",
				"source": map[string]string{
					"source": "local",
					"path":   ".",
				},
				"policy": map[string]string{
					"installation":   "AVAILABLE",
					"authentication": "ON_USE",
				},
				"category": "Developer Tools",
			},
		},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := filepath.Join(dir, "marketplace.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return output.Exit(output.ExitAPI, "write Codex marketplace.json: "+err.Error())
	}
	return nil
}

var codexNetcupSectionRE = regexp.MustCompile(`(?ms)^\[mcp_servers\.netcup\][^\[]*`)

func mergeCodexProjectMCP(path, command string, args []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir .codex: "+err.Error())
	}

	var b strings.Builder
	b.WriteString("[mcp_servers.netcup]\n")
	b.WriteString(fmt.Sprintf("command = %q\n", command))
	b.WriteString("args = [")
	for i, a := range args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%q", a))
	}
	b.WriteString("]\n")
	section := b.String()

	existing := ""
	if raw, err := os.ReadFile(path); err == nil {
		existing = string(raw)
	}
	existing = strings.TrimRight(existing, "\n\r\t ")
	if existing == "" {
		if err := os.WriteFile(path, []byte(section), 0o644); err != nil {
			return output.Exit(output.ExitAPI, "write "+path+": "+err.Error())
		}
		return nil
	}
	if codexNetcupSectionRE.MatchString(existing) {
		existing = codexNetcupSectionRE.ReplaceAllString(existing, strings.TrimSuffix(section, "\n")+"\n")
	} else {
		existing = existing + "\n\n" + section
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		return output.Exit(output.ExitAPI, "write "+path+": "+err.Error())
	}
	return nil
}

// runHostCommand runs a host CLI. If soft is true, non-zero exit returns the error
// without wrapping as ExitError (caller decides).
func runHostCommand(cmd *cobra.Command, argv []string, soft bool) error {
	if len(argv) == 0 {
		return nil
	}
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		err = fmt.Errorf("%s not found on PATH", argv[0])
		if soft {
			return err
		}
		return output.Exit(output.ExitUsage, err.Error())
	}
	c := exec.CommandContext(cmd.Context(), bin, argv[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		if soft {
			return err
		}
		return fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
	}
	return nil
}
