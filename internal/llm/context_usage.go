package llm

import (
	"encoding/json"
	"math"
	"unicode/utf8"
)

// ContextUsage describes the estimated prompt size for the next model request.
type ContextUsage struct {
	Tokens        int
	ContextWindow int
	Percent       float64
	Estimated     bool
}

// EstimateContextUsage estimates tokens for the request payload that is about
// to be sent to the provider. It intentionally stays provider-neutral: exact
// token accounting varies by API, model, and tool protocol.
func EstimateContextUsage(req Request, contextWindow int) ContextUsage {
	tokens := 3 // assistant reply priming overhead approximation
	for _, msg := range req.Messages {
		tokens += 3 // chat message framing overhead approximation
		tokens += estimateTokens(string(msg.Role))
		tokens += estimateTokens(msg.Content)
		for _, call := range msg.ToolCalls {
			tokens += estimateTokens(call.ID)
			tokens += estimateTokens(call.Name)
			tokens += estimateTokens(call.Arguments)
		}
		if msg.ToolCallID != "" {
			tokens += estimateTokens(msg.ToolCallID)
		}
	}
	if len(req.Tools) > 0 {
		if raw, err := json.Marshal(req.Tools); err == nil {
			tokens += estimateTokens(string(raw))
		}
	}

	usage := ContextUsage{Tokens: tokens, ContextWindow: contextWindow, Estimated: true}
	if contextWindow > 0 {
		usage.Percent = float64(tokens) / float64(contextWindow) * 100
	}
	return usage
}

func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	// Common rule of thumb for BPE tokenizers: ~4 UTF-8 chars per token.
	// Rune count prevents large underestimates for non-ASCII text.
	runes := utf8.RuneCountInString(s)
	return int(math.Ceil(float64(runes) / 4.0))
}
