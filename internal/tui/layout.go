package tui

import "charm.land/lipgloss/v2"

const (
	sidebarOuterWidth = 40
	inputOuterHeight  = 3
	statusHeight      = 1
)

// resize sizes the INNER widgets (viewport, text input). Their content area is
// outer - full frame (border + padding), so GetFrameSize is correct here.
func (m *Model) resize() {
	contentOuterWidth := safeSub(m.width, sidebarOuterWidth)
	chatOuterHeight := safeSub(m.height, inputOuterHeight+statusHeight)

	chatFrameW, chatFrameH := chatStyle().GetFrameSize()
	inputFrameW, _ := inputStyle().GetFrameSize()

	chatInnerWidth := safeSub(contentOuterWidth, chatFrameW)
	chatInnerHeight := safeSub(chatOuterHeight, chatFrameH)

	inputInnerWidth := safeSub(contentOuterWidth, inputFrameW)
	inputPromptWidth := lipgloss.Width(m.input.Prompt)

	m.chat.SetWidth(chatInnerWidth)
	m.chat.SetHeight(chatInnerHeight)

	m.input.SetWidth(safeSub(inputInnerWidth, inputPromptWidth+1))
}
