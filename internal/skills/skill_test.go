package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRootFindsNestedSkillsAndStoresMetadataOnly(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "graphify", "SKILL.md"), `---
name: graphify
description: "Builds knowledge graphs."
trigger: /graphify
---
`+strings.Repeat("body\n", 20000))
	writeSkill(t, filepath.Join(root, "nested", "skills", "caveman", "SKILL.md"), `---
name: caveman
description: |
  Terse response mode.
---
# Caveman
`)

	store, err := LoadRoot(root)
	if err != nil {
		t.Fatalf("LoadRoot failed: %v", err)
	}

	names := store.Names()
	if got, want := strings.Join(names, ","), "caveman,graphify"; got != want {
		t.Fatalf("expected sorted names %q, got %q", want, got)
	}

	skill, ok := store.Get("graphify")
	if !ok {
		t.Fatal("expected graphify skill")
	}
	if skill.Trigger != "/graphify" {
		t.Fatalf("expected trigger, got %q", skill.Trigger)
	}
	if strings.Contains(skill.Description, "body") {
		t.Fatalf("description contains body content: %q", skill.Description)
	}
}

func TestLoadRootMissingDirectoryReturnsEmptyStore(t *testing.T) {
	store, err := LoadRoot(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("expected missing root to be non-fatal, got %v", err)
	}
	if len(store.Names()) != 0 {
		t.Fatalf("expected no skills, got %v", store.Names())
	}
}

func TestIndexPromptContainsMetadataAndNameBasedToolInstruction(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "graphify", "SKILL.md"), `---
name: graphify
description: Builds knowledge graphs.
trigger: /graphify
---
body
`)
	store, err := LoadRoot(root)
	if err != nil {
		t.Fatalf("LoadRoot failed: %v", err)
	}

	prompt := store.IndexPrompt()
	for _, want := range []string{"graphify", "trigger: /graphify", `load_skill with {"name":"<skill-name>"}`} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
		}
	}
}

func writeSkill(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
