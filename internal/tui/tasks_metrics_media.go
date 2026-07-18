package tui

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateTasksKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.tasksList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	switch msg.String() {
	case "R":
		m.loading = true
		m.status = "refreshing tasks…"
		return m, loadTasksCmd(m.ctx, m.deps.Client)
	case "c":
		it, ok := m.tasksList.SelectedItem().(taskItem)
		if !ok || it.task.Uuid == nil {
			m.status = "no task selected"
			return m, nil
		}
		m.confirm = &confirmState{
			action: "task-cancel",
			names:  []string{derefStr(it.task.Name)},
			extra:  *it.task.Uuid,
		}
		return m, nil
	}
	return m.updateActiveList(msg)
}

func (m model) onTasksLoaded(msg tasksLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "failed to load tasks"
		return m, nil
	}
	items := make([]list.Item, 0, len(msg.tasks))
	for _, t := range msg.tasks {
		items = append(items, taskItem{task: t})
	}
	m.tasksList.SetItems(items)
	m.status = fmt.Sprintf("%d tasks", len(items))
	m.errMsg = ""
	return m, nil
}

func (m model) onTaskCancel(msg taskCancelMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "cancel failed"
		return m, nil
	}
	j := jobFromTask(0, shortUUID(msg.uuid), "cancel", msg.task, nil)
	m.status = "cancel requested"
	cmd := m.trackJob(j)
	return m, tea.Batch(cmd, loadTasksCmd(m.ctx, m.deps.Client))
}

func (m model) updateMetricsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	id, _ := m.focusedServer()
	switch msg.String() {
	case "c":
		m.metricsKind = "cpu"
	case "d":
		m.metricsKind = "disk"
	case "n":
		m.metricsKind = "network"
	case "p":
		m.metricsKind = "packets"
	case "h":
		switch m.metricsHours {
		case 1:
			m.metricsHours = 6
		case 6:
			m.metricsHours = 24
		case 24:
			m.metricsHours = 168
		default:
			m.metricsHours = 1
		}
	case "R":
		// refresh below
	default:
		return m.updateActiveList(msg)
	}
	if id == 0 {
		m.status = "select a server on Servers tab first"
		return m, nil
	}
	m.loading = true
	m.status = fmt.Sprintf("loading %s (%dh)…", m.metricsKind, m.metricsHours)
	return m, loadMetricsCmd(m.ctx, m.deps.Client, id, m.metricsKind, m.metricsHours)
}

func (m model) onMetricsLoaded(msg metricsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "metrics failed"
		return m, nil
	}
	m.metricsKind = msg.kind
	m.metricsData = msg.data
	m.errMsg = ""
	m.status = fmt.Sprintf("%s metrics (%dh)", msg.kind, m.metricsHours)
	m.viewport.SetContent(formatMetrics(msg.kind, msg.data))
	m.viewport.GotoTop()
	return m, nil
}

func (m model) metricsView() string {
	id, name := m.focusedServer()
	header := stylePanelTitle.Render("Metrics") + "  " +
		styleMuted.Render(fmt.Sprintf("%s · %s · %dh", name, m.metricsKind, m.metricsHours))
	if id == 0 {
		return m.onePanel(header + "\n\n" + styleMuted.Render("Select a server on Servers (1), then open Metrics (3)."))
	}
	body := m.viewport.View()
	if m.metricsData == nil && !m.loading {
		body = styleMuted.Render("No data yet. Press R, or c/d/n/p for a series.")
	}
	return m.onePanel(header + "\n\n" + body)
}

func formatMetrics(kind string, data any) string {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	s := string(raw)
	if len(s) > 12000 {
		s = s[:12000] + "\n…truncated"
	}
	return fmt.Sprintf("%s\n\n%s", kind, s)
}

func (m model) updateMediaKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mediaList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	id, name := m.focusedServer()
	switch msg.String() {
	case "R":
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.loading = true
		m.status = "refreshing media…"
		return m, loadMediaCmd(m.ctx, m.deps.Client, id, m.deps.UserID)
	case "D":
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		m.confirm = &confirmState{action: "iso-detach", ids: []int32{id}, names: []string{name}}
		return m, nil
	case "enter":
		if id == 0 {
			m.status = "no server"
			return m, nil
		}
		it, ok := m.mediaList.SelectedItem().(mediaItem)
		if !ok || it.kind == "attached" {
			m.status = "select a catalog or user ISO"
			return m, nil
		}
		extra := ""
		label := it.title
		if it.kind == "user" {
			extra = "user:" + it.userName + "|boot"
		} else if it.kind == "catalog" {
			extra = fmt.Sprintf("catalog:%d|boot", it.isoID)
		}
		m.confirm = &confirmState{
			action: "iso-attach",
			ids:    []int32{id},
			names:  []string{name + " ← " + label},
			extra:  extra,
		}
		return m, nil
	}
	return m.updateActiveList(msg)
}

