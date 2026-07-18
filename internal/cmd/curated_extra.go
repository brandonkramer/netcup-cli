package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/spf13/cobra"
)

func newNICsCreateCmd() *cobra.Command {
	var (
		vlanID int32
		driver string
	)
	c := &cobra.Command{
		Use:   "create [selector]",
		Short: "Create a VLAN NIC on a server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "nics.create"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			if vlanID == 0 {
				return output.Exit(output.ExitUsage, "--vlan-id is required")
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			body := scpclient.ServerCreateNicVlan{
				VlanId:        vlanID,
				NetworkDriver: scpclient.NetworkDriver(driver),
			}
			raw, err := json.Marshal(body)
			if err != nil {
				return err
			}
			resp, err := app.Client.PostApiV1ServersServerIdInterfacesWithBodyWithResponse(
				cmd.Context(), id, "application/json", bytes.NewReader(raw),
			)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "nics.create", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().Int32Var(&vlanID, "vlan-id", 0, "VLAN id")
	c.Flags().StringVar(&driver, "driver", string(scpclient.NetworkDriverVIRTIO), "network driver (VIRTIO, E1000, E1000E, RTL8139, VMXNET3)")
	return c
}

func newNICsUpdateCmd() *cobra.Command {
	var driver string
	c := &cobra.Command{
		Use:   "update <mac> [selector]",
		Short: "Update a server network interface",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "nics.update"
			if driver == "" {
				return output.Exit(output.ExitUsage, "--driver is required")
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			mac := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}
			body := scpclient.ServerInterfaceUpdate{
				Driver: ptr(scpclient.NetworkDriver(driver)),
			}
			resp, err := app.Client.PutApiV1ServersServerIdInterfacesMacWithResponse(cmd.Context(), id, mac, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "nics.update", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().StringVar(&driver, "driver", "", "network driver (VIRTIO, E1000, E1000E, RTL8139, VMXNET3)")
	return c
}

func newFirewallSetCmd() *cobra.Command {
	var (
		bodyFile       string
		active         bool
		userPolicies   []int32
		copiedPolicies []int32
	)
	c := &cobra.Command{
		Use:   "set <mac> [selector]",
		Short: "Set server interface firewall policies",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "firewall.set"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			mac := args[0]
			id, err := resolveServerArg(cmd.Context(), args[1:])
			if err != nil {
				return err
			}

			var body scpclient.ServerFirewallSave
			if bodyFile != "" {
				raw, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return err
				}
			} else {
				body.UserPolicies = idsToIdentifiers(userPolicies)
				body.CopiedPolicies = idsToIdentifiers(copiedPolicies)
				if cmd.Flags().Changed("active") {
					body.Active = &active
				}
			}

			resp, err := app.Client.PutApiV1ServersServerIdInterfacesMacFirewallWithResponse(cmd.Context(), id, mac, body)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "firewall.set", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().StringVar(&bodyFile, "body", "", "full JSON body or @file (overrides flags)")
	c.Flags().BoolVar(&active, "active", false, "firewall active state")
	c.Flags().Int32SliceVar(&userPolicies, "user-policy", nil, "user policy ids (repeatable)")
	c.Flags().Int32SliceVar(&copiedPolicies, "copied-policy", nil, "copied policy ids (repeatable)")
	return c
}

func newFirewallPoliciesCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		bodyFile    string
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a user firewall policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "firewall-policies.create"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}

			var body scpclient.FirewallPolicySave
			if bodyFile != "" {
				raw, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return err
				}
			} else {
				if name == "" {
					return output.Exit(output.ExitUsage, "--name is required (or --body)")
				}
				body.Name = name
				if description != "" {
					body.Description = &description
				}
			}

			resp, err := app.Client.PostApiV1UsersUserIdFirewallPoliciesWithResponse(cmd.Context(), uid, body)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 201 {
				return app.HandleAPIError("firewall-policies.create", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON201, resp.HALJSON201, resp.Body), output.WithHTTPStatus(201))
		},
	}
	c.Flags().StringVar(&name, "name", "", "policy name")
	c.Flags().StringVar(&description, "description", "", "policy description")
	c.Flags().StringVar(&bodyFile, "body", "", "full JSON body or @file (overrides flags)")
	return c
}

func newFirewallPoliciesUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		bodyFile    string
	)
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user firewall policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "firewall-policies.update"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			pid, err := parseInt32(args[0])
			if err != nil {
				return err
			}

			var body scpclient.FirewallPolicySave
			if bodyFile != "" {
				raw, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return err
				}
			} else {
				if name == "" {
					return output.Exit(output.ExitUsage, "--name is required (or --body)")
				}
				body.Name = name
				if description != "" {
					body.Description = &description
				}
			}

			resp, err := app.Client.PutApiV1UsersUserIdFirewallPoliciesIdWithResponse(cmd.Context(), uid, pid, body)
			if err != nil {
				return err
			}
			status := resp.StatusCode()
			if status == 202 {
				return handleTaskResp(
					cmd.Context(),
					"firewall-policies.update",
					status,
					taskFromFirewallPolicyResult(resp.HALJSON202, resp.JSON202),
					resp.Body,
				)
			}
			if status != 200 && status != 204 {
				return app.HandleAPIError("firewall-policies.update", status, resp.Body)
			}
			return app.Out.Success(firstJSON(resp.HALJSON202, resp.JSON202, resp.Body), output.WithHTTPStatus(status))
		},
	}
	c.Flags().StringVar(&name, "name", "", "policy name")
	c.Flags().StringVar(&description, "description", "", "policy description")
	c.Flags().StringVar(&bodyFile, "body", "", "full JSON body or @file (overrides flags)")
	return c
}

func newVLANsUpdateCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "update <vlanId>",
		Short: "Update a user VLAN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "vlans.update"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			vid, err := parseInt32(args[0])
			if err != nil {
				return err
			}
			body := scpclient.VLanSave{}
			if name != "" {
				body.Name = &name
			}
			resp, err := app.Client.PutApiV1UsersUserIdVlansVlanIdWithResponse(cmd.Context(), uid, vid, body)
			if err != nil {
				return err
			}
			status := resp.StatusCode()
			if status == 202 {
				return handleTaskResp(cmd.Context(), "vlans.update", status, firstTask(nil, nil), resp.Body)
			}
			if status != 200 && status != 204 {
				return app.HandleAPIError("vlans.update", status, resp.Body)
			}
			return app.Out.Success(map[string]any{"vlanId": vid, "name": name}, output.WithHTTPStatus(status))
		},
	}
	c.Flags().StringVar(&name, "name", "", "VLAN name")
	return c
}

func newUsersUpdateCmd() *cobra.Command {
	var (
		bodyFile        string
		language        string
		timezone        string
		apiIPRestrict   string
		secureMode      bool
		showNickname    bool
		passwordless    bool
		password        string
		oldPassword     string
	)
	c := &cobra.Command{
		Use:   "update",
		Short: "Update the current user profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "users.update"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}

			var body scpclient.UserSave
			if bodyFile != "" {
				raw, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return err
				}
			} else {
				if language == "" || timezone == "" {
					return output.Exit(output.ExitUsage, "--language and --timezone are required (or --body)")
				}
				body.Language = language
				body.TimeZone = timezone
				if apiIPRestrict != "" {
					body.ApiIpLoginRestrictions = &apiIPRestrict
				}
				if cmd.Flags().Changed("secure-mode") {
					body.SecureMode = &secureMode
				}
				if cmd.Flags().Changed("show-nickname") {
					body.ShowNickname = &showNickname
				}
				if cmd.Flags().Changed("passwordless") {
					body.PasswordlessMode = &passwordless
				}
				if password != "" {
					body.Password = &password
				}
				if oldPassword != "" {
					body.OldPassword = &oldPassword
				}
			}

			resp, err := app.Client.PutApiV1UsersUserIdWithResponse(cmd.Context(), uid, body)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("users.update", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
	c.Flags().StringVar(&bodyFile, "body", "", "full JSON body or @file (overrides flags)")
	c.Flags().StringVar(&language, "language", "", "user language")
	c.Flags().StringVar(&timezone, "timezone", "", "user time zone")
	c.Flags().StringVar(&apiIPRestrict, "api-ip-restrictions", "", "API IP login restrictions")
	c.Flags().BoolVar(&secureMode, "secure-mode", false, "secure mode")
	c.Flags().BoolVar(&showNickname, "show-nickname", false, "show nickname")
	c.Flags().BoolVar(&passwordless, "passwordless", false, "passwordless mode")
	c.Flags().StringVar(&password, "password", "", "new password")
	c.Flags().StringVar(&oldPassword, "old-password", "", "current password")
	return c
}

func newSSHKeysAddCmd() *cobra.Command {
	var (
		name    string
		key     string
		keyFile string
	)
	c := &cobra.Command{
		Use:   "add",
		Short: "Add an SSH public key",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "ssh-keys.add"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			if name == "" {
				return output.Exit(output.ExitUsage, "--name is required")
			}
			keyMaterial := strings.TrimSpace(key)
			if keyFile != "" {
				b, err := os.ReadFile(keyFile)
				if err != nil {
					return err
				}
				keyMaterial = strings.TrimSpace(string(b))
			}
			if keyMaterial == "" {
				return output.Exit(output.ExitUsage, "--key or --key-file is required")
			}

			body := scpclient.SSHKey{Name: name, Key: keyMaterial}
			resp, err := app.Client.PostApiV1UsersUserIdSshKeysWithResponse(cmd.Context(), uid, body)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 201 {
				return app.HandleAPIError("ssh-keys.add", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON201, resp.HALJSON201, resp.Body), output.WithHTTPStatus(201))
		},
	}
	c.Flags().StringVar(&name, "name", "", "key name")
	c.Flags().StringVar(&key, "key", "", "public key material")
	c.Flags().StringVar(&keyFile, "key-file", "", "path to public key file")
	return c
}

func idsToIdentifiers(ids []int32) []scpclient.IdentifierInt {
	out := make([]scpclient.IdentifierInt, len(ids))
	for i, id := range ids {
		out[i] = scpclient.IdentifierInt{Id: id}
	}
	return out
}

func taskFromFirewallPolicyResult(results ...*scpclient.FirewallPolicyUpdateResult) *scpclient.TaskInfo {
	for _, r := range results {
		if r != nil && r.TaskInfo != nil {
			return r.TaskInfo
		}
	}
	return nil
}
