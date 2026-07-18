package cmd

import (
	"os"

	"github.com/brandonkramer/netcup-cli/internal/tui"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive server operations TUI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd)
		},
	}
}

func runTUI(cmd *cobra.Command) error {
	if err := app.EnsureClient(cmd.Context()); err != nil {
		return err
	}
	uid, err := app.ResolveUserID()
	if err != nil {
		return err
	}
	opts := app.WaitOpts()
	return tui.Run(cmd.Context(), tui.Deps{
		Client:       app.Client,
		UserID:       uid,
		WaitTimeout:  opts.Timeout,
		PollInterval: opts.PollInterval,
		OnServersMutated: func() {
			app.Cache.BustTags("servers")
		},
	})
}

func isInteractiveTTY() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}
