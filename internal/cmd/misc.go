package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/apiraw"
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	selectserver "github.com/brandonkramer/netcup-cli/internal/select"
	"github.com/spf13/cobra"
)

func newNICsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "nics", Short: "Network interfaces"}
	cmd.AddCommand(
		&cobra.Command{
			Use:  "list [selector]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "nics.list"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				id, err := resolveServerArg(cmd.Context(), args)
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1ServersServerIdInterfacesWithResponse(cmd.Context(), id, &scpclient.GetApiV1ServersServerIdInterfacesParams{LoadRdns: ptr(true)})
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("nics.list", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "get <mac> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "nics.get"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				mac := args[0]
				id, err := resolveServerArg(cmd.Context(), args[1:])
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1ServersServerIdInterfacesMacWithResponse(cmd.Context(), id, mac, nil)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("nics.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		newNICsCreateCmd(),
		newNICsUpdateCmd(),
		&cobra.Command{
			Use:  "delete <mac> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "nics.delete"
				if err := app.Confirm("delete NIC"); err != nil {
					return err
				}
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				mac := args[0]
				id, err := resolveServerArg(cmd.Context(), args[1:])
				if err != nil {
					return err
				}
				resp, err := app.Client.DeleteApiV1ServersServerIdInterfacesMacWithResponse(cmd.Context(), id, mac)
				if err != nil {
					return err
				}
				return handleTaskResp(cmd.Context(), "nics.delete", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
			},
		},
	)
	return cmd
}

func newRDNSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rdns", Short: "Reverse DNS"}
	ipv4 := &cobra.Command{Use: "ipv4", Short: "IPv4 rDNS"}
	ipv4.AddCommand(
		&cobra.Command{
			Use:  "get <ip>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv4.get"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1RdnsIpv4IpWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("rdns.ipv4.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "set <ip> <rdns>",
			Args: cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv4.set"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				body := scpclient.SetRdnsIpv4{Ip: args[0], Rdns: args[1]}
				resp, err := app.Client.PostApiV1RdnsIpv4WithResponse(cmd.Context(), body)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 201 && resp.StatusCode() != 204 {
					return app.HandleAPIError("rdns.ipv4.set", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"ip": args[0], "rdns": args[1]}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
		&cobra.Command{
			Use:  "delete <ip>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv4.delete"
				if err := app.Confirm("delete rDNS"); err != nil {
					return err
				}
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.DeleteApiV1RdnsIpv4IpWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return app.HandleAPIError("rdns.ipv4.delete", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"deleted": args[0]}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
	)
	ipv6 := &cobra.Command{Use: "ipv6", Short: "IPv6 rDNS"}
	ipv6.AddCommand(
		&cobra.Command{
			Use:  "get <ip>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv6.get"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1RdnsIpv6IpWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("rdns.ipv6.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "set <ip> <rdns>",
			Args: cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv6.set"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				body := scpclient.SetRdnsIpv6{Ip: args[0], Rdns: args[1]}
				resp, err := app.Client.PostApiV1RdnsIpv6WithResponse(cmd.Context(), body)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 201 && resp.StatusCode() != 204 {
					return app.HandleAPIError("rdns.ipv6.set", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"ip": args[0], "rdns": args[1]}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
		&cobra.Command{
			Use:  "delete <ip>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "rdns.ipv6.delete"
				if err := app.Confirm("delete rDNS"); err != nil {
					return err
				}
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.DeleteApiV1RdnsIpv6IpWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return app.HandleAPIError("rdns.ipv6.delete", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"deleted": args[0]}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
	)
	cmd.AddCommand(ipv4, ipv6)
	return cmd
}

func newFailoverCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "failover", Short: "Failover IPs"}
	for _, ver := range []string{"ipv4", "ipv6"} {
		v := ver
		sub := &cobra.Command{Use: v}
		sub.AddCommand(&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "failover." + v + ".list"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				if v == "ipv4" {
					resp, err := app.Client.GetApiV1UsersUserIdFailoveripsV4WithResponse(cmd.Context(), uid, nil)
					if err != nil {
						return err
					}
					if resp.StatusCode() != 200 {
						return app.HandleAPIError(app.Out.Command, resp.StatusCode(), resp.Body)
					}
					return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
				}
				resp, err := app.Client.GetApiV1UsersUserIdFailoveripsV6WithResponse(cmd.Context(), uid, nil)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError(app.Out.Command, resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		})
		sub.AddCommand(&cobra.Command{
			Use:  "route <id> <serverId>",
			Args: cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "failover." + v + ".route"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				fid, err := parseInt32(args[0])
				if err != nil {
					return err
				}
				sid, err := parseInt32(args[1])
				if err != nil {
					return err
				}
				body, _ := json.Marshal(scpclient.RouteFailoverIp{ServerId: &sid})
				if v == "ipv4" {
					resp, err := app.Client.PatchApiV1UsersUserIdFailoveripsV4IdWithBodyWithResponse(cmd.Context(), uid, fid, "application/merge-patch+json", bytes.NewReader(body))
					if err != nil {
						return err
					}
					return handleTaskResp(cmd.Context(), app.Out.Command, resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
				}
				resp, err := app.Client.PatchApiV1UsersUserIdFailoveripsV6IdWithBodyWithResponse(cmd.Context(), uid, fid, "application/merge-patch+json", bytes.NewReader(body))
				if err != nil {
					return err
				}
				return handleTaskResp(cmd.Context(), app.Out.Command, resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
			},
		})
		cmd.AddCommand(sub)
	}
	return cmd
}

func newVLANsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vlans", Short: "VLANs"}
	cmd.AddCommand(
		newVLANsUpdateCmd(),
		&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "vlans.list"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1UsersUserIdVlansWithResponse(cmd.Context(), uid, nil)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("vlans.list", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "get <vlanId>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "vlans.get"
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
				resp, err := app.Client.GetApiV1UsersUserIdVlansVlanIdWithResponse(cmd.Context(), uid, vid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("vlans.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "get-global <vlanId>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "vlans.get-global"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				vid, err := parseInt32(args[0])
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1VlansVlanIdWithResponse(cmd.Context(), vid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("vlans.get-global", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
	)
	return cmd
}

func newFirewallCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "firewall", Short: "Server interface firewall"}
	cmd.AddCommand(
		newFirewallSetCmd(),
		&cobra.Command{
			Use:  "get <mac> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall.get"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				mac := args[0]
				id, err := resolveServerArg(cmd.Context(), args[1:])
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1ServersServerIdInterfacesMacFirewallWithResponse(cmd.Context(), id, mac, nil)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("firewall.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "reapply <mac> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall.reapply"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				mac := args[0]
				id, err := resolveServerArg(cmd.Context(), args[1:])
				if err != nil {
					return err
				}
				resp, err := app.Client.PostApiV1ServersServerIdInterfacesMacFirewallReapplyWithResponse(cmd.Context(), id, mac)
				if err != nil {
					return err
				}
				return handleTaskResp(cmd.Context(), "firewall.reapply", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
			},
		},
		&cobra.Command{
			Use:  "restore-copied <mac> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall.restore-copied"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				mac := args[0]
				id, err := resolveServerArg(cmd.Context(), args[1:])
				if err != nil {
					return err
				}
				resp, err := app.Client.PostApiV1ServersServerIdInterfacesMacFirewallRestoreCopiedPoliciesWithResponse(cmd.Context(), id, mac)
				if err != nil {
					return err
				}
				return handleTaskResp(cmd.Context(), "firewall.restore-copied", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
			},
		},
	)
	return cmd
}

func newFirewallPoliciesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "firewall-policies", Short: "User firewall policies"}
	cmd.AddCommand(
		newFirewallPoliciesCreateCmd(),
		newFirewallPoliciesUpdateCmd(),
		&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall-policies.list"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1UsersUserIdFirewallPoliciesWithResponse(cmd.Context(), uid, nil)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("firewall-policies.list", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "get <id>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall-policies.get"
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
				resp, err := app.Client.GetApiV1UsersUserIdFirewallPoliciesIdWithResponse(cmd.Context(), uid, pid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("firewall-policies.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "delete <id>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "firewall-policies.delete"
				if err := app.Confirm("delete firewall policy"); err != nil {
					return err
				}
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
				resp, err := app.Client.DeleteApiV1UsersUserIdFirewallPoliciesIdWithResponse(cmd.Context(), uid, pid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return app.HandleAPIError("firewall-policies.delete", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"deleted": pid}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
	)
	return cmd
}

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "metrics", Short: "Server metrics (uncached)"}
	for _, kind := range []string{"cpu", "disk", "network", "packets"} {
		k := kind
		var hours int32
		c := &cobra.Command{
			Use:  k + " [selector]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "metrics." + k
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				id, err := resolveServerArg(cmd.Context(), args)
				if err != nil {
					return err
				}
				var hoursPtr *int32
				if cmd.Flags().Changed("hours") {
					hoursPtr = &hours
				}
				var status int
				var body []byte
				var data any
				switch k {
				case "cpu":
					resp, err := app.Client.GetApiV1ServersServerIdMetricsCpuWithResponse(cmd.Context(), id, &scpclient.GetApiV1ServersServerIdMetricsCpuParams{Hours: hoursPtr})
					if err != nil {
						return err
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				case "disk":
					resp, err := app.Client.GetApiV1ServersServerIdMetricsDiskWithResponse(cmd.Context(), id, &scpclient.GetApiV1ServersServerIdMetricsDiskParams{Hours: hoursPtr})
					if err != nil {
						return err
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				case "network":
					resp, err := app.Client.GetApiV1ServersServerIdMetricsNetworkWithResponse(cmd.Context(), id, &scpclient.GetApiV1ServersServerIdMetricsNetworkParams{Hours: hoursPtr})
					if err != nil {
						return err
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				case "packets":
					resp, err := app.Client.GetApiV1ServersServerIdMetricsNetworkPacketWithResponse(cmd.Context(), id, &scpclient.GetApiV1ServersServerIdMetricsNetworkPacketParams{Hours: hoursPtr})
					if err != nil {
						return err
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
				if status != 200 {
					return app.HandleAPIError(app.Out.Command, status, body)
				}
				return app.Out.Success(data, output.WithHTTPStatus(200))
			},
		}
		c.Flags().Int32Var(&hours, "hours", 0, "metrics window in hours")
		cmd.AddCommand(c)
	}
	return cmd
}

func newTasksCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tasks", Short: "Async tasks"}
	cmd.AddCommand(
		newTasksListCmd(),
		&cobra.Command{
			Use:  "get <uuid>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "tasks.get"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1TasksUuidWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("tasks.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "cancel <uuid>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "tasks.cancel"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.PutApiV1TasksUuidCancelWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return handleTaskResp(cmd.Context(), "tasks.cancel", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
			},
		},
	)
	return cmd
}

func newTasksListCmd() *cobra.Command {
	var q, state string
	var serverID, limit, offset int32
	c := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "tasks.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			params := &scpclient.GetApiV1TasksParams{}
			if q != "" {
				params.Q = &q
			}
			if state != "" {
				st := scpclient.TaskState(state)
				params.State = &st
			}
			if cmd.Flags().Changed("server-id") {
				params.ServerId = &serverID
			}
			if cmd.Flags().Changed("limit") {
				params.Limit = &limit
			}
			if cmd.Flags().Changed("offset") {
				params.Offset = &offset
			}
			resp, err := app.Client.GetApiV1TasksWithResponse(cmd.Context(), params)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("tasks.list", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
	c.Flags().StringVar(&q, "q", "", "search name/uuid/server")
	c.Flags().StringVar(&state, "state", "", "filter by state (PENDING, RUNNING, FINISHED, ERROR, …)")
	c.Flags().Int32Var(&serverID, "server-id", 0, "filter by server id")
	c.Flags().Int32Var(&limit, "limit", 0, "page size")
	c.Flags().Int32Var(&offset, "offset", 0, "page offset")
	return c
}

func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "users", Short: "User profile"}
	cmd.AddCommand(
		newUsersUpdateCmd(),
		&cobra.Command{
			Use: "get",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "users.get"
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
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("users.get", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		newUsersLogsCmd(),
	)
	return cmd
}

func newUsersLogsCmd() *cobra.Command {
	var limit, offset int32
	c := &cobra.Command{
		Use:   "logs",
		Short: "User logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "users.logs"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, err := app.ResolveUserID()
			if err != nil {
				return err
			}
			params := &scpclient.GetApiV1UsersUserIdLogsParams{}
			if cmd.Flags().Changed("limit") {
				params.Limit = &limit
			}
			if cmd.Flags().Changed("offset") {
				params.Offset = &offset
			}
			resp, err := app.Client.GetApiV1UsersUserIdLogsWithResponse(cmd.Context(), uid, params)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("users.logs", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
	c.Flags().Int32Var(&limit, "limit", 0, "page size")
	c.Flags().Int32Var(&offset, "offset", 0, "page offset")
	return c
}

func newSSHKeysCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ssh-keys", Short: "SSH keys"}
	cmd.AddCommand(
		newSSHKeysAddCmd(),
		&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "ssh-keys.list"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1UsersUserIdSshKeysWithResponse(cmd.Context(), uid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("ssh-keys.list", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:  "delete <id>",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "ssh-keys.delete"
				if err := app.Confirm("delete SSH key"); err != nil {
					return err
				}
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				uid, err := app.ResolveUserID()
				if err != nil {
					return err
				}
				kid, err := parseInt32(args[0])
				if err != nil {
					return err
				}
				resp, err := app.Client.DeleteApiV1UsersUserIdSshKeysIdWithResponse(cmd.Context(), uid, kid)
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
					return app.HandleAPIError("ssh-keys.delete", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{"deleted": kid}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
	)
	return cmd
}

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Local read cache"}
	cmd.AddCommand(
		&cobra.Command{
			Use: "stats",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "cache.stats"
				st, err := app.Cache.Stats()
				if err != nil {
					return err
				}
				return app.Out.Success(st)
			},
		},
		&cobra.Command{
			Use:  "clear [prefix]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "cache.clear"
				prefix := ""
				if len(args) > 0 {
					prefix = args[0]
				}
				n, err := app.Cache.DeletePrefix(prefix)
				if err != nil {
					return err
				}
				return app.Out.Success(map[string]any{"cleared": n})
			},
		},
	)
	return cmd
}

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "spec", Short: "OpenAPI spec"}
	cmd.AddCommand(
		&cobra.Command{
			Use: "show",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "spec.show"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1OpenapiWithResponse(cmd.Context())
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("spec.show", resp.StatusCode(), resp.Body)
				}
				var doc any
				_ = json.Unmarshal(resp.Body, &doc)
				return app.Out.Success(doc, output.WithHTTPStatus(200))
			},
		},
		&cobra.Command{
			Use:   "update",
			Short: "Fetch live OpenAPI into config dir",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "spec.update"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.GetApiV1OpenapiWithResponse(cmd.Context())
				if err != nil {
					return err
				}
				if resp.StatusCode() != 200 {
					return app.HandleAPIError("spec.update", resp.StatusCode(), resp.Body)
				}
				path := app.Cfg.OpenAPIPath()
				if err := os.WriteFile(path, resp.Body, 0o644); err != nil {
					return err
				}
				return app.Out.Success(map[string]any{
					"path":       path,
					"bytes":      len(resp.Body),
					"regenerate": "run: make generate",
				})
			},
		},
		&cobra.Command{
			Use:   "mcp",
			Short: "Fetch OpenAPI MCP proxy (docs)",
			RunE: func(cmd *cobra.Command, args []string) error {
				app.Out.Command = "spec.mcp"
				if err := app.EnsureClient(cmd.Context()); err != nil {
					return err
				}
				resp, err := app.Client.PostApiV1OpenapiMcpWithResponse(cmd.Context())
				if err != nil {
					return err
				}
				var body any
				if len(resp.Body) > 0 && json.Unmarshal(resp.Body, &body) != nil {
					body = string(resp.Body)
				}
				if resp.StatusCode() >= 400 {
					return app.HandleAPIError("spec.mcp", resp.StatusCode(), resp.Body)
				}
				return app.Out.Success(map[string]any{
					"status": resp.StatusCode(),
					"body":   body,
				}, output.WithHTTPStatus(resp.StatusCode()))
			},
		},
	)
	return cmd
}

func newAPICmd() *cobra.Command {
	var (
		bodyFile string
		queryKV  []string
		headerKV []string
	)
	c := &cobra.Command{
		Use:   "api METHOD PATH",
		Short: "Raw authenticated HTTP against /scp-core",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			path := args[1]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			parsed, err := url.Parse("http://local" + path)
			if err != nil {
				return err
			}
			q := parsed.Query()
			for k, v := range kvMap(queryKV) {
				q.Set(k, v)
			}
			pathWithQuery := parsed.Path
			if enc := q.Encode(); enc != "" {
				pathWithQuery += "?" + enc
			}
			app.Out.Command = "api"
			if method != http.MethodGet && method != http.MethodHead {
				if err := app.Confirm(method + " " + path); err != nil {
					return err
				}
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			var bodyReader io.Reader
			if bodyFile != "" {
				b, err := readBodyArg(bodyFile)
				if err != nil {
					return err
				}
				bodyReader = bytes.NewReader(b)
			}
			reqURL := app.Cfg.APIRoot() + pathWithQuery
			req, err := http.NewRequestWithContext(cmd.Context(), method, reqURL, bodyReader)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "application/json")
			if bodyReader != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			for _, h := range headerKV {
				name, val, ok := strings.Cut(h, ":")
				if !ok || strings.TrimSpace(name) == "" {
					continue
				}
				req.Header.Set(strings.TrimSpace(name), strings.TrimSpace(val))
			}
			resp, err := app.HTTPClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			var data any
			if len(b) > 0 && json.Unmarshal(b, &data) != nil {
				data = string(b)
			}
			if resp.StatusCode >= 400 {
				return app.HandleAPIError("api", resp.StatusCode, b)
			}
			return app.Out.Success(data, output.WithHTTPStatus(resp.StatusCode))
		},
	}
	c.Flags().StringVar(&bodyFile, "body", "", "JSON body or @file or - for stdin")
	c.Flags().StringArrayVar(&queryKV, "query", nil, "query param k=v (repeatable)")
	c.Flags().StringArrayVar(&headerKV, "header", nil, "header Name: value (repeatable)")
	return c
}

func newCallCmd() *cobra.Command {
	var (
		bodyFile string
		dryRun   bool
		pathKV   []string
		queryKV  []string
	)
	c := &cobra.Command{
		Use:   "call OPERATION_HINT",
		Short: "Resolve an OpenAPI operation and invoke it",
		Long: `Resolve against the pinned/local OpenAPI and call the API.

Hints:
  netcup call "GET /api/v1/servers"
  netcup call "snapshot create" --path serverId=123 --body '{"name":"x"}'
  netcup call "/api/v1/tasks/{uuid}" --path uuid=...
  netcup call "servers list" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "call"
			idx, err := loadOpenAPIIndex()
			if err != nil {
				return err
			}
			op, matches, err := idx.Best(args[0])
			if err != nil {
				cands := make([]map[string]any, 0, len(matches))
				for _, m := range matches {
					cands = append(cands, map[string]any{
						"method": m.Op.Method,
						"path":   m.Op.Path,
						"summary": m.Op.Summary,
						"score":  m.Score,
						"reason": m.Reason,
					})
				}
				_ = app.Out.Fail(output.ExitUsage, "usage", "", err.Error(), 0, nil)
				// still print candidates in data via Success-like path for agents
				if app.Out.Format == output.FormatJSON || app.Out.Format == output.FormatJSONL {
					enc := json.NewEncoder(app.Out.Out)
					enc.SetIndent("", "  ")
					_ = enc.Encode(map[string]any{
						"ok":         false,
						"command":    "call",
						"error":      map[string]any{"type": "usage", "message": err.Error()},
						"candidates": cands,
					})
					return output.Exit(output.ExitUsage, err.Error())
				}
				return err
			}

			pathParams := kvMap(pathKV)
			// Convenience: --server fills {serverId}
			if app.Flags.Server != "" {
				if _, ok := pathParams["serverId"]; !ok {
					if err := app.EnsureClient(cmd.Context()); err == nil {
						if id, _, rerr := resolveServerQuiet(cmd.Context(), app.Flags.Server); rerr == nil {
							pathParams["serverId"] = fmt.Sprint(id)
						}
					}
				}
			}
			if _, ok := pathParams["userId"]; !ok {
				for _, p := range op.PathParams {
					if p == "userId" {
						if uid, err := app.ResolveUserID(); err == nil {
							pathParams["userId"] = fmt.Sprint(uid)
						}
					}
				}
			}

			filled, fillErr := apiraw.FillPath(op.Path, pathParams)
			q := url.Values{}
			for k, v := range kvMap(queryKV) {
				q.Set(k, v)
			}
			fullPath := filled
			if fillErr == nil {
				if enc := q.Encode(); enc != "" {
					fullPath += "?" + enc
				}
			}

			var body []byte
			if bodyFile != "" {
				body, err = readBodyArg(bodyFile)
				if err != nil {
					return err
				}
			}

			if dryRun {
				missing := []string{}
				if fillErr != nil {
					for _, p := range op.PathParams {
						if _, ok := pathParams[p]; !ok {
							missing = append(missing, p)
						}
					}
				}
				return app.Out.Success(map[string]any{
					"method":         op.Method,
					"path":           op.Path,
					"filled_path":    fullPath,
					"fill_error":     errString(fillErr),
					"missing_params": missing,
					"summary":        op.Summary,
					"tag":            op.Tag,
					"has_body":       op.HasBody,
					"path_params":    op.PathParams,
					"query_params":   op.QueryParams,
					"body":           jsonRawOrNil(body),
				})
			}

			if fillErr != nil {
				return app.Out.Fail(output.ExitUsage, "usage", "", fillErr.Error(), 0, nil)
			}

			if op.Method != http.MethodGet && op.Method != http.MethodHead {
				if err := app.Confirm(op.Method + " " + filled); err != nil {
					return err
				}
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}

			var bodyReader io.Reader
			if len(body) > 0 {
				bodyReader = bytes.NewReader(body)
			}
			req, err := http.NewRequestWithContext(cmd.Context(), op.Method, app.Cfg.APIRoot()+fullPath, bodyReader)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "application/json, application/hal+json")
			if len(body) > 0 {
				ct := "application/json"
				if op.Method == http.MethodPatch {
					ct = "application/merge-patch+json"
				}
				req.Header.Set("Content-Type", ct)
			}
			resp, err := app.HTTPClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			var data any
			if len(b) > 0 && json.Unmarshal(b, &data) != nil {
				data = string(b)
			}
			if resp.StatusCode == 202 {
				var task scpclient.TaskInfo
				if json.Unmarshal(b, &task) == nil && task.Uuid != nil {
					return handleTaskResp(cmd.Context(), "call", resp.StatusCode, &task, b)
				}
			}
			if resp.StatusCode >= 400 {
				return app.HandleAPIError("call", resp.StatusCode, b)
			}
			return app.Out.Success(map[string]any{
				"operation": map[string]any{"method": op.Method, "path": op.Path, "summary": op.Summary},
				"response":  data,
			}, output.WithHTTPStatus(resp.StatusCode))
		},
	}
	c.Flags().StringArrayVar(&pathKV, "path", nil, "path param k=v (repeatable)")
	c.Flags().StringArrayVar(&queryKV, "query", nil, "query param k=v (repeatable)")
	c.Flags().StringVar(&bodyFile, "body", "", "JSON body or @file or -")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "resolve only; do not call API")
	return c
}

func kvMap(pairs []string) map[string]string {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func jsonRawOrNil(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	var v any
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return string(b)
}

func errString(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func resolveServerQuiet(ctx context.Context, sel string) (int32, *scpclient.Server, error) {
	return selectserver.Resolve(ctx, app.Client, sel)
}

func loadOpenAPIIndex() (*apiraw.Index, error) {
	return apiraw.Load(app.Cfg.OpenAPIPath(), "openapi.json")
}

func newEndpointsCmd() *cobra.Command {
	var filter, tag string
	c := &cobra.Command{
		Use:   "endpoints",
		Short: "List OpenAPI operations from pinned/local spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "endpoints"
			doc, err := loadOpenAPIDoc()
			if err != nil {
				return err
			}
			type row struct {
				Method string `json:"method"`
				Path   string `json:"path"`
				Tag    string `json:"tag,omitempty"`
				Summary string `json:"summary,omitempty"`
			}
			var rows []row
			paths, _ := doc["paths"].(map[string]any)
			for p, item := range paths {
				ops, ok := item.(map[string]any)
				if !ok {
					continue
				}
				for m, op := range ops {
					ml := strings.ToUpper(m)
					if ml != "GET" && ml != "POST" && ml != "PUT" && ml != "PATCH" && ml != "DELETE" && ml != "HEAD" {
						continue
					}
					om, _ := op.(map[string]any)
					summary, _ := om["summary"].(string)
					tags, _ := om["tags"].([]any)
					t := ""
					if len(tags) > 0 {
						t, _ = tags[0].(string)
					}
					if tag != "" && !strings.EqualFold(t, tag) {
						continue
					}
					line := ml + " " + p + " " + summary
					if filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filter)) {
						continue
					}
					rows = append(rows, row{Method: ml, Path: p, Tag: t, Summary: summary})
				}
			}
			if app.Out.Format == output.FormatTable {
				td := output.TableData{Headers: []string{"method", "path", "tag", "summary"}, Raw: rows}
				for _, r := range rows {
					td.Rows = append(td.Rows, []string{r.Method, r.Path, r.Tag, r.Summary})
				}
				return app.Out.Success(td)
			}
			return app.Out.Success(rows)
		},
	}
	c.Flags().StringVar(&filter, "filter", "", "substring filter")
	c.Flags().StringVar(&tag, "tag", "", "OpenAPI tag filter")
	return c
}

func newDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe PATH_OR_OPERATION",
		Short: "Show OpenAPI operation or path item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "describe"
			idx, err := loadOpenAPIIndex()
			if err != nil {
				return err
			}
			op, matches, err := idx.Best(args[0])
			if err == nil && op != nil {
				return app.Out.Success(map[string]any{
					"method":       op.Method,
					"path":         op.Path,
					"summary":      op.Summary,
					"description":  op.Description,
					"tag":          op.Tag,
					"path_params":  op.PathParams,
					"query_params": op.QueryParams,
					"has_body":     op.HasBody,
				})
			}
			if len(matches) > 0 {
				out := make([]map[string]any, 0, len(matches))
				for _, m := range matches {
					out = append(out, map[string]any{
						"method":  m.Op.Method,
						"path":    m.Op.Path,
						"summary": m.Op.Summary,
						"score":   m.Score,
					})
				}
				return app.Out.Success(map[string]any{"candidates": out, "error": err.Error()})
			}
			return output.Exit(output.ExitNotFound, "path not found: "+args[0])
		},
	}
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}

func readBodyArg(s string) ([]byte, error) {
	if s == "-" {
		return io.ReadAll(os.Stdin)
	}
	if strings.HasPrefix(s, "@") {
		return os.ReadFile(s[1:])
	}
	return []byte(s), nil
}

func loadOpenAPIDoc() (map[string]any, error) {
	candidates := []string{
		app.Cfg.OpenAPIPath(),
		"openapi.json",
	}
	var last error
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			last = err
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(b, &doc); err != nil {
			return nil, err
		}
		return doc, nil
	}
	return nil, fmt.Errorf("openapi not found: %v", last)
}
