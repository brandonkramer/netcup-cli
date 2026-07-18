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
	tabMetrics
	tabMedia
)

var tabNames = []string{"Servers", "Tasks", "Metrics", "Media"}

type confirmState struct {
	action string
	ids    []int32
	names  []string
	extra  string // optional payload (e.g. user iso name)
}

type editMode struct {
	attr  string // hostname | nickname
	input textinput.Model
}

type model struct {
	ctx  context.Context
	deps Deps

	tab tabKind

	serversList list.Model
	tasksList   list.Model
	mediaList   list.Model
	viewport    viewport.Model
	spinner     spinner.Model

	loading bool
	detail  *scpclient.Server
	guest   any
	jobs    []job
	confirm *confirmState
	edit    *editMode

	metricsKind  string
	metricsHours int32
	metricsData  any
	mediaAttached any

	status   string
	errMsg   string
	width    int
	height   int
	bodyH    int // allocated body rows (panels sized to this)
	ready    bool
	deadline time.Time
}

func newModel(ctx context.Context, deps Deps) model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(colAccent).BorderForeground(colAccent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(colMuted).BorderForeground(colAccent)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(colValue)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(colMuted)

	servers := list.New([]list.Item{}, delegate, 0, 0)
	servers.Title = "Servers"
	servers.SetShowStatusBar(false)
	servers.SetShowPagination(false)
	servers.SetFilteringEnabled(true)
	servers.SetShowHelp(false)
	servers.DisableQuitKeybindings()
	servers.Styles.Title = styleAppTitle
	servers.Styles.FilterPrompt = styleFocus
	servers.Styles.FilterCursor = styleFocus

	tasks := list.New([]list.Item{}, delegate, 0, 0)
	tasks.Title = "Tasks"
	tasks.SetShowStatusBar(false)
	tasks.SetShowPagination(false)
	tasks.SetFilteringEnabled(true)
	tasks.SetShowHelp(false)
	tasks.DisableQuitKeybindings()
	tasks.Styles.Title = styleAppTitle

	media := list.New([]list.Item{}, delegate, 0, 0)
	media.Title = "Media"
	media.SetShowStatusBar(false)
	media.SetShowPagination(false)
	media.SetFilteringEnabled(true)
	media.SetShowHelp(false)
	media.DisableQuitKeybindings()
	media.Styles.Title = styleAppTitle

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styleFocus

	vp := viewport.New(0, 0)

	return model{
		ctx:          ctx,
		deps:         deps,
		tab:          tabServers,
		serversList:  servers,
		tasksList:    tasks,
		mediaList:    media,
		viewport:     vp,
		spinner:      sp,
		loading:      true,
		metricsKind:  "cpu",
		metricsHours: 24,
		status:       "loading servers…",
	}
}

func teaProgram(m model) *tea.Program {
	return tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(m.ctx))
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadServersCmd(m.ctx, m.deps.Client))
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
		return m.onServersLoaded(msg)
	case detailLoadedMsg:
		return m.onDetailLoaded(msg)
	case guestAgentMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.guest = msg.status
		}
		return m, nil
	case powerStartedMsg:
		return m.onPowerStarted(msg)
	case jobsPolledMsg:
		return m.onJobsPolled(msg)
	case tasksLoadedMsg:
		return m.onTasksLoaded(msg)
	case taskCancelMsg:
		return m.onTaskCancel(msg)
	case metricsLoadedMsg:
		return m.onMetricsLoaded(msg)
	case mediaLoadedMsg:
		return m.onMediaLoaded(msg)
	case isoActionMsg:
		return m.onISOAction(msg)
	case attrSetMsg:
		return m.onAttrSet(msg)
	case rescueMsg:
		return m.onRescue(msg)

	case tea.KeyMsg:
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
			return m.switchTab(tabMetrics)
		case "4":
			return m.switchTab(tabMedia)
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
		case tabMetrics:
			return m.updateMetricsKeys(msg)
		case tabMedia:
			return m.updateMediaKeys(msg)
		}
	}

	return m.updateActiveList(msg)
}

func (m model) View() string {
	if !m.ready {
		return styleMuted.Render(" starting… ")
	}
	if m.confirm != nil {
		return m.confirmView()
	}
	if m.edit != nil {
		return m.editView()
	}

	m.layout()

	// Build bottom first (clamped to 1 line each) so we never overflow the
	// terminal — overflow scrolls away the TOP (logo/tabs).
	var bottomParts []string
	if m.errMsg != "" {
		bottomParts = append(bottomParts, clampLines(styleErr.Render("✗ "+m.errMsg), 1, m.width))
	}
	bottomParts = append(bottomParts,
		clampLines(m.jobsView(), 1, m.width),
		clampLines(m.helpView(), 1, m.width),
		clampLines(m.statusLine(), 1, m.width),
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
	case tabMetrics:
		body = m.metricsView()
	case tabMedia:
		body = m.mediaView()
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

func (m model) statusLine() string {
	status := m.status
	if m.loading {
		status = m.spinner.View() + " " + status
	}
	return styleStatusBar.Width(m.width).Render(status)
}

func (m *model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// Estimate body height; View() recomputes exactly after measuring chrome.
	const tabsH, bottomLines = 2, 4 // jobs+help+status + optional err ≈ 3–4
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
	m.mediaList.SetSize(listInnerW, innerH)
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
	case tabMetrics:
		id, _ := m.focusedServer()
		if id == 0 {
			m.status = "select a server on Servers tab first"
			return m, nil
		}
		m.loading = true
		m.status = fmt.Sprintf("loading %s metrics…", m.metricsKind)
		return m, loadMetricsCmd(m.ctx, m.deps.Client, id, m.metricsKind, m.metricsHours)
	case tabMedia:
		id, _ := m.focusedServer()
		if id == 0 {
			m.status = "select a server on Servers tab first"
			return m, nil
		}
		m.loading = true
		m.status = "loading media…"
		return m, loadMediaCmd(m.ctx, m.deps.Client, id, m.deps.UserID)
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
	case tabMedia:
		m.mediaList, cmd = m.mediaList.Update(msg)
	case tabMetrics:
		m.viewport, cmd = m.viewport.Update(msg)
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
	logo := renderLogo()
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if tabKind(i) == m.tab {
			tabs = append(tabs, styleTabOn.Render(label))
		} else {
			tabs = append(tabs, styleTabOff.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	row := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", tabBar)
	row = clampLines(row, 1, m.width)
	line := clampLines(styleMuted.Render(strings.Repeat("─", max(0, m.width))), 1, m.width)
	return row + "\n" + line
}

func (m model) helpView() string {
	var help string
	switch m.tab {
	case tabServers:
		help = "↑↓ move · space select · / filter · s/t/r power · P/x/u hard · h/n edit · e/d rescue · R refresh · q quit"
	case tabTasks:
		help = "↑↓ move · / filter · c cancel · R refresh · q quit"
	case tabMetrics:
		help = "c/d/n/p series · h hours · R refresh · q quit"
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
		id, _ := m.focusedServer()
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.loading = true
		m.status = "setting " + e.attr
		return m, setAttrCmd(m.ctx, m.deps.Client, id, e.attr, e.input.Value())
	}
	var cmd tea.Cmd
	m.edit.input, cmd = m.edit.input.Update(msg)
	return m, cmd
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
