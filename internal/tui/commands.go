package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/cache"
	"github.com/brandonkramer/netcup-cli/internal/patch"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/wait"
	tea "github.com/charmbracelet/bubbletea"
)

func loadServersCmd(ctx context.Context, deps Deps, force bool) tea.Cmd {
	return func() tea.Msg {
		ck := cache.ServersListKey(fmt.Sprint(deps.UserID), "", "", "", nil, 0, 0, 0)
		if !force && deps.Cache != nil {
			if body, _, _, ok := deps.Cache.Get(ck); ok {
				var list []scpclient.ServerListMinimal
				if err := json.Unmarshal(body, &list); err == nil {
					return serversLoadedMsg{servers: list, cached: true}
				}
			}
		}
		if force && deps.Cache != nil {
			_ = deps.Cache.Delete(ck)
		}

		var (
			list []scpclient.ServerListMinimal
			err  error
		)
		err = withRetry(ctx, func(cctx context.Context) error {
			resp, e := deps.Client.GetApiV1ServersWithResponse(cctx, nil)
			if e != nil {
				return e
			}
			if resp == nil {
				return tuiError("empty response")
			}
			if resp.StatusCode() != 200 {
				return apiStatusErr("list servers", resp.StatusCode(), resp.Body)
			}
			list, e = decodeServerList(resp)
			return e
		})
		if err != nil {
			return serversLoadedMsg{err: friendlyNetErr("list servers", err)}
		}
		if deps.Cache != nil {
			_ = deps.Cache.Put(ck, list, cache.TTLServerList, cache.TagServers)
		}
		return serversLoadedMsg{servers: list}
	}
}

func loadDetailCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		var (
			s   *scpclient.Server
			err error
		)
		err = withRetry(ctx, func(cctx context.Context) error {
			live := true
			resp, e := client.GetApiV1ServersServerIdWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdParams{
				LoadServerLiveInfo: &live,
			})
			if e != nil {
				return e
			}
			if resp == nil {
				return tuiError("empty response")
			}
			if resp.StatusCode() != 200 {
				return apiStatusErr("get server", resp.StatusCode(), resp.Body)
			}
			s, e = decodeServer(resp)
			return e
		})
		if err != nil {
			return detailLoadedMsg{err: friendlyNetErr("get server", err)}
		}
		return detailLoadedMsg{server: s}
	}
}

func loadGuestAgentCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdGuestAgentStatusWithResponse(ctx, id)
		if err != nil {
			return guestAgentMsg{err: err}
		}
		if resp == nil {
			return guestAgentMsg{err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return guestAgentMsg{err: apiStatusErr("guest-agent", resp.StatusCode(), resp.Body)}
		}
		switch {
		case resp.JSON200 != nil:
			return guestAgentMsg{status: resp.JSON200}
		case resp.HALJSON200 != nil:
			return guestAgentMsg{status: resp.HALJSON200}
		default:
			return guestAgentMsg{status: nil}
		}
	}
}

