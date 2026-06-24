package chat

import (
	"strings"
	"testing"
)

func TestChatOutputSeparatesUserAndAssistant(t *testing.T) {
	m := New()
	m.AppendUser("Lets work!")
	m.AppendAssistantChunk("Test received.")
	m.CommitAssistant()

	if !strings.Contains(m.output, "**You:** Lets work!\n\n**Apex:** Test received.\n\n") {
		t.Fatalf("unexpected output spacing: %q", m.output)
	}
}

func TestChatOutputSeparatesStatusAndAssistant(t *testing.T) {
	m := New()
	m.AppendStatus("[tool] dir_tree .")
	m.AppendAssistantChunk("Here is the tree:")
	m.CommitAssistant()

	if !strings.Contains(m.output, "[tool] dir_tree .\n\n**Apex:** Here is the tree:\n\n") {
		t.Fatalf("unexpected status spacing: %q", m.output)
	}
}

func TestMarkdownStylePrefersApexEnv(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "dark")
	t.Setenv("APEX_MARKDOWN_STYLE", "light")

	if got := markdownStyle(); got != "light" {
		t.Fatalf("expected APEX_MARKDOWN_STYLE to win, got %q", got)
	}
}

func TestMarkdownStyleFallsBackToLight(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "")
	t.Setenv("APEX_MARKDOWN_STYLE", "")

	if got := markdownStyle(); got != "light" {
		t.Fatalf("expected light fallback, got %q", got)
	}
}
