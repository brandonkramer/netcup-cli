package tui

import (
	"errors"
	"io"
	"testing"
)

func TestIsTransientNetErr(t *testing.T) {
	if !isTransientNetErr(errors.New(`Get "https://x": read tcp 1.2.3.4:5->6.7.8.9:443: connection reset by peer`)) {
		t.Fatal("expected transient")
	}
	if !isTransientNetErr(io.ErrUnexpectedEOF) {
		t.Fatal("expected EOF transient")
	}
	if isTransientNetErr(errors.New("HTTP 404")) {
		t.Fatal("404 should not be transient")
	}
}

func TestFriendlyNetErr(t *testing.T) {
	err := friendlyNetErr("list tasks", errors.New(`Get "https://x": read tcp 1.2.3.4:5->6.7.8.9:443: connection reset by peer`))
	want := "list tasks: connection reset — press R to retry · check https://www.netcup-status.de/"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}
