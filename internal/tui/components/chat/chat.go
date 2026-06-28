package chat

import (
	"image/color"
	"os"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/cheetahbyte/apex/internal/tui/theme"
)

type blockKind int

const (
	kindUser blockKind = iota
	kindAssistant
	kindStatus
	kindError
)

type block struct {
	kind blockKind
	text string
}

// Model is the chat viewport component. It owns the list of conversation
// blocks (user, assistant, status, error), the scrollable viewport, and
// tracks whether an assistant block is currently streaming.
type Model struct {
	width  int
	height int

	viewport  viewport.Model
	blocks    []block
	streaming bool   // an assistant block is open and accumulating chunks
	model     string // shown in the reply meta footer
}

// New creates an empty chat component.
func New() Model {
	vp := viewport.New()
	return Model{viewport: vp}
}

// SetModel sets the model name shown under assistant replies.
func (m *Model) SetModel(model string) {
	m.model = model
}

// Update forwards messages to the viewport (scrolling, mouse, etc.).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the chat box (borderless, left gutter).
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	// Padding is counted INSIDE Width/Height, so pass the full outer size:
	// content area = outer − frame, which matches the viewport size set in
	// SetSize. Subtracting the frame here would double-count it and re-wrap.
	return Style().
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}

// SetSize sets the outer box dimensions. The viewport content area is
// computed by subtracting the full frame (padding).
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	style := Style()
	fw, fh := style.GetFrameSize()
	m.viewport.SetWidth(safeSub(width, fw))
	m.viewport.SetHeight(safeSub(height, fh))
	m.refreshOutput()
}

// AppendUser adds a user message block.
func (m *Model) AppendUser(text string) {
	m.blocks = append(m.blocks, block{kind: kindUser, text: text})
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// AppendAssistantChunk appends a streaming chunk to the open assistant block,
// opening a new one if needed.
func (m *Model) AppendAssistantChunk(chunk string) {
	if !m.streaming {
		m.blocks = append(m.blocks, block{kind: kindAssistant})
		m.streaming = true
	}
	m.blocks[len(m.blocks)-1].text += chunk
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// CommitAssistant closes the open assistant block and returns its text.
func (m *Model) CommitAssistant() string {
	if !m.streaming {
		return ""
	}
	m.streaming = false
	text := m.blocks[len(m.blocks)-1].text
	m.refreshOutput()
	m.viewport.GotoBottom()
	return text
}

// DiscardAssistant drops the open assistant block (e.g. on stream error).
func (m *Model) DiscardAssistant() {
	if m.streaming {
		m.blocks = m.blocks[:len(m.blocks)-1]
		m.streaming = false
	}
}

// AppendError adds an error block.
func (m *Model) AppendError(text string) {
	m.blocks = append(m.blocks, block{kind: kindError, text: text})
	m.refreshOutput()
	m.viewport.GotoBottom()
}

// AppendStatus adds a status block (e.g. tool execution).
func (m *Model) AppendStatus(text string) {
	m.blocks = append(m.blocks, block{kind: kindStatus, text: text})
	m.refreshOutput()
	m.viewport.GotoBottom()
}

func (m *Model) refreshOutput() {
	width := m.viewport.Width()
	if width <= 0 {
		width = 80
	}
	if len(m.blocks) == 0 {
		m.viewport.SetContent("")
		return
	}

	rendered := make([]string, 0, len(m.blocks))
	for _, b := range m.blocks {
		if strings.TrimSpace(b.text) == "" && b.kind != kindAssistant {
			continue
		}
		rendered = append(rendered, m.renderBlock(b, width))
	}
	m.viewport.SetContent(strings.Join(rendered, "\n\n"))
}

func (m *Model) renderBlock(b block, width int) string {
	switch b.kind {
	case kindUser:
		return userBlock(b.text, width)
	case kindStatus:
		return indent(statusStyle().Render("↳ "+cleanStatus(b.text)), 2)
	case kindError:
		return indent(errorStyle().Render("✕ "+b.text), 2)
	default:
		return m.assistantBlock(b.text, width)
	}
}

// userBlock renders a user message with a left accent bar spanning every
// line of the message and no label.
func userBlock(text string, width int) string {
	body := lipgloss.NewStyle().
		Foreground(theme.ForegroundEmphasis).
		Width(safeSub(width, 2)).
		Render(text)
	return barred(body, theme.Success)
}

// assistantBlock renders a reply with a dim left bar spanning the whole
// message, followed by a meta footer: ▣ apex · model.
func (m *Model) assistantBlock(text string, width int) string {
	body := barred(assistantText(text, safeSub(width, 2)), theme.BorderNormal)
	return body + "\n\n" + m.replyMeta()
}

// replyMeta is the small footer under a reply: a marker, the agent name and
// the model.
func (m *Model) replyMeta() string {
	marker := lipgloss.NewStyle().Foreground(theme.Primary).Render("▣")
	label := lipgloss.NewStyle().Foreground(theme.ForegroundEmphasis).Bold(true).Render("apex")
	out := marker + " " + label
	if m.model != "" {
		out += lipgloss.NewStyle().Foreground(theme.ForegroundMuted).Render(" · "+m.model)
	}
	return indent(out, 2)
}

// barred prefixes every line with a colored vertical bar.
func barred(text string, c color.Color) string {
	bar := lipgloss.NewStyle().Foreground(c).Render("┃ ")
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = bar + l
	}
	return strings.Join(lines, "\n")
}

// indent prefixes every line with n spaces.
func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

// cleanStatus strips the "[tool] " prefix that tool events carry so the
// footer marker reads cleanly.
func cleanStatus(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, "[tool]"))
}

func assistantText(text string, width int) string {
	r, err := markdownRenderer(width)
	if err != nil {
		return text
	}
	defer r.Close()
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return dedentOne(strings.TrimRight(out, "\n"))
}

// leadingIndent matches one leading space that sits after any ANSI escape
// codes at the start of a line.
var leadingIndent = regexp.MustCompile(`^((?:\x1b\[[0-9;]*m)*) `)

// dedentOne removes a single leading space from each line (the one-column
// indent glamour adds to paragraphs), keeping the body flush against the
// bar+space gutter so it lines up with the user message.
func dedentOne(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = leadingIndent.ReplaceAllString(l, "$1")
	}
	return strings.Join(lines, "\n")
}

func markdownRenderer(width int) (*glamour.TermRenderer, error) {
	if style := markdownStyle(); style != "" {
		return glamour.NewTermRenderer(
			glamour.WithStylePath(style),
			glamour.WithWordWrap(width),
		)
	}

	return glamour.NewTermRenderer(
		glamour.WithStyles(theme.MarkdownStyle()),
		glamour.WithWordWrap(width),
	)
}

func markdownStyle() string {
	if style := strings.TrimSpace(os.Getenv("APEX_MARKDOWN_STYLE")); style != "" {
		return style
	}
	if style := strings.TrimSpace(os.Getenv("GLAMOUR_STYLE")); style != "" {
		return style
	}
	return ""
}

// Style returns the chat box style: borderless, full-bleed with a left
// gutter so messages breathe.
func Style() lipgloss.Style {
	return theme.Base().
		Padding(1, 2)
}

func statusStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ForegroundMuted)
}

func errorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Error).Bold(true)
}

func safeSub(a, b int) int {
	if a-b < 0 {
		return 0
	}
	return a - b
}
