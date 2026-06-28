package tui

import "github.com/cheetahbyte/apex/internal/tui/components/statusline"

const (
	headerHeight = 1
	ruleHeight   = 1
	inputHeight  = 1
)

// chrome is every row the chat area does NOT get: header, the rule under it,
// the rule above the input, the input line, and the footer.
func chromeHeight() int {
	return headerHeight + ruleHeight + ruleHeight + inputHeight + statusline.Height
}

// resize tells each component how much space it gets. The root owns the
// layout budget; each component handles its own frame/padding math.
func (m *Model) resize() {
	chatHeight := safeSub(m.height, chromeHeight())

	m.chat.SetSize(m.width, chatHeight)
	m.prompt.SetSize(m.width, inputHeight)
	m.status = m.status.SetSize(m.width)
}
