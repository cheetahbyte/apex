package chat

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

// Model is the chat viewport component. It owns the raw output text,
// the scrollable viewport, and a buffer that accumulates the in-flight
// assistant response until the stream completes.
type Model struct {
	width  int
	height int

	viewport     viewport.Model
	output       string
	assistantBuf string
}

// New creates a chat component with an initial ready message.
func New() Model {
	output := "Apex ready."
	vp := viewport.New()
	m := Model{
		viewport: vp,
		output:   output,
	}
	m.refreshOutput()
	return m
}

// Update forwards messages to the viewport (scrolling, mouse, etc.).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the chat box with its border and padding.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	style := Style()
	bw, bh := borderSize(style)
	return style.
		Width(safeSub(m.width, bw)).
		Height(safeSub(m.height, bh)).
		Render(m.viewport.View())
}

// SetSize sets the outer box dimensions. The viewport content area is
// computed by subtracting the full frame (border + padding).
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	style := Style()
	fw, fh := style.GetFrameSize()
	m.viewport.SetWidth(safeSub(width, fw))
	m.viewport.SetHeight(safeSub(height, fh))
	m.refreshOutput()
}

// AppendUser appends user input to the raw output and rerenders markdown.
func (m *Model) AppendUser(text string) {
	m.output += "\n\n> " + text + "\n"
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// AppendAssistantChunk appends a streaming text chunk to the output and
// assistant buffer, then rerenders markdown.
func (m *Model) AppendAssistantChunk(chunk string) {
	m.output += chunk
	m.assistantBuf += chunk
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// CommitAssistant returns the accumulated assistant response and resets
// the buffer. Call this when the stream completes successfully.
func (m *Model) CommitAssistant() string {
	buf := m.assistantBuf
	m.assistantBuf = ""
	return buf
}

// DiscardAssistant drops the in-flight assistant buffer without adding
// it to the output. Call this when the stream errors.
func (m *Model) DiscardAssistant() {
	m.assistantBuf = ""
}

// AppendError adds an error line to the output and rerenders markdown.
func (m *Model) AppendError(text string) {
	m.output += "\n[error] " + text
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// AppendStatus adds a status line (e.g. tool execution) and rerenders markdown.
func (m *Model) AppendStatus(text string) {
	m.output += "\n" + text + "\n"
	m.refreshOutput()
	m.viewport.GotoBottom()
}

func (m *Model) refreshOutput() {
	width := m.viewport.Width()
	if width <= 0 {
		width = 80
	}

	rendered := m.output

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err == nil {
		if out, err := r.Render(m.output); err == nil {
			rendered = out
		}
		_ = r.Close()
	}

	m.viewport.SetContent(rendered)
}

// Style returns the chat box style. Border is added outside Width/Height,
// padding is counted inside.
func Style() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)
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
