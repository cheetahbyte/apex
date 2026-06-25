package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	stores "github.com/cheetahbyte/apex/internal/skills"
	"github.com/cheetahbyte/apex/internal/tools"
)

type args struct {
	Name string `json:"name"`
}

type SkillLoadTool struct {
	store *stores.Store
}

func New(store *stores.Store) SkillLoadTool {
	return SkillLoadTool{store: store}
}

func (t SkillLoadTool) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        "load_skill",
		Description: "Load detailed instructions for a named skill. Use before answering when a request matches an available skill.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The skill name to load, e.g. graphify or caveman.",
				},
			},
			"required":             []string{"name"},
			"additionalProperties": false,
		},
		ReadOnly: true,
	}
}

func (t SkillLoadTool) Execute(ctx context.Context, input json.RawMessage) (tools.ToolResult, error) {
	var parsedArgs args
	if err := json.Unmarshal(input, &parsedArgs); err != nil {
		return tools.ToolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	name := strings.TrimSpace(parsedArgs.Name)
	if name == "" {
		return tools.ToolResult{}, fmt.Errorf("name is required")
	}
	if t.store == nil {
		return tools.ToolResult{}, fmt.Errorf("no skill store configured")
	}

	skill, ok := t.store.Get(name)
	if !ok {
		return tools.ToolResult{}, fmt.Errorf("unknown skill %q; available skills: %s", name, strings.Join(sortedNames(t.store.Names()), ", "))
	}

	select {
	case <-ctx.Done():
		return tools.ToolResult{}, ctx.Err()
	default:
	}

	skillContent, err := os.ReadFile(skill.Path)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to read skill file: %w", err)
	}
	return tools.ToolResult{
		Content: fmt.Sprintf("<skill_content name=%q path=%q>\n%s\n</skill_content>", skill.Name, skill.Path, string(skillContent)),
	}, nil
}

func sortedNames(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}
