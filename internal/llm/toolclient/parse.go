package toolclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/conversation"
)

// ParseToolCalls attempts to extract tool calls from model text output.
// It tolerantly accepts several common JSON shapes that models produce
// and normalizes them into Apex's canonical ToolCall format.
//
// Accepted shapes:
//
//	{"tool_calls":[{"id":"call_1","name":"web_fetch","arguments":{"url":"..."}}]}
//	{"tool_calls":[{"name":"web_fetch","parameters":{"url":"..."}}]}
//	{"name":"web_fetch","arguments":{"url":"..."}}
//	{"name":"web_fetch","parameters":{"url":"..."}}
//	{"function":"web_fetch","input":{"url":"..."}}
//
// Normalizations:
//   - "parameters" and "input" → "arguments"
//   - fuzzy tool name matching (webfetch → web_fetch)
//   - missing IDs auto-generated
func ParseToolCalls(content string, specs []map[string]any) []conversation.ToolCall {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return nil
	}

	// Try the standard {"tool_calls":[...]} shape first.
	if calls := parseToolCallsArray(trimmed, specs); len(calls) > 0 {
		return calls
	}

	// Try a single tool call object: {"name":"...","arguments":{...}}
	if calls := parseSingleToolCall(trimmed, specs); len(calls) > 0 {
		return calls
	}

	return nil
}

// parseToolCallsArray parses {"tool_calls":[{...},{...}]}.
func parseToolCallsArray(content string, specs []map[string]any) []conversation.ToolCall {
	var parsed struct {
		ToolCalls []json.RawMessage `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil
	}
	if len(parsed.ToolCalls) == 0 {
		return nil
	}

	out := make([]conversation.ToolCall, 0, len(parsed.ToolCalls))
	for i, raw := range parsed.ToolCalls {
		call, ok := parseOneToolCall(raw, i, specs)
		if !ok {
			return nil
		}
		out = append(out, call)
	}
	return out
}

// parseSingleToolCall parses a bare {"name":"...","arguments":{...}}.
func parseSingleToolCall(content string, specs []map[string]any) []conversation.ToolCall {
	call, ok := parseOneToolCall(json.RawMessage(content), 0, specs)
	if !ok {
		return nil
	}
	return []conversation.ToolCall{call}
}

// parseOneToolCall parses a single tool call object from raw JSON.
// It accepts name/function and arguments/parameters/input fields.
func parseOneToolCall(raw json.RawMessage, index int, specs []map[string]any) (conversation.ToolCall, bool) {
	// Use a flexible struct with all known field aliases.
	var flexible struct {
		ID         string          `json:"id"`
		Name       string          `json:"name"`
		Function   string          `json:"function"`
		Arguments  json.RawMessage `json:"arguments"`
		Parameters json.RawMessage `json:"parameters"`
		Input      json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(raw, &flexible); err != nil {
		return conversation.ToolCall{}, false
	}

	// Resolve name from "name" or "function" field.
	name := flexible.Name
	if name == "" {
		name = flexible.Function
	}
	if name == "" {
		return conversation.ToolCall{}, false
	}

	// Normalize tool name against registered specs.
	name = normalizeToolName(name, specs)

	// Resolve arguments from "arguments", "parameters", or "input".
	args := "{}"
	if len(flexible.Arguments) > 0 && string(flexible.Arguments) != "null" {
		args = string(flexible.Arguments)
	} else if len(flexible.Parameters) > 0 && string(flexible.Parameters) != "null" {
		args = string(flexible.Parameters)
	} else if len(flexible.Input) > 0 && string(flexible.Input) != "null" {
		args = string(flexible.Input)
	}
	if strings.TrimSpace(args) == "" {
		args = "{}"
	}

	// Generate ID if missing.
	id := flexible.ID
	if id == "" {
		id = fmt.Sprintf("call_%d", index+1)
	}

	return conversation.ToolCall{
		ID:        id,
		Name:      name,
		Arguments: args,
	}, true
}

// normalizeToolName matches a model-produced tool name against registered
// tool specs, tolerating case differences and missing underscores.
func normalizeToolName(name string, specs []map[string]any) string {
	// Exact match.
	for _, s := range specs {
		if sn, ok := s["name"].(string); ok && sn == name {
			return sn
		}
	}

	lower := strings.ToLower(name)

	// Case-insensitive match.
	for _, s := range specs {
		if sn, ok := s["name"].(string); ok && strings.ToLower(sn) == lower {
			return sn
		}
	}

	// Underscore-insensitive match (webfetch → web_fetch).
	compact := strings.ReplaceAll(lower, "_", "")
	for _, s := range specs {
		if sn, ok := s["name"].(string); ok {
			if strings.ReplaceAll(strings.ToLower(sn), "_", "") == compact {
				return sn
			}
		}
	}

	return name
}
