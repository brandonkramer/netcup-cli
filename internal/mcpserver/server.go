package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer builds the netcup MCP tool server (stdio via Serve).
func NewServer(version string) *server.MCPServer {
	s := server.NewMCPServer(
		"netcup",
		version,
		server.WithToolCapabilities(true),
	)
	registerTools(s)
	return s
}

// Serve runs the MCP server on stdin/stdout.
func Serve(version string) error {
	return server.ServeStdio(NewServer(version))
}

func registerTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("netcup_ping",
		mcp.WithDescription("Check SCP API availability (netcup ping)."),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textResult(runCLI(ctx, []string{"ping"}, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_whoami",
		mcp.WithDescription("GET current user profile (netcup auth whoami)."),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textResult(runCLI(ctx, []string{"auth", "whoami"}, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_auth_status",
		mcp.WithDescription("Show local login status (netcup auth status)."),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textResult(runCLI(ctx, []string{"auth", "status"}, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_servers_list",
		mcp.WithDescription("List servers (netcup servers list)."),
		mcp.WithString("q", mcp.Description("search query")),
		mcp.WithString("name", mcp.Description("name filter")),
		mcp.WithString("ip", mcp.Description("ip filter")),
		mcp.WithInteger("limit", mcp.Description("max results")),
		mcp.WithInteger("offset", mcp.Description("list offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argv := []string{"servers", "list"}
		if q := req.GetString("q", ""); q != "" {
			argv = append(argv, "--q", q)
		}
		if name := req.GetString("name", ""); name != "" {
			argv = append(argv, "--name", name)
		}
		if ip := req.GetString("ip", ""); ip != "" {
			argv = append(argv, "--ip", ip)
		}
		if _, ok := req.GetArguments()["limit"]; ok {
			argv = append(argv, "--limit", fmt.Sprintf("%d", req.GetInt("limit", 0)))
		}
		if _, ok := req.GetArguments()["offset"]; ok {
			argv = append(argv, "--offset", fmt.Sprintf("%d", req.GetInt("offset", 0)))
		}
		return textResult(runCLI(ctx, argv, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_servers_get",
		mcp.WithDescription("Get one server by id|name|nickname|ip (netcup servers get)."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"servers", "get"}, serverSel), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_servers_power",
		mcp.WithDescription("Server power action: start|stop|reboot|poweroff|reset|suspend. Requires confirm=true."),
		mcp.WithString("action", mcp.Required(), mcp.Description("power action")),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
		mcp.WithBoolean("confirm", mcp.Description("must be true for destructive actions")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := strings.ToLower(strings.TrimSpace(req.GetString("action", "")))
		serverSel := req.GetString("server", "")
		if !powerActions[action] {
			return mcp.NewToolResultText("error: action must be one of start|stop|reboot|poweroff|reset|suspend"), nil
		}
		if !req.GetBool("confirm", false) {
			return mcp.NewToolResultText("error: destructive power action requires confirm=true"), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"servers", action}, serverSel), true, 600*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_tasks_list",
		mcp.WithDescription("List async tasks (netcup tasks list)."),
		mcp.WithString("server", mcp.Description("optional server filter")),
		mcp.WithInteger("limit", mcp.Description("max results")),
		mcp.WithInteger("offset", mcp.Description("list offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argv := []string{"tasks", "list"}
		if _, ok := req.GetArguments()["limit"]; ok {
			argv = append(argv, "--limit", fmt.Sprintf("%d", req.GetInt("limit", 0)))
		}
		if _, ok := req.GetArguments()["offset"]; ok {
			argv = append(argv, "--offset", fmt.Sprintf("%d", req.GetInt("offset", 0)))
		}
		return textResult(runCLI(ctx, withServer(argv, req.GetString("server", "")), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_tasks_get",
		mcp.WithDescription("Get one task by UUID (netcup tasks get)."),
		mcp.WithString("uuid", mcp.Required(), mcp.Description("task UUID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uuid, err := req.RequireString("uuid")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, []string{"tasks", "get", uuid}, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_tasks_cancel",
		mcp.WithDescription("Cancel a task (netcup tasks cancel). Requires confirm=true."),
		mcp.WithString("uuid", mcp.Required(), mcp.Description("task UUID")),
		mcp.WithBoolean("confirm", mcp.Description("must be true")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !req.GetBool("confirm", false) {
			return mcp.NewToolResultText("error: cancel requires confirm=true"), nil
		}
		uuid, err := req.RequireString("uuid")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, []string{"tasks", "cancel", uuid}, true, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_snapshots_list",
		mcp.WithDescription("List snapshots for a server (netcup snapshots list)."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"snapshots", "list"}, serverSel), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_iso_get",
		mcp.WithDescription("Show attached ISO (netcup iso get)."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"iso", "get"}, serverSel), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_iso_available",
		mcp.WithDescription("List ISOs available to attach (netcup iso available)."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"iso", "available"}, serverSel), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_iso_attach",
		mcp.WithDescription("Attach an ISO (netcup iso attach). Pass iso_id or user_iso. Requires confirm=true."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
		mcp.WithBoolean("confirm", mcp.Description("must be true")),
		mcp.WithInteger("iso_id", mcp.Description("catalog ISO id")),
		mcp.WithString("user_iso", mcp.Description("user ISO name")),
		mcp.WithBoolean("boot_cdrom", mcp.Description("set boot device to CDROM")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !req.GetBool("confirm", false) {
			return mcp.NewToolResultText("error: attach requires confirm=true"), nil
		}
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		_, hasID := req.GetArguments()["iso_id"]
		userISO := req.GetString("user_iso", "")
		if hasID == (userISO != "") {
			return mcp.NewToolResultText("error: provide exactly one of iso_id or user_iso"), nil
		}
		argv := []string{"iso", "attach"}
		if hasID {
			argv = append(argv, "--iso-id", fmt.Sprintf("%d", req.GetInt("iso_id", 0)))
		}
		if userISO != "" {
			argv = append(argv, "--user-iso", userISO)
		}
		if req.GetBool("boot_cdrom", false) {
			argv = append(argv, "--boot-cdrom")
		}
		return textResult(runCLI(ctx, withServer(argv, serverSel), true, 600*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_iso_detach",
		mcp.WithDescription("Detach ISO from a server (netcup iso detach). Requires confirm=true."),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
		mcp.WithBoolean("confirm", mcp.Description("must be true")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !req.GetBool("confirm", false) {
			return mcp.NewToolResultText("error: detach requires confirm=true"), nil
		}
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"iso", "detach"}, serverSel), true, 600*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_firewall_get",
		mcp.WithDescription("Get interface firewall for a MAC (netcup firewall get)."),
		mcp.WithString("mac", mcp.Required(), mcp.Description("NIC MAC")),
		mcp.WithString("server", mcp.Required(), mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac, err := req.RequireString("mac")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		serverSel, err := req.RequireString("server")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, withServer([]string{"firewall", "get", mac}, serverSel), false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_endpoints",
		mcp.WithDescription("List OpenAPI operations (netcup endpoints). Use before netcup_call."),
		mcp.WithString("filter", mcp.Description("substring filter")),
		mcp.WithString("tag", mcp.Description("OpenAPI tag filter")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argv := []string{"endpoints"}
		if f := req.GetString("filter", ""); f != "" {
			argv = append(argv, "--filter", f)
		}
		if t := req.GetString("tag", ""); t != "" {
			argv = append(argv, "--tag", t)
		}
		return textResult(runCLI(ctx, argv, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_describe",
		mcp.WithDescription("Show OpenAPI operation or path item (netcup describe)."),
		mcp.WithString("path_or_operation", mcp.Required(), mcp.Description("path or operation hint")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hint, err := req.RequireString("path_or_operation")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(runCLI(ctx, []string{"describe", hint}, false, 120*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_call",
		mcp.WithDescription("Resolve an OpenAPI operation and invoke it (netcup call). Destructive calls need confirm=true."),
		mcp.WithString("operation_hint", mcp.Required(), mcp.Description("operation hint")),
		mcp.WithArray("path", mcp.Description("path params as k=v"), mcp.WithStringItems()),
		mcp.WithArray("query", mcp.Description("query params as k=v"), mcp.WithStringItems()),
		mcp.WithString("body", mcp.Description("JSON body")),
		mcp.WithBoolean("dry_run", mcp.Description("resolve only")),
		mcp.WithBoolean("confirm", mcp.Description("skip destructive confirm (-y)")),
		mcp.WithString("server", mcp.Description("server selector")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hint, err := req.RequireString("operation_hint")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		argv := []string{"call", hint}
		for _, item := range req.GetStringSlice("path", nil) {
			argv = append(argv, "--path", item)
		}
		for _, item := range req.GetStringSlice("query", nil) {
			argv = append(argv, "--query", item)
		}
		if body := req.GetString("body", ""); body != "" {
			argv = append(argv, "--body", body)
		}
		dry := req.GetBool("dry_run", false)
		if dry {
			argv = append(argv, "--dry-run")
		}
		confirm := req.GetBool("confirm", false)
		argv = withServer(argv, req.GetString("server", ""))
		timeout := 120 * time.Second
		if confirm {
			timeout = 600 * time.Second
		}
		out, err := runCLI(ctx, argv, confirm, timeout)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "exited 2") && !confirm {
				return mcp.NewToolResultText(confirmRequired(msg)), nil
			}
			return mcp.NewToolResultText(errText(err)), nil
		}
		return mcp.NewToolResultText(out), nil
	})

	s.AddTool(mcp.NewTool("netcup_api",
		mcp.WithDescription("Raw authenticated HTTP against /scp-core (netcup api). Non-GET needs confirm=true."),
		mcp.WithString("method", mcp.Required(), mcp.Description("HTTP method")),
		mcp.WithString("path", mcp.Required(), mcp.Description("API path")),
		mcp.WithArray("query", mcp.Description("query params as k=v"), mcp.WithStringItems()),
		mcp.WithArray("header", mcp.Description("headers as Name: value"), mcp.WithStringItems()),
		mcp.WithString("body", mcp.Description("JSON body")),
		mcp.WithBoolean("confirm", mcp.Description("required for non-GET")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		method := strings.ToUpper(strings.TrimSpace(req.GetString("method", "")))
		path, err := req.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		confirm := req.GetBool("confirm", false)
		if method != "GET" && method != "HEAD" && method != "OPTIONS" && !confirm {
			return mcp.NewToolResultText("error: non-GET api calls require confirm=true"), nil
		}
		argv := []string{"api", method, path}
		for _, item := range req.GetStringSlice("query", nil) {
			argv = append(argv, "--query", item)
		}
		for _, item := range req.GetStringSlice("header", nil) {
			argv = append(argv, "--header", item)
		}
		if body := req.GetString("body", ""); body != "" {
			argv = append(argv, "--body", body)
		}
		return textResult(runCLI(ctx, argv, confirm, 600*time.Second))
	})

	s.AddTool(mcp.NewTool("netcup_cli",
		mcp.WithDescription("Run allowlisted netcup CLI argv (without the binary name). Destructive needs confirm=true."),
		mcp.WithArray("args", mcp.Required(), mcp.Description("argv after netcup"), mcp.WithStringItems()),
		mcp.WithBoolean("confirm", mcp.Description("maps to -y")),
		mcp.WithString("server", mcp.Description("server selector (-s)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetStringSlice("args", nil)
		if err := guardCLIArgs(args); err != nil {
			return mcp.NewToolResultText(errText(err)), nil
		}
		confirm := req.GetBool("confirm", false)
		argv := withServer(append([]string{}, args...), req.GetString("server", ""))
		timeout := 180 * time.Second
		if confirm {
			timeout = 600 * time.Second
		}
		out, err := runCLI(ctx, argv, confirm, timeout)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "exited 2") && !confirm {
				return mcp.NewToolResultText(confirmRequired(msg)), nil
			}
			return mcp.NewToolResultText(errText(err)), nil
		}
		return mcp.NewToolResultText(out), nil
	})
}

func textResult(out string, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultText(errText(err)), nil
	}
	return mcp.NewToolResultText(out), nil
}
