package llm

import (
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
)

func TestEstimateContextUsageIncludesMessagesAndTools(t *testing.T) {
	req := Request{
		Messages: []conversation.Message{
			{Role: conversation.RoleSystem, Content: "You are Apex."},
			{Role: conversation.RoleUser, Content: "Read README.md"},
			{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCall{{ID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`}}},
			{Role: conversation.RoleTool, ToolCallID: "call_1", Content: "file contents"},
		},
		Tools: []map[string]any{{"name": "read_file", "schema": map[string]any{"type": "object"}}},
	}

	usage := EstimateContextUsage(req, 1000)

	if usage.Tokens <= 0 {
		t.Fatalf("expected positive token estimate, got %d", usage.Tokens)
	}
	if usage.ContextWindow != 1000 {
		t.Fatalf("expected context window 1000, got %d", usage.ContextWindow)
	}
	if usage.Percent <= 0 {
		t.Fatalf("expected positive percent, got %f", usage.Percent)
	}
	if !usage.Estimated {
		t.Fatal("expected estimate flag")
	}
}
