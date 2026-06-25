package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	agentSystemPrompt = `You are Apex, a terminal coding agent.

Use tool results as authoritative data. If a tool call succeeds, answer from the returned content directly.
Do not describe tool mechanics (no function calls, IDs, names, schemas, object wrappers).
If the user asked for a file/web result, return the requested content unless they asked for a summary.
If user asks for direct output, output the content first.

If a tool call fails, report the exact failure text and what to try next.`
)

// Event is a single event from the agent loop. The TUI consumes these
// to update the chat view.
type Event struct {
	Delta   string // text chunk from the model
	Status  string // tool status line, e.g. "[tool] web_fetch https://..."
	Context *llm.ContextUsage
	Err     error
	Done    bool
}

// Agent orchestrates the model ↔ tool loop. It is provider-agnostic:
// all provider-specific translation happens inside llm.Client.
type Agent struct {
	client        llm.Client
	registry      *tools.Registry
	contextWindow int
	systemPrompt  string
}

func New(client llm.Client, registry *tools.Registry) *Agent {
	return NewWithContextWindowAndSystemPrompt(client, registry, 0, agentSystemPrompt)
}

func NewWithContextWindow(client llm.Client, registry *tools.Registry, contextWindow int) *Agent {
	return NewWithContextWindowAndSystemPrompt(client, registry, contextWindow, agentSystemPrompt)
}

func NewWithContextWindowAndSystemPrompt(client llm.Client, registry *tools.Registry, contextWindow int, systemPrompt string) *Agent {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = agentSystemPrompt
	}
	return &Agent{client: client, registry: registry, contextWindow: contextWindow, systemPrompt: systemPrompt}
}

// Run starts the agent loop for the current session and returns a
// channel of events. The channel is closed after Done or Err.
func (a *Agent) Run(ctx context.Context, session *conversation.Session) <-chan Event {
	ch := make(chan Event)

	go func() {
		defer close(ch)

		for i := 0; i < maxToolIterations; i++ {
			req := a.requestForSession(session)
			usage := a.contextUsage(req)
			ch <- Event{Context: &usage}

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
				usage := a.contextUsage(a.requestForSession(session))
				ch <- Event{Context: &usage}
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

func (a *Agent) requestForSession(session *conversation.Session) llm.Request {
	return llm.Request{
		Messages: withSystemPrompt(session.Messages(), a.systemPrompt),
		Tools:    a.registry.Specs(),
	}
}

func (a *Agent) contextUsage(req llm.Request) llm.ContextUsage {
	return llm.EstimateContextUsage(req, a.contextWindow)
}

func withAgentSystemPrompt(messages []conversation.Message) []conversation.Message {
	return withSystemPrompt(messages, agentSystemPrompt)
}

func BuildSystemPrompt(skillIndex string) string {
	if strings.TrimSpace(skillIndex) == "" {
		return agentSystemPrompt
	}
	return agentSystemPrompt + "\n\n" + strings.TrimSpace(skillIndex)
}

func withSystemPrompt(messages []conversation.Message, systemPrompt string) []conversation.Message {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = agentSystemPrompt
	}
	for i := range messages {
		if messages[i].Role == conversation.RoleSystem {
			if strings.Contains(messages[i].Content, systemPrompt) {
				return messages
			}
			if systemPrompt == agentSystemPrompt && strings.Contains(messages[i].Content, "You are Apex, a terminal coding agent") {
				return messages
			}
			out := make([]conversation.Message, len(messages))
			copy(out, messages)
			out[i].Content = out[i].Content + "\n\n" + systemPrompt
			return out
		}
	}

	out := make([]conversation.Message, 0, len(messages)+1)
	out = append(out, conversation.Message{Role: conversation.RoleSystem, Content: systemPrompt})
	out = append(out, messages...)
	return out
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
			} else if path, ok := parsed["path"].(string); ok {
				argsSummary = path
			} else {
				argsSummary = formatToolArgs(parsed)
			}
		} else {
			argsSummary = strings.TrimSpace(call.Arguments)
		}
	}
	if argsSummary == "" {
		return fmt.Sprintf("[tool] %s", call.Name)
	}
	return fmt.Sprintf("[tool] %s %s", call.Name, argsSummary)
}

func formatToolArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]any, len(args))
	for _, key := range keys {
		ordered[key] = args[key]
	}
	data, err := json.Marshal(ordered)
	if err != nil {
		return ""
	}
	return truncateStatusArg(string(data))
}

func truncateStatusArg(s string) string {
	const max = 160
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
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
	if err := validateRequiredToolArgs(tool.Spec(), args); err != nil {
		return fmt.Sprintf("tool error: %s", err)
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

func validateRequiredToolArgs(spec tools.ToolSpec, args json.RawMessage) error {
	required := requiredFields(spec.Parameters)
	if len(required) == 0 {
		return nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(args, &parsed); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	for _, field := range required {
		value, ok := parsed[field]
		if !ok || isEmptyToolArg(value) {
			return fmt.Errorf("missing required argument %q for %s. Include it in arguments, for example: %s", field, spec.Name, exampleArgs(spec.Name, field))
		}
	}
	return nil
}

func requiredFields(schema map[string]any) []string {
	raw, ok := schema["required"]
	if !ok {
		return nil
	}
	switch required := raw.(type) {
	case []string:
		return required
	case []any:
		fields := make([]string, 0, len(required))
		for _, value := range required {
			if field, ok := value.(string); ok {
				fields = append(fields, field)
			}
		}
		return fields
	default:
		return nil
	}
}

func isEmptyToolArg(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

func exampleArgs(toolName, field string) string {
	switch toolName {
	case "read_file":
		return `{"path":"README.md"}`
	case "web_fetch":
		return `{"url":"https://example.com"}`
	default:
		return fmt.Sprintf(`{"%s":"..."}`, field)
	}
}
