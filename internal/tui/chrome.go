package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const powerlineChevron = "\ue0b0"

type powerSeg struct {
	bg      lipgloss.Color
	fg      lipgloss.Color // default label fg when content is plain
	content string         // may include nested styles (keys); bg must match seg.bg
	plain   bool           // true → style content with fg/bg; false → content pre-styled
}

func joinPowerline(segs []powerSeg) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range segs {
		var body string
		if seg.plain {
			fg := seg.fg
			if fg == "" {
				fg = colPowerSegFg
			}
			body = lipgloss.NewStyle().
				Foreground(fg).
				Background(seg.bg).
				Padding(0, 1).
				Render(seg.content)
		} else {
			// Pre-styled content (keys); pad with matching bg.
			body = lipgloss.NewStyle().
				Background(seg.bg).
				Padding(0, 1).
				Render(seg.content)
		}
		b.WriteString(body)

		chev := lipgloss.NewStyle().Foreground(seg.bg)
		if i+1 < len(segs) {
			chev = chev.Background(segs[i+1].bg)
		}
		b.WriteString(chev.Render(powerlineChevron))
	}
	return b.String()
}

func keyContent(key, text string) string {
	keyPart := lipgloss.NewStyle().
		Bold(true).
		Foreground(colKey).
		Background(colPowerSegBg).
		Render(key)
	label := lipgloss.NewStyle().
		Foreground(colPowerSegFg).
		Background(colPowerSegBg).
		Render(" " + text)
	brackets := lipgloss.NewStyle().
		Foreground(colPowerSegFg).
		Background(colPowerSegBg)
	return brackets.Render("<") + keyPart + brackets.Render(">") + label
}

func (m model) tabsBarView() string {
	segs := make([]powerSeg, 0, len(tabNames)+1)
	segs = append(segs, powerSeg{bg: colPowerLeadBg, fg: colTitle, content: "netcup", plain: true})
	for i, name := range tabNames {
		bg := colPowerSegBg
		fg := colPowerSegFg
		if m.tab == tabKind(i) {
			bg = colPowerOnBg
			fg = colPowerOnFg
		}
		segs = append(segs, powerSeg{
			bg: bg, fg: fg,
			content: fmt.Sprintf("%d %s", i+1, name),
			plain:   true,
		})
	}
	return joinPowerline(segs)
}

func (m model) keysBarView() string {
	lead := "Keys"
	var items []struct{ key, text string }
	switch m.tab {
	case tabServers:
		if m.resourceMode == "metrics" {
			lead = "Metrics"
			items = []struct{ key, text string }{
				{"c", "cpu"}, {"d", "disk"}, {"n", "net"}, {"p", "pkt"},
				{"h", "hours"}, {"esc", "back"},
			}
		} else {
			lead = "Server"
			items = []struct{ key, text string }{
				{"↑↓", "move"}, {"enter", "reload"}, {"m", "metrics"},
				{"L", "logs"}, {"g", "guest"}, {"y", "copy"}, {"a", "clear"},
			}
		}
	case tabTasks:
		lead = "Tasks"
		items = []struct{ key, text string }{{"c", "cancel"}, {"R", "refresh"}}
	case tabSnapshots:
		lead = "Snap"
		items = []struct{ key, text string }{
			{"c", "create"}, {"r", "revert"}, {"d", "delete"},
			{"x", "export"}, {"D", "dry-run"}, {"R", "refresh"},
		}
	case tabDisks:
		lead = "Disks"
		items = []struct{ key, text string }{
			{"enter", "detail"}, {"v", "drivers"}, {"s", "driver"}, {"f", "format"},
		}
	case tabFirewall:
		lead = "FW"
		items = []struct{ key, text string }{
			{"enter", "get"}, {"r", "reapply"}, {"c", "restore"},
			{"s", "set"}, {"S", "SSH"}, {"H", "HTTP"}, {"n", "new"},
		}
	case tabMedia:
		lead = "Media"
		items = []struct{ key, text string }{
			{"enter", "attach"}, {"D", "detach"}, {"u", "ISO"},
			{"i", "image"}, {"F", "flavours"}, {"S", "setup"}, {"U", "user"},
		}
	case tabNetwork:
		lead = "Net"
		items = []struct{ key, text string }{
			{"enter", "detail"}, {"c", "NIC"}, {"e", "VLAN"},
			{"f", "route"}, {"r", "rDNS"}, {"d", "del"},
		}
	case tabAccount:
		lead = "Acct"
		items = []struct{ key, text string }{
			{"enter", "info"}, {"a", "add"}, {"d", "del"}, {"L", "logs"}, {"R", "reload"},
		}
	}

	segs := make([]powerSeg, 0, len(items)+1)
	segs = append(segs, powerSeg{bg: colPowerLeadBg, fg: colTitle, content: lead, plain: true})
	for _, it := range items {
		segs = append(segs, powerSeg{bg: colPowerSegBg, content: keyContent(it.key, it.text), plain: false})
	}
	return clampLines(joinPowerline(segs), 1, m.width)
}

func (m model) tipText() string {
	tip := "tab cycles tabs · q quits"
	switch m.tab {
	case tabServers:
		if id, _ := m.focusedServer(); id != 0 {
			tip = fmt.Sprintf("netcup servers get -s %d", id)
		} else {
			tip = "select a server, then open Snapshots/Disks/Firewall/Media"
		}
	case tabSnapshots, tabDisks:
		if m.server.ID == 0 {
			tip = "Select a server on Servers before using this tab"
		}
	case tabFirewall:
		if m.server.ID == 0 {
			tip = "Select a server on Servers before using this tab"
		} else {
			tip = "S/H create SSH or HTTP policy presets · s assign JSON to MAC"
		}
	case tabMedia:
		if m.server.ID == 0 {
			tip = "Select a server on Servers before using this tab"
		} else {
			tip = "S setup needs JSON with imageFlavourId · U setup-user on image row"
		}
	case tabNetwork:
		tip = "c create VLAN NIC · e rename VLAN · f route failover · g/x rDNS"
	}
	return styleTip.Render("Tip: ") + styleTipKey.Render(tip)
}

// metaBarView shows tip, jobs, and status on one horizontal line.
func (m model) metaBarView() string {
	sep := styleMuted.Render("  │  ")
	line := m.tipText() + sep + m.jobsView() + sep + styleStatusBar.Render(m.statusText())
	return clampLines(line, 1, m.width)
}

func (m model) serverContextChip() string {
	if m.server.ID == 0 {
		return styleMuted.Render(" no server selected ")
	}
	return styleContext.Render(fmt.Sprintf("server %d · %s", m.server.ID, m.server.Name))
}
