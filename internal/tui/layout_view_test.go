package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPanelBoxOuterSize(t *testing.T) {
	got := panelBox(40, 10, "hi")
	if w := lipgloss.Width(got); w != 40 {
		t.Fatalf("width: got %d want 40", w)
	}
	if h := lipgloss.Height(got); h != 10 {
		t.Fatalf("height: got %d want 10", h)
	}
}

func TestTwoPanelsFitWidth(t *testing.T) {
	m := model{width: 80, bodyH: 12}
	got := m.twoPanels("left", "right")
	if w := lipgloss.Width(got); w != 80 {
		t.Fatalf("width: got %d want 80", w)
	}
	if h := lipgloss.Height(got); h != 12 {
		t.Fatalf("height: got %d want 12", h)
	}
}
