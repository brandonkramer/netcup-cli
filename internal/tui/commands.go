package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/patch"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/brandonkramer/netcup-cli/internal/wait"
	tea "github.com/charmbracelet/bubbletea"
)

func loadServersCmd(ctx context.Context, client *scpclient.ClientWithResponses) tea.Cmd {
	return func() tea.Msg {
		var (
			list []scpclient.ServerListMinimal
			err  error
		)
		err = withRetry(func(cctx context.Context) error {
			resp, e := client.GetApiV1ServersWithResponse(cctx, nil)
			if e != nil {
				return e
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("list servers: HTTP %d", resp.StatusCode())
			}
			list, e = decodeServerList(resp)
			return e
		})
		if err != nil {
			return serversLoadedMsg{err: friendlyNetErr("list servers", err)}
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
		err = withRetry(func(cctx context.Context) error {
			live := true
			resp, e := client.GetApiV1ServersServerIdWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdParams{
				LoadServerLiveInfo: &live,
			})
			if e != nil {
				return e
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("get server: HTTP %d", resp.StatusCode())
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
		if resp.StatusCode() != 200 {
			return guestAgentMsg{err: fmt.Errorf("guest-agent: HTTP %d", resp.StatusCode())}
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
		task, err := patch.RequireOK(resp)
		if err != nil {
			return powerStartedMsg{serverID: id, serverName: name, action: action, err: err}
		}
		return powerStartedMsg{serverID: id, serverName: name, action: action, task: task}
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
		default:
			return attrSetMsg{attr: attr, err: fmt.Errorf("unknown attr %s", attr)}
		}
		if err != nil {
			return attrSetMsg{attr: attr, err: err}
		}
		task, err := patch.RequireOK(resp)
		if err != nil {
			return attrSetMsg{attr: attr, err: err}
		}
		return attrSetMsg{attr: attr, task: task}
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
			if resp.StatusCode() != 200 {
				return rescueMsg{action: action, err: fmt.Errorf("HTTP %d", resp.StatusCode())}
			}
			return rescueMsg{action: action, data: firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)}
		case "enable":
			resp, err := client.PostApiV1ServersServerIdRescuesystemWithResponse(ctx, id)
			if err != nil {
				return rescueMsg{action: action, err: err}
			}
			task := firstTask(resp.HALJSON202, resp.JSON202)
			if resp.StatusCode() != 202 && resp.StatusCode() != 200 {
				return rescueMsg{action: action, err: fmt.Errorf("HTTP %d", resp.StatusCode())}
			}
			return rescueMsg{action: action, task: task}
		case "disable":
			resp, err := client.DeleteApiV1ServersServerIdRescuesystemWithResponse(ctx, id)
			if err != nil {
				return rescueMsg{action: action, err: err}
			}
			task := firstTask(resp.HALJSON202, resp.JSON202)
			if resp.StatusCode() != 202 && resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return rescueMsg{action: action, err: fmt.Errorf("HTTP %d", resp.StatusCode())}
			}
			return rescueMsg{action: action, task: task}
		default:
			return rescueMsg{action: action, err: fmt.Errorf("unknown rescue action")}
		}
	}
}

func loadTasksCmd(ctx context.Context, client *scpclient.ClientWithResponses) tea.Cmd {
	return func() tea.Msg {
		var list []scpclient.TaskInfoMinimal
		err := withRetry(func(cctx context.Context) error {
			resp, e := client.GetApiV1TasksWithResponse(cctx, nil)
			if e != nil {
				return e
			}
			if resp.StatusCode() != 200 {
				return fmt.Errorf("list tasks: HTTP %d", resp.StatusCode())
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
		task := firstTask(resp.HALJSON202, resp.JSON202)
		if resp.StatusCode() != 202 && resp.StatusCode() != 200 {
			return taskCancelMsg{uuid: uuid, err: fmt.Errorf("HTTP %d", resp.StatusCode())}
		}
		return taskCancelMsg{uuid: uuid, task: task}
	}
}

func loadMetricsCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, kind string, hours int32) tea.Cmd {
	return func() tea.Msg {
		var hoursPtr *int32
		if hours > 0 {
			hoursPtr = &hours
		}
		var data any
		err := withRetry(func(cctx context.Context) error {
			var status int
			var body []byte
			var e error
			switch kind {
			case "cpu":
				resp, err2 := client.GetApiV1ServersServerIdMetricsCpuWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsCpuParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "disk":
				resp, err2 := client.GetApiV1ServersServerIdMetricsDiskWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsDiskParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "network":
				resp, err2 := client.GetApiV1ServersServerIdMetricsNetworkWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsNetworkParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			case "packets":
				resp, err2 := client.GetApiV1ServersServerIdMetricsNetworkPacketWithResponse(cctx, id, &scpclient.GetApiV1ServersServerIdMetricsNetworkPacketParams{Hours: hoursPtr})
				e = err2
				if err2 == nil {
					status, body, data = resp.StatusCode(), resp.Body, firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
			default:
				return fmt.Errorf("unknown metrics kind")
			}
			if e != nil {
				return e
			}
			if status != 200 {
				return fmt.Errorf("HTTP %d: %s", status, truncate(string(body), 120))
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
		err := withRetry(func(cctx context.Context) error {
			if serverID != 0 {
				resp, e := client.GetApiV1ServersServerIdIsoWithResponse(cctx, serverID)
				if e != nil {
					return e
				}
				if resp.StatusCode() == 200 {
					out.attached = firstJSON(resp.JSON200, resp.HALJSON200, resp.Body)
				}
				cat, e := client.GetApiV1ServersServerIdIsoimagesWithResponse(cctx, serverID)
				if e != nil {
					return e
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
				if resp.StatusCode() == 200 {
					switch {
					case resp.JSON200 != nil:
						out.userISOs = *resp.JSON200
					case resp.HALJSON200 != nil:
						out.userISOs = *resp.HALJSON200
					}
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
		task := firstTask(resp.HALJSON202, resp.JSON202)
		if resp.StatusCode() != 202 && resp.StatusCode() != 200 {
			return isoActionMsg{action: "attach", err: fmt.Errorf("HTTP %d", resp.StatusCode())}
		}
		return isoActionMsg{action: "attach", task: task}
	}
}

func isoDetachCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.DeleteApiV1ServersServerIdIsoWithResponse(ctx, serverID)
		if err != nil {
			return isoActionMsg{action: "detach", err: err}
		}
		if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
			return isoActionMsg{action: "detach", err: fmt.Errorf("HTTP %d", resp.StatusCode())}
		}
		return isoActionMsg{action: "detach"}
	}
}

func pollJobs(ctx context.Context, client *scpclient.ClientWithResponses, jobs []job) tea.Msg {
	out := make([]job, len(jobs))
	copy(out, jobs)
	for i := range out {
		j := &out[i]
		if j.Done || j.UUID == "" {
			continue
		}
		resp, err := client.GetApiV1TasksUuidWithResponse(ctx, j.UUID)
		if err != nil {
			j.Err = err.Error()
			j.Done = true
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
	return jobsPolledMsg{jobs: out}
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func jobFromTask(serverID int32, serverName, action string, task *scpclient.TaskInfo, err error) job {
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
	j.State = "OK"
	j.Done = true
	return j
}
