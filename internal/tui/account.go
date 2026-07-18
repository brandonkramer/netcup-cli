package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func loadAccountCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, username string) tea.Cmd {
	return func() tea.Msg {
		items := []resourceItem{{
			kind:        "session",
			id:          fmt.Sprint(uid),
			title:       fmt.Sprintf("session · user %d", uid),
			description: usernameOrID(username, uid),
			raw:         fmt.Sprintf("user_id=%d\nusername=%s\nsource=local credentials", uid, usernameOrID(username, uid)),
		}}

		notes := make([]string, 0, 3)
		u, err := c.GetApiV1UsersUserIdWithResponse(ctx, uid)
		switch {
		case err != nil:
			notes = append(notes, "profile unreachable")
		case u == nil:
			notes = append(notes, "profile empty response")
		case u.StatusCode() == 200:
			items = append(items, resourceItems("user", firstJSON(u.JSON200, u.HALJSON200, u.Body))...)
		case u.StatusCode() == 401:
			return resourceLoadedMsg{tab: "account", err: apiStatusErr("profile", u.StatusCode(), u.Body)}
		case u.StatusCode() == 403:
			// SCP often forbids GET /users/{id} even when other APIs work.
			notes = append(notes, "profile forbidden (403)")
			notes = append(notes, "showing local session only")
		default:
			notes = append(notes, fmt.Sprintf("profile HTTP %d", u.StatusCode()))
		}

		r, e := c.GetApiV1UsersUserIdSshKeysWithResponse(ctx, uid)
		switch {
		case e != nil:
			notes = append(notes, "ssh-keys unreachable")
		case r == nil:
			notes = append(notes, "ssh-keys empty response")
		case r.StatusCode() == 200:
			items = append(items, resourceItems("ssh-key", firstJSON(r.JSON200, r.HALJSON200, r.Body))...)
		case r.StatusCode() == 401:
			return resourceLoadedMsg{tab: "account", err: apiStatusErr("ssh-keys", r.StatusCode(), r.Body)}
		case r.StatusCode() == 404:
			notes = append(notes, "ssh-keys not found (404)")
		default:
			notes = append(notes, fmt.Sprintf("ssh-keys HTTP %d", r.StatusCode()))
		}

		return resourceLoadedMsg{tab: "account", items: items, detail: joinNotes(notes), mode: "account-notes"}
	}
}

func usernameOrID(username string, uid int32) string {
	if username != "" {
		return username
	}
	return fmt.Sprint(uid)
}

func accountLogsCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32) tea.Cmd {
	return func() tea.Msg {
		r, e := c.GetApiV1UsersUserIdLogsWithResponse(ctx, uid, nil)
		if e != nil {
			return resourceLoadedMsg{tab: "account", err: e}
		}
		if r == nil {
			return resourceLoadedMsg{tab: "account", err: tuiError("empty response")}
		}
		if r.StatusCode() != 200 {
			return resourceLoadedMsg{tab: "account", err: apiStatusErr("user logs", r.StatusCode(), r.Body)}
		}
		return resourceLoadedMsg{tab: "account", detail: prettyJSON(firstJSON(r.JSON200, r.HALJSON200, r.Body))}
	}
}

func (m model) updateAccountKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.accountList.FilterState() == list.Filtering {
		return m.updateActiveList(msg)
	}
	it, _ := m.accountList.SelectedItem().(resourceItem)
	switch msg.String() {
	case "R":
		m.loading = true
		return m, loadAccountCmd(m.ctx, m.deps.Client, m.deps.UserID, m.deps.Username)
	case "L":
		return m, accountLogsCmd(m.ctx, m.deps.Client, m.deps.UserID)
	case "enter":
		if it.raw != "" {
			m.resourceDetail = it.raw
			m.resourceMode = "account-detail"
			m.viewport.SetContent(it.raw)
			m.viewport.GotoTop()
			m.status = it.title
			return m, nil
		}
	case "k":
		m.status = "SSH keys in list (when API allows)"
		return m, nil
	case "a":
		return m.beginResourceEdit("ssh-add", "name|public key")
	case "d":
		if it.kind == "ssh-key" && it.id != "" {
			m.confirm = &confirmState{action: "ssh-delete", names: []string{it.title}, extra: it.id}
			return m, nil
		}
	}
	return m.updateActiveList(msg)
}

func (m model) accountView() string {
	right := stylePanelTitle.Render("Account") + "\n\n" +
		styleMuted.Render("Local session always shown.\n") +
		styleMuted.Render("Profile/SSH may be 403/404 on SCP.\n") +
		styleMuted.Render("enter detail · a add · d del · L logs")
	if m.resourceDetail != "" {
		right = m.viewport.View()
	}
	return m.twoPanels(m.accountList.View(), right)
}

func addSSHKeyCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, value string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.SplitN(value, "|", 2)
		if len(parts) != 2 {
			return resourceActionMsg{action: "ssh-add", err: strconvErr("use name|public key")}
		}
		body := scpclient.SSHKey{Name: parts[0], Key: parts[1]}
		r, err := c.PostApiV1UsersUserIdSshKeysWithResponse(ctx, uid, body)
		if err != nil {
			return resourceActionMsg{action: "ssh-add", err: err}
		}
		if r == nil {
			return resourceActionMsg{action: "ssh-add", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 201 && status != 202 {
			return resourceActionMsg{action: "ssh-add", status: status, err: apiStatusErr("add SSH key", status, r.Body)}
		}
		return resourceActionMsg{action: "ssh-add", status: status, task: extractTask(status, r.Body)}
	}
}

type tuiError string

func (e tuiError) Error() string { return string(e) }
func strconvErr(s string) error  { return tuiError(s) }

func deleteSSHKeyCmd(ctx context.Context, c *scpclient.ClientWithResponses, uid int32, id string) tea.Cmd {
	return func() tea.Msg {
		r, e := c.DeleteApiV1UsersUserIdSshKeysIdWithResponse(ctx, uid, parseID(id))
		if e != nil {
			return resourceActionMsg{action: "ssh-delete", err: e}
		}
		if r == nil {
			return resourceActionMsg{action: "ssh-delete", err: tuiError("empty response")}
		}
		status := r.StatusCode()
		if status != 200 && status != 204 {
			return resourceActionMsg{action: "ssh-delete", status: status, err: apiStatusErr("delete SSH key", status, r.Body)}
		}
		return resourceActionMsg{action: "ssh-delete", status: status}
	}
}
