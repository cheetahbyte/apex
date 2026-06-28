package statusline

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/cheetahbyte/apex/internal/llm"
)

func TestFooterShowsModelAndHints(t *testing.T) {
	m := New().SetSize(80)

	view := m.View("codex", "gpt-5.5", llm.ContextUsage{}, 80)

	for _, want := range []string{"gpt-5.5", "codex", "send", "quit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in view %q", want, view)
		}
	}
}

func TestFooterShowsContextUsage(t *testing.T) {
	m := New().SetSize(80)

	view := m.View("codex", "gpt-5.5", llm.ContextUsage{Tokens: 18600, ContextWindow: 400000, Percent: 5}, 80)

	if !strings.Contains(view, "18.6K") {
		t.Fatalf("expected token count in view %q", view)
	}
}

func TestFooterFitsWidth(t *testing.T) {
	for _, w := range []int{80, 24, 1} {
		m := New().SetSize(w)
		view := m.View("openai", "a-very-long-model-name", llm.ContextUsage{}, w)
		if got := lipgloss.Width(view); got > w {
			t.Fatalf("width %d: footer width %d exceeds %d in %q", w, got, w, view)
		}
	}
}
