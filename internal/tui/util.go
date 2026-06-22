package tui

import "fmt"

func safeSub(a, b int) int {
	if a-b < 0 {
		return 0
	}
	return a - b
}

func (m Model) Debug() string {
	return fmt.Sprintf("%dx%d", m.width, m.height)
}
