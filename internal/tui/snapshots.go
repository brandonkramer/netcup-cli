package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func resourceItems(kind string, data any) []resourceItem {
	raw, _ := json.Marshal(data)
	var values []map[string]any
	if json.Unmarshal(raw, &values) != nil {
		var one map[string]any
		if json.Unmarshal(raw, &one) == nil {
			values = []map[string]any{one}
		}
	}
	items := make([]resourceItem, 0, len(values))
	for i, v := range values {
		b, _ := json.MarshalIndent(v, "", "  ")
		id := fieldString(v, "id", "name", "key", "mac", "uuid")
		title := fieldString(v, "name", "key", "mac", "id")
		if title == "" {
			title = fmt.Sprintf("%s %d", kind, i+1)
		}
		items = append(items, resourceItem{kind: kind, id: id, title: title, description: compactJSON(v), raw: string(b)})
	}
	return items
}

func fieldString(v map[string]any, names ...string) string {
	for _, name := range names {
		if x, ok := v[name]; ok {
			return fmt.Sprint(x)
		}
	}
	return ""
}

func compactJSON(v map[string]any) string { b, _ := json.Marshal(v); return truncate(string(b), 110) }

func loadSnapshotsCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdSnapshotsWithResponse(ctx, id)
		if err != nil {
			return resourceLoadedMsg{tab: "snapshots", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "snapshots", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "snapshots", err: apiStatusErr("snapshots", resp.StatusCode(), resp.Body)}
		}
		return resourceLoadedMsg{tab: "snapshots", items: resourceItems("snapshot", firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
	}
}

func snapshotActionCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, action, name string) tea.Cmd {
	return func() tea.Msg {
		var (
			status int
			body   []byte
			task   *scpclient.TaskInfo
			err    error
		)
		switch action {
		case "snapshot-create":
			resp, e := client.PostApiV1ServersServerIdSnapshotsWithResponse(ctx, id, scpclient.ServerSnapshotCreate{Name: name})
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status, body = resp.StatusCode(), resp.Body
			task = extractTask(status, body, resp.JSON202, resp.HALJSON202)
		case "snapshot-delete":
			resp, e := client.DeleteApiV1ServersServerIdSnapshotsNameWithResponse(ctx, id, name)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status, body = resp.StatusCode(), resp.Body
			task = extractTask(status, body, resp.JSON202, resp.HALJSON202)
		case "snapshot-revert":
			resp, e := client.PostApiV1ServersServerIdSnapshotsNameRevertWithResponse(ctx, id, name)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status, body = resp.StatusCode(), resp.Body
			task = extractTask(status, body, resp.JSON202, resp.HALJSON202)
		case "snapshot-export":
			resp, e := client.PostApiV1ServersServerIdSnapshotsNameExportWithResponse(ctx, id, name)
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status, body = resp.StatusCode(), resp.Body
			task = extractTask(status, body, resp.JSON202, resp.HALJSON202)
		case "snapshot-dry-run":
			resp, e := client.PostApiV1ServersServerIdSnapshotsDryrunWithResponse(ctx, id, scpclient.ServerSnapshotCreateCheck{})
			if e != nil {
				return resourceActionMsg{action: action, err: e}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			if resp.StatusCode() != 200 && resp.StatusCode() != 204 {
				return resourceActionMsg{action: action, err: apiStatusErr(action, resp.StatusCode(), resp.Body)}
			}
			return resourceActionMsg{action: action, status: resp.StatusCode()}
		default:
			return resourceActionMsg{action: action, err: tuiError("unknown snapshot action")}
		}
		if err != nil {
			return resourceActionMsg{action: action, err: err}
		}
		if status != 200 && status != 202 && status != 204 {
			return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, body)}
		}
		return resourceActionMsg{action: action, status: status, task: task}
	}
}

func (m model) updateSnapshotsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.snapshotsList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	it, _ := m.snapshotsList.SelectedItem().(resourceItem)
	switch msg.String() {
	case "R":
		m.loading = true
		return m, loadSnapshotsCmd(m.ctx, m.deps.Client, m.server.ID)
	case "enter":
		if it.raw != "" {
			m.resourceDetail = it.raw
			m.viewport.SetContent(it.raw)
			m.viewport.GotoTop()
			return m, nil
		}
	case "c":
		return m.beginResourceEdit("snapshot-create", "snapshot name")
	case "D":
		return m, snapshotActionCmd(m.ctx, m.deps.Client, m.server.ID, "snapshot-dry-run", "")
	case "d", "r", "x":
		if it.id == "" {
			m.status = "select a snapshot"
			return m, nil
		}
		action := map[string]string{"d": "snapshot-delete", "r": "snapshot-revert", "x": "snapshot-export"}[msg.String()]
		if action == "snapshot-export" {
			return m, snapshotActionCmd(m.ctx, m.deps.Client, m.server.ID, action, it.id)
		}
		m.confirm = &confirmState{action: action, ids: []int32{m.server.ID}, names: []string{it.title}, extra: it.id}
		return m, nil
	}
	return m.updateActiveList(msg)
}

func (m model) snapshotsView() string {
	right := stylePanelTitle.Render("Snapshot detail") + "\n\n" + styleMuted.Render("Enter opens selected snapshot; c creates; D dry-run.")
	if m.resourceDetail != "" {
		right = m.viewport.View()
	}
	return m.twoPanels(m.snapshotsList.View(), right)
}
