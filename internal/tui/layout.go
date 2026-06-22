package tui

import "github.com/cheetahbyte/apex/internal/tui/components/statusline"

const (
	sidebarOuterWidth = 40
	inputOuterHeight  = 3
)

// resize tells each component how much space it gets. The root owns the
// layout budget; each component handles its own frame/padding math.
func (m *Model) resize() {
	contentOuterWidth := safeSub(m.width, sidebarOuterWidth)
	chatOuterHeight := safeSub(m.height, inputOuterHeight+statusline.Height)

	m.chat.SetSize(contentOuterWidth, chatOuterHeight)
	m.prompt.SetSize(contentOuterWidth, inputOuterHeight)
	m.status = m.status.SetSize(contentOuterWidth)
}
