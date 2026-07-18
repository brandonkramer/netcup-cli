package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/cache"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	tea "github.com/charmbracelet/bubbletea"
)

// Deps are the runtime dependencies for the interactive TUI.
type Deps struct {
	Client           *scpclient.ClientWithResponses
	Cache            *cache.Store
	UserID           int32
	Username         string
	APIRoot          string
	WaitTimeout      time.Duration
	PollInterval     time.Duration
	OnServersMutated func()

	// Auth helpers (optional). Login/Reconnect return sessionReadyMsg via tea.Cmd.
	ProbeAuth         func(ctx context.Context) error
	Login             func(ctx context.Context) tea.Cmd
	Reconnect         func(ctx context.Context) tea.Cmd
	InitialAuthReason string
}

func (d Deps) pollInterval() time.Duration {
	if d.PollInterval > 0 {
		return d.PollInterval
	}
	return 2 * time.Second
}

func (d Deps) waitTimeout() time.Duration {
	if d.WaitTimeout > 0 {
		return d.WaitTimeout
	}
	return 30 * time.Minute
}

// Run starts the Bubble Tea program.
func Run(ctx context.Context, deps Deps) error {
	m := newModel(ctx, deps)
	if deps.Client == nil {
		m.authGate = true
		m.authReason = deps.InitialAuthReason
		if m.authReason == "" {
			m.authReason = "not logged in"
		}
	}
	p := teaProgram(m)
	_, err := p.Run()
	return err
}

func (d Deps) requireClient() error {
	if d.Client == nil {
		return fmt.Errorf("not logged in")
	}
	return nil
}
