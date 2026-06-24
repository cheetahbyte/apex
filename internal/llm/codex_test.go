package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
)

type codexTestSource struct {
	token     string
	refreshed string
	accountID string
	refreshes int
}

func (s *codexTestSource) Token(ctx context.Context) (string, error) { return s.token, nil }
func (s *codexTestSource) Refresh(ctx context.Context) (string, error) {
	s.refreshes++
	return s.refreshed, nil
}
func (s *codexTestSource) AccountID(ctx context.Context) (string, error) { return s.accountID, nil }

func TestCodexClientStreamsTextAndToolCalls(t *testing.T) {
	source := &codexTestSource{token: "access", refreshed: "new-access", accountID: "acct"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access" {
			t.Fatalf("missing auth header %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("ChatGPT-Account-Id") != "acct" {
			t.Fatalf("missing account header %q", r.Header.Get("ChatGPT-Account-Id"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5.5" || body["stream"] != true || body["store"] != false {
			t.Fatalf("unexpected body %+v", body)
		}
		if _, ok := body["max_output_tokens"]; ok {
			t.Fatal("codex body must not set max_output_tokens")
		}
		if tools, ok := body["tools"].([]any); !ok || len(tools) != 1 {
			t.Fatalf("missing tools %+v", body["tools"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\"}\"}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewCodexClient("gpt-5.5", server.URL, source)
	stream := client.Stream(context.Background(), Request{
		Messages: []conversation.Message{{Role: conversation.RoleSystem, Content: "sys"}, {Role: conversation.RoleUser, Content: "hello"}},
		Tools: []map[string]any{{
			"name":        "read_file",
			"description": "read file",
			"schema":      map[string]any{"type": "object"},
		}},
	})

	var sawDelta bool
	var turn *Turn
	for ev := range stream {
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		if ev.Delta == "hi" {
			sawDelta = true
		}
		if ev.Turn != nil {
			turn = ev.Turn
		}
	}
	if !sawDelta {
		t.Fatal("missing text delta")
	}
	if turn == nil || turn.Content != "hi" || len(turn.ToolCalls) != 1 {
		t.Fatalf("unexpected turn %+v", turn)
	}
	if turn.ToolCalls[0].ID != "call_1" || turn.ToolCalls[0].Name != "read_file" || turn.ToolCalls[0].Arguments != `{"path":"README.md"}` {
		t.Fatalf("unexpected tool call %+v", turn.ToolCalls[0])
	}
}

func TestCodexClientRefreshesOnUnauthorized(t *testing.T) {
	source := &codexTestSource{token: "old", refreshed: "new", accountID: "acct"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") != "Bearer new" {
			t.Fatalf("expected refreshed token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewCodexClient("gpt-5.5", server.URL, source)
	for ev := range client.Stream(context.Background(), Request{Messages: []conversation.Message{{Role: conversation.RoleUser, Content: "hello"}}}) {
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
	}
	if source.refreshes != 1 || requests != 2 {
		t.Fatalf("expected refresh retry, refreshes=%d requests=%d", source.refreshes, requests)
	}
}
