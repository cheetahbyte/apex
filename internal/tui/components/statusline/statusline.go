package statusline

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Height is the vertical space the status line occupies.
const Height = 1

// Model is the status bar component. It renders a single line with app
// name on the left and runtime stats on the right.
type Model struct {
	width int
}

// New creates an empty status line.
func New() Model {
	return Model{}
}

// SetSize sets the width of the status line. Height is always Height.
func (m Model) SetSize(width int) Model {
	m.width = width
	return m
}

// View renders the status line. The caller provides the runtime stats
// (message count, input length, terminal size) since they live on the
// root model and the session.
func (m Model) View(provider, model string, msgCount, inputLen, inputLimit, termW, termH int) string {
	left := fmt.Sprintf(" Apex  %s/%s ", provider, model)
	right := fmt.Sprintf(
		" msgs:%d  input:%d/%d  size:%dx%d ",
		msgCount, inputLen, inputLimit, termW, termH,
	)

	spaces := safeSub(m.width, lipgloss.Width(left)+lipgloss.Width(right))
	line := left + strings.Repeat(" ", spaces) + right

	return lipgloss.NewStyle().
		Width(m.width).
		Height(Height).
		Background(lipgloss.Color("8")).
		Render(line)
}

func safeSub(a, b int) int {
	if a < b {
		return 0
	}
	return a - b
}
