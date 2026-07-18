package tui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type healthMsg struct {
	pingOK      bool
	pingMS      int64
	maintenance string // empty = none/unknown
	err         string
}

func loadHealthCmd(ctx context.Context, deps Deps) tea.Cmd {
	return func() tea.Msg {
		msg := healthMsg{}
		root := deps.APIRoot
		if root == "" {
			root = "https://www.servercontrolpanel.de/scp-core"
		}

		pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()

		start := time.Now()
		req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, root+"/api/ping", nil)
		if err != nil {
			msg.err = err.Error()
			return msg
		}
		req.Header.Set("Accept", "*/*")
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		msg.pingMS = time.Since(start).Milliseconds()
		if err != nil {
			msg.pingOK = false
			msg.err = err.Error()
		} else {
			_ = resp.Body.Close()
			msg.pingOK = resp.StatusCode >= 200 && resp.StatusCode < 300
		}

		if deps.Client != nil {
			maintCtx, maintCancel := context.WithTimeout(ctx, 8*time.Second)
			r, e := deps.Client.GetApiV1MaintenanceWithResponse(maintCtx)
			maintCancel()
			if e == nil && r != nil && r.StatusCode() == 200 {
				m := r.JSON200
				if m == nil {
					m = r.HALJSON200
				}
				if m != nil && (m.StartAt != nil || m.FinishAt != nil) {
					switch {
					case m.FinishAt != nil:
						msg.maintenance = "until " + m.FinishAt.Local().Format("15:04")
					case m.StartAt != nil:
						msg.maintenance = "from " + m.StartAt.Local().Format("15:04")
					}
				} else {
					msg.maintenance = "none"
				}
			}
		}
		return msg
	}
}

func (m model) onHealth(msg healthMsg) (tea.Model, tea.Cmd) {
	m.health = msg
	return m, nil
}

func (m model) healthChip() string {
	if m.health.pingMS == 0 && !m.health.pingOK && m.health.err == "" && m.health.maintenance == "" {
		return ""
	}
	ping := "scp · down"
	style := styleErr
	if m.health.pingOK {
		ping = fmt.Sprintf("scp · ok %dms", m.health.pingMS)
		style = styleOk
	}
	chip := style.Render(ping)
	if m.health.maintenance != "" && m.health.maintenance != "none" {
		chip += "  " + styleWarn.Render("maint "+m.health.maintenance)
	}
	return chip
}
