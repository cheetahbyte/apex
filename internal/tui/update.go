package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func SyncChat(m Model, message string) {
	m.output += string(message)
	m.chat.SetContent(m.output)
	m.chat.GotoBottom()
}

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
			prompt := strings.TrimSpace(m.input.Value())
			if prompt == "" {
				return m, nil
			}
			m.input.Reset()
			m.streaming = true
			m.output += "\n\n>" + prompt + "\n"
			m.chat.SetContent(m.output)
			m.chat.GotoBottom()
			return m, tea.Batch(m.spawnStream(prompt), waitForChunk(m.chunks))
		}
	case streamChunkMsg:
		SyncChat(m, string(msg))
		return m, waitForChunk(m.chunks)
	case streamDoneMsg:
		m.streaming = false
		return m, nil
	case errMsg:
		m.streaming = false
		m.output += "\n[error] " + msg.err.Error()
		m.chat.SetContent(m.output)
		m.chat.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
