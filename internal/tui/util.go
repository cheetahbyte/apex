package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// borderSize returns the horizontal and vertical space consumed by a style's
// border only (no padding). Used to convert a desired OUTER box size into the
// Width/Height to set on the style.
func borderSize(s lipgloss.Style) (w, h int) {
	return s.GetBorderLeftSize() + s.GetBorderRightSize(),
		s.GetBorderTopSize() + s.GetBorderBottomSize()
}

func safeSub(a, b int) int {
	if a-b < 0 {
		return 0
	}
	return a - b
}

func (m Model) Debug() string {
	return fmt.Sprintf("%dx%d", m.width, m.height)
}
