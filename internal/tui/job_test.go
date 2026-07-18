package tui

import (
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestJobFromTaskStatusUntracked202(t *testing.T) {
	j := jobFromTaskStatus(1, "srv", "nic-create", 202, nil, nil)
	if j.State != "UNTRACKED" || !j.Done || j.UUID != "" {
		t.Fatalf("got %+v", j)
	}
	if j.Message == "" {
		t.Fatal("expected message")
	}
}

func TestJobFromTaskStatusOK200(t *testing.T) {
	j := jobFromTaskStatus(1, "srv", "policy-create", 200, nil, nil)
	if j.State != "OK" || !j.Done {
		t.Fatalf("got %+v", j)
	}
}

func TestJobFromTaskStatusWithUUID(t *testing.T) {
	uuid := "abc"
	state := scpclient.TaskStateRUNNING
	j := jobFromTaskStatus(1, "srv", "start", 202, &scpclient.TaskInfo{Uuid: &uuid, State: &state}, nil)
	if j.UUID != "abc" || j.Done {
		t.Fatalf("got %+v", j)
	}
}

func TestJobFromTaskWithoutStatusIsOK(t *testing.T) {
	// Legacy callers that omit status still get OK for nil-task success paths.
	j := jobFromTask(1, "srv", "detach", nil, nil)
	if j.State != "OK" || !j.Done {
		t.Fatalf("got %+v", j)
	}
}

func TestJobFromTaskStatusBodyless202ByAction(t *testing.T) {
	actions := []string{"cancel", "iso.attach", "rescue.enable", "start", "set.cpu"}
	for _, action := range actions {
		j := jobFromTaskStatus(1, "srv", action, 202, nil, nil)
		if j.State != "UNTRACKED" || !j.Done {
			t.Fatalf("%s: got %+v", action, j)
		}
	}
}
