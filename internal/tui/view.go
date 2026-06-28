package tui

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cheetahbyte/apex/internal/tui/theme"
)

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		rule(m.width),
		m.chat.View(),
		rule(m.width),
		m.prompt.View(),
		m.status.View(
			m.runtime.Provider,
			m.runtime.Model,
			m.usage,
			m.width,
		),
	)

	rendered := theme.Base().
		Width(m.width).
		Height(m.height).
		Render(body)

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}

// header renders the top bar: the app mark on the left, the working
// directory on the right.
func (m Model) header() string {
	left := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Render("✳ apex")

	right := lipgloss.NewStyle().
		Foreground(theme.ForegroundMuted).
		Render(shortenPath(m.cwd, safeSub(m.width, lipgloss.Width(left)+4)))

	gap := safeSub(m.width, lipgloss.Width(left)+lipgloss.Width(right)+2)
	line := " " + left + strings.Repeat(" ", gap) + right + " "
	return theme.Base().Width(m.width).Render(line)
}

func rule(width int) string {
	return lipgloss.NewStyle().
		Foreground(theme.BorderDim).
		Render(strings.Repeat("─", max(width, 0)))
}

func shortenPath(p string, max int) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	r := []rune(p)
	if max <= 0 {
		return ""
	}
	if len(r) <= max {
		return p
	}
	if max <= 1 {
		return string(r[len(r)-max:])
	}
	return "…" + string(r[len(r)-(max-1):])
}
