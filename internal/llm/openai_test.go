package llm

import (
	"encoding/json"
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

func TestToOpenAIMessages_assistantToolCallNoContent(t *testing.T) {
	msgs := []conversation.Message{
		{
			Role:    conversation.RoleAssistant,
			Content: "",
			ToolCalls: []conversation.ToolCall{
				{ID: "call_1", Name: "web_fetch", Arguments: `{"url":"https://example.com"}`},
			},
		},
		{
			Role:       conversation.RoleTool,
			Content:    "tool result",
			ToolCallID: "call_1",
		},
	}

	result := toOpenAIMessages(msgs)

	// assistant message with empty content but tool calls
	assistant := result[0].OfAssistant
	if assistant == nil {
		t.Fatal("expected assistant message")
	}
	j, err := json.Marshal(assistant)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(j, &raw); err != nil {
		t.Fatal(err)
	}

	// content must be present as string, not nil/missing
	if _, ok := raw["content"]; !ok {
		t.Fatalf("content field missing; got %s", j)
	}
	if raw["content"] != "" {
		t.Fatalf("expected empty content string, got %v", raw["content"])
	}
	if tc, ok := raw["tool_calls"]; !ok || tc == nil {
		t.Fatal("expected tool_calls")
	}

	// tool message
	toolMsg := result[1].OfTool
	if toolMsg == nil {
		t.Fatal("expected tool message")
	}
	if toolMsg.ToolCallID != "call_1" {
		t.Fatalf("expected tool_call_id call_1, got %q", toolMsg.ToolCallID)
	}

	// round-trip: unmarshal assistant back through openai types
	var ap openai.ChatCompletionAssistantMessageParam
	if err := json.Unmarshal(j, &ap); err != nil {
		t.Fatalf("unmarshal assistant message: %v", err)
	}
	if len(ap.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(ap.ToolCalls))
	}
	fn := ap.ToolCalls[0].GetFunction()
	if fn == nil || fn.Name != "web_fetch" {
		t.Fatalf("expected tool call web_fetch, got %+v", fn)
	}
	if string(ap.Content.OfString.Value) != "" {
		t.Fatalf("expected empty content, got %q", ap.Content.OfString.Value)
	}
	// Content.OfString must NOT be omitted (nil would be omitted, empty "" is present)
	if param.IsOmitted(ap.Content.OfString) {
		t.Fatal("content must be present string (not nil); Ollama rejects nil content")
	}
}
