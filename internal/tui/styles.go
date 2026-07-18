package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Bright blue accents (avoid bubbles' default magenta/purple).
	colAccent  = lipgloss.Color("39") // dodger blue
	colMuted   = lipgloss.Color("244")
	colOk      = lipgloss.Color("42")
	colWarn    = lipgloss.Color("214")
	colErr     = lipgloss.Color("196")
	colBorder  = lipgloss.Color("238")
	colTitle   = lipgloss.Color("15")
	colLabel   = lipgloss.Color("245")
	colValue   = lipgloss.Color("252")
	colTabOnBg = lipgloss.Color("236")

	// Zellij-like powerline: light segment fill, dark label, blue key, dark lead.
	colPowerLeadBg = lipgloss.Color("236") // dark first segment
	colPowerSegBg  = lipgloss.Color("250") // light grey segments
	colPowerSegFg  = lipgloss.Color("232") // near-black on light
	colPowerOnBg   = lipgloss.Color("33")  // active tab (blue)
	colPowerOnFg   = lipgloss.Color("15")
	colKey         = lipgloss.Color("33") // blue key letter (was pink)
	colTip         = lipgloss.Color("42")

	styleAppTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colTitle).
			Background(colAccent).
			Padding(0, 1)
	styleMuted      = lipgloss.NewStyle().Foreground(colMuted)
	styleErr        = lipgloss.NewStyle().Foreground(colErr)
	styleOk         = lipgloss.NewStyle().Foreground(colOk).Bold(true)
	styleWarn       = lipgloss.NewStyle().Foreground(colWarn).Bold(true)
	styleLabel      = lipgloss.NewStyle().Foreground(colLabel).Width(11)
	styleValue      = lipgloss.NewStyle().Foreground(colValue)
	stylePanelTitle = lipgloss.NewStyle().Bold(true).Foreground(colTitle)
	stylePanel      = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)
	styleTabOn = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent).
			Background(colTabOnBg).
			Padding(0, 1)
	styleTabOff = lipgloss.NewStyle().
			Foreground(colMuted).
			Padding(0, 1)
	styleStatusBar = lipgloss.NewStyle().Foreground(colMuted)
	styleHelp      = lipgloss.NewStyle().Foreground(colMuted)
	styleJobsTitle = lipgloss.NewStyle().Bold(true).Foreground(colMuted)
	styleFocus     = lipgloss.NewStyle().Foreground(colAccent)
	styleTip       = lipgloss.NewStyle().Foreground(colMuted)
	styleTipKey    = lipgloss.NewStyle().Bold(true).Foreground(colTip)
	styleContext   = lipgloss.NewStyle().Foreground(colValue).Background(colTabOnBg).Padding(0, 1)
)

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "RUNNING", "ON", "FINISHED", "OK":
		return styleOk
	case "STOPPED", "OFF", "SUSPENDED", "PMSUSPENDED", "UNTRACKED", "ACCEPTED":
		return styleWarn
	case "ERROR", "ROLLBACK", "CANCELED":
		return styleErr
	default:
		return styleValue
	}
}

func kv(label, value string) string {
	if value == "" {
		value = "—"
	}
	return styleLabel.Render(label) + " " + styleValue.Render(value)
}
