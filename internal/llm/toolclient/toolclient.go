// Package toolclient provides a unified tool-calling layer that wraps any
// llm.Client. It handles all tool protocol decisions — native API, text
// JSON prompt, automatic fallback, or disabled — behind one clean
// interface. The agent and TUI never need to know which protocol is in
// use.
//
// Usage:
//
//	base := llm.NewOpenAIClient(model, baseURL, apiKey)
//	client := toolclient.New(base, toolclient.ModeAuto)
//	agent := agent.New(client, registry)
package toolclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
)

// Mode controls how tools are exposed to the model.
type Mode string

const (
	// ModeAuto tries native tool calling first, falls back to text JSON
	// protocol if the provider returns an empty response. This is the
	// recommended default — works with OpenAI, Anthropic, Ollama, LM
	// Studio, and any provider that can follow text instructions.
	ModeAuto Mode = "auto"

	// ModeNative uses the provider's native tool-calling API only. No
	// fallback. Best for providers known to support tools well.
	ModeNative Mode = "native"

	// ModeText injects tool instructions as a system prompt and parses
	// JSON tool calls from the model's text output. Works with any
	// provider that can follow instructions.
	ModeText Mode = "text"

	// ModeNone disables tools entirely.
	ModeNone Mode = "none"
)

// ModeFromString parses a mode string, defaulting to ModeAuto.
func ModeFromString(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "native":
		return ModeNative
	case "text":
		return ModeText
	case "none":
		return ModeNone
	default:
		return ModeAuto
	}
}

// Client wraps an inner llm.Client with unified tool-calling support.
type Client struct {
	inner llm.Client
	mode  Mode
}

// New wraps inner with tool-calling support governed by mode.
func New(inner llm.Client, mode Mode) *Client {
	return &Client{inner: inner, mode: mode}
}

// Capabilities reports what this toolclient exposes to the agent.
// It always claims native tools so the agent sends tool specs — the
// toolclient handles translation internally.
func (c *Client) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		NativeTools:        true,
		StreamingToolCalls: c.inner.Capabilities().StreamingToolCalls,
	}
}

// Stream routes the request through the selected tool mode.
func (c *Client) Stream(ctx context.Context, req llm.Request) <-chan llm.StreamEvent {
	if len(req.Tools) == 0 || c.mode == ModeNone {
		return c.inner.Stream(ctx, llm.Request{
			Messages: req.Messages,
			Tools:    nil,
		})
	}

	switch c.mode {
	case ModeNative:
		return c.inner.Stream(ctx, req)
	case ModeText:
		return c.textStream(ctx, req)
	default: // ModeAuto
		return c.autoStream(ctx, req)
	}
}

// autoStream tries native tool calling first. If the provider returns
// an empty turn (no content, no tool calls), it falls back to the text
// JSON protocol. This handles providers that claim native tool support
// but don't actually work with it (common with local models).
func (c *Client) autoStream(ctx context.Context, req llm.Request) <-chan llm.StreamEvent {
	if !c.inner.Capabilities().NativeTools {
		return c.textStream(ctx, req)
	}

	ch := make(chan llm.StreamEvent)
	go func() {
		defer close(ch)

		// Try native first.
		var turn *llm.Turn
		for ev := range c.inner.Stream(ctx, req) {
			if ev.Err != nil {
				ch <- ev
				return
			}
			if ev.Turn != nil {
				turn = ev.Turn
				break
			}
			if ev.Delta != "" {
				ch <- ev
			}
		}

		// If native gave us content or tool calls, use it.
		if turn != nil && (turn.Content != "" || len(turn.ToolCalls) > 0) {
			ch <- llm.StreamEvent{Turn: turn}
			return
		}

		// Native returned empty — fall back to text protocol.
		c.forwardTextStream(ctx, ch, req)
	}()
	return ch
}

// textStream injects tool instructions as a system prompt, sends the
// request without the tools field, and parses the response for JSON
// tool calls.
func (c *Client) textStream(ctx context.Context, req llm.Request) <-chan llm.StreamEvent {
	ch := make(chan llm.StreamEvent)
	go func() {
		defer close(ch)
		c.forwardTextStream(ctx, ch, req)
	}()
	return ch
}

// forwardTextStream is the shared text-protocol implementation used by
// both ModeText and ModeAuto fallback.
func (c *Client) forwardTextStream(ctx context.Context, ch chan<- llm.StreamEvent, req llm.Request) {
	textReq := llm.Request{
		Messages: injectToolPrompt(req.Messages, buildToolPrompt(req.Tools)),
		Tools:    nil,
	}

	var content strings.Builder
	var innerTurn *llm.Turn

	for ev := range c.inner.Stream(ctx, textReq) {
		if ev.Err != nil {
			ch <- ev
			return
		}
		if ev.Turn != nil {
			innerTurn = ev.Turn
			break
		}
		if ev.Delta != "" {
			content.WriteString(ev.Delta)
		}
	}

	if innerTurn == nil {
		ch <- llm.StreamEvent{Err: fmt.Errorf("stream closed without turn")}
		return
	}

	fullContent := innerTurn.Content
	if fullContent == "" {
		fullContent = content.String()
	}

	// Try to parse tool calls from the response.
	if calls := ParseToolCalls(fullContent, req.Tools); len(calls) > 0 {
		ch <- llm.StreamEvent{Turn: &llm.Turn{
			Content:   "",
			ToolCalls: calls,
		}}
		return
	}

	// Not a tool call — pass through as normal text.
	if fullContent != "" {
		ch <- llm.StreamEvent{Delta: fullContent}
	}
	ch <- llm.StreamEvent{Turn: &llm.Turn{
		Content: fullContent,
	}}
}

// buildToolPrompt creates the system prompt instructing the model how
// to request tool calls via JSON.
func buildToolPrompt(specs []map[string]any) string {
	var b strings.Builder
	b.WriteString("You have access to tools. To call a tool, respond with ONLY a JSON object in this exact format, no other text:\n")
	b.WriteString(`{"tool_calls":[{"id":"call_1","name":"<tool_name>","arguments":{<args>}}]}`)
	b.WriteString("\n\nIf you don't need a tool, respond normally with text.\n\nAvailable tools:\n")
	for _, spec := range specs {
		name, _ := spec["name"].(string)
		desc, _ := spec["description"].(string)
		schema, _ := spec["schema"].(map[string]any)
		b.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
		if schema != nil {
			schemaJSON, _ := json.Marshal(schema)
			b.WriteString(fmt.Sprintf("  args schema: %s\n", string(schemaJSON)))
		}
	}
	return b.String()
}

// injectToolPrompt merges the tool prompt into the message list. If a
// system message already exists, the tool prompt is appended to it.
// Otherwise a new system message is prepended.
func injectToolPrompt(messages []conversation.Message, prompt string) []conversation.Message {
	out := make([]conversation.Message, 0, len(messages)+1)
	injected := false
	for _, msg := range messages {
		if msg.Role == conversation.RoleSystem && !injected {
			out = append(out, conversation.Message{
				Role:    conversation.RoleSystem,
				Content: msg.Content + "\n\n" + prompt,
			})
			injected = true
		} else {
			out = append(out, msg)
		}
	}
	if !injected {
		out = append([]conversation.Message{{Role: conversation.RoleSystem, Content: prompt}}, out...)
	}
	return out
}
