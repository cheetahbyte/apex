package toolclient

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
	"github.com/cheetahbyte/apex/internal/llm"
)

// mockClient returns pre-programmed turns in sequence.
type mockClient struct {
	caps     llm.Capabilities
	turns    []llm.Turn
	callIdx  int
	requests []llm.Request
}

func (m *mockClient) Capabilities() llm.Capabilities { return m.caps }

func (m *mockClient) Stream(ctx context.Context, req llm.Request) <-chan llm.StreamEvent {
	m.requests = append(m.requests, req)
	ch := make(chan llm.StreamEvent)
	go func() {
		defer close(ch)
		if m.callIdx >= len(m.turns) {
			ch <- llm.StreamEvent{Err: fmt.Errorf("no more mock turns")}
			return
		}
		turn := m.turns[m.callIdx]
		m.callIdx++
		if turn.Content != "" {
			ch <- llm.StreamEvent{Delta: turn.Content}
		}
		ch <- llm.StreamEvent{Turn: &turn}
	}()
	return ch
}

var testTools = []map[string]any{
	{"name": "web_fetch", "description": "fetch a URL", "schema": map[string]any{"type": "object"}},
}

func TestModeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Mode
	}{
		{"auto", ModeAuto},
		{"native", ModeNative},
		{"text", ModeText},
		{"none", ModeNone},
		{"AUTO", ModeAuto},
		{"", ModeAuto},
		{"invalid", ModeAuto},
	}
	for _, tt := range tests {
		if got := ModeFromString(tt.input); got != tt.want {
			t.Errorf("ModeFromString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClient_noneMode_stripsTools(t *testing.T) {
	inner := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "hello"}},
	}
	client := New(inner, ModeNone)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hi"}},
		Tools:    testTools,
	}

	for ev := range client.Stream(context.Background(), req) {
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
	}

	if len(inner.requests) != 1 {
		t.Fatal("expected 1 inner request")
	}
	if len(inner.requests[0].Tools) != 0 {
		t.Fatalf("expected tools stripped, got %d tools", len(inner.requests[0].Tools))
	}
}

func TestClient_nativeMode_passthrough(t *testing.T) {
	inner := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "hello", ToolCalls: []conversation.ToolCall{
			{ID: "c1", Name: "web_fetch", Arguments: `{}`},
		}}},
	}
	client := New(inner, ModeNative)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hi"}},
		Tools:    testTools,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || len(turn.ToolCalls) != 1 {
		t.Fatalf("expected passthrough with 1 tool call, got %+v", turn)
	}
	// Native mode should pass tools through to inner client
	if len(inner.requests[0].Tools) != 1 {
		t.Fatalf("expected tools passed through in native mode")
	}
}

func TestClient_textMode_parsesToolCall(t *testing.T) {
	inner := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{
			{Content: `{"tool_calls":[{"id":"c1","name":"web_fetch","arguments":{"url":"https://example.com"}}]}`},
		},
	}
	client := New(inner, ModeText)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "fetch example.com"}},
		Tools:    testTools,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || len(turn.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call from text protocol, got %+v", turn)
	}
	if turn.ToolCalls[0].Name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", turn.ToolCalls[0].Name)
	}
	// Text mode should NOT pass tools to inner client
	if len(inner.requests[0].Tools) != 0 {
		t.Fatalf("expected no tools in inner request for text mode")
	}
}

func TestClient_textMode_normalText(t *testing.T) {
	inner := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "This is a normal response."}},
	}
	client := New(inner, ModeText)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hi"}},
		Tools:    testTools,
	}

	var deltas []string
	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || turn.Content != "This is a normal response." {
		t.Fatalf("expected normal text passthrough, got %+v", turn)
	}
	if len(turn.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(turn.ToolCalls))
	}
}

func TestClient_autoMode_nativeWorks(t *testing.T) {
	inner := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{
			{Content: "", ToolCalls: []conversation.ToolCall{
				{ID: "c1", Name: "web_fetch", Arguments: `{}`},
			}},
		},
	}
	client := New(inner, ModeAuto)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "fetch"}},
		Tools:    testTools,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || len(turn.ToolCalls) != 1 {
		t.Fatalf("expected native tool call, got %+v", turn)
	}
	if len(inner.requests) != 1 {
		t.Fatalf("expected only 1 inner request (no fallback), got %d", len(inner.requests))
	}
}

