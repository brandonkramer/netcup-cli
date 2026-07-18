package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/output"
	selectserver "github.com/brandonkramer/netcup-cli/internal/select"
	"github.com/spf13/cobra"
)

func newDisksCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "disks", Short: "Server disks"}
	cmd.AddCommand(newDisksListCmd(), newDisksGetCmd(), newDisksDriversCmd(), newDisksSetDriverCmd(), newDisksFormatCmd())
	return cmd
}

func newDisksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "list [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "disks.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdDisksWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("disks.list: empty response")
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("disks.list", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newDisksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "get <diskName> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "disks.get"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			disk := args[0]
			selArgs := []string{}
			if len(args) > 1 {
				selArgs = args[1:]
			}
			id, err := resolveServerArg(cmd.Context(), selArgs)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdDisksDiskNameWithResponse(cmd.Context(), id, disk)
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("disks.get: empty response")
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("disks.get", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newDisksDriversCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "drivers [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "disks.drivers"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdDisksSupportedDriversWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("disks.drivers: empty response")
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("disks.drivers", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newDisksSetDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "set-driver <driver> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "disks.set-driver"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			driver := args[0]
			selArgs := []string{}
			if len(args) > 1 {
				selArgs = args[1:]
			}
			id, err := resolveServerArg(cmd.Context(), selArgs)
			if err != nil {
				return err
			}
			body, _ := json.Marshal(map[string]any{"driver": driver})
			resp, err := app.Client.PatchApiV1ServersServerIdDisksWithBodyWithResponse(cmd.Context(), id, "application/merge-patch+json", bytes.NewReader(body))
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("disks.set-driver: empty response")
			}
			return handleTaskResp(cmd.Context(), "disks.set-driver", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
}

func newDisksFormatCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "format <diskName> [selector]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "disks.format"
			if err := app.Confirm("format disk (ALL DATA WILL BE LOST)"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			disk := args[0]
			selArgs := []string{}
			if len(args) > 1 {
				selArgs = args[1:]
			}
			id, _, err := selectserver.Resolve(cmd.Context(), app.Client, func() string {
				if len(selArgs) > 0 {
					return selArgs[0]
				}
				return app.Flags.Server
			}())
			if err != nil {
				return err
			}
			resp, err := app.Client.PostApiV1ServersServerIdDisksDiskNameFormatWithResponse(cmd.Context(), id, disk)
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("disks.format: empty response")
			}
			return handleTaskResp(cmd.Context(), "disks.format", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
}
