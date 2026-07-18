package tui

import (
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

type serversLoadedMsg struct {
	servers []scpclient.ServerListMinimal
	cached  bool
	err     error
}

type detailLoadedMsg struct {
	server *scpclient.Server
	err    error
}

type guestAgentMsg struct {
	status any
	err    error
}

type powerStartedMsg struct {
	serverID   int32
	serverName string
	action     string
	status     int
	task       *scpclient.TaskInfo
	err        error
}

type jobsPolledMsg struct {
	jobs []job
	err  error
}

type job struct {
	ServerID   int32
	ServerName string
	Action     string
	UUID       string
	State      string
	Progress   float32
	Message    string
	Err        string
	Done       bool
}

type tasksLoadedMsg struct {
	tasks []scpclient.TaskInfoMinimal
	err   error
}

type taskCancelMsg struct {
	uuid   string
	status int
	task   *scpclient.TaskInfo
	err    error
}

type metricsLoadedMsg struct {
	kind string
	data any
	err  error
}

type mediaLoadedMsg struct {
	attached   any
	catalog    []scpclient.IsoImage
	userISOs   any
	userImages any
	err        error
}

type isoActionMsg struct {
	action string
	status int
	task   *scpclient.TaskInfo
	err    error
}

type attrSetMsg struct {
	attr   string
	status int
	task   *scpclient.TaskInfo
	err    error
}

type rescueMsg struct {
	action string
	status int
	data   any
	task   *scpclient.TaskInfo
	err    error
}

type resourceLoadedMsg struct {
	tab, mode string
	items     []resourceItem
	detail    string
	warning   string
	err       error
}

type resourceActionMsg struct {
	action string
	status int
	task   *scpclient.TaskInfo
	err    error
}
