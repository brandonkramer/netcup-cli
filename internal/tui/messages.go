package tui

import (
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

type serversLoadedMsg struct {
	servers []scpclient.ServerListMinimal
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
	uuid string
	task *scpclient.TaskInfo
	err  error
}

type metricsLoadedMsg struct {
	kind string
	data any
	err  error
}

type mediaLoadedMsg struct {
	attached any
	catalog  []scpclient.IsoImage
	userISOs any
	err      error
}

type isoActionMsg struct {
	action string
	task   *scpclient.TaskInfo
	err    error
}

type attrSetMsg struct {
	attr string
	task *scpclient.TaskInfo
	err  error
}

type rescueMsg struct {
	action string
	data   any
	task   *scpclient.TaskInfo
	err    error
}
