package cmd

import (
	"context"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Authentication"}
	cmd.AddCommand(newAuthLoginCmd(), newAuthLogoutCmd(), newAuthStatusCmd(), newAuthWhoamiCmd(), newAuthRefreshCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var noBrowser, noSave bool
	c := &cobra.Command{
		Use:   "login",
		Short: "Log in via OAuth device code",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "auth.login"
			creds, err := app.Auth.Login(cmd.Context(), !noBrowser, !noSave)
			if err != nil {
				return app.Out.Fail(output.ExitAuth, "auth", "", err.Error(), 0, nil)
			}
			return app.Out.Success(map[string]any{
				"user_id":  creds.UserID,
				"username": creds.Username,
				"saved":    !noSave,
			})
		},
	}
	c.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open a browser")
	c.Flags().BoolVar(&noSave, "no-save", false, "do not persist refresh token")
	return c
}

func newAuthLogoutCmd() *cobra.Command {
	var revoke bool
	c := &cobra.Command{
		Use:   "logout",
		Short: "Remove local credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "auth.logout"
			if err := app.Auth.Logout(revoke); err != nil {
				return err
			}
			return app.Out.Success(map[string]any{"logged_out": true})
		},
	}
	c.Flags().BoolVar(&revoke, "revoke", false, "attempt remote revoke (best-effort)")
	return c
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show login status",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "auth.status"
			st, err := app.Auth.Status(cmd.Context())
			if err != nil {
				return err
			}
			if app.Out.Format == output.FormatTable {
				return app.Out.Success(output.TableData{
					Headers: []string{"logged_in", "user_id", "username", "refresh_age"},
					Rows: [][]string{{
						fmtBool(st["logged_in"]),
						fmtAny(st["user_id"]),
						fmtAny(st["username"]),
						fmtAny(st["refresh_age"]),
					}},
					Raw: st,
				})
			}
			return app.Out.Success(st)
		},
	}
}

func newAuthWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "GET current user profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "auth.whoami"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1UsersUserIdWithResponse(cmd.Context(), uid)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 || resp.JSON200 == nil {
				return app.HandleAPIError("auth.whoami", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newAuthRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Force access token refresh",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "auth.refresh"
			tok, err := app.Auth.Refresh(context.Background())
			if err != nil {
				return app.Out.Fail(output.ExitAuth, "auth", "", err.Error(), 0, nil)
			}
			return app.Out.Success(map[string]any{
				"expires_at": tok.Expiry,
				"token_type": tok.TokenType,
			})
		},
	}
}
