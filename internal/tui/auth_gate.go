package tui

import (
	"context"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type authGateMsg struct {
	reason string
	ok     bool
}

type sessionReadyMsg struct {
	client   *scpclient.ClientWithResponses
	userID   int32
	username string
	err      error
}

type deviceLoginPromptMsg struct {
	verifyURL string
	userCode  string
	wait      func(context.Context) tea.Msg
}

// SessionReady builds a session-ready message for wiring from cmd.
func SessionReady(client *scpclient.ClientWithResponses, userID int32, username string, err error) tea.Msg {
	return sessionReadyMsg{client: client, userID: userID, username: username, err: err}
}

// DeviceLoginPrompt shows the device-code screen, then runs wait in a follow-up cmd.
func DeviceLoginPrompt(verifyURL, userCode string, wait func(context.Context) tea.Msg) tea.Msg {
	return deviceLoginPromptMsg{verifyURL: verifyURL, userCode: userCode, wait: wait}
}

func (m model) authGateView() string {
	reason := m.authReason
	if reason == "" {
		reason = "not logged in or session expired"
	}
	body := stylePanelTitle.Render("Authentication required") + "\n\n" +
		styleErr.Render(reason) + "\n\n"
	if m.deviceURL != "" {
		body += styleMuted.Render("Open this URL to authorize:\n") +
			styleFocus.Render(m.deviceURL) + "\n\n" +
			styleMuted.Render("User code: ") + styleFocus.Render(m.deviceCode) + "\n\n" +
			styleMuted.Render("Waiting for authorization…") + "\n\n" +
			styleMuted.Render("Press ") + styleFocus.Render("q") + styleMuted.Render(" to quit")
	} else {
		body += styleMuted.Render("Press ") + styleFocus.Render("l") + styleMuted.Render(" to login (device code)\n") +
			styleMuted.Render("Press ") + styleFocus.Render("r") + styleMuted.Render(" to retry session\n") +
			styleMuted.Render("Press ") + styleFocus.Render("q") + styleMuted.Render(" to quit\n\n") +
			styleMuted.Render("Or run: netcup auth login")
	}
	box := stylePanel.Padding(1, 2).Width(min(72, max(40, m.width-4))).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) handleAuthGateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "l":
		if m.deviceURL != "" {
			return m, nil // already waiting
		}
		if m.deps.Login == nil {
			m.authReason = "login not available — run: netcup auth login"
			return m, nil
		}
		m.loading = true
		m.status = "starting device login…"
		return m, m.deps.Login(m.ctx)
	case "r":
		if m.deviceURL != "" {
			return m, nil
		}
		if m.deps.Reconnect == nil {
			m.authReason = "reconnect not available — run: netcup auth login"
			return m, nil
		}
		m.loading = true
		m.status = "retrying session…"
		return m, m.deps.Reconnect(m.ctx)
	}
	return m, nil
}

func (m model) onDeviceLoginPrompt(msg deviceLoginPromptMsg) (tea.Model, tea.Cmd) {
	m.authGate = true
	m.loading = false
	m.deviceURL = msg.verifyURL
	m.deviceCode = msg.userCode
	m.authReason = "authorize in browser, then return here"
	m.status = "waiting for device login…"
	wait := msg.wait
	ctx := m.ctx
	return m, func() tea.Msg {
		return wait(ctx)
	}
}

func (m model) onSessionReady(msg sessionReadyMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.deviceURL = ""
	m.deviceCode = ""
	if msg.err != nil {
		m.authGate = true
		m.authReason = msg.err.Error()
		m.status = "auth failed"
		return m, nil
	}
	if msg.client != nil {
		m.deps.Client = msg.client
	}
	if msg.userID != 0 {
		m.deps.UserID = msg.userID
	}
	if msg.username != "" {
		m.deps.Username = msg.username
	}
	m.authGate = false
	m.authReason = ""
	m.errMsg = ""
	m.status = "signed in"
	return m, tea.Batch(
		loadServersCmd(m.ctx, m.deps, true),
		loadHealthCmd(m.ctx, m.deps),
	)
}

func checkSessionCmd(ctx context.Context, deps Deps) tea.Cmd {
	return func() tea.Msg {
		if deps.Client == nil {
			return authGateMsg{ok: false, reason: "not logged in"}
		}
		if deps.ProbeAuth != nil {
			if err := deps.ProbeAuth(ctx); err != nil {
				return authGateMsg{ok: false, reason: err.Error()}
			}
		}
		return authGateMsg{ok: true}
	}
}
