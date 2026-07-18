package cmd

import (
	"encoding/json"
	"path/filepath"
	"strconv"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/upload"
	"github.com/spf13/cobra"
)

func newISOCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "iso", Short: "Attached ISO on a server"}
	cmd.AddCommand(newISOGetCmd(), newISOAttachCmd(), newISODetachCmd(), newISOAvailableCmd())
	return cmd
}

func newISOGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "get [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "iso.get"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdIsoWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("iso.get", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newISOAttachCmd() *cobra.Command {
	var isoID int32
	var userIso string
	var bootCDROM bool
	c := &cobra.Command{
		Use:  "attach [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "iso.attach"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			body := scpclient.ServerAttachIso{ChangeBootDeviceToCdrom: &bootCDROM}
			if isoID != 0 {
				body.IsoId = &isoID
			}
			if userIso != "" {
				body.UserIsoName = &userIso
			}
			resp, err := app.Client.PostApiV1ServersServerIdIsoWithResponse(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "iso.attach", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().Int32Var(&isoID, "iso-id", 0, "catalog ISO id")
	c.Flags().StringVar(&userIso, "user-iso", "", "user ISO name")
	c.Flags().BoolVar(&bootCDROM, "boot-cdrom", false, "change boot device to CDROM")
	return c
}

func newISODetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "detach [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "iso.detach"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.DeleteApiV1ServersServerIdIsoWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return app.HandleAPIError("iso.detach", resp.StatusCode(), resp.Body)
			}
			app.Cache.BustTags("servers")
			return app.Out.Success(map[string]any{"detached": true}, output.WithHTTPStatus(resp.StatusCode()))
		},
	}
}

func newISOAvailableCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "available [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "iso.available"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdIsoimagesWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("iso.available", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newISOsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "isos", Short: "User ISO library"}
	cmd.AddCommand(
		newISOsListCmd(),
		newISOsUploadCmd(),
		newISOsDeleteCmd(),
		newISOsDownloadURLCmd(),
		newISOsPrepareUploadCmd(),
		newISOsPartURLCmd(),
		newISOsCompleteUploadCmd(),
	)
	return cmd
}

func newISOsListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1UsersUserIdIsosWithResponse(cmd.Context(), uid)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("isos.list", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newISOsUploadCmd() *cobra.Command {
	var key string
	var partSize int64
	c := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a user ISO (multipart S3)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.upload"
			path := args[0]
			if key == "" {
				key = filepath.Base(path)
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			res, err := upload.File(cmd.Context(), s3API(s3ISO, uid), key, path, partSize, uploadProgress())
			if err != nil {
				return err
			}
			return app.Out.Success(res, output.WithHTTPStatus(201))
		},
	}
	c.Flags().StringVar(&key, "key", "", "object key (default: filename)")
	c.Flags().Int64Var(&partSize, "part-size", upload.DefaultPartSize, "multipart part size in bytes")
	return c
}

func newISOsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "delete <key>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.delete"
			if err := app.Confirm("delete ISO"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.DeleteApiV1UsersUserIdIsosKeyWithResponse(cmd.Context(), uid, args[0])
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return app.HandleAPIError("isos.delete", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(map[string]any{"deleted": args[0]}, output.WithHTTPStatus(resp.StatusCode()))
		},
	}
}

func newISOsDownloadURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "download-url <key>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.download-url"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1UsersUserIdIsosKeyWithResponse(cmd.Context(), uid, args[0])
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("isos.download-url", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newISOsPrepareUploadCmd() *cobra.Command {
	var multipart bool
	c := &cobra.Command{
		Use:   "prepare-upload <key>",
		Short: "Prepare ISO upload (returns uploadId or presignedUrl)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.prepare-upload"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			up, err := s3API(s3ISO, uid).Prepare(cmd.Context(), args[0], multipart)
			if err != nil {
				return err
			}
			return app.Out.Success(up, output.WithHTTPStatus(201))
		},
	}
	c.Flags().BoolVar(&multipart, "multipart", true, "multipart upload")
	return c
}

func newISOsPartURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "part-url <key> <uploadId> <partNumber>",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.part-url"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			n, err := strconv.ParseInt(args[2], 10, 32)
			if err != nil {
				return output.Exit(output.ExitUsage, "invalid part number")
			}
			url, err := s3API(s3ISO, uid).PartURL(cmd.Context(), args[0], args[1], int32(n))
			if err != nil {
				return err
			}
			return app.Out.Success(map[string]any{"url": url}, output.WithHTTPStatus(200))
		},
	}
}

func newISOsCompleteUploadCmd() *cobra.Command {
	var bodyFile string
	c := &cobra.Command{
		Use:   "complete-upload <key> <uploadId>",
		Short: "Complete multipart ISO upload with parts JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "isos.complete-upload"
			if bodyFile == "" {
				return output.Exit(output.ExitUsage, "--body with JSON array of {ETag,partNumber} required")
			}
			raw, err := readBodyArg(bodyFile)
			if err != nil {
				return err
			}
			var parts []scpclient.S3CompletedPart
			if err := json.Unmarshal(raw, &parts); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			if err := s3API(s3ISO, uid).Complete(cmd.Context(), args[0], args[1], parts); err != nil {
				return err
			}
			return app.Out.Success(map[string]any{"completed": true, "parts": len(parts)}, output.WithHTTPStatus(200))
		},
	}
	c.Flags().StringVar(&bodyFile, "body", "", "JSON parts array or @file")
	return c
}
