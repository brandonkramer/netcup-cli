package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	tea "github.com/charmbracelet/bubbletea"
)

func resourceEditCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid, sid int32, attr, value, contextID string) tea.Cmd {
	switch attr {
	case "firewall-set":
		return firewallSetCmd(ctx, c, sid, contextID, value)
	case "policy-create", "policy-update":
		return policyEditCmd(ctx, c, uid, attr, contextID, value)
	case "ssh-add":
		return addSSHKeyCmd(ctx, c, uid, value)
	case "iso-upload":
		return uploadMediaCmd(ctx, c, uid, "iso", value)
	case "image-upload":
		return uploadMediaCmd(ctx, c, uid, "image", value)
	case "failover-route":
		return routeFailoverCmd(ctx, c, uid, contextID, value)
	case "nic-create":
		return nicCreateCmd(ctx, c, sid, value)
	case "vlan-rename":
		return vlanRenameCmd(ctx, c, uid, contextID, value)
	case "rdns-set":
		return rdnsSetCmd(ctx, c, value)
	case "rdns-get":
		return rdnsGetCmd(ctx, c, value)
	case "rdns-delete":
		return rdnsDeleteCmd(ctx, c, value)
	}
	return func() tea.Msg { return resourceActionMsg{action: attr, err: fmt.Errorf("unknown editor %s", attr)} }
}
func resourceConfirmCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid, sid int32, action, value string) tea.Cmd {
	switch action {
	case "firewall-reapply":
		return func() tea.Msg {
			r, e := c.PostApiV1ServersServerIdInterfacesMacFirewallReapplyWithResponse(ctx, sid, value)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if r == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 202 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
		}
	case "firewall-restore":
		return func() tea.Msg {
			r, e := c.PostApiV1ServersServerIdInterfacesMacFirewallRestoreCopiedPoliciesWithResponse(ctx, sid, value)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if r == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 202 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
		}
	case "policy-delete":
		return func() tea.Msg {
			r, e := c.DeleteApiV1UsersUserIdFirewallPoliciesIdWithResponse(ctx, uid, parseID(value))
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if r == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 204 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
			}
			return resourceActionMsg{action: action, status: status}
		}
	case "iso-delete":
		return deleteMediaCmd(ctx, c, uid, "iso", value)
	case "image-delete":
		return deleteMediaCmd(ctx, c, uid, "image", value)
	case "ssh-delete":
		return deleteSSHKeyCmd(ctx, c, uid, value)
	case "nic-delete":
		return func() tea.Msg {
			r, e := c.DeleteApiV1ServersServerIdInterfacesMacWithResponse(ctx, sid, value)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if r == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 202 && status != 204 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
		}
	case "image-setup-user":
		return imageSetupUserCmd(ctx, c, sid, value)
	case "policy-preset-ssh", "policy-preset-http":
		return firewallPresetCmd(ctx, c, uid, action)
	}
	return nil
}
func policyEditCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, action, id, value string) tea.Cmd {
	return func() tea.Msg {
		raw, e := bodyArg(value)
		if e != nil {
			return resourceActionMsg{action: action, err: e}
		}
		if action == "policy-create" {
			r, e := c.PostApiV1UsersUserIdFirewallPoliciesWithBodyWithResponse(ctx, uid, "application/json", bytes.NewReader(raw))
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if r == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 201 && status != 202 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, r.Body)}
		}
		r, e := c.PutApiV1UsersUserIdFirewallPoliciesIdWithBodyWithResponse(ctx, uid, parseID(id), "application/json", bytes.NewReader(raw))
		if e != nil {
			return resourceActionMsg{action: action, err: e}
		}
		if r == nil {
			return resourceActionMsg{action: action, err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, r.Body)}
		}
		var task *scpclient.TaskInfo
		if r.JSON202 != nil {
			task = r.JSON202.TaskInfo
		} else if r.HALJSON202 != nil {
			task = r.HALJSON202.TaskInfo
		}
		if task == nil {
			task = extractTask(status, r.Body)
		}
		return resourceActionMsg{action: action, status: status, task: task}
	}
}
func rdnsSetCmd(ctx context.Context, c *scpclient.ClientWithResponses, value string) tea.Cmd {
	return func() tea.Msg {
		p := strings.Fields(value)
		if len(p) != 2 {
			return resourceActionMsg{action: "rdns-set", err: tuiError("use IP hostname")}
		}
		if strings.Contains(p[0], ":") {
			body := scpclient.SetRdnsIpv6{Ip: p[0], Rdns: p[1]}
			r, e := c.PostApiV1RdnsIpv6WithResponse(ctx, body)
			if e != nil {
				return resourceActionMsg{action: "rdns-set", err: e}
			}
			if r == nil {
				return resourceActionMsg{action: "rdns-set", err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 201 {
				return resourceActionMsg{action: "rdns-set", status: status, err: apiStatusErr("set rDNS", status, r.Body)}
			}
			return resourceActionMsg{action: "rdns-set", status: status}
		}
		body := scpclient.SetRdnsIpv4{Ip: p[0], Rdns: p[1]}
		r, e := c.PostApiV1RdnsIpv4WithResponse(ctx, body)
		if e != nil {
			return resourceActionMsg{action: "rdns-set", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "rdns-set", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 201 {
			return resourceActionMsg{action: "rdns-set", status: status, err: apiStatusErr("set rDNS", status, r.Body)}
		}
		return resourceActionMsg{action: "rdns-set", status: status}
	}
}

func rdnsGetCmd(ctx context.Context, c *scpclient.ClientWithResponses, ip string) tea.Cmd {
	return func() tea.Msg {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return resourceLoadedMsg{tab: "network", err: tuiError("IP required")}
		}
		if strings.Contains(ip, ":") {
			r, e := c.GetApiV1RdnsIpv6IpWithResponse(ctx, ip)
			if e != nil {
				return resourceLoadedMsg{tab: "network", err: e}
			}
			if r == nil {
				return resourceLoadedMsg{tab: "network", err: tuiError("empty response")}
			}
			if r.StatusCode() != 200 {
				return resourceLoadedMsg{tab: "network", err: apiStatusErr("get rDNS", r.StatusCode(), r.Body)}
			}
			return resourceLoadedMsg{tab: "network", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
		}
		r, e := c.GetApiV1RdnsIpv4IpWithResponse(ctx, ip)
		if e != nil {
			return resourceLoadedMsg{tab: "network", err: e}
		}
		if r == nil {
			return resourceLoadedMsg{tab: "network", err: tuiError("empty response")}
		}
		if r.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "network", err: apiStatusErr("get rDNS", r.StatusCode(), r.Body)}
		}
		return resourceLoadedMsg{tab: "network", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
	}
}

func rdnsDeleteCmd(ctx context.Context, c *scpclient.ClientWithResponses, ip string) tea.Cmd {
	return func() tea.Msg {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return resourceActionMsg{action: "rdns-delete", err: tuiError("IP required")}
		}
		if strings.Contains(ip, ":") {
			r, e := c.DeleteApiV1RdnsIpv6IpWithResponse(ctx, ip)
			if e != nil {
				return resourceActionMsg{action: "rdns-delete", err: e}
			}
			if r == nil {
				return resourceActionMsg{action: "rdns-delete", err: tuiError("empty response")}
			}
			status := r.StatusCode()
			if status != 200 && status != 204 {
				return resourceActionMsg{action: "rdns-delete", status: status, err: apiStatusErr("delete rDNS", status, r.Body)}
			}
			return resourceActionMsg{action: "rdns-delete", status: status}
		}
		r, e := c.DeleteApiV1RdnsIpv4IpWithResponse(ctx, ip)
		if e != nil {
			return resourceActionMsg{action: "rdns-delete", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "rdns-delete", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 204 {
			return resourceActionMsg{action: "rdns-delete", status: status, err: apiStatusErr("delete rDNS", status, r.Body)}
		}
		return resourceActionMsg{action: "rdns-delete", status: status}
	}
}

func imageSetupCmd(ctx context.Context, c *scpclient.ClientWithResponses, sid int32, value string) tea.Cmd {
	return func() tea.Msg {
		raw, err := bodyArg(value)
		if err != nil {
			return resourceActionMsg{action: "image-setup", err: err}
		}
		r, e := c.PostApiV1ServersServerIdImageWithBodyWithResponse(ctx, sid, "application/json", bytes.NewReader(raw))
		if e != nil {
			return resourceActionMsg{action: "image-setup", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "image-setup", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "image-setup", status: status, err: apiStatusErr("image setup", status, r.Body)}
		}
		return resourceActionMsg{action: "image-setup", status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
	}
}

func imageSetupUserCmd(ctx context.Context, c *scpclient.ClientWithResponses, sid int32, name string) tea.Cmd {
	return func() tea.Msg {
		body := scpclient.ServerUserImageSetup{UserImageName: name}
		r, e := c.PostApiV1ServersServerIdUserImageWithResponse(ctx, sid, body)
		if e != nil {
			return resourceActionMsg{action: "image-setup-user", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "image-setup-user", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "image-setup-user", status: status, err: apiStatusErr("image setup-user", status, r.Body)}
		}
		return resourceActionMsg{action: "image-setup-user", status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
	}
}

func loadFlavoursCmd(ctx context.Context, c *scpclient.ClientWithResponses, sid int32) tea.Cmd {
	return func() tea.Msg {
		r, e := c.GetApiV1ServersServerIdImageflavoursWithResponse(ctx, sid)
		if e != nil {
			return resourceLoadedMsg{tab: "media", err: e}
		}
		if r == nil {
			return resourceLoadedMsg{tab: "media", err: tuiError("empty response")}
		}
		if r.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "media", err: apiStatusErr("flavours", r.StatusCode(), r.Body)}
		}
		return resourceLoadedMsg{tab: "media", mode: "flavours", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
	}
}
func storageOptimizeCmd(ctx context.Context, c *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		r, e := c.PostApiV1ServersServerIdStorageoptimizationWithResponse(ctx, id, nil)
		if e != nil {
			return resourceActionMsg{action: "storage-optimize", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "storage-optimize", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "storage-optimize", status: status, err: apiStatusErr("storage optimize", status, r.Body)}
		}
		return resourceActionMsg{action: "storage-optimize", status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
	}
}
func loadServerLogsCmd(ctx context.Context, c *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		r, e := c.GetApiV1ServersServerIdLogsWithResponse(ctx, id, nil)
		if e != nil {
			return resourceLoadedMsg{tab: "servers", err: e}
		}
		if r == nil {
			return resourceLoadedMsg{tab: "servers", err: tuiError("empty response")}
		}
		if r.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "servers", err: apiStatusErr("server logs", r.StatusCode(), r.Body)}
		}
		return resourceLoadedMsg{tab: "servers", mode: "server-logs", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
	}
}
func loadGPUCmd(ctx context.Context, c *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		r, e := c.GetApiV1ServersServerIdGpuDriverWithResponse(ctx, id)
		if e != nil {
			return resourceLoadedMsg{tab: "servers", err: e}
		}
		if r == nil {
			return resourceLoadedMsg{tab: "servers", err: tuiError("empty response")}
		}
		if r.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "servers", err: apiStatusErr("GPU driver", r.StatusCode(), r.Body)}
		}
		return resourceLoadedMsg{tab: "servers", mode: "gpu", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
	}
}