func (m model) onMediaLoaded(msg mediaLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "media failed"
		return m, nil
	}
	m.mediaAttached = msg.attached
	items := make([]list.Item, 0)
	if msg.attached != nil {
		items = append(items, mediaItem{
			kind:        "attached",
			title:       "● Currently attached",
			description: fmt.Sprintf("%v", msg.attached),
			filter:      "attached",
		})
	} else {
		items = append(items, mediaItem{
			kind:        "attached",
			title:       "○ No ISO attached",
			description: "Select a catalog/user ISO and press Enter to attach",
			filter:      "none",
		})
	}
	for _, img := range msg.catalog {
		id := derefInt(img.Id)
		items = append(items, mediaItem{
			kind:        "catalog",
			title:       "catalog · " + img.Name,
			description: fmt.Sprintf("id %d · %s", id, derefStr(img.Description)),
			isoID:       id,
			filter:      img.Name + " " + derefStr(img.Description),
		})
	}
	switch objs := msg.userISOs.(type) {
	case []scpclient.S3Object:
		for _, o := range objs {
			key := derefStr(o.Key)
			sz := ""
			if o.SizeInB != nil {
				sz = fmt.Sprintf("%d B", *o.SizeInB)
			}
			items = append(items, mediaItem{
				kind: "user", title: "user · " + key, description: sz, userName: key, filter: key,
			})
		}
	case *[]scpclient.S3Object:
		if objs != nil {
			for _, o := range *objs {
				key := derefStr(o.Key)
				items = append(items, mediaItem{
					kind: "user", title: "user · " + key, description: "", userName: key, filter: key,
				})
			}
		}
	}
	m.mediaList.SetItems(items)
	m.status = fmt.Sprintf("media: %d items", len(items))
	m.errMsg = ""
	return m, nil
}

func (m model) mediaView() string {
	_, name := m.focusedServer()
	attached := "none"
	if m.mediaAttached != nil {
		attached = fmt.Sprintf("%v", m.mediaAttached)
	}
	right := stylePanelTitle.Render("ISO") + "\n" +
		styleMuted.Render(name) + "\n\n" +
		kv("Attached", attached) + "\n\n" +
		styleMuted.Render("Enter attach (boot→CDROM)\nD detach")
	return m.twoPanels(m.mediaList.View(), right)
}

func (m model) onISOAction(msg isoActionMsg) (tea.Model, tea.Cmd) {
	id, name := m.focusedServer()
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = msg.action + " failed"
		return m, nil
	}
	j := jobFromTask(id, name, "iso."+msg.action, msg.task, nil)
	m.status = "iso " + msg.action
	cmd := m.trackJob(j)
	reload := loadMediaCmd(m.ctx, m.deps.Client, id, m.deps.UserID)
	return m, tea.Batch(cmd, reload)
}

func (m model) onAttrSet(msg attrSetMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	id, name := m.focusedServer()
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "set " + msg.attr + " failed"
		return m, nil
	}
	j := jobFromTask(id, name, "set."+msg.attr, msg.task, nil)
	m.status = "set " + msg.attr
	cmd := m.trackJob(j)
	return m, tea.Batch(cmd, m.loadSelectedDetail())
}

func (m model) onRescue(msg rescueMsg) (tea.Model, tea.Cmd) {
	id, name := m.focusedServer()
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "rescue " + msg.action + " failed"
		return m, nil
	}
	if msg.action == "status" {
		m.status = fmt.Sprintf("rescue: %v", msg.data)
		return m, nil
	}
	j := jobFromTask(id, name, "rescue."+msg.action, msg.task, nil)
	m.status = "rescue " + msg.action
	cmd := m.trackJob(j)
	return m, tea.Batch(cmd, m.loadSelectedDetail())
}

func nowTime() time.Time { return time.Now() }
