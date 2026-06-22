package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tools"
)

func newTestRegistry(tool tools.Tool) *tools.Registry {
	r := tools.NewRegistry()
	r.Register(tool)
	return r
}

func TestAgent_normalTextResponse(t *testing.T) {
	client := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "Hello from the model"}},
	}
	session := conversation.NewSession()
	session.AppendUser("hi")
	agent := New(client, newTestRegistry(mockTool{name: "noop", result: ""}))

	var deltas []string
	var done bool
	var errEvent error
	for ev := range agent.Run(context.Background(), session) {
		if ev.Err != nil {
			errEvent = ev.Err
		}
		if ev.Done {
			done = true
		}
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
	}

	if errEvent != nil {
		t.Fatalf("unexpected error: %v", errEvent)
	}
	if !done {
		t.Fatal("expected Done event")
	}
	if len(deltas) != 1 || deltas[0] != "Hello from the model" {
		t.Fatalf("expected delta 'Hello from the model', got %v", deltas)
	}
	if session.Len() != 2 {
		t.Fatalf("expected 2 messages (user+assistant), got %d", session.Len())
	}
}

func TestAgent_emptyResponseReturnsError(t *testing.T) {
	client := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "", ToolCalls: nil}},
	}
	session := conversation.NewSession()
	session.AppendUser("hi")
	agent := New(client, newTestRegistry(mockTool{name: "noop", result: ""}))

	var errEvent error
	for ev := range agent.Run(context.Background(), session) {
		if ev.Err != nil {
			errEvent = ev.Err
		}
	}

	if errEvent == nil {
		t.Fatal("expected error for empty response")
	}
	if !strings.Contains(errEvent.Error(), "empty model response") {
		t.Fatalf("expected 'empty model response' error, got %v", errEvent)
	}
	// Session must not be poisoned with empty assistant message
	if session.Len() != 1 {
		t.Fatalf("expected 1 message (user only), got %d — session was poisoned", session.Len())
	}
}

func TestAgent_toolCallLoop(t *testing.T) {
	client := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{
			{Content: "", ToolCalls: []conversation.ToolCall{
				{ID: "call_1", Name: "mock", Arguments: `{}`},
			}},
			{Content: "Here is the result"},
		},
	}
	session := conversation.NewSession()
	session.AppendUser("use the tool")
	agent := New(client, newTestRegistry(mockTool{name: "mock", result: "tool output"}))

	var statuses []string
	var deltas []string
	var done bool
	for ev := range agent.Run(context.Background(), session) {
		if ev.Status != "" {
			statuses = append(statuses, ev.Status)
		}
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
		if ev.Done {
			done = true
		}
	}

	if !done {
		t.Fatal("expected Done")
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status event, got %v", statuses)
	}
	if !strings.Contains(statuses[0], "mock") {
		t.Fatalf("expected status to contain tool name, got %q", statuses[0])
	}
	if len(deltas) != 1 || deltas[0] != "Here is the result" {
		t.Fatalf("expected final delta 'Here is the result', got %v", deltas)
	}
	// user + assistant(tool_call) + tool(result) + assistant(text) = 4
	if session.Len() != 4 {
		t.Fatalf("expected 4 messages, got %d", session.Len())
	}
}

func TestAgent_unknownToolReturnsToolError(t *testing.T) {
	client := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{
			{Content: "", ToolCalls: []conversation.ToolCall{
				{ID: "call_1", Name: "nonexistent", Arguments: `{}`},
			}},
			{Content: "OK"},
		},
	}
	session := conversation.NewSession()
	session.AppendUser("use unknown tool")
	agent := New(client, newTestRegistry(mockTool{name: "mock", result: ""}))

	var done bool
	var errEvent error
	for ev := range agent.Run(context.Background(), session) {
		if ev.Err != nil {
			errEvent = ev.Err
		}
		if ev.Done {
			done = true
		}
	}

	if errEvent != nil {
		t.Fatalf("unexpected error: %v", errEvent)
	}
	if !done {
		t.Fatal("expected Done — unknown tool should be a tool result error, not a fatal error")
	}
	// Check tool result message contains error text
	msgs := session.Messages()
	toolMsg := msgs[2] // user, assistant, tool
	if toolMsg.Role != conversation.RoleTool {
		t.Fatalf("expected tool message at index 2, got role %s", toolMsg.Role)
	}
	if !strings.Contains(toolMsg.Content, "unknown tool") {
		t.Fatalf("expected 'unknown tool' in tool result, got %q", toolMsg.Content)
	}
}

func TestAgent_streamError(t *testing.T) {
	client := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{}, // no turns → error
	}
	session := conversation.NewSession()
	session.AppendUser("hi")
	agent := New(client, newTestRegistry(mockTool{name: "noop", result: ""}))

	var errEvent error
	for ev := range agent.Run(context.Background(), session) {
		if ev.Err != nil {
			errEvent = ev.Err
		}
	}

	if errEvent == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(errEvent, errEvent) && errEvent.Error() == "" {
		t.Fatal("error should have message")
	}
	// Session should not be poisoned
	if session.Len() != 1 {
		t.Fatalf("expected 1 message (user only), got %d", session.Len())
	}
}
