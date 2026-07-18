package tui

import (
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestFormatGuest(t *testing.T) {
	yes := true
	no := false
	if got := formatGuest(&scpclient.GuestAgentStatus{Available: &yes}); got != "available" {
		t.Fatalf("got %q", got)
	}
	if got := formatGuest(&scpclient.GuestAgentStatus{Available: &no}); got != "unavailable" {
		t.Fatalf("got %q", got)
	}
	if got := formatGuest(nil); got != "…" {
		t.Fatalf("got %q", got)
	}
}
