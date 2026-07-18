package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// AuthError is returned for missing/expired credentials or HTTP 401 responses.
type AuthError struct {
	Msg string
}

func (e *AuthError) Error() string {
	if e == nil || e.Msg == "" {
		return "not logged in — run: netcup auth login"
	}
	return e.Msg
}

func newAuthError(msg string) error {
	return &AuthError{Msg: msg}
}

func isAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	var ae *AuthError
	if errors.As(err, &ae) {
		return true
	}
	s := err.Error()
	lower := strings.ToLower(s)
	return strings.Contains(s, "HTTP 401") ||
		strings.Contains(lower, "not logged in") ||
		(strings.Contains(lower, "token") && (strings.Contains(lower, "refresh") || strings.Contains(lower, "expired"))) ||
		(strings.Contains(lower, "oauth2:") && strings.Contains(lower, "cannot fetch token"))
}

func (m model) withAuthGate(err error) model {
	if !isAuthFailure(err) {
		return m
	}
	m.authGate = true
	m.authReason = err.Error()
	m.loading = false
	m.errMsg = ""
	return m
}

func applyAuthGate(mod tea.Model, cmd tea.Cmd, err error) (tea.Model, tea.Cmd) {
	mm, ok := mod.(model)
	if !ok {
		return mod, cmd
	}
	return mm.withAuthGate(err), cmd
}
