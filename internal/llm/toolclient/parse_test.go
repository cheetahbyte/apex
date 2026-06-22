package toolclient

import (
	"testing"

	"github.com/cheetahbyte/apex/internal/conversation"
)

var testSpecs = []map[string]any{
	{"name": "web_fetch", "description": "fetch a URL", "schema": map[string]any{"type": "object"}},
	{"name": "read_file", "description": "read a file", "schema": map[string]any{"type": "object"}},
}

func TestParseToolCalls_standardFormat(t *testing.T) {
	content := `{"tool_calls":[{"id":"call_1","name":"web_fetch","arguments":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].ID != "call_1" || calls[0].Name != "web_fetch" {
		t.Fatalf("unexpected: %+v", calls[0])
	}
	if !contains(calls[0].Arguments, "example.com") {
		t.Fatalf("expected url in args, got %q", calls[0].Arguments)
	}
}

func TestParseToolCalls_parametersAlias(t *testing.T) {
	content := `{"tool_calls":[{"name":"web_fetch","parameters":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", calls[0].Name)
	}
	if !contains(calls[0].Arguments, "example.com") {
		t.Fatalf("expected url in args, got %q", calls[0].Arguments)
	}
}

func TestParseToolCalls_inputAlias(t *testing.T) {
	content := `{"tool_calls":[{"name":"web_fetch","input":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if !contains(calls[0].Arguments, "example.com") {
		t.Fatalf("expected url in args, got %q", calls[0].Arguments)
	}
}

func TestParseToolCalls_singleObjectWithName(t *testing.T) {
	content := `{"name":"web_fetch","arguments":{"url":"https://example.com"}}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", calls[0].Name)
	}
	if calls[0].ID != "call_1" {
		t.Fatalf("expected auto ID call_1, got %q", calls[0].ID)
	}
}

func TestParseToolCalls_singleObjectWithFunction(t *testing.T) {
	content := `{"function":"web_fetch","input":{"url":"https://example.com"}}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", calls[0].Name)
	}
}

func TestParseToolCalls_fuzzyNameMatch(t *testing.T) {
	// "webfetch" should normalize to "web_fetch"
	content := `{"tool_calls":[{"name":"webfetch","arguments":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" {
		t.Fatalf("expected fuzzy match to web_fetch, got %q", calls[0].Name)
	}
}

func TestParseToolCalls_caseInsensitiveName(t *testing.T) {
	content := `{"tool_calls":[{"name":"WEB_FETCH","arguments":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" {
		t.Fatalf("expected case-insensitive match to web_fetch, got %q", calls[0].Name)
	}
}

func TestParseToolCalls_missingID(t *testing.T) {
	content := `{"tool_calls":[{"name":"web_fetch","arguments":{"url":"https://example.com"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].ID != "call_1" {
		t.Fatalf("expected auto-generated ID call_1, got %q", calls[0].ID)
	}
}

func TestParseToolCalls_emptyArguments(t *testing.T) {
	content := `{"tool_calls":[{"id":"call_1","name":"web_fetch"}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Arguments != "{}" {
		t.Fatalf("expected default '{}', got %q", calls[0].Arguments)
	}
}

func TestParseToolCalls_multipleCalls(t *testing.T) {
	content := `{"tool_calls":[{"id":"c1","name":"web_fetch","arguments":{"url":"https://a.com"}},{"id":"c2","name":"read_file","arguments":{"path":"/tmp/x"}}]}`
	calls := ParseToolCalls(content, testSpecs)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "web_fetch" || calls[1].Name != "read_file" {
		t.Fatalf("unexpected names: %q, %q", calls[0].Name, calls[1].Name)
	}
}

func TestParseToolCalls_notJSON(t *testing.T) {
	calls := ParseToolCalls("This is a normal text response.", testSpecs)
	if calls != nil {
		t.Fatalf("expected nil for non-JSON, got %v", calls)
	}
}

func TestParseToolCalls_noToolCallsField(t *testing.T) {
	calls := ParseToolCalls(`{"foo":"bar"}`, testSpecs)
	if calls != nil {
		t.Fatalf("expected nil for JSON without tool_calls, got %v", calls)
	}
}

func TestParseToolCalls_emptyToolCallsArray(t *testing.T) {
	calls := ParseToolCalls(`{"tool_calls":[]}`, testSpecs)
	if calls != nil {
		t.Fatalf("expected nil for empty tool_calls, got %v", calls)
	}
}

func TestNormalizeToolName_exactMatch(t *testing.T) {
	if name := normalizeToolName("web_fetch", testSpecs); name != "web_fetch" {
		t.Fatalf("expected web_fetch, got %q", name)
	}
}

func TestNormalizeToolName_noMatch(t *testing.T) {
	if name := normalizeToolName("nonexistent", testSpecs); name != "nonexistent" {
		t.Fatalf("expected passthrough for unknown tool, got %q", name)
	}
}

func TestNormalizeToolName_underscoreInsensitive(t *testing.T) {
	if name := normalizeToolName("readfile", testSpecs); name != "read_file" {
		t.Fatalf("expected read_file, got %q", name)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && containsStr(s, substr)))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure conversation.ToolCall is used (silences unused import if tests evolve)
var _ conversation.ToolCall
