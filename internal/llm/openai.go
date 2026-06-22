package llm

import (
	"context"
	"fmt"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
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

// Capabilities reports OpenAI-compatible native tool support.
func (c *OpenAIClient) Capabilities() Capabilities {
	return Capabilities{
		NativeTools:        true,
		StreamingToolCalls: true,
	}
}

// Stream sends the conversation to the model and returns a channel of
// stream events. The channel is closed after a Turn or Err event.
func (c *OpenAIClient) Stream(ctx context.Context, req Request) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		params := openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: toOpenAIMessages(req.Messages),
		}
		if len(req.Tools) > 0 {
			params.Tools = toOpenAITools(req.Tools)
			params.ParallelToolCalls = openai.Bool(false)
		}

		stream := c.client.Chat.Completions.NewStreaming(ctx, params)
		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			if !acc.AddChunk(chunk) {
				ch <- StreamEvent{Err: fmt.Errorf("failed to accumulate stream chunk")}
				return
			}
			if choices := chunk.Choices; len(choices) > 0 && choices[0].Delta.Content != "" {
				ch <- StreamEvent{Delta: choices[0].Delta.Content}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Err: err}
			return
		}

		turn := &Turn{}
		if len(acc.Choices) > 0 {
			turn.Content = acc.Choices[0].Message.Content
			turn.ToolCalls = fromOpenAIToolCalls(acc.Choices[0].Message.ToolCalls)
		}
		ch <- StreamEvent{Turn: turn}
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
			assistant := *openai.AssistantMessage(msg.Content).OfAssistant
			for _, call := range msg.ToolCalls {
				assistant.ToolCalls = append(assistant.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: call.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Arguments,
						},
					},
				})
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
		case conversation.RoleTool:
			out = append(out, openai.ToolMessage(msg.Content, msg.ToolCallID))
		default:
			out = append(out, openai.UserMessage(msg.Content))
		}
	}
	return out
}

func toOpenAITools(specs []map[string]any) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(specs))
	for _, spec := range specs {
		name, _ := spec["name"].(string)
		description, _ := spec["description"].(string)
		schema, _ := spec["schema"].(map[string]any)
		if name == "" || schema == nil {
			continue
		}
		out = append(out, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        name,
					Description: openai.String(description),
					Parameters:  shared.FunctionParameters(schema),
				},
			},
		})
	}
	return out
}

func fromOpenAIToolCalls(calls []openai.ChatCompletionMessageToolCallUnion) []conversation.ToolCall {
	out := make([]conversation.ToolCall, 0, len(calls))
	for _, call := range calls {
		fn := call.AsFunction()
		if fn.ID == "" || fn.Function.Name == "" {
			continue
		}
		out = append(out, conversation.ToolCall{
			ID:        fn.ID,
			Name:      fn.Function.Name,
			Arguments: fn.Function.Arguments,
		})
	}
	return out
}
