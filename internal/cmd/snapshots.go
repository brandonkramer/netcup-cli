package cmd

import (
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/spf13/cobra"
)

func newSnapshotsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "snapshots", Short: "Server snapshots"}
	cmd.AddCommand(
		newSnapshotsListCmd(),
		newSnapshotsGetCmd(),
		newSnapshotsCreateCmd(),
		newSnapshotsDryRunCmd(),
		newSnapshotsDeleteCmd(),
		newSnapshotsExportCmd(),
		newSnapshotsRevertCmd(),
	)
	return cmd
}

func newSnapshotsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "list [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdSnapshotsWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("snapshots.list", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newSnapshotsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "get <name> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.get"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			name := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdSnapshotsNameWithResponse(cmd.Context(), id, name)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("snapshots.get", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newSnapshotsCreateCmd() *cobra.Command {
		var name, disk, description string
	var online bool
	c := &cobra.Command{
		Use:  "create [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.create"
			if name == "" {
				return output.Exit(output.ExitUsage, "--name is required")
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			body := scpclient.ServerSnapshotCreate{Name: name, OnlineSnapshot: &online}
			if disk != "" {
				body.DiskName = &disk
			}
			if description != "" {
				body.Description = &description
			}
			resp, err := app.Client.PostApiV1ServersServerIdSnapshotsWithResponse(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "snapshots.create", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().StringVar(&name, "name", "", "snapshot name (required)")
	c.Flags().StringVar(&disk, "disk", "", "disk name")
	c.Flags().StringVar(&description, "description", "", "description")
	c.Flags().BoolVar(&online, "online", true, "online snapshot")
	_ = c.MarkFlagRequired("name")
	return c
}

func newSnapshotsDryRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "dry-run [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.dry-run"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			body := scpclient.ServerSnapshotCreateCheck{OnlineSnapshot: ptr(true)}
			resp, err := app.Client.PostApiV1ServersServerIdSnapshotsDryrunWithResponse(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return app.HandleAPIError("snapshots.dry-run", resp.StatusCode(), resp.Body)
			}
			data := any(map[string]any{"ok": true})
			if resp.JSON200 != nil {
				data = resp.JSON200
			}
			return app.Out.Success(data, output.WithHTTPStatus(resp.StatusCode()))
		},
	}
}

func newSnapshotsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "delete <name> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.delete"
			if err := app.Confirm("delete snapshot"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			name := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}
			resp, err := app.Client.DeleteApiV1ServersServerIdSnapshotsNameWithResponse(cmd.Context(), id, name)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "snapshots.delete", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
}

func newSnapshotsExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "export <name> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.export"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			name := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}
			resp, err := app.Client.PostApiV1ServersServerIdSnapshotsNameExportWithResponse(cmd.Context(), id, name)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "snapshots.export", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
}

func newSnapshotsRevertCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "revert <name> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "snapshots.revert"
			if err := app.Confirm("revert snapshot"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			name := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}
			resp, err := app.Client.PostApiV1ServersServerIdSnapshotsNameRevertWithResponse(cmd.Context(), id, name)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "snapshots.revert", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
}
