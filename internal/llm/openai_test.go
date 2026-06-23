package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

type testBearerSource struct {
	tokenCalls   int
	refreshCalls int
}

func (s *testBearerSource) Token(ctx context.Context) (string, error) {
	s.tokenCalls++
	return "old", nil
}

func (s *testBearerSource) Refresh(ctx context.Context) (string, error) {
	s.refreshCalls++
	return "new", nil
}

func TestToOpenAIMessages_assistantToolCallNoContent(t *testing.T) {
	msgs := []conversation.Message{
		{
			Role:    conversation.RoleAssistant,
			Content: "",
			ToolCalls: []conversation.ToolCall{
				{ID: "call_1", Name: "web_fetch", Arguments: `{"url":"https://example.com"}`},
			},
		},
		{
			Role:       conversation.RoleTool,
			Content:    "tool result",
			ToolCallID: "call_1",
		},
	}

	result := toOpenAIMessages(msgs)

	// assistant message with empty content but tool calls
	assistant := result[0].OfAssistant
	if assistant == nil {
		t.Fatal("expected assistant message")
	}
	j, err := json.Marshal(assistant)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(j, &raw); err != nil {
		t.Fatal(err)
	}

	// content must be present as string, not nil/missing
	if _, ok := raw["content"]; !ok {
		t.Fatalf("content field missing; got %s", j)
	}
	if raw["content"] != "" {
		t.Fatalf("expected empty content string, got %v", raw["content"])
	}
	if tc, ok := raw["tool_calls"]; !ok || tc == nil {
		t.Fatal("expected tool_calls")
	}

	// tool message
	toolMsg := result[1].OfTool
	if toolMsg == nil {
		t.Fatal("expected tool message")
	}
	if toolMsg.ToolCallID != "call_1" {
		t.Fatalf("expected tool_call_id call_1, got %q", toolMsg.ToolCallID)
	}

	// round-trip: unmarshal assistant back through openai types
	var ap openai.ChatCompletionAssistantMessageParam
	if err := json.Unmarshal(j, &ap); err != nil {
		t.Fatalf("unmarshal assistant message: %v", err)
	}
	if len(ap.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(ap.ToolCalls))
	}
	fn := ap.ToolCalls[0].GetFunction()
	if fn == nil || fn.Name != "web_fetch" {
		t.Fatalf("expected tool call web_fetch, got %+v", fn)
	}
	if string(ap.Content.OfString.Value) != "" {
		t.Fatalf("expected empty content, got %q", ap.Content.OfString.Value)
	}
	// Content.OfString must NOT be omitted (nil would be omitted, empty "" is present)
	if param.IsOmitted(ap.Content.OfString) {
		t.Fatal("content must be present string (not nil); Ollama rejects nil content")
	}
}

func TestAuthMiddlewareRefreshesOnUnauthorized(t *testing.T) {
	source := &testBearerSource{}
	middleware := authMiddleware(source)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("{}")), nil
	}
	var authHeaders []string
	next := func(req *http.Request) (*http.Response, error) {
		authHeaders = append(authHeaders, req.Header.Get("Authorization"))
		status := http.StatusOK
		if len(authHeaders) == 1 {
			status = http.StatusUnauthorized
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}
	resp, err := middleware(req, next)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected retry response 200, got %d", resp.StatusCode)
	}
	if source.tokenCalls != 1 || source.refreshCalls != 1 {
		t.Fatalf("unexpected calls token=%d refresh=%d", source.tokenCalls, source.refreshCalls)
	}
	if len(authHeaders) != 2 || authHeaders[0] != "Bearer old" || authHeaders[1] != "Bearer new" {
		t.Fatalf("unexpected auth headers %#v", authHeaders)
	}
}