func TestClient_autoMode_fallsBackOnEmpty(t *testing.T) {
	// First call (native) returns empty, second call (text) returns JSON tool call
	inner := &mockClient{
		caps: llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{
			{Content: "", ToolCalls: nil}, // empty native response
			{Content: `{"tool_calls":[{"id":"c1","name":"web_fetch","arguments":{"url":"https://example.com"}}]}`},
		},
	}
	client := New(inner, ModeAuto)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "fetch example.com"}},
		Tools:    testTools,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil {
		t.Fatal("expected turn")
	}
	if len(turn.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call from fallback, got %d", len(turn.ToolCalls))
	}
	if turn.ToolCalls[0].Name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", turn.ToolCalls[0].Name)
	}
	// Should have made 2 inner requests: native attempt + text fallback
	if len(inner.requests) != 2 {
		t.Fatalf("expected 2 inner requests (native + text fallback), got %d", len(inner.requests))
	}
	// First request should have tools (native), second should not (text)
	if len(inner.requests[0].Tools) != 1 {
		t.Fatalf("expected tools in first (native) request")
	}
	if len(inner.requests[1].Tools) != 0 {
		t.Fatalf("expected no tools in second (text) request")
	}
}

func TestClient_autoMode_noNativeTools_usesTextDirectly(t *testing.T) {
	inner := &mockClient{
		caps: llm.Capabilities{NativeTools: false},
		turns: []llm.Turn{
			{Content: `{"tool_calls":[{"id":"c1","name":"web_fetch","arguments":{"url":"https://example.com"}}]}`},
		},
	}
	client := New(inner, ModeAuto)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "fetch"}},
		Tools:    testTools,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || len(turn.ToolCalls) != 1 {
		t.Fatalf("expected tool call via text (no native), got %+v", turn)
	}
	// Should only make 1 request (text, no native attempt)
	if len(inner.requests) != 1 {
		t.Fatalf("expected 1 inner request, got %d", len(inner.requests))
	}
	if len(inner.requests[0].Tools) != 0 {
		t.Fatalf("expected no tools in text request")
	}
}

func TestClient_autoMode_nativeTextResponse(t *testing.T) {
	inner := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "Hello from native"}},
	}
	client := New(inner, ModeAuto)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hi"}},
		Tools:    testTools,
	}

	var deltas []string
	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || turn.Content != "Hello from native" {
		t.Fatalf("expected native text response, got %+v", turn)
	}
	if len(deltas) != 1 || deltas[0] != "Hello from native" {
		t.Fatalf("expected 1 delta, got %v", deltas)
	}
	if len(inner.requests) != 1 {
		t.Fatalf("expected only 1 request (no fallback needed), got %d", len(inner.requests))
	}
}

func TestClient_noTools_passthrough(t *testing.T) {
	inner := &mockClient{
		caps:  llm.Capabilities{NativeTools: true},
		turns: []llm.Turn{{Content: "hello"}},
	}
	client := New(inner, ModeAuto)

	req := llm.Request{
		Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hi"}},
		Tools:    nil,
	}

	var turn *llm.Turn
	for ev := range client.Stream(context.Background(), req) {
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}

	if turn == nil || turn.Content != "hello" {
		t.Fatalf("expected passthrough, got %+v", turn)
	}
}

func TestClient_capabilities(t *testing.T) {
	inner := &mockClient{caps: llm.Capabilities{NativeTools: true, StreamingToolCalls: true}}
	client := New(inner, ModeAuto)
	caps := client.Capabilities()
	if !caps.NativeTools {
		t.Fatal("toolclient should always claim native tools to agent")
	}
	if !caps.StreamingToolCalls {
		t.Fatal("should inherit streaming tool calls from inner")
	}
}

func TestInjectToolPrompt_appendsToExistingSystem(t *testing.T) {
	msgs := []conversation.Message{
		{Role: conversation.RoleSystem, Content: "You are helpful."},
		{Role: conversation.RoleUser, Content: "hi"},
	}
	result := injectToolPrompt(msgs, "TOOL INSTRUCTIONS")

	if result[0].Role != conversation.RoleSystem {
		t.Fatal("expected system message first")
	}
	if !strings.Contains(result[0].Content, "You are helpful.") {
		t.Fatal("should preserve original system content")
	}
	if !strings.Contains(result[0].Content, "TOOL INSTRUCTIONS") {
		t.Fatal("should append tool prompt")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}

func TestInjectToolPrompt_prependsIfNoSystem(t *testing.T) {
	msgs := []conversation.Message{
		{Role: conversation.RoleUser, Content: "hi"},
	}
	result := injectToolPrompt(msgs, "TOOL INSTRUCTIONS")

	if result[0].Role != conversation.RoleSystem {
		t.Fatal("expected system message prepended")
	}
	if result[0].Content != "TOOL INSTRUCTIONS" {
		t.Fatalf("expected tool prompt as content, got %q", result[0].Content)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
}
