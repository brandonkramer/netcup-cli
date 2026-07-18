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

func newImagesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "images", Short: "Images and flavours"}
	cmd.AddCommand(
		newImagesFlavoursCmd(),
		newImagesSetupCmd(),
		newImagesSetupUserCmd(),
		newImagesListCmd(),
		newImagesUploadCmd(),
		newImagesDeleteCmd(),
		newImagesDownloadURLCmd(),
		newImagesPrepareUploadCmd(),
		newImagesPartURLCmd(),
		newImagesCompleteUploadCmd(),
	)
	return cmd
}

func newImagesFlavoursCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "flavours [selector]",
		Short: "List image flavours for a server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.flavours"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdImageflavoursWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("images.flavours", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newImagesSetupCmd() *cobra.Command {
	var (
		flavourID                             int32
		disk, hostname, locale, timezone      string
		user, password, customScript, bodyFile string
		sshKeys                               []int32
		rootFull, email, sshPassAuth bool
	)
	c := &cobra.Command{
		Use:   "setup [selector]",
		Short: "Setup OS image on a server (destructive)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.setup"
			if err := app.Confirm("setup image (ALL DATA WILL BE LOST)"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}

			var body scpclient.ServerImageSetup
			if bodyFile != "" {
				raw, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return err
				}
			} else {
				if flavourID == 0 {
					return output.Exit(output.ExitUsage, "--flavour-id is required (or --body)")
				}
				body.ImageFlavourId = &flavourID
				if disk != "" {
					body.DiskName = &disk
				}
				if hostname != "" {
					body.Hostname = &hostname
				}
				if locale != "" {
					body.Locale = &locale
				}
				if timezone != "" {
					body.Timezone = &timezone
				}
				if user != "" {
					body.AdditionalUserUsername = &user
				}
				if password != "" {
					body.AdditionalUserPassword = &password
				}
				if customScript != "" {
					body.CustomScript = &customScript
				}
				if len(sshKeys) > 0 {
					body.SshKeyIds = &sshKeys
				}
				if cmd.Flags().Changed("root-full-disk") {
					body.RootPartitionFullDiskSize = &rootFull
				}
				if cmd.Flags().Changed("email") {
					body.EmailToExecutingUser = &email
				}
				if cmd.Flags().Changed("ssh-password-auth") {
					body.SshPasswordAuthentication = &sshPassAuth
				}
			}

			resp, err := app.Client.PostApiV1ServersServerIdImageWithResponse(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "images.setup", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().Int32Var(&flavourID, "flavour-id", 0, "imageFlavourId")
	c.Flags().StringVar(&disk, "disk", "", "target disk name")
	c.Flags().StringVar(&hostname, "hostname", "", "hostname")
	c.Flags().StringVar(&locale, "locale", "", "locale")
	c.Flags().StringVar(&timezone, "timezone", "", "timezone")
	c.Flags().StringVar(&user, "user", "", "additional user username")
	c.Flags().StringVar(&password, "password", "", "additional user password")
	c.Flags().StringVar(&customScript, "custom-script", "", "cloud-init / custom script")
	c.Flags().Int32SliceVar(&sshKeys, "ssh-key-id", nil, "SSH key ids (repeatable)")
	c.Flags().BoolVar(&rootFull, "root-full-disk", false, "root partition uses full disk")
	c.Flags().BoolVar(&email, "email", false, "email executing user")
	c.Flags().BoolVar(&sshPassAuth, "ssh-password-auth", false, "enable SSH password auth")
	c.Flags().StringVar(&bodyFile, "body", "", "full JSON body or @file (overrides flags)")
	return c
}

func newImagesSetupUserCmd() *cobra.Command {
	var name, disk string
	var email bool
	c := &cobra.Command{
		Use:   "setup-user [selector]",
		Short: "Setup a user image on a server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.setup-user"
			if name == "" {
				return output.Exit(output.ExitUsage, "--name is required")
			}
			if err := app.Confirm("setup user image (data on selected disk may be lost)"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			body := scpclient.ServerUserImageSetup{UserImageName: name, EmailNotification: &email}
			if disk != "" {
				body.DiskName = &disk
			}
			resp, err := app.Client.PostApiV1ServersServerIdUserImageWithResponse(cmd.Context(), id, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "images.setup-user", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().StringVar(&name, "name", "", "user image name/key (required)")
	c.Flags().StringVar(&disk, "disk", "", "target disk name")
	c.Flags().BoolVar(&email, "email", false, "email notification")
	_ = c.MarkFlagRequired("name")
	return c
}

func newImagesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List user images",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1UsersUserIdImagesWithResponse(cmd.Context(), uid)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("images.list", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newImagesUploadCmd() *cobra.Command {
	var key string
	var partSize int64
	c := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a user image (multipart S3)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.upload"
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
			res, err := upload.File(cmd.Context(), s3API(s3Image, uid), key, path, partSize, uploadProgress())
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

func newImagesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.delete"
			if err := app.Confirm("delete user image"); err != nil {
				return err
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.DeleteApiV1UsersUserIdImagesKeyWithResponse(cmd.Context(), uid, args[0])
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return app.HandleAPIError("images.delete", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(map[string]any{"deleted": args[0]}, output.WithHTTPStatus(resp.StatusCode()))
		},
	}
}

func newImagesDownloadURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download-url <key>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.download-url"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1UsersUserIdImagesKeyWithResponse(cmd.Context(), uid, args[0])
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("images.download-url", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
}

func newImagesPrepareUploadCmd() *cobra.Command {
	var multipart bool
	c := &cobra.Command{
		Use:   "prepare-upload <key>",
		Short: "Prepare image upload (returns uploadId or presignedUrl)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.prepare-upload"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			up, err := s3API(s3Image, uid).Prepare(cmd.Context(), args[0], multipart)
			if err != nil {
				return err
			}
			return app.Out.Success(up, output.WithHTTPStatus(201))
		},
	}
	c.Flags().BoolVar(&multipart, "multipart", true, "multipart upload")
	return c
}

func newImagesPartURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "part-url <key> <uploadId> <partNumber>",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.part-url"
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
			url, err := s3API(s3Image, uid).PartURL(cmd.Context(), args[0], args[1], int32(n))
			if err != nil {
				return err
			}
			return app.Out.Success(map[string]any{"url": url}, output.WithHTTPStatus(200))
		},
	}
}

func newImagesCompleteUploadCmd() *cobra.Command {
	var bodyFile string
	c := &cobra.Command{
		Use:   "complete-upload <key> <uploadId>",
		Short: "Complete multipart image upload with parts JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "images.complete-upload"
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
			if err := s3API(s3Image, uid).Complete(cmd.Context(), args[0], args[1], parts); err != nil {
				return err
			}
			return app.Out.Success(map[string]any{"completed": true, "parts": len(parts)}, output.WithHTTPStatus(200))
		},
	}
	c.Flags().StringVar(&bodyFile, "body", "", "JSON parts array or @file")
	return c
}
