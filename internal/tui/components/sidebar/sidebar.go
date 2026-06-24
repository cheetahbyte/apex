package sidebar

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/cheetahbyte/apex/internal/llm"
)

var (
	enumeratorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginRight(1)
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	cwdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Faint(true)
)

type Model struct {
	width  int // OUTER box width (border included)
	height int // OUTER box height
	mcps   []string
	cwd    string
	usage  llm.ContextUsage
}

func New() Model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "?"
	}
	return Model{
		mcps: make([]string, 0),
		cwd:  cwd,
	}
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m
}

func (m Model) SetCWD(cwd string) Model {
	m.cwd = cwd
	return m
}

func (m Model) SetContext(usage llm.ContextUsage) Model {
	m.usage = usage
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	m.mcps = []string{"Craft", "Linear", "abc"}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	style := boxStyle()

	bw, bh := borderSize(style)  // border only, added OUTSIDE Width/Height
	pw, ph := paddingSize(style) // padding only, counted INSIDE Width/Height

	// lipgloss Width/Height include padding but not the border, so the box
	// dimensions subtract the border only.
	boxW := safeSub(m.width, bw)
	boxH := safeSub(m.height, bh)

	// The actual text region is smaller still: subtract padding too. This is
	// what the body must fit into, and what truncation measures against.
	innerW := safeSub(boxW, pw)
	innerH := safeSub(boxH, ph)

	return style.
		Width(boxW).
		Height(boxH).
		Render(m.body(innerW, innerH))
}

func (m Model) body(innerW, innerH int) string {
	top := lipgloss.JoinVertical(lipgloss.Left, m.contextView(), "", m.mcpList())
	bottom := cwdStyle.Render(shortenPath(m.cwd, innerW))
	fill := safeSub(innerH, lipgloss.Height(top)+lipgloss.Height(bottom))
	parts := []string{top}
	if fill > 0 {
		parts = append(parts, lipgloss.NewStyle().Height(fill).Render(""))
	}
	parts = append(parts, bottom)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) contextView() string {
	if m.usage.Tokens == 0 {
		return itemStyle.Render("Context\n—")
	}
	used := formatTokens(m.usage.Tokens)
	if m.usage.ContextWindow <= 0 {
		return itemStyle.Render(fmt.Sprintf("Context\n%s / ?", used))
	}
	total := formatTokens(m.usage.ContextWindow)
	return itemStyle.Render(fmt.Sprintf("Context\n%s / %s\n%.1f%%", used, total, m.usage.Percent))
}

func formatTokens(n int) string {
	switch {
	case n >= 1000000:
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (m Model) mcpList() string {
	if len(m.mcps) == 0 {
		return "No MCPs connected"
	}
	l := list.New().
		Enumerator(list.Dash).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)
	for _, mcp := range m.mcps {
		l = l.Item(mcp)
	}
	return l.String()
}

func boxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2) // vertical 1, horizontal 2
}

// borderSize is added OUTSIDE the style's Width/Height.
func borderSize(s lipgloss.Style) (w, h int) {
	return s.GetHorizontalBorderSize(), s.GetVerticalBorderSize()
}

// paddingSize is counted INSIDE the style's Width/Height.
func paddingSize(s lipgloss.Style) (w, h int) {
	return s.GetHorizontalPadding(), s.GetVerticalPadding()
}

func safeSub(a, b int) int {
	if a < b {
		return 0
	}
	return a - b
}

func shortenPath(p string, max int) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	r := []rune(p)
	if len(r) <= max {
		return p
	}
	if max <= 1 {
		return string(r[len(r)-max:])
	}
	return "…" + string(r[len(r)-(max-1):])
}
