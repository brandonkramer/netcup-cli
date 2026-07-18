package tui

import (
	"fmt"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) serversView() string {
	left := m.serversList.View()
	if m.loading && len(m.serversList.Items()) == 0 {
		left = styleMuted.Render("loading servers…")
	}
	return m.twoPanels(left, m.detailContent())
}

func (m model) detailContent() string {
	if m.detail == nil {
		return stylePanelTitle.Render("Detail") + "\n\n" +
			styleMuted.Render("↑↓ select a server · Enter to reload")
	}

	s := m.detail
	live := s.ServerLiveInfo
	state, uptime, cpus, ram := "", "", "", ""
	boot, uefi := "", ""
	if live != nil {
		state = derefState(live.State)
		uptime = formatUptime(live.UptimeInSeconds)
		if live.CpuCount != nil {
			cpus = fmt.Sprintf("%d", *live.CpuCount)
		}
		if live.MaxServerMemoryInMiB != nil {
			ram = fmt.Sprintf("%d MiB", *live.MaxServerMemoryInMiB)
		}
		if live.Bootorder != nil {
			parts := make([]string, 0, len(*live.Bootorder))
			for _, b := range *live.Bootorder {
				parts = append(parts, string(b))
			}
			boot = strings.Join(parts, " → ")
		}
		if live.Uefi != nil {
			if *live.Uefi {
				uefi = "yes"
			} else {
				uefi = "no"
			}
		}
	}
	site := ""
	if s.Site != nil {
		site = s.Site.City
	}
	ips := formatIPv4(s)
	nick := derefStr(s.Nickname)
	rescue := "no"
	if derefBool(s.RescueSystemActive) {
		rescue = "yes"
	}

	name := derefStr(s.Name)
	header := stylePanelTitle.Render(name)
	if state != "" {
		header += "  " + stateStyle(state).Render(state)
	}

	rows := []string{
		header,
		styleMuted.Render(fmt.Sprintf("id %d", derefInt(s.Id))),
		"",
		kv("Hostname", derefStr(s.Hostname)),
		kv("Nickname", nick),
		kv("Site", site),
		kv("IPv4", ips),
		"",
		kv("Uptime", uptime),
		kv("CPUs", cpus),
		kv("RAM", ram),
		kv("UEFI", uefi),
		kv("Boot", boot),
		kv("Rescue", rescue),
		kv("Guest", formatGuest(m.guest)),
	}
	return strings.Join(rows, "\n")
}

func formatIPv4(s *scpclient.Server) string {
	if s == nil || s.Ipv4Addresses == nil {
		return ""
	}
	parts := make([]string, 0, len(*s.Ipv4Addresses))
	for _, ip := range *s.Ipv4Addresses {
		if ip.Ip != nil {
			parts = append(parts, *ip.Ip)
		}
	}
	return strings.Join(parts, ", ")
}

func formatGuest(g any) string {
	if g == nil {
		return "…"
	}
	switch v := g.(type) {
	case *scpclient.GuestAgentStatus:
		if v == nil {
			return "—"
		}
		if v.Available == nil {
			return "unknown"
		}
		if *v.Available {
			return "available"
		}
		return "unavailable"
	case scpclient.GuestAgentStatus:
		if v.Available == nil {
			return "unknown"
		}
		if *v.Available {
			return "available"
		}
		return "unavailable"
	case map[string]any:
		if a, ok := v["available"].(bool); ok {
			if a {
				return "available"
			}
			return "unavailable"
		}
	}
	return "—"
}

func (m model) updateServersKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.serversList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	switch msg.String() {
	case " ":
		m.toggleSelect()
		return m, nil
	case "enter":
		m.guest = nil
		return m, m.loadSelectedDetail()
	case "j", "down", "k", "up":
		var cmd tea.Cmd
		m.serversList, cmd = m.serversList.Update(msg)
		// Auto-refresh detail when moving selection
		return m, tea.Batch(cmd, m.loadSelectedDetail())
	case "R":
		m.loading = true
		m.status = "refreshing servers…"
		return m, loadServersCmd(m.ctx, m.deps.Client)
	case "s":
		return m.beginAction("start")
	case "t":
		return m.beginAction("stop")
	case "r":
		return m.beginAction("reboot")
	case "P":
		return m.beginAction("poweroff")
	case "x":
		return m.beginAction("reset")
	case "u":
		return m.beginAction("suspend")
	case "a":
		m.clearSelection()
		return m, nil
	case "h":
		return m.beginEdit("hostname")
	case "n":
		return m.beginEdit("nickname")
	case "g":
		id, _ := m.focusedServer()
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.loading = true
		m.status = "loading guest agent…"
		return m, loadGuestAgentCmd(m.ctx, m.deps.Client, id)
	case "e":
		id, name := m.focusedServer()
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.confirm = &confirmState{action: "rescue-enable", ids: []int32{id}, names: []string{name}}
		return m, nil
	case "d":
		id, name := m.focusedServer()
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.confirm = &confirmState{action: "rescue-disable", ids: []int32{id}, names: []string{name}}
		return m, nil
	}
	return m.updateActiveList(msg)
}

