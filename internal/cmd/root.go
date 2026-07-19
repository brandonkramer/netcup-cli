package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	app     = &App{}
	version = "0.1.0-dev"
)

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := NewRoot()
	if err := root.ExecuteContext(ctx); err != nil {
		if isInterrupted(ctx, err) {
			fmt.Fprintln(os.Stderr, "interrupted")
			os.Exit(output.ExitInterrupted)
		}
		var ee *output.ExitError
		if errors.As(err, &ee) {
			if !ee.Rendered && ee.Message != "" {
				renderExitError(ee)
			}
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(output.ExitAPI)
	}
}

func isInterrupted(ctx context.Context, err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	return false
}

func renderExitError(ee *output.ExitError) {
	if app.Out != nil {
		_ = app.Out.WriteError(ee.Code, "cli", "", ee.Message, 0, nil)
		return
	}
	fmt.Fprintln(os.Stderr, "Error: "+ee.Message)
}

func NewRoot() *cobra.Command {
	var jsonFlag bool
	root := &cobra.Command{
		Use:           "netcup",
		Short:         "CLI for the netcup SCP (Server Control Panel) REST API",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if jsonFlag {
				app.Flags.Format = "json"
			}
			return app.Init()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return cmd.Help()
			}
			return runTUI(cmd)
		},
	}

	f := root.PersistentFlags()
	f.StringVar(&app.Flags.Format, "format", "", "output format: table|json|jsonl|yaml|brief")
	f.BoolVarP(&app.Flags.Yes, "yes", "y", false, "skip destructive confirmations")
	f.BoolVar(&app.Flags.NoWait, "no-wait", false, "return TaskInfo immediately on 202")
	f.DurationVar(&app.Flags.WaitTimeout, "wait-timeout", 0, "task wait timeout (default 30m)")
	f.DurationVar(&app.Flags.PollInterval, "poll-interval", 0, "task poll interval (default 2s)")
	f.BoolVar(&app.Flags.NoCache, "no-cache", false, "disable read cache")
	f.DurationVar(&app.Flags.CacheTTL, "cache-ttl", 0, "override cache TTL for this process")
	f.StringVar(&app.Flags.Profile, "profile", "", "credential profile (default: default)")
	f.StringVar(&app.Flags.UserID, "user-id", "", "override SCP userId")
	f.StringVarP(&app.Flags.Server, "server", "s", "", "server selector (id|name|nickname|ip)")
	f.BoolVarP(&app.Flags.Quiet, "quiet", "q", false, "suppress non-essential stderr")
	f.BoolVarP(&app.Flags.Verbose, "verbose", "v", false, "verbose logging")
	f.StringVar(&app.Flags.ConfigDir, "config-dir", "", "config directory (default: ~/.config/netcup)")
	f.BoolVar(&app.Flags.Full, "full", false, "include full API objects in curated views")
	f.BoolVar(&jsonFlag, "json", false, "alias for --format json")

	root.AddCommand(
		newTUICmd(),
		newAuthCmd(),
		newUpdateCmd(),
		newInstallMCPCmd(),
		newMCPCmd(),
		newPingCmd(),
		newMaintenanceCmd(),
		newServersCmd(),
		newDisksCmd(),
		newSnapshotsCmd(),
		newISOCmd(),
		newISOsCmd(),
		newImagesCmd(),
		newNICsCmd(),
		newRDNSCmd(),
		newFailoverCmd(),
		newVLANsCmd(),
		newFirewallCmd(),
		newFirewallPoliciesCmd(),
		newMetricsCmd(),
		newTasksCmd(),
		newUsersCmd(),
		newSSHKeysCmd(),
		newCacheCmd(),
		newSpecCmd(),
		newAPICmd(),
		newCallCmd(),
		newEndpointsCmd(),
		newDescribeCmd(),
		newCompletionCmd(),
	)
	return root
}

func withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 60*time.Second)
}
