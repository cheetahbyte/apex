package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar = m.sidebar.SetSize(sidebarOuterWidth, msg.Height)
		m.resize()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if m.streaming {
				return m, nil
			}
			text := strings.TrimSpace(m.prompt.Value())
			if text == "" {
				return m, nil
			}
			m.prompt.Reset()
			m.streaming = true
			m.session.AppendUser(text)
			m.chat.AppendUser(text)
			return m, tea.Batch(m.spawnStream(), waitForChunk(m.chunks))
		}

	case streamChunkMsg:
		m.chat.AppendAssistantChunk(string(msg))
		return m, waitForChunk(m.chunks)

	case statusMsg:
		m.chat.AppendStatus(string(msg))
		return m, waitForChunk(m.chunks)

	case streamDoneMsg:
		m.streaming = false
		m.chat.CommitAssistant()
		return m, nil

	case errMsg:
		m.streaming = false
		m.chat.DiscardAssistant()
		m.chat.AppendError(msg.err.Error())
		return m, nil
	}

	var cmd tea.Cmd

	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	m.prompt, cmd = m.prompt.Update(msg)
	cmds = append(cmds, cmd)

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