func (m *model) toggleSelect() {
	idx := m.serversList.Index()
	items := m.serversList.Items()
	if idx < 0 || idx >= len(items) {
		return
	}
	it, ok := items[idx].(serverItem)
	if !ok {
		return
	}
	it.selected = !it.selected
	items[idx] = it
	m.serversList.SetItems(items)
}

func (m *model) clearSelection() {
	items := m.serversList.Items()
	for i, it := range items {
		si, ok := it.(serverItem)
		if !ok {
			continue
		}
		si.selected = false
		items[i] = si
	}
	m.serversList.SetItems(items)
	m.status = "selection cleared"
}

func (m model) beginAction(action string) (tea.Model, tea.Cmd) {
	ids, names := m.selectedTargets()
	if len(ids) == 0 {
		m.status = "no server selected"
		return m, nil
	}
	needsConfirm := len(ids) > 1 ||
		action == "reboot" || action == "poweroff" || action == "reset" || action == "suspend"
	if needsConfirm {
		m.confirm = &confirmState{action: action, ids: ids, names: names}
		return m, nil
	}
	return m, m.dispatchPower(action, ids, names)
}

func (m model) beginEdit(attr string) (tea.Model, tea.Cmd) {
	id, _ := m.focusedServer()
	if id == 0 {
		m.status = "no server"
		return m, nil
	}
	ti := textinput.New()
	ti.Placeholder = attr
	ti.CharLimit = 128
	ti.Width = 40
	ti.Focus()
	if m.detail != nil {
		switch attr {
		case "hostname":
			ti.SetValue(derefStr(m.detail.Hostname))
		case "nickname":
			ti.SetValue(derefStr(m.detail.Nickname))
		}
	}
	m.edit = &editMode{attr: attr, input: ti}
	return m, textinput.Blink
}

func (m model) loadSelectedDetail() tea.Cmd {
	id, _ := m.focusedServer()
	if id == 0 {
		return nil
	}
	return tea.Batch(
		loadDetailCmd(m.ctx, m.deps.Client, id),
		loadGuestAgentCmd(m.ctx, m.deps.Client, id),
	)
}

func (m model) dispatchPower(action string, ids []int32, names []string) tea.Cmd {
	var state scpclient.ServerState1
	var opt *string
	switch action {
	case "start":
		state = scpclient.ON
	case "stop":
		state = scpclient.OFF
	case "reboot":
		state = scpclient.ON
		v := "POWERCYCLE"
		opt = &v
	case "poweroff":
		state = scpclient.OFF
		v := "POWEROFF"
		opt = &v
	case "reset":
		state = scpclient.ON
		v := "RESET"
		opt = &v
	case "suspend":
		state = scpclient.SUSPENDED
	default:
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(ids))
	for i, id := range ids {
		cmds = append(cmds, powerCmd(m.ctx, m.deps.Client, id, names[i], action, state, opt))
	}
	return tea.Batch(cmds...)
}

func (m model) onServersLoaded(msg serversLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "failed to load servers"
		return m, nil
	}
	m.errMsg = ""
	items := make([]list.Item, 0, len(msg.servers))
	for _, s := range msg.servers {
		items = append(items, serverItem{server: s})
	}
	m.serversList.SetItems(items)
	m.status = fmt.Sprintf("%d servers", len(items))
	if len(items) > 0 {
		return m, m.loadSelectedDetail()
	}
	return m, nil
}

func (m model) onDetailLoaded(msg detailLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.detail = msg.server
	m.errMsg = ""
	m.status = "ready"
	return m, nil
}

func (m model) onPowerStarted(msg powerStartedMsg) (tea.Model, tea.Cmd) {
	j := jobFromTask(msg.serverID, msg.serverName, msg.action, msg.task, msg.err)
	m.status = fmt.Sprintf("%s → %s", msg.action, msg.serverName)
	cmd := m.trackJob(j)
	if j.Done && j.Err == "" {
		return m, tea.Batch(cmd, loadServersCmd(m.ctx, m.deps.Client), m.loadSelectedDetail())
	}
	return m, cmd
}

func (m model) onJobsPolled(msg jobsPolledMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	updates := make(map[string]job, len(msg.jobs))
	for _, j := range msg.jobs {
		if j.UUID != "" {
			updates[j.UUID] = j
		}
	}
	for i, j := range m.jobs {
		if u, ok := updates[j.UUID]; ok {
			m.jobs[i] = u
		}
	}
	allDone := true
	for _, j := range m.jobs {
		if !j.Done {
			allDone = false
			break
		}
	}
	if allDone {
		m.status = "jobs finished"
		return m, tea.Batch(loadServersCmd(m.ctx, m.deps.Client), m.loadSelectedDetail())
	}
	if !m.deadline.IsZero() && timeAfterDeadline(m) {
		m.errMsg = "task wait timed out"
		return m, nil
	}
	return m, m.pollJobsCmd()
}

func timeAfterDeadline(m model) bool {
	return !m.deadline.IsZero() && m.deadline.Before(nowTime())
}
