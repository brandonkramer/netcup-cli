package tui

import (
	"fmt"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
	"github.com/charmbracelet/lipgloss"
)

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

func derefBool(v *bool) bool {
	return v != nil && *v
}

// formatAttachedISO renders GET /iso payloads for the Media tab.
// Avoids fmt %v on *Iso, which prints pointer addresses for *string/*bool fields.
func formatAttachedISO(v any) string {
	if v == nil {
		return ""
	}
	switch iso := v.(type) {
	case *scpclient.Iso:
		if iso == nil {
			return ""
		}
		return formatAttachedISO(*iso)
	case scpclient.Iso:
		name := derefStr(iso.Iso)
		if name == "" {
			if iso.IsoAttached != nil && !*iso.IsoAttached {
				return ""
			}
			return ""
		}
		if iso.IsoAttached != nil && !*iso.IsoAttached {
			return name + " (not attached)"
		}
		return name
	case map[string]any:
		if name, ok := iso["iso"].(string); ok && name != "" {
			return name
		}
		return strings.TrimSpace(prettyJSON(iso))
	default:
		s := strings.TrimSpace(prettyJSON(v))
		if s == "" || s == "null" {
			return ""
		}
		// Collapse multiline JSON to a one-line label for list descriptions.
		s = strings.ReplaceAll(s, "\n", " ")
		s = strings.Join(strings.Fields(s), " ")
		return s
	}
}

func derefState(s *scpclient.ServerState) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func derefTaskState(s *scpclient.TaskState) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func formatUptime(sec *int32) string {
	if sec == nil {
		return ""
	}
	s := *sec
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func shortUUID(u string) string {
	if len(u) > 8 {
		return u[:8]
	}
	return u
}

func isTerminalTask(state scpclient.TaskState) bool {
	switch state {
	case scpclient.TaskStateFINISHED, scpclient.TaskStateERROR, scpclient.TaskStateCANCELED, scpclient.TaskStateROLLBACK:
		return true
	default:
		return false
	}
}

func serverLabel(s scpclient.ServerListMinimal) string {
	name := derefStr(s.Name)
	if name == "" {
		return fmt.Sprintf("#%d", derefInt(s.Id))
	}
	return name
}

// wrapText soft-wraps s to width columns (viewport does not wrap).
func wrapText(s string, width int) string {
	if width < 8 {
		width = 8
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

func joinNotes(notes []string) string {
	if len(notes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("API notes\n\n")
	for _, n := range notes {
		b.WriteString("· ")
		b.WriteString(n)
		b.WriteString("\n")
	}
	return b.String()
}
