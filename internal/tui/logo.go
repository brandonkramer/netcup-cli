package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI-16-friendly bright colors (visible on almost every terminal).
var logoGradient = []lipgloss.Color{
	lipgloss.Color("13"), // bright magenta
	lipgloss.Color("5"),  // magenta
	lipgloss.Color("12"), // bright blue
	lipgloss.Color("14"), // bright cyan
}

func renderLogo() string {
	word := colorizeGradient("netcup")
	tag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(" scp")
	return word + tag
}

func colorizeGradient(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	n := len(runes)
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		idx := 0
		if n > 1 {
			idx = i * (len(logoGradient) - 1) / (n - 1)
		}
		b.WriteString(lipgloss.NewStyle().
			Foreground(logoGradient[idx]).
			Bold(true).
			Render(string(r)))
	}
	return b.String()
}

func logoHeight() int { return 1 }

func logoWidth() int {
	return lipgloss.Width("netcup scp")
}
