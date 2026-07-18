package tui

import (
	"bytes"
	"context"
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func loadDisksCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdDisksWithResponse(ctx, id)
		if err != nil {
			return resourceLoadedMsg{tab: "disks", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "disks", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "disks", err: apiStatusErr("disks", resp.StatusCode(), resp.Body)}
		}
		return resourceLoadedMsg{tab: "disks", items: resourceItems("disk", firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
	}
}

func disksDriversCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetApiV1ServersServerIdDisksSupportedDriversWithResponse(ctx, id)
		if err != nil {
			return resourceLoadedMsg{tab: "disks", mode: "drivers", err: err}
		}
		if resp == nil {
			return resourceLoadedMsg{tab: "disks", mode: "drivers", err: tuiError("empty response")}
		}
		if resp.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "disks", mode: "drivers", err: apiStatusErr("disk drivers", resp.StatusCode(), resp.Body)}
		}
		return resourceLoadedMsg{tab: "disks", mode: "drivers", detail: prettyJSON(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
	}
}

func diskActionCmd(ctx context.Context, client *scpclient.ClientWithResponses, id int32, action, disk, value string) tea.Cmd {
	return func() tea.Msg {
		switch action {
		case "disk-detail":
			resp, err := client.GetApiV1ServersServerIdDisksDiskNameWithResponse(ctx, id, disk)
			if err != nil {
				return resourceLoadedMsg{tab: "disks", err: err}
			}
			if resp == nil {
				return resourceLoadedMsg{tab: "disks", err: tuiError("empty response")}
			}
			if resp.StatusCode() != 200 {
				return resourceLoadedMsg{tab: "disks", err: apiStatusErr("disk", resp.StatusCode(), resp.Body)}
			}
			return resourceLoadedMsg{tab: "disks", detail: prettyJSON(firstJSON(resp.JSON200, resp.HALJSON200, resp.Body))}
		case "disk-format":
			resp, err := client.PostApiV1ServersServerIdDisksDiskNameFormatWithResponse(ctx, id, disk)
			if err != nil {
				return resourceActionMsg{action: action, err: err}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := resp.StatusCode()
			if status != 200 && status != 202 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, resp.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, resp.Body, resp.JSON202, resp.HALJSON202)}
		case "disk-driver":
			body := []byte(fmt.Sprintf(`{"driver":%q}`, value))
			resp, err := client.PatchApiV1ServersServerIdDisksWithBodyWithResponse(ctx, id, "application/merge-patch+json", bytes.NewReader(body))
			if err != nil {
				return resourceActionMsg{action: action, err: err}
			}
			if resp == nil {
				return resourceActionMsg{action: action, err: tuiError("empty response")}
			}
			status := resp.StatusCode()
			if status != 200 && status != 202 {
				return resourceActionMsg{action: action, status: status, err: apiStatusErr(action, status, resp.Body)}
			}
			return resourceActionMsg{action: action, status: status, task: extractTask(status, resp.Body, resp.JSON202, resp.HALJSON202)}
		}
		return resourceActionMsg{action: action, err: fmt.Errorf("unknown disk action")}
	}
}

func (m model) updateDisksKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.disksList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	it, _ := m.disksList.SelectedItem().(resourceItem)
	switch msg.String() {
	case "R":
		m.loading = true
		return m, loadDisksCmd(m.ctx, m.deps.Client, m.server.ID)
	case "enter":
		if it.id != "" {
			return m, diskActionCmd(m.ctx, m.deps.Client, m.server.ID, "disk-detail", it.id, "")
		}
	case "v":
		return m, disksDriversCmd(m.ctx, m.deps.Client, m.server.ID)
	case "s":
		return m.beginResourceEdit("disk-driver", "driver")
	case "f":
		if it.id == "" {
			m.status = "select a disk"
			return m, nil
		}
		m.confirm = &confirmState{action: "disk-format", ids: []int32{m.server.ID}, names: []string{it.title}, extra: it.id}
		return m, nil
	}
	return m.updateActiveList(msg)
}

func (m model) disksView() string {
	right := stylePanelTitle.Render("Disk detail") + "\n\n" + styleMuted.Render("Enter detail · v drivers · s set driver · f format")
	if m.resourceDetail != "" {
		right = m.viewport.View()
	}
	return m.twoPanels(m.disksList.View(), right)
}
