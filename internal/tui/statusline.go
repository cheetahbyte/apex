package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderStatusLine(width int) string {
	left := " Apex "
	right := fmt.Sprintf(
		" msgs:%d  input:%d/%d  size:%dx%d ",
		m.session.Len(),
		len(m.input.Value()),
		m.input.CharLimit,
		m.width,
		m.height,
	)

	spaces := safeSub(width, lipgloss.Width(left)+lipgloss.Width(right))
	line := left + strings.Repeat(" ", spaces) + right

	return lipgloss.NewStyle().
		Width(width).
		Height(statusHeight).
		Background(lipgloss.Color("8")).
		Render(line)
}
