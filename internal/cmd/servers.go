package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/cache"
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/patch"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	selectserver "github.com/brandonkramer/netcup-cli/internal/select"
	"github.com/brandonkramer/netcup-cli/internal/wait"
	"github.com/spf13/cobra"
)

func newServersCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "servers", Short: "Manage servers"}
	cmd.AddCommand(
		newServersListCmd(),
		newServersGetCmd(),
		newServersPowerCmd("start", scpclient.ON, nil, false),
		newServersPowerCmd("stop", scpclient.OFF, nil, false),
		newServersPowerCmd("poweroff", scpclient.OFF, ptr("POWEROFF"), true),
		newServersPowerCmd("reboot", scpclient.ON, ptr("POWERCYCLE"), false),
		newServersPowerCmd("reset", scpclient.ON, ptr("RESET"), true),
		newServersPowerCmd("suspend", scpclient.SUSPENDED, nil, false),
		newServersSetCmd(),
		newServersRescueCmd(),
		newServersGPUDriverCmd(),
		newServersGuestAgentCmd(),
		newServersLogsCmd(),
		newServersStorageOptimizeCmd(),
	)
	return cmd
}

func newServersListCmd() *cobra.Command {
	var q, name, ip string
	var sortBy []string
	var limit, offset, firewallPolicyID int32
	c := &cobra.Command{
		Use:   "list",
		Short: "List servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.list"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			uid, _ := app.ResolveUserID()
			ck := cache.Key("GET", "/api/v1/servers",
				fmt.Sprintf("%s|%s|%s|%v|%d|%d|%d", q, name, ip, sortBy, limit, offset, firewallPolicyID),
				fmt.Sprint(uid))
			if body, age, ttl, ok := app.Cache.Get(ck); ok {
				var list []scpclient.ServerListMinimal
				if err := json.Unmarshal(body, &list); err == nil {
					td := serversTable(list)
					meta := &output.CacheMeta{Key: ck, AgeMS: age.Milliseconds(), TTLMS: ttl.Milliseconds()}
					return app.Out.Success(successData(app.Out.Format, td), output.WithHTTPStatus(200), output.WithCached(meta))
				}
			}
			params := &scpclient.GetApiV1ServersParams{}
			if q != "" {
				params.Q = &q
			}
			if name != "" {
				params.Name = &name
			}
			if ip != "" {
				params.Ip = &ip
			}
			if len(sortBy) > 0 {
				params.Sort = &sortBy
			}
			if cmd.Flags().Changed("limit") {
				params.Limit = &limit
			}
			if cmd.Flags().Changed("offset") {
				params.Offset = &offset
			}
			if cmd.Flags().Changed("firewall-policy-id") {
				params.FirewallPolicyId = &firewallPolicyID
			}
			resp, err := app.Client.GetApiV1ServersWithResponse(cmd.Context(), params)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("servers.list", resp.StatusCode(), resp.Body)
			}
			list, err := decodeServersList(resp)
			if err != nil {
				return err
			}
			ttl := cache.TTLServerList
			if app.Flags.CacheTTL > 0 {
				ttl = app.Flags.CacheTTL
			}
			_ = app.Cache.Put(ck, list, ttl)
			td := serversTable(list)
			return app.Out.Success(successData(app.Out.Format, td), output.WithHTTPStatus(200))
		},
	}
	c.Flags().StringVar(&q, "q", "", "search name/nickname/ipv4")
	c.Flags().StringVar(&name, "name", "", "filter by name")
	c.Flags().StringVar(&ip, "ip", "", "filter by ip")
	c.Flags().StringSliceVar(&sortBy, "sort", nil, "sort fields (name, nickname; prefix - for desc)")
	c.Flags().Int32Var(&limit, "limit", 0, "page size")
	c.Flags().Int32Var(&offset, "offset", 0, "page offset")
	c.Flags().Int32Var(&firewallPolicyID, "firewall-policy-id", 0, "filter by assigned firewall policy")
	return c
}

func newServersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [selector]",
		Short: "Get one server",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.get"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			sel := app.Flags.Server
			if len(args) > 0 {
				sel = args[0]
			}
			_, srv, err := selectserver.Resolve(cmd.Context(), app.Client, sel)
			if err != nil {
				return err
			}
			if srv == nil {
				return output.Exit(output.ExitNotFound, "server not found")
			}
			p := serverProjection(*srv)
			if app.Out.Format == output.FormatTable {
				return app.Out.Success(output.TableData{
					Headers: []string{"id", "name", "nickname", "state", "ipv4", "hostname"},
					Rows: [][]string{{
						fmtAny(p["id"]), fmtAny(p["name"]), fmtAny(p["nickname"]),
						fmtAny(p["state"]), fmtAny(p["ipv4"]), fmtAny(p["hostname"]),
					}},
					Raw: p,
				}, output.WithHTTPStatus(200))
			}
			if app.Flags.Full {
				return app.Out.Success(srv, output.WithHTTPStatus(200))
			}
			return app.Out.Success(p, output.WithHTTPStatus(200))
		},
	}
}

func newServersPowerCmd(name string, state scpclient.ServerState1, stateOpt *string, destructive bool) *cobra.Command {
	return &cobra.Command{
		Use:   name + " [selector]",
		Short: "Server power: " + name,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := "servers." + name
			app.Out.Command = command
			if destructive {
				if err := app.Confirm(name); err != nil {
					return err
				}
			}
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			sel := app.Flags.Server
			if len(args) > 0 {
				sel = args[0]
			}
			id, _, err := selectserver.Resolve(cmd.Context(), app.Client, sel)
			if err != nil {
				return err
			}
			resp, err := patch.SetState(cmd.Context(), app.Client, id, state, stateOpt)
			if err != nil {
				return err
			}
			task, err := patch.RequireOK(resp)
			if err != nil {
				return app.HandleAPIError(command, resp.StatusCode(), resp.Body)
			}
			app.Cache.BustTags("servers")
			if task != nil {
				task, err = wait.Task(cmd.Context(), app.Client, *task, app.WaitOpts())
				if err != nil {
					_ = emitTaskResult(command, 202, task, nil)
					return err
				}
			}
			return emitTaskResult(command, resp.StatusCode(), task, map[string]any{"server_id": id, "action": name})
		},
	}
}

func newServersSetCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "set", Short: "Set server attributes (one per request)"}
	cmd.AddCommand(
		&cobra.Command{
			Use:  "hostname <value> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "hostname", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					return patch.SetHostname(cmd.Context(), app.Client, id, val)
				})
			},
		},
		&cobra.Command{
			Use:  "nickname <value> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "nickname", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					return patch.SetNickname(cmd.Context(), app.Client, id, val)
				})
			},
		},
		&cobra.Command{
			Use:  "uefi <true|false> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "uefi", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					b := strings.EqualFold(val, "true") || val == "1"
					return patch.SetUEFI(cmd.Context(), app.Client, id, b)
				})
			},
		},
		&cobra.Command{
			Use:  "bootorder <CDROM,HDD,NETWORK> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "bootorder", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					parts := strings.Split(val, ",")
					order := make([]scpclient.Bootorder, 0, len(parts))
					for _, p := range parts {
						order = append(order, scpclient.Bootorder(strings.TrimSpace(p)))
					}
					return patch.SetBootorder(cmd.Context(), app.Client, id, order)
				})
			},
		},
		&cobra.Command{
			Use:  "root-password <value> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := app.Confirm("set root password"); err != nil {
					return err
				}
				return serverSet(cmd, args, "root-password", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					return patch.SetRootPassword(cmd.Context(), app.Client, id, val)
				})
			},
		},
		&cobra.Command{
			Use:  "autostart <true|false> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "autostart", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					b := strings.EqualFold(val, "true") || val == "1"
					return patch.SetAutostart(cmd.Context(), app.Client, id, b)
				})
			},
		},
		&cobra.Command{
			Use:  "os-optimization <LINUX|WINDOWS|BSD|LINUX_LEGACY|UNKNOWN> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "os-optimization", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					return patch.SetOsOptimization(cmd.Context(), app.Client, id, scpclient.OsOptimization(val))
				})
			},
		},
		&cobra.Command{
			Use:  "cpu-topology <sockets>,<coresPerSocket> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSetCpuTopology(cmd, args)
			},
		},
		&cobra.Command{
			Use:  "keyboard-layout <value> [selector]",
			Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverSet(cmd, args, "keyboard-layout", func(id int32, val string) (*scpclient.PatchApiV1ServersServerIdResponse, error) {
					return patch.SetKeyboardLayout(cmd.Context(), app.Client, id, val)
				})
			},
		},
	)
	return cmd
}

