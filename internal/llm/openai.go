package llm

import (
	"context"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAIClient is an LLM client that talks to any OpenAI-compatible API
// (OpenAI, Ollama's /v1 endpoint, LM Studio, etc.).
type OpenAIClient struct {
	client openai.Client
	model  string
}

// NewOpenAIClient creates a client configured for an OpenAI-compatible
// endpoint. For local Ollama, pass baseURL="http://localhost:11434/v1"
// and apiKey="ollama".
func NewOpenAIClient(model, baseURL, apiKey string) *OpenAIClient {
	c := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
	return &OpenAIClient{client: c, model: model}
}

// Stream sends the conversation to the model and returns a channel of
// stream events. The channel is closed after a Done or Err event.
func (c *OpenAIClient) Stream(ctx context.Context, messages []conversation.Message) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		stream := c.client.Chat.Completions.NewStreaming(ctx,
			openai.ChatCompletionNewParams{
				Model:    c.model,
				Messages: toOpenAIMessages(messages),
			})

		for stream.Next() {
			if choices := stream.Current().Choices; len(choices) > 0 {
				ch <- StreamEvent{Delta: choices[0].Delta.Content}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Err: err}
			return
		}

		ch <- StreamEvent{Done: true}
	}()

	return ch
}

func toOpenAIMessages(messages []conversation.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case conversation.RoleSystem:
			out = append(out, openai.SystemMessage(msg.Content))
		case conversation.RoleAssistant:
			out = append(out, openai.AssistantMessage(msg.Content))
		default:
			out = append(out, openai.UserMessage(msg.Content))
		}
	}
	return out
}
