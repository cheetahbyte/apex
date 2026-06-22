package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	contentOuterWidth := safeSub(m.width, sidebarOuterWidth)
	chatOuterHeight := safeSub(m.height, inputOuterHeight+statusHeight)

	chat := chatStyle()
	input := inputStyle()
	chatBorderW, chatBorderH := borderSize(chat)
	inputBorderW, inputBorderH := borderSize(input)

	chatBox := chat.
		Width(safeSub(contentOuterWidth, chatBorderW)).
		Height(safeSub(chatOuterHeight, chatBorderH)).
		Render(m.chat.View())
	inputBox := input.
		Width(safeSub(contentOuterWidth, inputBorderW)).
		Height(safeSub(inputOuterHeight, inputBorderH)).
		Render(m.input.View())
	statusLine := m.renderStatusLine(contentOuterWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, chatBox, inputBox, statusLine)
	rendered := lipgloss.JoinHorizontal(lipgloss.Top, content, m.sidebar.View())

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}
