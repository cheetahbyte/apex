package llm

import (
	"context"

	"github.com/cheetahbyte/apex/internal/conversation"
)

// Capabilities describes what a provider supports. The agent uses this
// to decide how to orchestrate tool calls without being coupled to any
// specific provider.
type Capabilities struct {
	NativeTools        bool // provider has a native tool-calling API
	StreamingToolCalls bool // provider streams tool calls incrementally
	RequiresToolPrompt bool // provider needs tool instructions injected as text
}

// Request is a provider-neutral LLM request. The client decides how to
// translate Tools based on its Capabilities — native API, text prompt,
// or ignored.
type Request struct {
	Messages []conversation.Message
	Tools    []map[string]any
}

// Turn is the complete result of one model invocation. The agent treats
// this as the atomic unit of a model response: either text, tool calls,
// or both.
type Turn struct {
	Content   string
	ToolCalls []conversation.ToolCall
}

// StreamEvent is a single event from a streaming LLM response.
//   - Delta: a text chunk to append to the assistant response.
//   - Turn:  non-nil when the turn is complete; no further events follow.
//   - Err:   the stream failed; no further events follow.
type StreamEvent struct {
	Delta string
	Turn  *Turn
	Err   error
}

// Client is the abstraction over LLM providers. The TUI and agent depend
// on this interface, not on any concrete provider, so providers can be
// swapped (OpenAI, Anthropic, Ollama, LM Studio) without touching upper
// layers. Each client is responsible for translating between provider
// formats and Apex's provider-neutral types.
type Client interface {
	Capabilities() Capabilities
	Stream(ctx context.Context, req Request) <-chan StreamEvent
}
