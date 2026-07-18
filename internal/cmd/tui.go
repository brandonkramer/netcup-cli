package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/brandonkramer/netcup-cli/internal/auth"
	"github.com/brandonkramer/netcup-cli/internal/cache"
	"github.com/brandonkramer/netcup-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
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
	opts := app.WaitOpts()
	deps := tui.Deps{
		Cache:        app.Cache,
		APIRoot:      app.Cfg.APIRoot(),
		WaitTimeout:  opts.Timeout,
		PollInterval: opts.PollInterval,
		OnServersMutated: func() {
			if app.Cache != nil {
				app.Cache.BustTags(cache.TagServers)
			}
		},
		ProbeAuth: func(ctx context.Context) error {
			if !app.Auth.HasCredentials() {
				return fmt.Errorf("not logged in")
			}
			if _, err := app.Auth.Refresh(ctx); err != nil {
				return err
			}
			return nil
		},
		Login:     tuiDeviceLogin,
		Reconnect: tuiReconnect,
	}

	// Prefer a live session, but still open the TUI when auth is missing/expired.
	if app.Auth.HasCredentials() {
		if err := app.EnsureClient(cmd.Context()); err != nil {
			deps.InitialAuthReason = err.Error()
		} else {
			deps.Client = app.Client
			if uid, err := app.ResolveUserID(cmd.Context()); err == nil {
				deps.UserID = uid
			}
			if creds := app.Auth.Credentials(); creds != nil {
				deps.Username = creds.Username
			}
		}
	} else {
		deps.InitialAuthReason = "not logged in — press l or run: netcup auth login"
	}

	return tui.Run(cmd.Context(), deps)
}

func tuiReconnect(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		app.Client = nil
		if !app.Auth.HasCredentials() {
			return tui.SessionReady(nil, 0, "", fmt.Errorf("not logged in"))
		}
		if err := app.EnsureClient(ctx); err != nil {
			return tui.SessionReady(nil, 0, "", err)
		}
		uid, err := app.ResolveUserID(ctx)
		if err != nil {
			return tui.SessionReady(app.Client, 0, "", err)
		}
		username := ""
		if creds := app.Auth.Credentials(); creds != nil {
			username = creds.Username
		}
		return tui.SessionReady(app.Client, uid, username, nil)
	}
}

func tuiDeviceLogin(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		dev, err := app.Auth.StartDeviceLogin(ctx)
		if err != nil {
			return tui.SessionReady(nil, 0, "", err)
		}
		_ = auth.OpenURL(dev.VerifyURL)
		return tui.DeviceLoginPrompt(dev.VerifyURL, dev.UserCode, func(waitCtx context.Context) tea.Msg {
			creds, err := dev.Wait(waitCtx, true)
			if err != nil {
				return tui.SessionReady(nil, 0, "", err)
			}
			app.Client = nil
			if err := app.EnsureClient(waitCtx); err != nil {
				return tui.SessionReady(nil, 0, "", err)
			}
			uid, err := app.ResolveUserID(waitCtx)
			if err != nil {
				return tui.SessionReady(app.Client, 0, creds.Username, err)
			}
			return tui.SessionReady(app.Client, uid, creds.Username, nil)
		})
	}
}

func isInteractiveTTY() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}
