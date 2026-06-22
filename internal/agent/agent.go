package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tools"
)

const (
	maxToolIterations = 5
	toolTimeout       = 30 * time.Second
	maxToolResultSize = 64 * 1024
)

// Event is a single event from the agent loop. The TUI consumes these
// to update the chat view.
type Event struct {
	Delta  string // text chunk from the model
	Status string // tool status line, e.g. "[tool] web_fetch https://..."
	Err    error
	Done   bool
}

// Agent orchestrates the model ↔ tool loop. It is provider-agnostic:
// all provider-specific translation happens inside llm.Client.
type Agent struct {
	client   llm.Client
	registry *tools.Registry
}

func New(client llm.Client, registry *tools.Registry) *Agent {
	return &Agent{client: client, registry: registry}
}

// Run starts the agent loop for the current session and returns a
// channel of events. The channel is closed after Done or Err.
func (a *Agent) Run(ctx context.Context, session *conversation.Session) <-chan Event {
	ch := make(chan Event)

	go func() {
		defer close(ch)

		for i := 0; i < maxToolIterations; i++ {
			req := llm.Request{
				Messages: session.Messages(),
				Tools:    a.registry.Specs(),
			}

			turn, err := a.streamTurn(ctx, ch, req)
			if err != nil {
				ch <- Event{Err: err}
				return
			}

			// Reject empty turns — don't poison the session with
			// a contentless, toolless assistant message.
			if turn.Content == "" && len(turn.ToolCalls) == 0 {
				ch <- Event{Err: fmt.Errorf("empty model response")}
				return
			}

			session.AppendMessage(conversation.Message{
				Role:      conversation.RoleAssistant,
				Content:   turn.Content,
				ToolCalls: turn.ToolCalls,
			})

			if len(turn.ToolCalls) == 0 {
				ch <- Event{Done: true}
				return
			}

			for _, call := range turn.ToolCalls {
				ch <- Event{Status: formatToolStatus(call)}
				result := a.executeTool(ctx, call)
				session.AppendMessage(conversation.Message{
					Role:       conversation.RoleTool,
					Content:    result,
					ToolCallID: call.ID,
				})
			}
		}

		ch <- Event{Err: fmt.Errorf("too many tool iterations")}
	}()

	return ch
}

// streamTurn consumes the LLM stream, forwarding deltas to the agent
// channel, and returns the final turn.
func (a *Agent) streamTurn(ctx context.Context, ch chan<- Event, req llm.Request) (llm.Turn, error) {
	for ev := range a.client.Stream(ctx, req) {
		if ev.Err != nil {
			return llm.Turn{}, ev.Err
		}
		if ev.Turn != nil {
			return *ev.Turn, nil
		}
		if ev.Delta != "" {
			ch <- Event{Delta: ev.Delta}
		}
	}
	return llm.Turn{}, fmt.Errorf("stream closed without turn")
}

// formatToolStatus creates a human-readable status line for a tool call.
func formatToolStatus(call conversation.ToolCall) string {
	var argsSummary string
	if strings.TrimSpace(call.Arguments) != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(call.Arguments), &parsed); err == nil {
			if url, ok := parsed["url"].(string); ok {
				argsSummary = url
			}
		}
	}
	if argsSummary == "" {
		return fmt.Sprintf("[tool] %s", call.Name)
	}
	return fmt.Sprintf("[tool] %s %s", call.Name, argsSummary)
}

// executeTool runs a single tool call and returns the result string.
// Errors are returned as tool result strings, not Go errors, so the
// model can see and recover from them.
func (a *Agent) executeTool(ctx context.Context, call conversation.ToolCall) string {
	tool, ok := a.registry.Get(call.Name)
	if !ok {
		return fmt.Sprintf("tool error: unknown tool %q", call.Name)
	}

	args := json.RawMessage("{}")
	if strings.TrimSpace(call.Arguments) != "" {
		args = json.RawMessage(call.Arguments)
		if !json.Valid(args) {
			return "tool error: invalid arguments"
		}
	}

	toolCtx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	result, err := tool.Execute(toolCtx, args)
	if err != nil {
		return fmt.Sprintf("tool error: %s", err)
	}
	if len(result.Content) > maxToolResultSize {
		result.Content = result.Content[:maxToolResultSize] + "\n\n[truncated]"
	}
	return result.Content
}
