package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tools"
)

// mockClient is a test double for llm.Client. It returns pre-programmed
// turns in sequence, one per Stream call.
type mockClient struct {
	caps     llm.Capabilities
	turns    []llm.Turn
	callIdx  int
	requests []llm.Request
}

func (m *mockClient) Capabilities() llm.Capabilities { return m.caps }

func (m *mockClient) Stream(ctx context.Context, req llm.Request) <-chan llm.StreamEvent {
	m.requests = append(m.requests, req)
	ch := make(chan llm.StreamEvent)
	go func() {
		defer close(ch)
		if m.callIdx >= len(m.turns) {
			ch <- llm.StreamEvent{Err: fmt.Errorf("no more mock turns")}
			return
		}
		turn := m.turns[m.callIdx]
		m.callIdx++
		if turn.Content != "" {
			ch <- llm.StreamEvent{Delta: turn.Content}
		}
		ch <- llm.StreamEvent{Turn: &turn}
	}()
	return ch
}

// mockTool is a test double for tools.Tool.
type mockTool struct {
	name    string
	result  string
	execErr error
}

func (t mockTool) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        t.name,
		Description: "mock tool",
		Parameters:  map[string]any{"type": "object"},
	}
}

func (t mockTool) Execute(ctx context.Context, input json.RawMessage) (tools.ToolResult, error) {
	if t.execErr != nil {
		return tools.ToolResult{}, t.execErr
	}
	return tools.ToolResult{Content: t.result}, nil
}
