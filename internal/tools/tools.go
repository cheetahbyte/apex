package tools

import (
	"context"
	"encoding/json"
)

type ToolCall struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Args []string `json:"args"`
}

type ToolResult struct {
	ID         string `json:"id"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"params"`
	ReadOnly    bool
}

type Tool interface {
	Spec() ToolSpec
	Execute(ctx context.Context, input json.RawMessage) (ToolResult, error)
}
