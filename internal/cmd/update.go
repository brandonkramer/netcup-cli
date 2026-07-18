package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	installHomebrew = "homebrew"
	installGo       = "go"
	installManual   = "manual"

	goInstallPkg = "github.com/brandonkramer/netcup-cli/cmd/netcup@latest"
	brewFormula  = "brandonkramer/tap/netcup"
	releasesURL  = "https://github.com/brandonkramer/netcup-cli/releases"
	defaultGoBin = "go/bin"
)

func newUpdateCmd() *cobra.Command {
	var (
		dryRun bool
		method string
	)
	c := &cobra.Command{
		Use:   "update",
		Short: "Update this CLI via Homebrew, go install, or show Releases instructions",
		Long: `Update the netcup binary.

Detects how this executable was installed:

  Homebrew         brew update && brew upgrade brandonkramer/tap/netcup
  go install       go install github.com/brandonkramer/netcup-cli/cmd/netcup@latest
  other / Releases print download instructions

Use --method to force a path, or --dry-run to print the plan without running it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "update"

			exe, err := os.Executable()
			if err != nil {
				return output.Exit(output.ExitUsage, "resolve executable: "+err.Error())
			}
			resolved := exe
			if r, err := filepath.EvalSymlinks(exe); err == nil {
				resolved = r
			}
			resolved = filepath.Clean(resolved)

			detected := detectInstallMethod(resolved)
			chosen := detected
			if method != "" {
				chosen = strings.ToLower(strings.TrimSpace(method))
			}
			switch chosen {
			case installHomebrew, installGo, installManual, "releases", "github":
				if chosen == "releases" || chosen == "github" {
					chosen = installManual
				}
			default:
				return output.Exit(output.ExitUsage, "invalid --method (want homebrew|go|manual)")
			}

			plan := updatePlan(chosen)
			cmdLines := make([]string, 0, len(plan.commands))
			for _, c := range plan.commands {
				cmdLines = append(cmdLines, strings.Join(c, " "))
			}
			data := map[string]any{
				"version":  version,
				"method":   chosen,
				"detected": detected,
				"path":     resolved,
				"commands": cmdLines,
				"dry_run":  dryRun,
			}
			if plan.note != "" {
				data["note"] = plan.note
			}

			forced := method != ""
			app.Out.Info(fmt.Sprintf("current version: %s", version))
			if forced {
				app.Out.Info(fmt.Sprintf("install method:  %s (forced; detected %s)", chosen, detected))
			} else {
				app.Out.Info(fmt.Sprintf("install method:  %s (detected)", chosen))
			}
			app.Out.Info(fmt.Sprintf("path:            %s", resolved))

			if chosen == installManual || dryRun {
				if dryRun && chosen != installManual {
					app.Out.Info("dry-run; would run:")
					for _, line := range cmdLines {
						app.Out.Info("  " + line)
					}
				} else {
					for _, line := range strings.Split(plan.note, "\n") {
						app.Out.Info(line)
					}
					app.Out.Info("")
					app.Out.Info("Releases: " + releasesURL)
					app.Out.Info("Then check with: netcup --version")
				}
				return emitUpdateResult(data)
			}

			for _, argv := range plan.commands {
				app.Out.Info("")
				app.Out.Info("→ " + strings.Join(argv, " "))
				if err := runUpdateCommand(cmd, argv); err != nil {
					return err
				}
			}
			app.Out.Info("")
			app.Out.Info("Done. Check with: netcup --version")
			data["updated"] = true
			return emitUpdateResult(data)
		},
	}
	c.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print the update plan without running it")
	c.Flags().StringVar(&method, "method", "", "force update method: homebrew|go|manual")
	return c
}

func emitUpdateResult(data map[string]any) error {
	if app.Out.Format == output.FormatTable {
		return app.Out.Success(output.TableData{
			Headers: []string{"method", "detected", "version", "path"},
			Rows: [][]string{{
				fmt.Sprint(data["method"]),
				fmt.Sprint(data["detected"]),
				fmt.Sprint(data["version"]),
				fmt.Sprint(data["path"]),
			}},
			Raw: data,
		})
	}
	return app.Out.Success(data)
}

type updatePlanResult struct {
	commands [][]string
	note     string
}

func updatePlan(method string) updatePlanResult {
	switch method {
	case installHomebrew:
		return updatePlanResult{commands: [][]string{
			{"brew", "update"},
			{"brew", "upgrade", brewFormula},
		}}
	case installGo:
		return updatePlanResult{commands: [][]string{
			{"go", "install", goInstallPkg},
		}}
	default:
		return updatePlanResult{
			note: "This binary does not look like a Homebrew or go install.\n" +
				"Download the latest archive from GitHub Releases and replace the binary on your PATH.",
		}
	}
}

func runUpdateCommand(cmd *cobra.Command, argv []string) error {
	if len(argv) == 0 {
		return nil
	}
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return output.Exit(output.ExitUsage, argv[0]+" not found on PATH")
	}
	c := exec.CommandContext(cmd.Context(), bin, argv[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		return output.Exit(output.ExitAPI, fmt.Sprintf("%s failed: %v", strings.Join(argv, " "), err))
	}
	return nil
}

func detectInstallMethod(resolvedPath string) string {
	p := filepath.ToSlash(resolvedPath)
	if isHomebrewPath(p) {
		return installHomebrew
	}
	if isGoInstallPath(resolvedPath) {
		return installGo
	}
	return installManual
}

func isHomebrewPath(slashPath string) bool {
	if strings.Contains(slashPath, "/Cellar/netcup/") {
		return true
	}
	if strings.Contains(slashPath, "/opt/homebrew/") && strings.Contains(slashPath, "/netcup/") {
		return true
	}
	if strings.Contains(slashPath, "/linuxbrew/") && strings.Contains(slashPath, "/netcup/") {
		return true
	}
	if strings.Contains(slashPath, "/usr/local/Cellar/netcup/") {
		return true
	}
	return false
}

func isGoInstallPath(resolvedPath string) bool {
	base := filepath.Base(resolvedPath)
	if base != "netcup" && base != "netcup.exe" {
		return false
	}
	for _, dir := range goBinDirs() {
		if dir == "" {
			continue
		}
		if filepath.Clean(filepath.Dir(resolvedPath)) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

func goBinDirs() []string {
	var dirs []string
	if v := os.Getenv("GOBIN"); v != "" {
		dirs = append(dirs, v)
	}
	if v := os.Getenv("GOPATH"); v != "" {
		for _, p := range filepath.SplitList(v) {
			if p != "" {
				dirs = append(dirs, filepath.Join(p, "bin"))
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, defaultGoBin))
	}
	return dirs
}
