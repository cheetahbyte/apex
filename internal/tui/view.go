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

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.chat.View(),
		m.prompt.View(),
		m.status.View(
			m.session.Len(),
			len(m.prompt.Value()),
			m.prompt.CharLimit(),
			m.width,
			m.height,
		),
	)
	rendered := lipgloss.JoinHorizontal(lipgloss.Top, content, m.sidebar.View())

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}
