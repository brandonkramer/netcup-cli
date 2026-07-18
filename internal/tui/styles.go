package tui

import "github.com/charmbracelet/lipgloss"

var (
	colAccent  = lipgloss.Color("12")
	colMuted   = lipgloss.Color("244")
	colOk      = lipgloss.Color("42")
	colWarn    = lipgloss.Color("214")
	colErr     = lipgloss.Color("196")
	colBorder  = lipgloss.Color("238")
	colTitle   = lipgloss.Color("15")
	colLabel   = lipgloss.Color("245")
	colValue   = lipgloss.Color("252")
	colTabOnBg = lipgloss.Color("236")

	styleAppTitle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleMuted    = lipgloss.NewStyle().Foreground(colMuted)
	styleErr      = lipgloss.NewStyle().Foreground(colErr)
	styleOk       = lipgloss.NewStyle().Foreground(colOk).Bold(true)
	styleWarn     = lipgloss.NewStyle().Foreground(colWarn).Bold(true)
	styleLabel    = lipgloss.NewStyle().Foreground(colLabel).Width(11)
	styleValue    = lipgloss.NewStyle().Foreground(colValue)
	stylePanelTitle = lipgloss.NewStyle().Bold(true).Foreground(colTitle)
	stylePanel = lipgloss.NewStyle().
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
)

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "RUNNING", "ON", "FINISHED":
		return styleOk
	case "STOPPED", "OFF", "SUSPENDED", "PMSUSPENDED":
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
