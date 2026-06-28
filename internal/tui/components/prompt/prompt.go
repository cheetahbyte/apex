package prompt

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cheetahbyte/apex/internal/tui/theme"
)

// Model is the text input prompt component: a single clean line with an
// accent prompt glyph.
type Model struct {
	width  int
	height int
	input  textinput.Model
}

// New creates a focused prompt with a character limit.
func New() Model {
	input := textinput.New()
	input.Placeholder = ""
	input.Focus()
	input.CharLimit = 4000
	input.Prompt = ""
	input.SetStyles(inputStyles())
	return Model{input: input}
}

// Update forwards messages to the text input (typing, blinking, etc.).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the prompt line: accent glyph + input.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	glyph := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Render("❯ ")
	line := glyph + m.input.View()
	return theme.Base().Width(m.width).Render(" " + line)
}

// SetSize sets the available width. The text input width is the remaining
// space after the leading padding and prompt glyph.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(safeSub(width, 4)) // " " + "❯ " + a little slack
}

// Value returns the current input text.
func (m Model) Value() string {
	return m.input.Value()
}

// CharLimit returns the maximum number of characters the input accepts.
func (m Model) CharLimit() int {
	return m.input.CharLimit
}

// Reset clears the input text.
func (m *Model) Reset() {
	m.input.Reset()
}

// Blink returns the cursor blink command.
func Blink() tea.Cmd {
	return textinput.Blink
}

func inputStyles() textinput.Styles {
	return textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(theme.ForegroundEmphasis),
			Placeholder: lipgloss.NewStyle().Foreground(theme.ForegroundMuted).Faint(true),
			Suggestion:  lipgloss.NewStyle().Foreground(theme.ForegroundMuted),
			Prompt:      lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(theme.Foreground),
			Placeholder: lipgloss.NewStyle().Foreground(theme.ForegroundMuted).Faint(true),
			Suggestion:  lipgloss.NewStyle().Foreground(theme.ForegroundMuted),
			Prompt:      lipgloss.NewStyle().Foreground(theme.ForegroundMuted),
		},
		Cursor: textinput.CursorStyle{
			Color: theme.Primary,
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
}

func safeSub(a, b int) int {
	if a-b < 0 {
		return 0
	}
	return a - b
}
