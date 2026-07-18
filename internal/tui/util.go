package tui

import (
	"fmt"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
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
