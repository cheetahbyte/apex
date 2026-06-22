package files

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cheetahbyte/apex/internal/tools"
)

type TreeTool struct{}

func (TreeTool) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        "dir_tree",
		Description: "Lists files and directories recursively as a tree structure",
		ReadOnly:    true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The directory path to list (default is current working directory)",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Maximum recursion depth (default is 3)",
				},
			},
			"additionalProperties": false,
		},
	}
}

func (TreeTool) Execute(ctx context.Context, input json.RawMessage) (tools.ToolResult, error) {
	var args struct {
		Path     string `json:"path"`
		MaxDepth int    `json:"max_depth"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return tools.ToolResult{}, err
	}

	if args.Path == "" {
		args.Path = "."
	}
	if args.MaxDepth <= 0 {
		args.MaxDepth = 3
	}

	root, err := filepath.Abs(args.Path)
	if err != nil {
		return tools.ToolResult{}, err
	}

	// TODO: Hier deine apex.yaml Sandbox-Prüfung einbauen, ob der Pfad erlaubt ist!

	var sb strings.Builder
	fileCount := 0
	const maxFiles = 500

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		// Harte ignores für bekannte pfade
		parts := strings.Split(rel, string(filepath.Separator))
		for _, part := range parts {
			if part == ".git" || part == "node_modules" || part == "dist" || part == "bin" {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		depth := len(parts)
		if depth > args.MaxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if fileCount >= maxFiles {
			sb.WriteString("... (tree truncated, too many files)\n")
			return filepath.SkipAll
		}

		indent := strings.Repeat("  ", depth-1)
		if d.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
			fileCount++
		}

		return nil
	})

	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to walk directory: %w", err)
	}

	return tools.ToolResult{Content: sb.String()}, nil
}
