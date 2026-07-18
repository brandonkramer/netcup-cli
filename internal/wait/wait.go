package wait

import (
	"context"
	"fmt"
	"time"

	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

type Options struct {
	Timeout      time.Duration
	PollInterval time.Duration
	NoWait       bool
	OnProgress   func(task scpclient.TaskInfo)
}

func terminal(state scpclient.TaskState) bool {
	switch state {
	case scpclient.TaskStateFINISHED, scpclient.TaskStateERROR, scpclient.TaskStateCANCELED, scpclient.TaskStateROLLBACK:
		return true
	default:
		return false
	}
}

func Task(
	ctx context.Context,
	client *scpclient.ClientWithResponses,
	task scpclient.TaskInfo,
	opts Options,
) (*scpclient.TaskInfo, error) {
	if opts.NoWait {
		return &task, nil
	}
	if task.Uuid == nil {
		return &task, fmt.Errorf("task missing uuid")
	}
	if task.State != nil && terminal(*task.State) {
		return finish(&task)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Minute
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}

	deadline := time.Now().Add(opts.Timeout)
	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return &task, output.Exit(output.ExitTimeout, fmt.Sprintf("timed out waiting for task %s", *task.Uuid))
		}
		select {
		case <-ctx.Done():
			return &task, output.Exit(output.ExitInterrupted, "interrupted")
		case <-ticker.C:
			resp, err := client.GetApiV1TasksUuidWithResponse(ctx, *task.Uuid)
			if err != nil {
				return &task, err
			}
			if resp.StatusCode() != 200 || resp.JSON200 == nil {
				return &task, fmt.Errorf("task poll failed: HTTP %d", resp.StatusCode())
			}
			task = *resp.JSON200
			if opts.OnProgress != nil {
				opts.OnProgress(task)
			}
			if task.State != nil && terminal(*task.State) {
				return finish(&task)
			}
		}
	}
}

func finish(task *scpclient.TaskInfo) (*scpclient.TaskInfo, error) {
	if task.State == nil {
		return task, nil
	}
	switch *task.State {
	case scpclient.TaskStateFINISHED:
		return task, nil
	case scpclient.TaskStateERROR, scpclient.TaskStateROLLBACK:
		msg := "task failed"
		if task.Message != nil && *task.Message != "" {
			msg = *task.Message
		} else if task.ResponseError != nil && task.ResponseError.Message != nil {
			msg = *task.ResponseError.Message
		}
		return task, output.Exit(output.ExitAPI, msg)
	case scpclient.TaskStateCANCELED:
		return task, output.Exit(output.ExitAPI, "task canceled")
	default:
		return task, nil
	}
}

func ProgressPercent(task scpclient.TaskInfo) float32 {
	if task.TaskProgress != nil && task.TaskProgress.ProgressInPercent != nil {
		return *task.TaskProgress.ProgressInPercent
	}
	return 0
}

func TaskEnvelope(task *scpclient.TaskInfo) map[string]any {
	if task == nil {
		return nil
	}
	out := map[string]any{
		"uuid":             deref(task.Uuid),
		"state":            derefState(task.State),
		"progress_percent": ProgressPercent(*task),
		"message":          deref(task.Message),
		"result":           task.Result,
	}
	return out
}

func deref(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func derefState(s *scpclient.TaskState) any {
	if s == nil {
		return nil
	}
	return string(*s)
}
