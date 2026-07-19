package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
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
		Short: "Wire the netcup agent plugin into Claude, Cursor, and/or Codex",
		Long: `Install or refresh the netcup MCP plugin for agent hosts.

Resolves the plugin root in order: --root, NETCUP_PLUGIN_ROOT, then a git
checkout (cwd / executable). Needs .codex-plugin/plugin.json. Hosts launch
netcup mcp (binary on PATH or bin/netcup from make build).

  Claude   marketplace add + plugin install (scope: user|project|local)
  Cursor   merge mcp.json (project: .cursor/mcp.json, user: ~/.cursor/mcp.json)
  Codex    local marketplace + plugin add (user config; re-run to refresh)

Re-run after git pull to refresh hosts.`,
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

			root, err := resolvePluginRoot(pluginRoot)
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

			results := make([]map[string]any, 0, len(selected))
			app.Out.Info(fmt.Sprintf("plugin root: %s", root))
			app.Out.Info(fmt.Sprintf("scope:       %s", scope))
			app.Out.Info(fmt.Sprintf("hosts:       %s", strings.Join(selected, ",")))
			if dryRun {
				app.Out.Info("dry-run; no changes will be written")
			}

			for _, host := range selected {
				var (
					res map[string]any
					err error
				)
				switch host {
				case mcpHostClaude:
					res, err = installMCPClaude(cmd, root, scope, proj, dryRun)
				case mcpHostCursor:
					res, err = installMCPCursor(root, scope, proj, dryRun)
				case mcpHostCodex:
					res, err = installMCPCodex(cmd, root, dryRun)
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
				"plugin_root": root,
				"scope":       scope,
				"project_dir": proj,
				"dry_run":     dryRun,
				"hosts":       results,
				"version":     version,
			}
			return emitInstallMCPResult(data)
		},
	}
	c.Flags().StringSliceVar(&hosts, "host", []string{mcpHostAll}, "hosts: all|claude|cursor|codex (repeatable)")
	c.Flags().StringVar(&scope, "scope", mcpScopeUser, "install scope: user|project|local (Claude; Cursor uses user|project)")
	c.Flags().StringVar(&pluginRoot, "root", "", "plugin root (default: NETCUP_PLUGIN_ROOT or detect from cwd/executable)")
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

func resolvePluginRoot(explicit string) (string, error) {
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
	return "", output.Exit(output.ExitUsage,
		"plugin root not found (need .codex-plugin/plugin.json); "+
			"run from a checkout, or pass --root / set NETCUP_PLUGIN_ROOT")
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
	local := filepath.Join(pluginRoot, "bin", "netcup")
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		return local, []string{"mcp"}
	}
	if p, err := exec.LookPath("netcup"); err == nil {
		return p, []string{"mcp"}
	}
	return "netcup", []string{"mcp"}
}

func installMCPClaude(cmd *cobra.Command, pluginRoot, scope, projectDir string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": mcpHostClaude, "scope": scope}
	if _, err := exec.LookPath("claude"); err != nil {
		res["status"] = "skipped"
		res["note"] = "claude not found on PATH"
		return res, nil
	}

	marketDir, err := claudeMarketplaceDir(scope, projectDir)
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

	// Add marketplace (ok if already present).
	_ = runHostCommand(cmd, []string{"claude", "plugin", "marketplace", "add", marketDir, "--scope", scope}, true)
	installErr := runHostCommand(cmd, []string{
		"claude", "plugin", "install", "netcup@" + claudeMarketplaceName, "--scope", scope,
	}, true)
	// Refresh cached copy when already installed.
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

func claudeMarketplaceDir(scope, projectDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", output.Exit(output.ExitUsage, "home dir: "+err.Error())
	}
	switch scope {
	case mcpScopeUser:
		return filepath.Join(home, ".claude", "netcup-marketplace"), nil
	case mcpScopeProject, mcpScopeLocal:
		return filepath.Join(projectDir, ".claude", "netcup-marketplace"), nil
	default:
		return "", output.Exit(output.ExitUsage, "invalid scope")
	}
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

func installMCPCursor(pluginRoot, scope, projectDir string, dryRun bool) (map[string]any, error) {
	res := map[string]any{"host": mcpHostCursor, "scope": scope}
	command, mcpArgs := resolveMCPCommand(pluginRoot)

	var mcpPath string
	switch scope {
	case mcpScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, output.Exit(output.ExitUsage, "home dir: "+err.Error())
		}
		mcpPath = filepath.Join(home, ".cursor", "mcp.json")
	case mcpScopeProject, mcpScopeLocal:
		mcpPath = filepath.Join(projectDir, ".cursor", "mcp.json")
		if scope == mcpScopeLocal {
			res["note"] = "Cursor has no separate local scope; wrote project .cursor/mcp.json (consider gitignoring if personal-only)"
		}
	default:
		return nil, output.Exit(output.ExitUsage, "invalid scope for cursor")
	}
	res["mcp_json"] = mcpPath
	res["steps"] = []string{fmt.Sprintf("merge netcup server into %s", mcpPath)}

	if dryRun {
		res["status"] = "dry-run"
		if res["note"] == nil {
			res["note"] = fmt.Sprintf("would merge mcp server command=%s args=%v", command, mcpArgs)
		}
		return res, nil
	}

	if err := mergeCursorMCPJSON(mcpPath, command, mcpArgs); err != nil {
		return nil, err
	}
	res["status"] = "ok"
	note := "restart Cursor or reload MCP servers"
	if existing, ok := res["note"].(string); ok && existing != "" {
		note = existing + "\n" + note
	}
	res["note"] = note
	return res, nil
}

func mergeCursorMCPJSON(path, command string, args []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return output.Exit(output.ExitAPI, "mkdir cursor config: "+err.Error())
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

func installMCPCodex(cmd *cobra.Command, pluginRoot string, dryRun bool) (map[string]any, error) {
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
		// Already configured is fine — try continue to plugin add.
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
