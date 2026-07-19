package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tabKind int

const (
	tabServers tabKind = iota
	tabTasks
	tabSnapshots
	tabDisks
	tabFirewall
	tabMedia
	tabNetwork
	tabAccount
)

var tabNames = []string{"Servers", "Tasks", "Snapshots", "Disks", "Firewall", "Media", "Network", "Account"}

type serverContext struct {
	ID   int32
	Name string
}

type confirmState struct {
	action string
	ids    []int32
	names  []string
	extra  string // optional payload (e.g. user iso name)
}

type editMode struct {
	attr    string
	context string
	input   textinput.Model
}

type model struct {
	ctx  context.Context
	deps Deps

	tab tabKind

	serversList   list.Model
	tasksList     list.Model
	snapshotsList list.Model
	disksList     list.Model
	firewallList  list.Model
	mediaList     list.Model
	networkList   list.Model
	accountList   list.Model
	viewport      viewport.Model
	spinner       spinner.Model

	loading bool
	detail  *scpclient.Server
	server  serverContext
	guest   any
	jobs    []job
	confirm *confirmState
	edit    *editMode

	metricsKind    string // Servers sub-view, never a top tab.
	metricsHours   int32
	metricsData    any
	mediaAttached  any
	resourceDetail string
	resourceMode   string

	status   string
	errMsg   string
	width    int
	height   int
	bodyH    int // allocated body rows (panels sized to this)
	ready    bool
	deadline time.Time

	authGate   bool
	authReason string
	deviceURL  string
	deviceCode string
	health     healthMsg
}

func newModel(ctx context.Context, deps Deps) model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colAccent).
		Foreground(colAccent).
		Bold(true).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colAccent).
		Foreground(colMuted).
		Padding(0, 0, 0, 1)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(colValue).Padding(0, 0, 0, 2)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 0, 0, 2)

	styleList := func(l *list.Model, title string) {
		l.Title = title
		l.SetShowStatusBar(false)
		l.SetShowPagination(false)
		l.SetFilteringEnabled(true)
		l.SetShowHelp(false)
		l.DisableQuitKeybindings()
		l.Styles.Title = styleAppTitle
		l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
		l.Styles.FilterPrompt = styleFocus
		l.Styles.FilterCursor = styleFocus
	}

	servers := list.New([]list.Item{}, delegate, 0, 0)
	styleList(&servers, "Servers")

	tasks := list.New([]list.Item{}, delegate, 0, 0)
	styleList(&tasks, "Tasks")

	newList := func(title string) list.Model {
		l := list.New([]list.Item{}, delegate, 0, 0)
		styleList(&l, title)
		return l
	}
	media := newList("Media")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styleFocus

	vp := viewport.New(0, 0)

	return model{
		ctx:           ctx,
		deps:          deps,
		tab:           tabServers,
		serversList:   servers,
		tasksList:     tasks,
		snapshotsList: newList("Snapshots"),
		disksList:     newList("Disks"),
		firewallList:  newList("Firewall"),
		mediaList:     media,
		networkList:   newList("Network"),
		accountList:   newList("Account"),
		viewport:      vp,
		spinner:       sp,
		loading:       true,
		metricsKind:   "cpu",
		metricsHours:  24,
		status:        "loading servers…",
	}
}

