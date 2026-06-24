package sidebar

import (
	"strings"
	"testing"

	"github.com/cheetahbyte/apex/internal/llm"
)

func TestContextViewShowsPercent(t *testing.T) {
	m := New().SetContext(llm.ContextUsage{Tokens: 12500, ContextWindow: 400000, Percent: 3.125, Estimated: true})

	view := m.contextView()

	for _, want := range []string{"Context", "12.5k / 400.0k", "3.1%"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in view %q", want, view)
		}
	}
}

func TestContextViewUnknownWindow(t *testing.T) {
	m := New().SetContext(llm.ContextUsage{Tokens: 900, Estimated: true})

	view := m.contextView()

	if !strings.Contains(view, "900 / ?") {
		t.Fatalf("expected unknown context window, got %q", view)
	}
}