func powerCmd(
	ctx context.Context,
	client *scpclient.ClientWithResponses,
	id int32,
	name string,
	action string,
	state scpclient.ServerState1,
	opt *string,
) tea.Cmd {
	return func() tea.Msg {
		resp, err := patch.SetState(ctx, client, id, state, opt)
		if err != nil {
			return powerStartedMsg{serverID: id, serverName: name, action: action, err: err}
		}
		if resp == nil {
			return powerStartedMsg{serverID: id, serverName: name, action: action, err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		task, err := patch.RequireOK(resp)
		if err != nil {
			if status == 202 {
				return powerStartedMsg{serverID: id, serverName: name, action: action, status: status}
			}
			return powerStartedMsg{serverID: id, serverName: name, action: action, status: status, err: err}
		}
		return powerStartedMsg{serverID: id, serverName: name, action: action, status: status, task: task}
	}
}

func setAttrCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, attr, value string) tea.Cmd {
	return func() tea.Msg {
		var resp *scpclient.PatchApiV1ServersServerIdResponse
		var err error
		switch attr {
		case "hostname":
			resp, err = patch.SetHostname(ctx, client, id, value)
		case "nickname":
			resp, err = patch.SetNickname(ctx, client, id, value)
		case "autostart":
			v, e := strconv.ParseBool(value)
			if e != nil {
				return attrSetMsg{attr: attr, err: e}
			}
			resp, err = patch.SetAutostart(ctx, client, id, v)
		case "uefi":
			v, e := strconv.ParseBool(value)
			if e != nil {
				return attrSetMsg{attr: attr, err: e}
			}
			resp, err = patch.SetUEFI(ctx, client, id, v)
		case "bootorder":
			parts := strings.Split(value, ",")
			order := make([]scpclient.Bootorder, 0, len(parts))
			for _, p := range parts {
				order = append(order, scpclient.Bootorder(strings.TrimSpace(p)))
			}
			resp, err = patch.SetBootorder(ctx, client, id, order)
		case "keyboard-layout":
			resp, err = patch.SetKeyboardLayout(ctx, client, id, value)
		default:
			return attrSetMsg{attr: attr, err: fmt.Errorf("unknown attr %s", attr)}
		}
		if err != nil {
			return attrSetMsg{attr: attr, err: err}
		}
		if resp == nil {
			return attrSetMsg{attr: attr, err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		task, err := patch.RequireOK(resp)
		if err != nil {
			if status == 202 {
				return attrSetMsg{attr: attr, status: status}
			}
			return attrSetMsg{attr: attr, status: status, err: err}
		}
		return attrSetMsg{attr: attr, status: status, task: task}
	}
}

func rescueCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, action string) tea.Cmd {
	return func() tea.Msg {
		switch action {
		case "status":
			resp, err := client.GetApiV1ServersServerIdRescuesystemWithResponse(ctx, id)
			if err != nil {
				return rescueMsg{action: action, err: err}
			}
			if resp == nil {
				return rescueMsg{action: action, err: tuiError("empty response")}
			}
			if resp.StatusCode() != 200 {
				return rescueMsg{action: action, status: resp.StatusCode(), err: apiStatusErr(action, resp.StatusCode(), resp.Body)}
			}
			return rescueMsg{action: action, status: resp.StatusCode(), data: firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)}
		case "enable":
			resp, err := client.PostApiV1ServersServerIdRescuesystemWithResponse(ctx, id)
			if err != nil {
				return rescueMsg{action: action, err: err}
			}
			if resp == nil {
				return rescueMsg{action: action, err: tuiError("empty response")}
			}
			status := resp.StatusCode()
			if status != 202 && status != 200 {
				return rescueMsg{action: action, status: status, err: apiStatusErr(action, status, resp.Body)}
			}
			return rescueMsg{action: action, status: status, task: extractTask(status, resp.Body, resp.HALJSON202, resp.JSON202)}
		case "disable":
			resp, err := client.DeleteApiV1ServersServerIdRescuesystemWithResponse(ctx, id)
			if err != nil {
				return rescueMsg{action: action, err: err}
			}
			if resp == nil {
				return rescueMsg{action: action, err: tuiError("empty response")}
			}
			status := resp.StatusCode()
			if status != 202 && status != 200 && status != 204 {
				return rescueMsg{action: action, status: status, err: apiStatusErr(action, status, resp.Body)}
			}
			return rescueMsg{action: action, status: status, task: extractTask(status, resp.Body, resp.HALJSON202, resp.JSON202)}
		default:
			return rescueMsg{action: action, err: fmt.Errorf("unknown rescue action")}
		}
	}
}

func loadTasksCmd(ctx context.Context, client *scpclient.ClientWithResponses) tea.Cmd {
	return func() tea.Msg {
		var list []scpclient.TaskInfoMinimal
		err := withRetry(ctx, func(cctx context.Context) error {
			resp, e := client.GetApiV1TasksWithResponse(cctx, nil)
			if e != nil {
				return e
			}
			if resp == nil {
				return tuiError("empty response")
			}
			if resp.StatusCode() != 200 {
				return apiStatusErr("list tasks", resp.StatusCode(), resp.Body)
			}
			switch {
			case resp.JSON200 != nil:
				list = *resp.JSON200
			case resp.HALJSON200 != nil:
				list = *resp.HALJSON200
			default:
				list = nil
			}
			return nil
		})
		if err != nil {
			return tasksLoadedMsg{err: friendlyNetErr("list tasks", err)}
		}
		return tasksLoadedMsg{tasks: list}
	}
}

func cancelTaskCmd(ctx context.Context, client *scpclient.ClientWithResponses, uuid string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.PutApiV1TasksUuidCancelWithResponse(ctx, uuid)
		if err != nil {
			return taskCancelMsg{uuid: uuid, err: err}
		}
		if resp == nil {
			return taskCancelMsg{uuid: uuid, err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		if status != 202 && status != 200 {
			return taskCancelMsg{uuid: uuid, status: status, err: apiStatusErr("cancel task", status, resp.Body)}
		}
		return taskCancelMsg{uuid: uuid, status: status, task: extractTask(status, resp.Body, resp.HALJSON202, resp.JSON202)}
	}
}

func loadMetricsCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, kind string, hours int32) tea.Cmd {
	return func() tea.Msg {
		var hoursPtr *int32
		if hours > 0 {
			hoursPtr = &hours
		}
		var data any
		err := withRetry(ctx, func(cctx context.Context) error {
			var status int
			var body []byte
			var e error
			switch kind {
			case "cpu":
				resp, err2 := client.GetApiV1ServersServerIdMetricsCpuWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsCpuParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					if resp == nil {
						return tuiError("empty response")
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "disk":
				resp, err2 := client.GetApiV1ServersServerIdMetricsDiskWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsDiskParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					if resp == nil {
						return tuiError("empty response")
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "network":
				resp, err2 := client.GetApiV1ServersServerIdMetricsNetworkWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsNetworkParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					if resp == nil {
						return tuiError("empty response")
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "packets":
				resp, err2 := client.GetApiV1ServersServerIdMetricsNetworkPacketWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsNetworkPacketParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					if resp == nil {
						return tuiError("empty response")
					}
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			default:
				return fmt.Errorf("unknown metrics kind")
			}
			if e != nil {
				return e
			}
			if status != 200 {
				return apiStatusErr("metrics", status, body)
			}
			return nil
		})
		if err != nil {
			return metricsLoadedMsg{kind: kind, err: friendlyNetErr("metrics", err)}
		}
		return metricsLoadedMsg{kind: kind, data: data}
	}
}

func loadMediaCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID, userID int32) tea.Cmd {
	return func() tea.Msg {
		out := mediaLoadedMsg{}
		err := withRetry(ctx, func(cctx context.Context) error {
			if serverID != 0 {
				resp, e := client.GetApiV1ServersServerIdIsoWithResponse(cctx, serverID)
				if e != nil {
					return e
				}
				if resp == nil {
					return tuiError("empty response")
				}
				if resp.StatusCode() == 401 {
					return apiStatusErr("load media", resp.StatusCode(), resp.Body)
				}
				if resp.StatusCode() == 200 {
					out.attached = firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
				cat, e := client.GetApiV1ServersServerIdIsoimagesWithResponse(cctx, serverID)
				if e != nil {
					return e
				}
				if cat == nil {
					return tuiError("empty response")
				}
				if cat.StatusCode() == 401 {
					return apiStatusErr("load media", cat.StatusCode(), cat.Body)
				}
				if cat.StatusCode() == 200 {
					switch {
					case cat.JSON200 != nil:
						out.catalog = *cat.JSON200
					case cat.HALJSON200 != nil:
						out.catalog = *cat.HALJSON200
					}
				}
			}
			if userID != 0 {
				resp, e := client.GetApiV1UsersUserIdIsosWithResponse(cctx, userID)
				if e != nil {
					return e
				}
				if resp == nil {
					return tuiError("empty response")
				}
				if resp.StatusCode() == 401 {
					return apiStatusErr("load media", resp.StatusCode(), resp.Body)
				}
				if resp.StatusCode() == 200 {
					switch {
					case resp.JSON200 != nil:
						out.userISOs = *resp.JSON200
					case resp.HALJSON200 != nil:
						out.userISOs = *resp.HALJSON200
					}
				}
				images, e := client.GetApiV1UsersUserIdImagesWithResponse(cctx, userID)
				if e != nil {
					return e
				}
				if images == nil {
					return tuiError("empty response")
				}
				if images.StatusCode() == 401 {
					return apiStatusErr("load media", images.StatusCode(), images.Body)
				}
				if images.StatusCode() == 200 {
					out.userImages = firstJSON(images.JSON200, images.HALJSON200, images.Body)
				}
			}
			return nil
		})
		if err != nil {
			return mediaLoadedMsg{err: friendlyNetErr("load media", err)}
		}
		return out
	}
}

func isoAttachCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32, isoID int32, userName string, bootCDROM bool) tea.Cmd {
	return func() tea.Msg {
		body := scpclient.ServerAttachIso{ChangeBootDeviceToCdrom: &bootCDROM}
		if isoID != 0 {
			body.IsoId = &isoID
		}
		if userName != "" {
			body.UserIsoName = &userName
		}
		resp, err := client.PostApiV1ServersServerIdIsoWithResponse(ctx, serverID, body)
		if err != nil {
			return isoActionMsg{action: "attach", err: err}
		}
		if resp == nil {
			return isoActionMsg{action: "attach", err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		if status != 202 && status != 200 {
			return isoActionMsg{action: "attach", status: status, err: apiStatusErr("iso attach", status, resp.Body)}
		}
		return isoActionMsg{action: "attach", status: status, task: extractTask(status, resp.Body, resp.HALJSON202, resp.JSON202)}
	}
}

func isoDetachCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.DeleteApiV1ServersServerIdIsoWithResponse(ctx, serverID)
		if err != nil {
			return isoActionMsg{action: "detach", err: err}
		}
		if resp == nil {
			return isoActionMsg{action: "detach", err: tuiError("empty response")}
		}
		status := resp.StatusCode()
		if status != 200 && status != 204 {
			return isoActionMsg{action: "detach", status: status, err: apiStatusErr("iso detach", status, resp.Body)}
		}
		return isoActionMsg{action: "detach", status: status}
	}
}

func pollJobs(ctx context.Context, client *scpclient.ClientWithResponses, jobs []job) tea.Msg {
	out := make([]job, len(jobs))
	copy(out, jobs)
	var authErr error
	for i := range out {
		j := &out[i]
		if j.Done || j.UUID == "" {
			continue
		}
		resp, err := client.GetApiV1TasksUuidWithResponse(ctx, j.UUID)
		if err != nil {
			if isAuthFailure(err) {
				authErr = err
				j.Err = err.Error()
				continue
			}
			if isTransientNetErr(err) {
				j.Message = "poll retry…"
				continue
			}
			j.Err = err.Error()
			j.Done = true
			continue
		}
		if resp == nil {
			j.Message = "poll empty response — retrying"
			continue
		}
		if resp.StatusCode() == 401 {
			authErr = apiStatusErr("task poll", resp.StatusCode(), resp.Body)
			j.Err = authErr.Error()
			continue
		}
		if resp.StatusCode() >= 500 || resp.StatusCode() == 429 {
			j.Message = fmt.Sprintf("poll HTTP %d — retrying", resp.StatusCode())
			continue
		}
		if resp.StatusCode() != 200 {
			j.Err = fmt.Sprintf("HTTP %d", resp.StatusCode())
			j.Done = true
			continue
		}
		task := firstTask(resp.JSON200, resp.HALJSON200)
		if task == nil {
			j.Err = "empty task"
			j.Done = true
			continue
		}
		if task.State != nil {
			j.State = string(*task.State)
			if isTerminalTask(*task.State) {
				j.Done = true
			}
		}
		j.Progress = wait.ProgressPercent(*task)
		if task.Message != nil {
			j.Message = *task.Message
		}
		if j.Done && task.State != nil && (*task.State == scpclient.TaskStateERROR || *task.State == scpclient.TaskStateROLLBACK) {
			j.Err = j.Message
			if j.Err == "" {
				j.Err = "task failed"
			}
		}
	}
	return jobsPolledMsg{jobs: out, err: authErr}
}

func decodeServerList(resp *scpclient.GetApiV1ServersResponse) ([]scpclient.ServerListMinimal, error) {
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return *resp.HALJSON200, nil
	}
	var list []scpclient.ServerListMinimal
	if err := json.Unmarshal(resp.Body, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func decodeServer(resp *scpclient.GetApiV1ServersServerIdResponse) (*scpclient.Server, error) {
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.HALJSON200 != nil {
		return resp.HALJSON200, nil
	}
	return nil, fmt.Errorf("get server: empty body")
}

func firstJSON(a, b any, body []byte) any {
	if a != nil {
		return a
	}
	if b != nil {
		return b
	}
	if len(body) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body)
	}
	return v
}

func firstTask(a, b *scpclient.TaskInfo) *scpclient.TaskInfo {
	if a != nil {
		return a
	}
	return b
}

func extractTask(status int, body []byte, candidates ...*scpclient.TaskInfo) *scpclient.TaskInfo {
	for _, t := range candidates {
		if t != nil {
			return t
		}
	}
	if status == 202 && len(body) > 0 {
		var t scpclient.TaskInfo
		if err := json.Unmarshal(body, &t); err == nil && t.Uuid != nil {
			return &t
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func jobFromTask(serverID int32, serverName, action string, task *scpclient.TaskInfo, err error) job {
	return jobFromTaskStatus(serverID, serverName, action, 0, task, err)
}

func jobFromTaskStatus(serverID int32, serverName, action string, status int, task *scpclient.TaskInfo, err error) job {
	j := job{ServerID: serverID, ServerName: serverName, Action: action}
	if err != nil {
		j.Err = err.Error()
		j.Done = true
		j.State = "ERROR"
		return j
	}
	if task != nil && task.Uuid != nil {
		j.UUID = *task.Uuid
		if task.State != nil {
			j.State = string(*task.State)
			if isTerminalTask(*task.State) {
				j.Done = true
			}
		}
		j.Progress = wait.ProgressPercent(*task)
		if task.Message != nil {
			j.Message = *task.Message
		}
		return j
	}
	if status == 202 {
		// Accepted by API but no pollable UUID — not a confirmed success.
		j.State = "UNTRACKED"
		j.Done = true
		j.Message = "accepted, no task uuid — refresh to verify"
		return j
	}
	j.State = "OK"
	j.Done = true
	return j
}