func teaProgram(m model) *tea.Program {
	return tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(m.ctx))
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, loadHealthCmd(m.ctx, m.deps), checkSessionCmd(m.ctx, m.deps)}
	if !m.authGate && m.deps.Client != nil {
		cmds = append(cmds, loadServersCmd(m.ctx, m.deps, false))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case serversLoadedMsg:
		mod, cmd := m.onServersLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case detailLoadedMsg:
		mod, cmd := m.onDetailLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case guestAgentMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m.withAuthGate(msg.err), nil
		}
		m.guest = msg.status
		return m, nil
	case powerStartedMsg:
		mod, cmd := m.onPowerStarted(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case jobsPolledMsg:
		mod, cmd := m.onJobsPolled(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case tasksLoadedMsg:
		mod, cmd := m.onTasksLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case taskCancelMsg:
		mod, cmd := m.onTaskCancel(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case metricsLoadedMsg:
		mod, cmd := m.onMetricsLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case mediaLoadedMsg:
		mod, cmd := m.onMediaLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case isoActionMsg:
		mod, cmd := m.onISOAction(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case attrSetMsg:
		mod, cmd := m.onAttrSet(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case rescueMsg:
		mod, cmd := m.onRescue(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case resourceLoadedMsg:
		mod, cmd := m.onResourceLoaded(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case resourceActionMsg:
		mod, cmd := m.onResourceAction(msg)
		return applyAuthGate(mod, cmd, msg.err)
	case authGateMsg:
		if !msg.ok {
			m.authGate = true
			m.authReason = msg.reason
			m.loading = false
		}
		return m, nil
	case deviceLoginPromptMsg:
		return m.onDeviceLoginPrompt(msg)
	case sessionReadyMsg:
		return m.onSessionReady(msg)
	case healthMsg:
		return m.onHealth(msg)

	case tea.KeyMsg:
		if m.authGate {
			return m.handleAuthGateKey(msg)
		}
		if m.edit != nil {
			return m.handleEditKey(msg)
		}
		if m.confirm != nil {
			return m.handleConfirmKey(msg)
		}
		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			return m.switchTab(tabServers)
		case "2":
			return m.switchTab(tabTasks)
		case "3":
			return m.switchTab(tabSnapshots)
		case "4":
			return m.switchTab(tabDisks)
		case "5":
			return m.switchTab(tabFirewall)
		case "6":
			return m.switchTab(tabMedia)
		case "7":
			return m.switchTab(tabNetwork)
		case "8":
			return m.switchTab(tabAccount)
		case "tab":
			return m.switchTab((m.tab + 1) % tabKind(len(tabNames)))
		case "shift+tab":
			t := m.tab - 1
			if t < 0 {
				t = tabKind(len(tabNames) - 1)
			}
			return m.switchTab(t)
		}
		switch m.tab {
		case tabServers:
			return m.updateServersKeys(msg)
		case tabTasks:
			return m.updateTasksKeys(msg)
		case tabSnapshots:
			return m.updateSnapshotsKeys(msg)
		case tabDisks:
			return m.updateDisksKeys(msg)
		case tabFirewall:
			return m.updateFirewallKeys(msg)
		case tabMedia:
			return m.updateMediaKeys(msg)
		case tabNetwork:
			return m.updateNetworkKeys(msg)
		case tabAccount:
			return m.updateAccountKeys(msg)
		}
	}

	return m.updateActiveList(msg)
}

func (m model) View() string {
	if !m.ready {
		return styleMuted.Render(" starting… ")
	}
	if m.authGate {
		return m.authGateView()
	}
	if m.confirm != nil {
		return m.confirmView()
	}
	if m.edit != nil {
		return m.editView()
	}

	m.layout()

	// Build bottom first so we never overflow the terminal — overflow scrolls
	// away the TOP (tabs). Tip, jobs, and status share one meta line.
	var bottomParts []string
	if m.errMsg != "" {
		bottomParts = append(bottomParts, clampLines(styleErr.Render("✗ "+m.errMsg), 1, m.width))
	}
	bottomParts = append(bottomParts,
		clampLines(m.keysBarView(), 1, m.width),
		m.metaBarView(),
	)
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomParts...)
	top := m.tabsView()

	chrome := lipgloss.Height(top) + lipgloss.Height(bottom)
	bodyH := m.height - chrome
	if bodyH < 4 {
		bodyH = 4
	}
	m.bodyH = bodyH
	// Re-apply list sizes for the final bodyH (layout used an estimate).
	m.layoutBodySizes()

	var body string
	switch m.tab {
	case tabServers:
		body = m.serversView()
	case tabTasks:
		body = m.onePanel(m.tasksList.View())
	case tabSnapshots:
		body = m.snapshotsView()
	case tabDisks:
		body = m.disksView()
	case tabFirewall:
		body = m.firewallView()
	case tabMedia:
		body = m.mediaView()
	case tabNetwork:
		body = m.networkView()
	case tabAccount:
		body = m.accountView()
	}

	out := lipgloss.JoinVertical(lipgloss.Left, top, body, bottom)
	// If body still overshoots, trim body only — never clip help/status.
	if h := lipgloss.Height(out); h > m.height && m.height > 0 {
		maxBody := m.height - lipgloss.Height(top) - lipgloss.Height(bottom)
		if maxBody < 1 {
			maxBody = 1
		}
		body = lipgloss.NewStyle().MaxHeight(maxBody).MaxWidth(m.width).Render(body)
		out = lipgloss.JoinVertical(lipgloss.Left, top, body, bottom)
	}
	return out
}

func (m model) statusText() string {
	status := m.status
	if m.loading {
		status = m.spinner.View() + " " + status
	}
	return status
}

func (m *model) setErr(err error, fallback string) {
	if err == nil {
		return
	}
	m.errMsg = err.Error()
	m.status = fallback
	if strings.Contains(m.errMsg, "HTTP 401") || strings.Contains(m.errMsg, "HTTP 403") {
		m.status = "run netcup auth login"
	}
}

func (m *model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// Estimate body height; View() recomputes exactly after measuring chrome.
	const tabsH, bottomLines = 2, 3 // tabs/context + keys + tip|jobs|status
	bodyH := m.height - tabsH - bottomLines
	if bodyH < 4 {
		bodyH = 4
	}
	m.bodyH = bodyH
	m.layoutBodySizes()
}

func (m *model) layoutBodySizes() {
	listOuter, detailOuter := m.panelSplit()
	// Content area inside border (2) + horizontal padding (2).
	listInnerW := listOuter - 4
	detailInnerW := detailOuter - 4
	innerH := m.bodyH - 2
	if listInnerW < 8 {
		listInnerW = 8
	}
	if detailInnerW < 8 {
		detailInnerW = 8
	}
	if innerH < 3 {
		innerH = 3
	}
	m.serversList.SetSize(listInnerW, innerH)
	m.tasksList.SetSize(m.width-4, innerH)
	m.snapshotsList.SetSize(listInnerW, innerH)
	m.disksList.SetSize(listInnerW, innerH)
	m.firewallList.SetSize(listInnerW, innerH)
	m.mediaList.SetSize(listInnerW, innerH)
	m.networkList.SetSize(listInnerW, innerH)
	m.accountList.SetSize(listInnerW, innerH)
	m.viewport.Width = detailInnerW
	m.viewport.Height = innerH
}

func (m model) jobsHeight() int {
	n := len(m.jobs)
	if n == 0 {
		return 1
	}
	if n > 3 {
		n = 3
	}
	return n + 1
}

func (m model) switchTab(t tabKind) (tea.Model, tea.Cmd) {
	m.tab = t
	m.errMsg = ""
	m.layout()
	switch t {
	case tabServers:
		m.status = "servers"
		return m, nil
	case tabTasks:
		m.loading = true
		m.status = "loading tasks…"
		return m, loadTasksCmd(m.ctx, m.deps.Client)
	case tabSnapshots, tabDisks, tabFirewall, tabMedia, tabNetwork:
		if m.server.ID == 0 {
			m.status = "select a server on Servers tab first"
			return m, nil
		}
		m.loading = true
		return m, m.loadTabCmd(t)
	case tabAccount:
		m.loading = true
		return m, m.loadTabCmd(t)
	}
	return m, nil
}

func (m model) updateActiveList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.tab {
	case tabServers:
		if m.serversList.FilterState() == list.Filtering {
			m.serversList, cmd = m.serversList.Update(msg)
			return m, cmd
		}
		m.serversList, cmd = m.serversList.Update(msg)
	case tabTasks:
		m.tasksList, cmd = m.tasksList.Update(msg)
	case tabSnapshots:
		m.snapshotsList, cmd = m.snapshotsList.Update(msg)
	case tabDisks:
		m.disksList, cmd = m.disksList.Update(msg)
	case tabFirewall:
		m.firewallList, cmd = m.firewallList.Update(msg)
	case tabMedia:
		m.mediaList, cmd = m.mediaList.Update(msg)
	case tabNetwork:
		m.networkList, cmd = m.networkList.Update(msg)
	case tabAccount:
		m.accountList, cmd = m.accountList.Update(msg)
	}
	return m, cmd
}

func (m model) focusedServer() (int32, string) {
	items := m.serversList.Items()
	idx := m.serversList.Index()
	if idx < 0 || idx >= len(items) {
		return 0, ""
	}
	si, ok := items[idx].(serverItem)
	if !ok || si.server.Id == nil {
		return 0, ""
	}
	return *si.server.Id, serverLabel(si.server)
}

func (m model) selectedTargets() (ids []int32, names []string) {
	items := m.serversList.Items()
	for _, it := range items {
		si, ok := it.(serverItem)
		if !ok || !si.selected || si.server.Id == nil {
			continue
		}
		ids = append(ids, *si.server.Id)
		names = append(names, serverLabel(si.server))
	}
	if len(ids) > 0 {
		return ids, names
	}
	id, name := m.focusedServer()
	if id == 0 {
		return nil, nil
	}
	return []int32{id}, []string{name}
}

func (m model) tabsView() string {
	// Powerline tabs replace the old gradient logo at the top.
	row := m.tabsBarView()
	right := strings.TrimSpace(m.healthChip() + " " + m.serverContextChip())
	if right != "" {
		gap := m.width - lipgloss.Width(row) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
		row = row + strings.Repeat(" ", gap) + right
	}
	row = clampLines(row, 1, m.width)
	return row
}

func (m model) helpView() string {
	var help string
	switch m.tab {
	case tabServers:
		help = "↑↓ move · space select · / filter · s/t/r power · P/x/u hard · h/n edit · e/d rescue · R refresh · q quit"
	case tabTasks:
		help = "↑↓ move · / filter · c cancel · R refresh · q quit"
	case tabSnapshots:
		help = "c create · D dry-run · r revert · d delete · x export · R refresh"
	case tabMedia:
		help = "↑↓ move · enter attach · D detach · R refresh · q quit"
	}
	return styleHelp.Width(m.width).MaxWidth(m.width).Render(help)
}

func (m model) jobsView() string {
	if len(m.jobs) == 0 {
		return styleMuted.Render("jobs · none")
	}
	start := 0
	if len(m.jobs) > 3 {
		start = len(m.jobs) - 3
	}
	var bits []string
	for _, j := range m.jobs[start:] {
		st := stateStyle(j.State).Render(j.State)
		bit := fmt.Sprintf("%s %s %s", j.Action, j.ServerName, st)
		if j.Progress > 0 {
			bit += fmt.Sprintf(" %.0f%%", j.Progress)
		}
		if j.Err != "" {
			bit += " " + styleErr.Render(j.Err)
		}
		bits = append(bits, bit)
	}
	return styleJobsTitle.Render("jobs") + "  " + strings.Join(bits, " · ")
}

func (m model) confirmView() string {
	c := m.confirm
	var b strings.Builder
	b.WriteString(stylePanelTitle.Render("Confirm " + c.action))
	if len(c.ids) > 0 {
		b.WriteString(styleMuted.Render(fmt.Sprintf("  ·  %d server(s)", len(c.ids))))
	}
	b.WriteString("\n\n")
	for i, name := range c.names {
		if i >= 8 {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  … and %d more\n", len(c.names)-8)))
			break
		}
		b.WriteString("  · " + name + "\n")
	}
	if c.extra != "" {
		b.WriteString("\n  " + styleMuted.Render(c.extra) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styleOk.Render("[y] yes") + "   " + styleMuted.Render("[n] no"))
	box := stylePanel.Padding(1, 2).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) editView() string {
	e := m.edit
	body := stylePanelTitle.Render("Set "+e.attr) + "\n\n" + e.input.View() + "\n\n" +
		styleMuted.Render("enter save · esc cancel")
	box := stylePanel.Padding(1, 2).Width(min(64, m.width-4)).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		c := m.confirm
		m.confirm = nil
		return m, m.dispatchConfirm(c)
	case "n", "N", "esc", "q":
		m.confirm = nil
		m.status = "cancelled"
		return m, nil
	}
	return m, nil
}

func (m model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.edit = nil
		m.status = "edit cancelled"
		return m, nil
	case "enter":
		e := m.edit
		m.edit = nil
		if e.attr == "image-setup" {
			if m.server.ID == 0 {
				m.status = "select a server first"
				return m, nil
			}
			m.confirm = &confirmState{
				action: "image-setup",
				ids:    []int32{m.server.ID},
				names:  []string{m.server.Name},
				extra:  e.input.Value(),
			}
			return m, nil
		}
		m.loading = true
		m.status = "setting " + e.attr
		return m, m.dispatchEdit(e)
	}
	var cmd tea.Cmd
	m.edit.input, cmd = m.edit.input.Update(msg)
	return m, cmd
}

func (m model) dispatchEdit(e *editMode) tea.Cmd {
	switch e.attr {
	case "snapshot-create":
		return snapshotActionCmd(m.ctx, m.deps.Client, m.server.ID, "snapshot-create", e.input.Value())
	case "disk-driver":
		return diskActionCmd(m.ctx, m.deps.Client, m.server.ID, "disk-driver", "", e.input.Value())
	case "firewall-set", "policy-create", "policy-update", "rdns-set", "rdns-get", "rdns-delete", "ssh-add", "iso-upload", "image-upload", "failover-route", "nic-create", "vlan-rename":
		return resourceEditCmd(m.ctx, m.deps.Client, m.deps.UserID, m.server.ID, e.attr, e.input.Value(), e.context)
	default:
		if m.server.ID == 0 {
			return nil
		}
		return setAttrCmd(m.ctx, m.deps.Client, m.server.ID, e.attr, e.input.Value())
	}
}

func (m model) dispatchConfirm(c *confirmState) tea.Cmd {
	switch c.action {
	case "start", "stop", "reboot", "poweroff", "reset", "suspend":
		return m.dispatchPower(c.action, c.ids, c.names)
	case "rescue-enable":
		id := c.ids[0]
		return rescueCmd(m.ctx, m.deps.Client, id, "enable")
	case "rescue-disable":
		id := c.ids[0]
		return rescueCmd(m.ctx, m.deps.Client, id, "disable")
	case "iso-attach":
		return m.dispatchISOAttach(c.ids[0], c)
	case "iso-detach":
		return isoDetachCmd(m.ctx, m.deps.Client, c.ids[0])
	case "task-cancel":
		return cancelTaskCmd(m.ctx, m.deps.Client, c.extra)
	case "snapshot-delete", "snapshot-revert":
		return snapshotActionCmd(m.ctx, m.deps.Client, c.ids[0], c.action, c.extra)
	case "disk-format":
		return diskActionCmd(m.ctx, m.deps.Client, c.ids[0], c.action, c.extra, "")
	case "firewall-reapply", "firewall-restore", "policy-delete", "iso-delete", "image-delete", "ssh-delete", "nic-delete", "image-setup-user", "policy-preset-ssh", "policy-preset-http":
		return resourceConfirmCmd(m.ctx, m.deps.Client, m.deps.UserID, m.server.ID, c.action, c.extra)
	case "storage-optimize":
		return storageOptimizeCmd(m.ctx, m.deps.Client, c.ids[0])
	case "image-setup":
		return imageSetupCmd(m.ctx, m.deps.Client, m.server.ID, c.extra)
	default:
		return nil
	}
}

func (m model) dispatchISOAttach(id int32, c *confirmState) tea.Cmd {
	// extra format: "user:NAME" | "catalog:ID" | "user:NAME|boot" handled via names
	boot := strings.Contains(c.extra, "|boot")
	extra := strings.TrimSuffix(c.extra, "|boot")
	if strings.HasPrefix(extra, "user:") {
		return isoAttachCmd(m.ctx, m.deps.Client, id, 0, strings.TrimPrefix(extra, "user:"), boot)
	}
	if strings.HasPrefix(extra, "catalog:") {
		var isoID int32
		_, _ = fmt.Sscanf(strings.TrimPrefix(extra, "catalog:"), "%d", &isoID)
		return isoAttachCmd(m.ctx, m.deps.Client, id, isoID, "", boot)
	}
	return nil
}

func (m model) pollJobsCmd() tea.Cmd {
	active := make([]job, 0)
	for _, j := range m.jobs {
		if !j.Done && j.UUID != "" {
			active = append(active, j)
		}
	}
	if len(active) == 0 {
		return nil
	}
	interval := m.deps.pollInterval()
	client := m.deps.Client
	ctx := m.ctx
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return pollJobs(ctx, client, active)
	})
}

func (m *model) trackJob(j job) tea.Cmd {
	m.jobs = append(m.jobs, j)
	if m.deps.OnServersMutated != nil {
		m.deps.OnServersMutated()
	}
	if !j.Done {
		m.deadline = time.Now().Add(m.deps.waitTimeout())
		return m.pollJobsCmd()
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
