package wait

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
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
	transientFails := 0

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
				if isTransient(err) {
					transientFails++
					backoff(ctx, transientFails, opts.PollInterval)
					continue
				}
				return &task, err
			}
			if resp == nil {
				transientFails++
				backoff(ctx, transientFails, opts.PollInterval)
				continue
			}
			status := resp.StatusCode()
			if status == 401 {
				return &task, output.Exit(output.ExitAuth, fmt.Sprintf("task poll: HTTP 401 — run netcup auth login"))
			}
			if status == 429 || status >= 500 {
				transientFails++
				backoff(ctx, transientFails, opts.PollInterval)
				continue
			}
			if status != 200 {
				return &task, fmt.Errorf("task poll failed: HTTP %d", status)
			}
			next := resp.JSON200
			if next == nil {
				next = resp.HALJSON200
			}
			if next == nil {
				transientFails++
				backoff(ctx, transientFails, opts.PollInterval)
				continue
			}
			transientFails = 0
			task = *next
			if opts.OnProgress != nil {
				opts.OnProgress(task)
			}
			if task.State != nil && terminal(*task.State) {
				return finish(&task)
			}
		}
	}
}

func backoff(ctx context.Context, fails int, base time.Duration) {
	if fails < 1 {
		fails = 1
	}
	d := base
	if fails > 1 {
		d = time.Duration(fails) * base / 2
	}
	if d > 10*time.Second {
		d = 10 * time.Second
	}
	timer := time.NewTimer(d)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
	}
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isTransient(urlErr.Err)
	}
	msg := strings.ToLower(err.Error())
	for _, frag := range []string{
		"connection reset", "broken pipe", "i/o timeout", "tls handshake timeout",
		"server closed idle connection", "http2: client connection force closed",
		"read tcp", "write tcp", "eof", "temporary failure", "connection refused",
	} {
		if strings.Contains(msg, frag) {
			return true
		}
	}
	return false
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
