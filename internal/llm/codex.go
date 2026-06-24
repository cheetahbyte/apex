package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/cheetahbyte/apex/internal/conversation"
)

const defaultCodexBaseURL = "https://chatgpt.com/backend-api/codex"

// CodexClient talks to ChatGPT's Codex Responses backend with ChatGPT OAuth.
type CodexClient struct {
	model  string
	base   string
	source BearerTokenSource
	http   *http.Client
}

func NewCodexClient(model, baseURL string, source BearerTokenSource) *CodexClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultCodexBaseURL
	}
	return &CodexClient{
		model:  model,
		base:   baseURL,
		source: source,
		http:   &http.Client{Timeout: 0},
	}
}

func (c *CodexClient) Capabilities() Capabilities {
	return Capabilities{NativeTools: true, StreamingToolCalls: true}
}

func (c *CodexClient) Stream(ctx context.Context, req Request) <-chan StreamEvent {
	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)
		body, err := c.requestBody(req)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		resp, err := c.do(ctx, body, false)
		if err == nil && resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			resp, err = c.do(ctx, body, true)
		}
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			ch <- StreamEvent{Err: fmt.Errorf("codex request failed: %s: %s", resp.Status, strings.TrimSpace(string(msg)))}
			return
		}
		turn, err := parseCodexSSE(resp.Body, ch)
		if err != nil {
			ch <- StreamEvent{Err: err}
			return
		}
		ch <- StreamEvent{Turn: &turn}
	}()
	return ch
}

func (c *CodexClient) do(ctx context.Context, body []byte, refresh bool) (*http.Response, error) {
	token, err := c.source.Token(ctx)
	if refresh {
		token, err = c.source.Refresh(ctx)
	}
	if err != nil {
		return nil, err
	}
	accountID, err := accountID(ctx, c.source)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("ChatGPT-Account-Id", accountID)
	httpReq.Header.Set("OpenAI-Beta", "responses=experimental")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("originator", "apex")
	httpReq.Header.Set("User-Agent", fmt.Sprintf("apex/dev (%s %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	return c.http.Do(httpReq)
}

func accountID(ctx context.Context, source BearerTokenSource) (string, error) {
	withAccount, ok := source.(AccountIDSource)
	if !ok {
		return "", fmt.Errorf("codex token source does not expose ChatGPT account id")
	}
	return withAccount.AccountID(ctx)
}

func (c *CodexClient) requestBody(req Request) ([]byte, error) {
	instructions, input := toCodexInput(req.Messages)
	body := map[string]any{
		"model":               c.model,
		"input":               input,
		"stream":              true,
		"store":               false,
		"parallel_tool_calls": false,
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	if tools := toCodexTools(req.Tools); len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	return json.Marshal(body)
}

func toCodexInput(messages []conversation.Message) (string, []map[string]any) {
	var instructions []string
	input := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case conversation.RoleSystem:
			if strings.TrimSpace(msg.Content) != "" {
				instructions = append(instructions, msg.Content)
			}
		case conversation.RoleTool:
			if msg.ToolCallID != "" {
				input = append(input, map[string]any{"type": "function_call_output", "call_id": msg.ToolCallID, "output": msg.Content})
			}
		case conversation.RoleAssistant:
			if strings.TrimSpace(msg.Content) != "" {
				input = append(input, textMessage("assistant", "output_text", msg.Content))
			}
			for _, call := range msg.ToolCalls {
				input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": call.Arguments})
			}
		default:
			input = append(input, textMessage("user", "input_text", msg.Content))
		}
	}
	return strings.Join(instructions, "\n\n"), input
}

func textMessage(role, contentType, text string) map[string]any {
	return map[string]any{"type": "message", "role": role, "content": []map[string]any{{"type": contentType, "text": text}}}
}

func toCodexTools(specs []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		name, _ := spec["name"].(string)
		if name == "" {
			continue
		}
		description, _ := spec["description"].(string)
		schema, _ := spec["schema"].(map[string]any)
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{"type": "function", "name": name, "description": description, "parameters": schema})
	}
	return out
}

type codexStreamEvent struct {
	Type      string          `json:"type"`
	Delta     string          `json:"delta"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Item      codexOutputItem `json:"item"`
	Response  struct {
		Output []codexOutputItem `json:"output"`
	} `json:"response"`
}

type codexOutputItem struct {
	Type      string          `json:"type"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func parseCodexSSE(r io.Reader, ch chan<- StreamEvent) (Turn, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var text strings.Builder
	calls := map[string]conversation.ToolCall{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var ev codexStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return Turn{}, err
		}
		switch ev.Type {
		case "response.output_text.delta":
			if ev.Delta != "" {
				text.WriteString(ev.Delta)
				ch <- StreamEvent{Delta: ev.Delta}
			}
		case "response.output_item.done", "response.output_item.added":
			collectCodexItem(ev.Item, &text, calls)
		case "response.completed":
			for _, item := range ev.Response.Output {
				collectCodexItem(item, &text, calls)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Turn{}, err
	}
	out := make([]conversation.ToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, call)
	}
	return Turn{Content: text.String(), ToolCalls: out}, nil
}

func collectCodexItem(item codexOutputItem, text *strings.Builder, calls map[string]conversation.ToolCall) {
	switch item.Type {
	case "function_call":
		if item.CallID != "" && item.Name != "" {
			calls[item.CallID] = conversation.ToolCall{ID: item.CallID, Name: item.Name, Arguments: rawJSONString(item.Arguments)}
		}
	case "message":
		for _, part := range item.Content {
			if part.Type == "output_text" && part.Text != "" && text.Len() == 0 {
				text.WriteString(part.Text)
			}
		}
	}
}

func rawJSONString(raw json.RawMessage) string {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "{}"
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}
