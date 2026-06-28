package chat

import (
	"strings"
	"testing"
)

func TestChatSeparatesUserAndAssistant(t *testing.T) {
	m := New()
	m.AppendUser("Lets work!")
	m.AppendAssistantChunk("Test received.")
	m.CommitAssistant()

	if len(m.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(m.blocks))
	}
	if m.blocks[0].kind != kindUser || m.blocks[0].text != "Lets work!" {
		t.Fatalf("unexpected user block: %+v", m.blocks[0])
	}
	if m.blocks[1].kind != kindAssistant || m.blocks[1].text != "Test received." {
		t.Fatalf("unexpected assistant block: %+v", m.blocks[1])
	}
}

func TestChatTracksStatusAndAssistant(t *testing.T) {
	m := New()
	m.AppendStatus("[tool] dir_tree .")
	m.AppendAssistantChunk("Here is the tree:")
	m.CommitAssistant()

	if len(m.blocks) != 2 || m.blocks[0].kind != kindStatus || m.blocks[1].kind != kindAssistant {
		t.Fatalf("unexpected blocks: %+v", m.blocks)
	}
}

func TestChatStartsWithBlankState(t *testing.T) {
	m := New()
	m.SetSize(60, 12)

	// Blank state paints background but shows no message content.
	if len(m.blocks) != 0 {
		t.Fatalf("expected no blocks, got %d", len(m.blocks))
	}
	if strings.Contains(m.viewport.View(), "▌") {
		t.Fatalf("expected no role labels in blank state, got %q", m.viewport.View())
	}
}

func TestMarkdownStylePrefersApexEnv(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "dark")
	t.Setenv("APEX_MARKDOWN_STYLE", "light")

	if got := markdownStyle(); got != "light" {
		t.Fatalf("expected APEX_MARKDOWN_STYLE to win, got %q", got)
	}
}

func TestMarkdownStyleFallsBackToTheme(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "")
	t.Setenv("APEX_MARKDOWN_STYLE", "")

	if got := markdownStyle(); got != "" {
		t.Fatalf("expected built-in theme fallback, got %q", got)
	}
}
