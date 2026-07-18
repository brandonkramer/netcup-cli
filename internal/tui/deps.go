package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

// Deps are the runtime dependencies for the interactive TUI.
type Deps struct {
	Client           *scpclient.ClientWithResponses
	UserID           int32
	WaitTimeout      time.Duration
	PollInterval     time.Duration
	OnServersMutated func()
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

// Run starts the Bubble Tea program. Caller must supply an authenticated client.
func Run(ctx context.Context, deps Deps) error {
	if deps.Client == nil {
		return fmt.Errorf("tui: missing API client")
	}
	m := newModel(ctx, deps)
	p := teaProgram(m)
	_, err := p.Run()
	return err
}
