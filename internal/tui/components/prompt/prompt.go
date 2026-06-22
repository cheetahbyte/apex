package prompt

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model is the text input prompt component. It wraps a textinput widget
// and renders it inside a bordered box.
type Model struct {
	width  int
	height int
	input  textinput.Model
}

// New creates a focused prompt with a placeholder and character limit.
func New() Model {
	input := textinput.New()
	input.Placeholder = "Lets work!"
	input.Focus()
	input.CharLimit = 4000
	input.Prompt = "> "
	return Model{input: input}
}

// Update forwards messages to the text input (typing, blinking, etc.).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the prompt box with its border and padding.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	style := Style()
	bw, bh := borderSize(style)
	return style.
		Width(safeSub(m.width, bw)).
		Height(safeSub(m.height, bh)).
		Render(m.input.View())
}

// SetSize sets the outer box dimensions. The text input width is
// computed by subtracting the frame and the prompt string width.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	style := Style()
	fw, _ := style.GetFrameSize()
	promptWidth := lipgloss.Width(m.input.Prompt)
	m.input.SetWidth(safeSub(safeSub(width, fw), promptWidth+1))
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

// Blink returns the cursor blink command. The root model should include
// this in its Init so the prompt cursor blinks on startup.
func Blink() tea.Cmd {
	return textinput.Blink
}

// Style returns the prompt box style.
func Style() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
}

func borderSize(s lipgloss.Style) (w, h int) {
	return s.GetBorderLeftSize() + s.GetBorderRightSize(),
		s.GetBorderTopSize() + s.GetBorderBottomSize()
}

func safeSub(a, b int) int {
	if a-b < 0 {
		return 0
	}
	return a - b
}
