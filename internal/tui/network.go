package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func parseID(s string) int32 { n, _ := strconv.ParseInt(s, 10, 32); return int32(n) }

func loadNetworkCmd(ctx context.Context, client *scpclient.ClientWithResponses, serverID, userID int32) tea.Cmd {
	return func() tea.Msg {
		items := make([]resourceItem, 0)
		var warns []string

		if serverID != 0 {
			r, e := client.GetApiV1ServersServerIdInterfacesWithResponse(ctx, serverID, &scpclient.GetApiV1ServersServerIdInterfacesParams{})
			if e != nil {
				return resourceLoadedMsg{tab: "network", err: e}
			}
			if r == nil {
				return resourceLoadedMsg{tab: "network", err: tuiError("empty NIC response")}
			}
			if r.StatusCode() == 401 {
				return resourceLoadedMsg{tab: "network", err: apiStatusErr("nics", r.StatusCode(), r.Body)}
			}
			if r.StatusCode() == 200 {
				items = append(items, resourceItems("nic", firstJSON(r.JSON200, r.HALJSON200, r.Body))...)
			} else {
				warns = append(warns, fmt.Sprintf("NICs HTTP %d", r.StatusCode()))
			}
		}

		appendOptional := func(kind, label string, status int, body []byte, data any, err error) error {
			if err != nil {
				warns = append(warns, label+": "+err.Error())
				return nil
			}
			if status == 401 {
				return apiStatusErr(label, status, body)
			}
			if status == 200 {
				items = append(items, resourceItems(kind, data)...)
			} else {
				warns = append(warns, fmt.Sprintf("%s HTTP %d", label, status))
			}
			return nil
		}

		if userID != 0 {
			r, e := client.GetApiV1UsersUserIdFailoveripsV4WithResponse(ctx, userID, nil)
			data := any(nil)
			body := []byte(nil)
			status := 0
			if r != nil {
				status, body, data = r.StatusCode(), r.Body, firstJSON(r.JSON200, r.HALJSON200, r.Body)
			}
			if err := appendOptional("failover-v4", "failover-v4", status, body, data, e); err != nil {
				return resourceLoadedMsg{tab: "network", err: err}
			}

			r6, e6 := client.GetApiV1UsersUserIdFailoveripsV6WithResponse(ctx, userID, nil)
			data, body, status = nil, nil, 0
			if r6 != nil {
				status, body, data = r6.StatusCode(), r6.Body, firstJSON(r6.JSON200, r6.HALJSON200, r6.Body)
			}
			if err := appendOptional("failover-v6", "failover-v6", status, body, data, e6); err != nil {
				return resourceLoadedMsg{tab: "network", err: err}
			}

			rv, ev := client.GetApiV1UsersUserIdVlansWithResponse(ctx, userID, nil)
			data, body, status = nil, nil, 0
			if rv != nil {
				status, body, data = rv.StatusCode(), rv.Body, firstJSON(rv.JSON200, rv.HALJSON200, rv.Body)
			}
			if err := appendOptional("vlan", "vlans", status, body, data, ev); err != nil {
				return resourceLoadedMsg{tab: "network", err: err}
			}
		}

		msg := resourceLoadedMsg{tab: "network", items: items}
		if len(warns) > 0 {
			msg.warning = strings.Join(warns, " · ")
		}
		return msg
	}
}

func (m model) updateNetworkKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.networkList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	it, _ := m.networkList.SelectedItem().(resourceItem)
	switch msg.String() {
	case "R":
		m.loading = true
		return m, loadNetworkCmd(m.ctx, m.deps.Client, m.server.ID, m.deps.UserID)
	case "enter":
		return m, networkDetailCmd(m.ctx, m.deps.Client, m.deps.UserID, m.server.ID, it)
	case "c":
		if m.server.ID == 0 {
			m.status = "select a server first"
			return m, nil
		}
		ph := "vlanId [driver]"
		if it.kind == "vlan" && it.id != "" {
			ph = it.id + " [driver]"
		}
		return m.beginResourceEdit("nic-create", ph)
	case "e":
		if it.kind == "vlan" && it.id != "" {
			return m.beginResourceEditCtx("vlan-rename", it.id, "new VLAN name")
		}
		m.status = "select a VLAN row to rename"
		return m, nil
	case "d":
		if it.kind == "nic" && it.id != "" {
			m.confirm = &confirmState{action: "nic-delete", names: []string{it.title}, extra: it.id}
			return m, nil
		}
	case "r":
		return m.beginResourceEdit("rdns-set", "IP hostname")
	case "g":
		return m.beginResourceEdit("rdns-get", "IP")
	case "x":
		return m.beginResourceEdit("rdns-delete", "IP to clear")
	case "f":
		kind := it.kind
		if kind != "failover-v4" && kind != "failover-v6" {
			kind = "failover-v4"
		}
		ph := "id targetServerId"
		if it.id != "" {
			ph = it.id + " <targetServerId>"
		}
		return m.beginResourceEditCtx("failover-route", kind, ph)
	}
	return m.updateActiveList(msg)
}

func networkDetailCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid, sid int32, it resourceItem) tea.Cmd {
	return func() tea.Msg {
		var data any
		var status int
		var body []byte
		var err error
		switch it.kind {
		case "nic":
			r, e := c.GetApiV1ServersServerIdInterfacesMacWithResponse(ctx, sid, it.id, nil)
			if e != nil {
				return resourceLoadedMsg{tab: "network", err: e}
			}
			if r == nil {
				return resourceLoadedMsg{tab: "network", err: tuiError("empty NIC response")}
			}
			err, status, body = nil, r.StatusCode(), r.Body
			data = firstJSON(r.JSON200, r.HALJSON200, r.Body)
		case "vlan":
			r, e := c.GetApiV1UsersUserIdVlansVlanIdWithResponse(ctx, uid, parseID(it.id))
			if e != nil {
				return resourceLoadedMsg{tab: "network", err: e}
			}
			if r == nil {
				return resourceLoadedMsg{tab: "network", err: tuiError("empty VLAN response")}
			}
			err, status, body = nil, r.StatusCode(), r.Body
			data = firstJSON(r.JSON200, r.HALJSON200, r.Body)
		default:
			return resourceLoadedMsg{tab: "network", detail: it.raw}
		}
		if err != nil {
			return resourceLoadedMsg{tab: "network", err: err}
		}
		if status != 200 {
			return resourceLoadedMsg{tab: "network", err: apiStatusErr("network", status, body)}
		}
		return resourceLoadedMsg{tab: "network", detail: prettyJSON(data)}
	}
}

func (m model) networkView() string {
	right := stylePanelTitle.Render("Network") + "\n\n" + styleMuted.Render("NICs · failover v4/v6 · VLANs\nenter detail · c create NIC · e rename VLAN · d delete NIC · r rDNS · f route")
	if m.resourceDetail != "" {
		right = m.viewport.View()
	}
	return m.twoPanels(m.networkList.View(), right)
}

func nicCreateCmd(ctx context.Context, c *scpclient.ClientWithResponses, sid int32, value string) tea.Cmd {
	return func() tea.Msg {
		parts := bytes.Fields([]byte(value))
		if len(parts) < 1 {
			return resourceActionMsg{action: "nic-create", err: tuiError("use: vlanId [driver]")}
		}
		vlanID := parseID(string(parts[0]))
		if vlanID == 0 {
			return resourceActionMsg{action: "nic-create", err: tuiError("vlanId required")}
		}
		driver := string(scpclient.NetworkDriverVIRTIO)
		if len(parts) > 1 {
			driver = string(parts[1])
		}
		body := scpclient.ServerCreateNicVlan{
			VlanId:        vlanID,
			NetworkDriver: scpclient.NetworkDriver(driver),
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return resourceActionMsg{action: "nic-create", err: err}
		}
		r, e := c.PostApiV1ServersServerIdInterfacesWithBodyWithResponse(ctx, sid, "application/json", bytes.NewReader(raw))
		if e != nil {
			return resourceActionMsg{action: "nic-create", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "nic-create", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "nic-create", status: status, err: apiStatusErr("nic create", status, r.Body)}
		}
		return resourceActionMsg{action: "nic-create", status: status, task: extractTask(status, r.Body, r.JSON202, r.HALJSON202)}
	}
}

func vlanRenameCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, vlanID, name string) tea.Cmd {
	return func() tea.Msg {
		name = strings.TrimSpace(name)
		if name == "" {
			return resourceActionMsg{action: "vlan-rename", err: tuiError("name required")}
		}
		vid := parseID(vlanID)
		if vid == 0 {
			return resourceActionMsg{action: "vlan-rename", err: tuiError("vlan id required")}
		}
		body := scpclient.VLanSave{Name: &name}
		r, e := c.PutApiV1UsersUserIdVlansVlanIdWithResponse(ctx, uid, vid, body)
		if e != nil {
			return resourceActionMsg{action: "vlan-rename", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "vlan-rename", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 202 && status != 204 {
			return resourceActionMsg{action: "vlan-rename", status: status, err: apiStatusErr("vlan rename", status, r.Body)}
		}
		return resourceActionMsg{action: "vlan-rename", status: status, task: extractTask(status, r.Body)}
	}
}

func routeFailoverCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, kind, value string) tea.Cmd {
	return func() tea.Msg {
		parts := bytes.Fields([]byte(value))
		if len(parts) != 2 {
			return resourceActionMsg{action: "failover-route", err: tuiError("use: id targetServerId")}
		}
		id := parseID(string(parts[0]))
		body := []byte(`{"serverId":` + string(parts[1]) + `}`)
		var (
			status   int
			respBody []byte
			task     *scpclient.TaskInfo
			err      error
		)
		if kind == "failover-v6" {
			r, e := c.PatchApiV1UsersUserIdFailoveripsV6IdWithBodyWithResponse(ctx, uid, id, "application/merge-patch+json", bytes.NewReader(body))
			err = e
			if r != nil {
				status, respBody = r.StatusCode(), r.Body
				task = extractTask(status, respBody, r.JSON202, r.HALJSON202)
			}
		} else {
			r, e := c.PatchApiV1UsersUserIdFailoveripsV4IdWithBodyWithResponse(ctx, uid, id, "application/merge-patch+json", bytes.NewReader(body))
			err = e
			if r != nil {
				status, respBody = r.StatusCode(), r.Body
				task = extractTask(status, respBody, r.JSON202, r.HALJSON202)
			}
		}
		if err != nil {
			return resourceActionMsg{action: "failover-route", err: err}
		}
		if status != 200 && status != 202 {
			return resourceActionMsg{action: "failover-route", err: apiStatusErr("route failover", status, respBody)}
		}
		return resourceActionMsg{action: "failover-route", task: task}
	}
}
