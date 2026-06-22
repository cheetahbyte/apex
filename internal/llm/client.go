package llm

import (
	"context"

	"github.com/cheetahbyte/apex/internal/conversation"
)

// StreamEvent is a single event from a streaming LLM response. Exactly one
// of Delta, Err, or Done is meaningful per event:
//   - Delta: a text chunk to append to the assistant response.
//   - Err:   the stream failed; no further events will arrive.
//   - Done:  the stream completed successfully; no further events will arrive.
type StreamEvent struct {
	Delta string
	Err   error
	Done  bool
}

// Client is the abstraction over LLM providers. The TUI depends on this
// interface, not on any concrete provider, so providers can be swapped
// (Ollama, OpenAI, Anthropic) without touching UI code.
type Client interface {
	Stream(ctx context.Context, messages []conversation.Message) <-chan StreamEvent
}
