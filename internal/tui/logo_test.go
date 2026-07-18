package tui

import (
	"strings"
	"testing"
)

func TestRenderLogo(t *testing.T) {
	got := renderLogo()
	if got == "" {
		t.Fatal("empty logo")
	}
	plain := colorizeGradient("netcup")
	for _, r := range []rune("netcup") {
		if !strings.ContainsRune(plain, r) {
			t.Fatalf("missing %c in wordmark", r)
		}
	}
	if !strings.Contains(got, "scp") {
		t.Fatalf("missing scp tag: %q", got)
	}
	if logoHeight() != 1 {
		t.Fatalf("height %d", logoHeight())
	}
}
