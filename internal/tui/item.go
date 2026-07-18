package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

type serverItem struct {
	server   scpclient.ServerListMinimal
	selected bool
}

func (i serverItem) FilterValue() string {
	parts := []string{
		derefStr(i.server.Name),
		derefStr(i.server.Nickname),
		derefStr(i.server.Hostname),
	}
	if i.server.Id != nil {
		parts = append(parts, strconv.FormatInt(int64(*i.server.Id), 10))
	}
	return strings.Join(parts, " ")
}

func (i serverItem) Title() string {
	mark := "·"
	if i.selected {
		mark = "✓"
	}
	name := serverLabel(i.server)
	nick := derefStr(i.server.Nickname)
	if nick != "" {
		return fmt.Sprintf("%s %s  %s", mark, name, nick)
	}
	return fmt.Sprintf("%s %s", mark, name)
}

func (i serverItem) Description() string {
	host := derefStr(i.server.Hostname)
	id := derefInt(i.server.Id)
	if host != "" {
		return fmt.Sprintf("%d  %s", id, host)
	}
	return fmt.Sprintf("%d", id)
}

type taskItem struct {
	task scpclient.TaskInfoMinimal
}

func (i taskItem) FilterValue() string {
	return strings.Join([]string{
		derefStr(i.task.Name),
		derefStr(i.task.Uuid),
		derefTaskState(i.task.State),
		derefStr(i.task.Message),
	}, " ")
}

func (i taskItem) Title() string {
	name := derefStr(i.task.Name)
	if name == "" {
		name = shortUUID(derefStr(i.task.Uuid))
	}
	return fmt.Sprintf("%s · %s", name, derefTaskState(i.task.State))
}

func (i taskItem) Description() string {
	parts := []string{shortUUID(derefStr(i.task.Uuid))}
	if i.task.TaskProgress != nil && i.task.TaskProgress.ProgressInPercent != nil {
		parts = append(parts, fmt.Sprintf("%.0f%%", *i.task.TaskProgress.ProgressInPercent))
	}
	if msg := derefStr(i.task.Message); msg != "" {
		parts = append(parts, msg)
	}
	return strings.Join(parts, " · ")
}

type mediaItem struct {
	kind        string // "attached" | "catalog" | "user"
	title       string
	description string
	isoID       int32
	userName    string
	filter      string
}

func (i mediaItem) FilterValue() string { return i.filter }
func (i mediaItem) Title() string       { return i.title }
func (i mediaItem) Description() string { return i.description }