func serverSet(cmd *cobra.Command, args []string, attr string, fn func(int32, string) (*scpclient.PatchApiV1ServersServerIdResponse, error)) error {
	command := "servers.set." + attr
	app.Out.Command = command
	if err := app.EnsureClient(cmd.Context()); err != nil {
		return err
	}
	val := args[0]
	sel := app.Flags.Server
	if len(args) > 1 {
		sel = args[1]
	}
	id, _, err := selectserver.Resolve(cmd.Context(), app.Client, sel)
	if err != nil {
		return err
	}
	resp, err := fn(id, val)
	if err != nil {
		return err
	}
	task, err := patch.RequireOK(resp)
	if err != nil {
		return app.HandleAPIError(command, resp.StatusCode(), resp.Body)
	}
	app.Cache.BustTags("servers")
	if task != nil {
		task, err = wait.Task(cmd.Context(), app.Client, *task, app.WaitOpts())
		if err != nil {
			_ = emitTaskResult(command, 202, task, nil)
			return err
		}
	}
	return emitTaskResult(command, resp.StatusCode(), task, map[string]any{"server_id": id, "attribute": attr, "value": val})
}

func serverSetCpuTopology(cmd *cobra.Command, args []string) error {
	const attr = "cpu-topology"
	command := "servers.set." + attr
	app.Out.Command = command
	if err := app.EnsureClient(cmd.Context()); err != nil {
		return err
	}
	parts := strings.Split(args[0], ",")
	if len(parts) != 2 {
		return output.Exit(output.ExitUsage, "expected sockets,coresPerSocket (e.g. 2,4)")
	}
	sockets, err := parseInt32(strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}
	cores, err := parseInt32(strings.TrimSpace(parts[1]))
	if err != nil {
		return err
	}
	sel := app.Flags.Server
	if len(args) > 1 {
		sel = args[1]
	}
	id, _, err := selectserver.Resolve(cmd.Context(), app.Client, sel)
	if err != nil {
		return err
	}
	resp, err := patch.SetCpuTopology(cmd.Context(), app.Client, id, sockets, cores)
	if err != nil {
		return err
	}
	task, err := patch.RequireOK(resp)
	if err != nil {
		return app.HandleAPIError(command, resp.StatusCode(), resp.Body)
	}
	app.Cache.BustTags("servers")
	if task != nil {
		task, err = wait.Task(cmd.Context(), app.Client, *task, app.WaitOpts())
		if err != nil {
			_ = emitTaskResult(command, 202, task, nil)
			return err
		}
	}
	return emitTaskResult(command, resp.StatusCode(), task, map[string]any{
		"server_id": id, "attribute": attr, "sockets": sockets, "cores_per_socket": cores,
	})
}

func newServersRescueCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rescue", Short: "Rescue system"}
	cmd.AddCommand(
		&cobra.Command{
			Use:  "status [selector]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverRescue(cmd, args, "status")
			},
		},
		&cobra.Command{
			Use:  "enable [selector]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := app.Confirm("enable rescue"); err != nil {
					return err
				}
				return serverRescue(cmd, args, "enable")
			},
		},
		&cobra.Command{
			Use:  "disable [selector]",
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return serverRescue(cmd, args, "disable")
			},
		},
	)
	return cmd
}

