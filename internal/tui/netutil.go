package tui

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

func apiTimeout() time.Duration { return 45 * time.Second }

// apiCtx uses a fresh timeout so Bubble Tea UI events cannot cancel in-flight GETs.
func apiCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), apiTimeout())
}

func isTransientNetErr(err error) bool {
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
		return isTransientNetErr(urlErr.Err)
	}
	msg := strings.ToLower(err.Error())
	for _, frag := range []string{
		"connection reset",
		"broken pipe",
		"i/o timeout",
		"tls handshake timeout",
		"server closed idle connection",
		"http2: client connection force closed",
		"read tcp",
		"write tcp",
		"eof",
		"temporary failure",
		"connection refused",
	} {
		if strings.Contains(msg, frag) {
			return true
		}
	}
	return false
}

func friendlyNetErr(action string, err error) error {
	if err == nil {
		return nil
	}
	if !isTransientNetErr(err) {
		return err
	}
	kind := "network error"
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || errors.Is(err, context.DeadlineExceeded):
		kind = "timed out"
	case strings.Contains(msg, "reset"):
		kind = "connection reset"
	case strings.Contains(msg, "refused"):
		kind = "connection refused"
	case strings.Contains(msg, "eof"):
		kind = "connection closed"
	}
	return errors.New(action + ": " + kind + " — press R to retry · check https://www.netcup-status.de/")
}

// withRetry runs fn up to 3 times on transient network errors.
func withRetry(fn func(ctx context.Context) error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := apiCtx()
		err = fn(ctx)
		cancel()
		if err == nil || !isTransientNetErr(err) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 400 * time.Millisecond)
	}
	return err
}
