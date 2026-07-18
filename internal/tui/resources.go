package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func apiStatusErr(action string, status int, body []byte) error {
	msg := truncate(string(body), 160)
	if status == 401 {
		return newAuthError(fmt.Sprintf("%s: HTTP 401 — run netcup auth login", action))
	}
	if status == 403 {
		if detail := apiErrorMessage(body); detail != "" {
			return fmt.Errorf("%s: HTTP 403 — %s (token ok; API denied this resource)", action, detail)
		}
		return fmt.Errorf("%s: HTTP 403 — access not allowed (token ok; API denied this resource)", action)
	}
	return fmt.Errorf("%s: HTTP %d: %s", action, status, msg)
}

func apiErrorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if payload.Message != "" {
		return payload.Message
	}
	return payload.Code
}

func prettyJSON(value any) string {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(b)
}

func (m model) loadTabCmd(t tabKind) tea.Cmd {
	switch t {
	case tabSnapshots:
		m.status = "loading snapshots…"
		return loadSnapshotsCmd(m.ctx, m.deps.Client, m.server.ID)
	case tabDisks:
		m.status = "loading disks…"
		return loadDisksCmd(m.ctx, m.deps.Client, m.server.ID)
	case tabFirewall:
		m.status = "loading firewall…"
		return loadFirewallCmd(m.ctx, m.deps.Client, m.server.ID, m.deps.UserID)
	case tabMedia:
		m.status = "loading media…"
		return loadMediaCmd(m.ctx, m.deps.Client, m.server.ID, m.deps.UserID)
	case tabNetwork:
		m.status = "loading network…"
		return loadNetworkCmd(m.ctx, m.deps.Client, m.server.ID, m.deps.UserID)
	case tabAccount:
		m.status = "loading account…"
		return loadAccountCmd(m.ctx, m.deps.Client, m.deps.UserID, m.deps.Username)
	}
	return nil
}

func (m model) onResourceLoaded(msg resourceLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = "request failed"
		if isAuthFailure(msg.err) {
			m.status = "run netcup auth login"
		}
		return m.withAuthGate(msg.err), nil
	}
	m.errMsg = ""
	if msg.warning != "" {
		m.status = msg.warning
	}
	if msg.detail != "" {
		m.resourceDetail = msg.detail
		m.resourceMode = msg.mode
		w := m.viewport.Width
		if w < 1 {
			_, detailOuter := m.panelSplit()
			w = detailOuter - 4
		}
		m.viewport.SetContent(wrapText(msg.detail, w))
		m.viewport.GotoTop()
	}
	switch msg.tab {
	case "snapshots":
		m.snapshotsList.SetItems(itemsAsList(msg.items))
	case "disks":
		m.disksList.SetItems(itemsAsList(msg.items))
	case "firewall":
		m.firewallList.SetItems(itemsAsList(msg.items))
	case "network":
		m.networkList.SetItems(itemsAsList(msg.items))
	case "account":
		m.accountList.SetItems(itemsAsList(msg.items))
	}
	if len(msg.items) > 0 {
		m.status = fmt.Sprintf("%d %s items", len(msg.items), msg.tab)
	} else if msg.detail != "" {
		m.status = msg.tab + " detail"
	}
	if msg.tab == "account" && msg.detail != "" && strings.Contains(msg.detail, "403") {
		m.status = "account · local session (profile API forbidden)"
	}
	return m, nil
}

func itemsAsList(items []resourceItem) []list.Item {
	out := make([]list.Item, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func (m model) onResourceAction(msg resourceActionMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.status = msg.action + " failed"
		return m.withAuthGate(msg.err), nil
	}
	name := m.server.Name
	j := jobFromTaskStatus(m.server.ID, name, msg.action, msg.status, msg.task, nil)
	switch {
	case j.UUID != "":
		m.status = msg.action
	case j.State == "UNTRACKED":
		m.status = msg.action + " accepted (untracked — refresh to verify)"
	default:
		m.status = msg.action + " ok"
	}
	cmd := m.trackJob(j)
	return m, tea.Batch(cmd, m.loadTabCmd(m.tab))
}

func (m model) beginResourceEdit(attr, placeholder string) (tea.Model, tea.Cmd) {
	return m.beginResourceEditCtx(attr, "", placeholder)
}

func (m model) beginResourceEditCtx(attr, context, placeholder string) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = 64
	ti.CharLimit = 4096
	ti.Focus()
	m.edit = &editMode{attr: attr, context: context, input: ti}
	return m, textinput.Blink
}
