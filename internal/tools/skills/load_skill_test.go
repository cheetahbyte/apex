package skills

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	stores "github.com/cheetahbyte/apex/internal/skills"
)

func TestLoadSkillLoadsContentByName(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, "graphify", "SKILL.md")
	content := `---
name: graphify
description: Builds knowledge graphs.
---
# Graphify
Full body loaded only by tool.
`
	writeTestFile(t, skillPath, content)

	store, err := stores.LoadRoot(root)
	if err != nil {
		t.Fatalf("LoadRoot failed: %v", err)
	}

	result, err := New(store).Execute(context.Background(), json.RawMessage(`{"name":"graphify"}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Content, "Full body loaded only by tool.") {
		t.Fatalf("expected skill body in result, got %q", result.Content)
	}
	if !strings.Contains(result.Content, `<skill_content name="graphify"`) {
		t.Fatalf("expected skill wrapper, got %q", result.Content)
	}
}

func TestLoadSkillUnknownNameListsAvailableSkills(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "caveman", "SKILL.md"), `---
name: caveman
description: Terse response mode.
---
body
`)
	store, err := stores.LoadRoot(root)
	if err != nil {
		t.Fatalf("LoadRoot failed: %v", err)
	}

	_, err = New(store).Execute(context.Background(), json.RawMessage(`{"name":"missing"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unknown skill "missing"`) || !strings.Contains(err.Error(), "caveman") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