func serverRescue(cmd *cobra.Command, args []string, action string) error {
	command := "servers.rescue." + action
	app.Out.Command = command
	if err := app.EnsureClient(cmd.Context()); err != nil {
		return err
	}
	sel := app.Flags.Server
	if len(args) > 0 {
		sel = args[0]
	}
	id, _, err := selectserver.Resolve(cmd.Context(), app.Client, sel)
	if err != nil {
		return err
	}
	switch action {
	case "status":
		resp, err := app.Client.GetApiV1ServersServerIdRescuesystemWithResponse(cmd.Context(), id)
		if err != nil {
			return err
		}
		if resp.StatusCode() != 200 {
			return app.HandleAPIError(command, resp.StatusCode(), resp.Body)
		}
		return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
	case "enable":
		resp, err := app.Client.PostApiV1ServersServerIdRescuesystemWithResponse(cmd.Context(), id)
		if err != nil {
			return err
		}
		return handleTaskResp(cmd.Context(), command, resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
	case "disable":
		resp, err := app.Client.DeleteApiV1ServersServerIdRescuesystemWithResponse(cmd.Context(), id)
		if err != nil {
			return err
		}
		return handleTaskResp(cmd.Context(), command, resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
	}
	return output.Exit(output.ExitUsage, "unknown rescue action")
}

func handleTaskResp(ctx context.Context, command string, status int, task *scpclient.TaskInfo, body []byte) error {
	task = taskFromBody(status, body, task)
	app.Cache.BustTags("servers")
	if status != 202 && status != 200 && status != 204 {
		return app.HandleAPIError(command, status, body)
	}
	if task != nil {
		var err error
		task, err = wait.Task(ctx, app.Client, *task, app.WaitOpts())
		if err != nil {
			_ = emitTaskResult(command, status, task, nil)
			return err
		}
	}
	return emitTaskResult(command, status, task, nil)
}

func resolveServerArg(ctx context.Context, args []string) (int32, error) {
	sel := app.Flags.Server
	if len(args) > 0 {
		sel = args[0]
	}
	id, _, err := selectserver.Resolve(ctx, app.Client, sel)
	return id, err
}

func newServersGPUDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gpu-driver [selector]",
		Short: "GPU driver download URL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.gpu-driver"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdGpuDriverWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("servers.gpu-driver", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	}
}

func newServersGuestAgentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "guest-agent", Short: "Guest agent"}
	cmd.AddCommand(&cobra.Command{
		Use:  "status [selector]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.guest-agent.status"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			resp, err := app.Client.GetApiV1ServersServerIdGuestAgentStatusWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("servers.guest-agent.status", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
		},
	})
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		app.Out.Command = "servers.guest-agent"
		if err := app.EnsureClient(cmd.Context()); err != nil {
			return err
		}
		id, err := resolveServerArg(cmd.Context(), args)
		if err != nil {
			return err
		}
		resp, err := app.Client.GetApiV1ServersServerIdGuestAgentWithResponse(cmd.Context(), id)
		if err != nil {
			return err
		}
		if resp.StatusCode() != 200 {
			return app.HandleAPIError("servers.guest-agent", resp.StatusCode(), resp.Body)
		}
		return app.Out.Success(resp.JSON200, output.WithHTTPStatus(200))
	}
	return cmd
}

func newServersLogsCmd() *cobra.Command {
	var limit, offset int32
	c := &cobra.Command{
		Use:   "logs [selector]",
		Short: "Server logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.logs"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			params := &scpclient.GetApiV1ServersServerIdLogsParams{}
			if cmd.Flags().Changed("limit") {
				params.Limit = &limit
			}
			if cmd.Flags().Changed("offset") {
				params.Offset = &offset
			}
			resp, err := app.Client.GetApiV1ServersServerIdLogsWithResponse(cmd.Context(), id, params)
			if err != nil {
				return err
			}
			if resp.StatusCode() != 200 {
				return app.HandleAPIError("servers.logs", resp.StatusCode(), resp.Body)
			}
			return app.Out.Success(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body), output.WithHTTPStatus(200))
		},
	}
	c.Flags().Int32Var(&limit, "limit", 0, "page size")
	c.Flags().Int32Var(&offset, "offset", 0, "page offset")
	return c
}

func newServersStorageOptimizeCmd() *cobra.Command {
	var disks []string
	var startAfter bool
	c := &cobra.Command{
		Use:   "storage-optimize [selector]",
		Short: "Optimize server storage",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Out.Command = "servers.storage-optimize"
			if err := app.EnsureClient(cmd.Context()); err != nil {
				return err
			}
			id, err := resolveServerArg(cmd.Context(), args)
			if err != nil {
				return err
			}
			params := &scpclient.PostApiV1ServersServerIdStorageoptimizationParams{}
			if len(disks) > 0 {
				params.Disks = &disks
			}
			if cmd.Flags().Changed("start-after") {
				params.StartAfterOptimization = &startAfter
			}
			resp, err := app.Client.PostApiV1ServersServerIdStorageoptimizationWithResponse(cmd.Context(), id, params)
			if err != nil {
				return err
			}
			return handleTaskResp(cmd.Context(), "servers.storage-optimize", resp.StatusCode(), firstTask(resp.HALJSON202, resp.JSON202), resp.Body)
		},
	}
	c.Flags().StringSliceVar(&disks, "disk", nil, "disk names to optimize (repeatable)")
	c.Flags().BoolVar(&startAfter, "start-after", false, "start server after optimization")
	return c
}
