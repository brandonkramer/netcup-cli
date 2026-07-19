package tui

import (
	"strings"
	"testing"

	"github.com/brandonkramer/netcup-cli/internal/scpclient"
)

func TestFormatAttachedISO(t *testing.T) {
	name := "debian-13-amd64.iso"
	yes := true
	no := false

	if got := formatAttachedISO(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
	if got := formatAttachedISO(&scpclient.Iso{Iso: &name, IsoAttached: &yes}); got != name {
		t.Fatalf("attached: got %q want %q", got, name)
	}
	if got := formatAttachedISO(scpclient.Iso{Iso: &name, IsoAttached: &no}); got != name+" (not attached)" {
		t.Fatalf("not attached: %q", got)
	}
	if got := formatAttachedISO(&scpclient.Iso{}); got != "" {
		t.Fatalf("empty: %q", got)
	}
	// Regression: %v on *Iso used to show pointer addresses like &{0x… 0x…}.
	raw := formatAttachedISO(&scpclient.Iso{Iso: &name, IsoAttached: &yes})
	if strings.Contains(raw, "0x") || strings.Contains(raw, "&{") {
		t.Fatalf("looks like pointer dump: %q", raw)
	}
}
