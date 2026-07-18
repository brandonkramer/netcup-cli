package tui

import "github.com/charmbracelet/lipgloss"

// panelBox sizes a bordered pane to an exact outer width/height.
// lipgloss Width/Height cover the content box (padding inside); the border
// adds 2 columns and 2 rows outside that.
func panelBox(outerW, outerH int, content string) string {
	if outerW < 8 {
		outerW = 8
	}
	if outerH < 4 {
		outerH = 4
	}
	return stylePanel.Width(outerW - 2).Height(outerH - 2).Render(content)
}

// panelSplit returns outer widths for list + detail that sum to m.width.
func (m model) panelSplit() (listOuter, detailOuter int) {
	listOuter = m.width * 40 / 100
	if listOuter < 28 {
		listOuter = 28
	}
	if listOuter > m.width-24 {
		listOuter = m.width - 24
	}
	if listOuter < 8 {
		listOuter = m.width / 2
	}
	detailOuter = m.width - listOuter
	if detailOuter < 8 {
		detailOuter = 8
		listOuter = m.width - detailOuter
	}
	return listOuter, detailOuter
}

// twoPanels is the dual bordered layout (list + detail).
func (m model) twoPanels(left, right string) string {
	h := m.bodyH
	if h < 4 {
		h = 4
	}
	listW, detailW := m.panelSplit()
	return lipgloss.JoinHorizontal(lipgloss.Top,
		panelBox(listW, h, left),
		panelBox(detailW, h, right),
	)
}

func (m model) onePanel(content string) string {
	return panelBox(m.width, m.bodyH, content)
}

func clampLines(s string, maxLines, width int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	if width < 1 {
		width = 1
	}
	return lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		MaxHeight(maxLines).
		Render(s)
}
