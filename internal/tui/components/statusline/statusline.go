package statusline

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tui/theme"
)

// Height is the vertical space the footer occupies.
const Height = 1

// Model is the footer component: model/provider on the left, context usage
// and key hints on the right.
type Model struct {
	width int
}

// New creates an empty footer.
func New() Model {
	return Model{}
}

// SetSize sets the footer width.
func (m Model) SetSize(width int) Model {
	m.width = width
	return m
}

// View renders the footer line.
func (m Model) View(provider, model string, usage llm.ContextUsage, _ int) string {
	left := m.left(provider, model)
	right := m.right(usage)

	gap := safeSub(m.width, lipgloss.Width(left)+lipgloss.Width(right)+2)
	if gap == 0 && lipgloss.Width(left)+lipgloss.Width(right)+2 > m.width {
		// Not enough room for both: keep the hints, drop the model.
		left = ""
		gap = safeSub(m.width, lipgloss.Width(right)+2)
	}

	line := " " + left + strings.Repeat(" ", gap) + right + " "
	return theme.Base().Width(m.width).Height(Height).Render(line)
}

func (m Model) left(provider, model string) string {
	if model == "" && provider == "" {
		return muted("no model")
	}
	out := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).Render(orDash(model))
	if provider != "" {
		out += muted("  " + provider)
	}
	return out
}

func (m Model) right(usage llm.ContextUsage) string {
	hints := muted("↵ send") + dim("  ·  ") + muted("^C quit")
	if usage.Tokens <= 0 {
		return hints
	}
	ctx := formatTokens(usage.Tokens)
	if usage.ContextWindow > 0 {
		ctx += fmt.Sprintf(" (%.0f%%)", usage.Percent)
	}
	return dim(ctx) + dim("  ·  ") + hints
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func muted(text string) string {
	return lipgloss.NewStyle().Foreground(theme.ForegroundMuted).Render(text)
}

func dim(text string) string {
	return lipgloss.NewStyle().Foreground(theme.BorderNormal).Render(text)
}

func safeSub(a, b int) int {
	if a < b {
		return 0
	}
	return a - b
}
