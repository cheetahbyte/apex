package files

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cheetahbyte/apex/internal/tools"
)

type ReadFileTool struct{}

const (
	maxReadLines  = 2000
	maxReadBytes  = 50 * 1024
	maxLineLength = 2000

	maxLineSuffix = "... (line truncated to 2000 chars)"
)

var binaryExtensions = map[string]struct{}{
	".7z": {}, ".a": {}, ".bin": {}, ".class": {}, ".dat": {}, ".dll": {}, ".doc": {}, ".docx": {},
	".dylib": {}, ".exe": {}, ".gif": {}, ".gz": {}, ".ico": {}, ".jar": {}, ".jpeg": {}, ".jpg": {},
	".lib": {}, ".o": {}, ".obj": {}, ".odp": {}, ".ods": {}, ".odt": {}, ".pdf": {}, ".png": {},
	".ppt": {}, ".pptx": {}, ".pyc": {}, ".pyo": {}, ".so": {}, ".tar": {}, ".wasm": {}, ".webp": {},
	".xls": {}, ".xlsx": {}, ".zip": {},
}

type readFileArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (ReadFileTool) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        "read_file",
		Description: "Read a UTF-8 text file with optional line paging, or list a directory, returning plain text output.",
		ReadOnly:    true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The file or directory path to read. Relative paths resolve from the current working directory.",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "The 1-based directory entry or text line offset to start reading from.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "The maximum number of directory entries or text lines to read. Capped at 2000.",
				},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	}
}

func (ReadFileTool) Execute(ctx context.Context, input json.RawMessage) (tools.ToolResult, error) {
	var args readFileArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return tools.ToolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	requestedPath := strings.TrimSpace(args.Path)
	if requestedPath == "" {
		return tools.ToolResult{}, fmt.Errorf("missing path")
	}

	abs, err := filepath.Abs(requestedPath)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("unable to stat %s: %w", abs, err)
	}

	offset, limit, err := normalizePage(args.Offset, args.Limit)
	if err != nil {
		return tools.ToolResult{}, err
	}

	select {
	case <-ctx.Done():
		return tools.ToolResult{}, ctx.Err()
	default:
	}

	if info.IsDir() {
		content, err := readDirectory(requestedPath, abs, offset, limit)
		if err != nil {
			return tools.ToolResult{}, err
		}
		return tools.ToolResult{Content: content}, nil
	}
	if !info.Mode().IsRegular() {
		return tools.ToolResult{}, fmt.Errorf("path is not a file or directory: %s", abs)
	}

	content, err := readTextFile(requestedPath, abs, info.Size(), offset, limit, args.Offset != 0 || args.Limit != 0)
	if err != nil {
		return tools.ToolResult{}, err
	}
	return tools.ToolResult{Content: content}, nil
}

func normalizePage(offset, limit int) (int, int, error) {
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must be positive")
	}
	if limit < 0 {
		return 0, 0, fmt.Errorf("limit must be positive")
	}
	if offset == 0 {
		offset = 1
	}
	if limit == 0 || limit > maxReadLines {
		limit = maxReadLines
	}
	return offset, limit, nil
}

func readDirectory(requestedPath, absPath string, offset, limit int) (string, error) {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("unable to read directory %s: %w", absPath, err)
	}

	type visibleEntry struct {
		name  string
		isDir bool
	}
	visible := make([]visibleEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		isDir := info.IsDir()
		if !isDir && !info.Mode().IsRegular() {
			continue
		}
		name := entry.Name()
		if isDir {
			name += string(os.PathSeparator)
		}
		visible = append(visible, visibleEntry{name: name, isDir: isDir})
	}
	sort.Slice(visible, func(i, j int) bool {
		if visible[i].isDir != visible[j].isDir {
			return visible[i].isDir
		}
		return visible[i].name < visible[j].name
	})

	start := offset - 1
	if start >= len(visible) && offset != 1 {
		return "", fmt.Errorf("offset %d is out of range", offset)
	}
	end := start + limit
	if end > len(visible) {
		end = len(visible)
	}
	truncated := end < len(visible)
	var next int
	if truncated {
		next = offset + (end - start)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Read directory: %s\nType: directory\nOffset: %d\nTruncated: %t\n", requestedPath, offset, truncated)
	if truncated {
		fmt.Fprintf(&b, "Next: %d\n", next)
	}
	b.WriteString("Entries:\n")
	for _, entry := range visible[start:end] {
		b.WriteString(entry.name)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func readTextFile(requestedPath, absPath string, size int64, offset, limit int, pageRequested bool) (string, error) {
	if isBinaryExtension(absPath) {
		return "", fmt.Errorf("cannot read binary file: %s", absPath)
	}

	file, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("unable to open %s: %w", absPath, err)
	}
	defer file.Close()

	first, err := readPrefix(file, 64*1024)
	if err != nil {
		return "", err
	}
	if looksBinary(first) {
		return "", fmt.Errorf("cannot read binary file: %s", absPath)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("unable to seek %s: %w", absPath, err)
	}

	if !pageRequested && size <= maxReadBytes {
		data, err := io.ReadAll(io.LimitReader(file, maxReadBytes+1))
		if err != nil {
			return "", fmt.Errorf("unable to read %s: %w", absPath, err)
		}
		if len(data) > maxReadBytes {
			return formatPagedFile(requestedPath, absPath, offset, limit, data)
		}
		if !utf8.Valid(data) {
			return "", fmt.Errorf("file is not valid UTF-8: %s", absPath)
		}
		return formatTextFile(requestedPath, string(data)), nil
	}

	return readPagedFile(requestedPath, file, offset, limit)
}

func readPrefix(file *os.File, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(file, max))
	if err != nil {
		return nil, fmt.Errorf("unable to read file prefix: %w", err)
	}
	return data, nil
}

func formatTextFile(requestedPath, content string) string {
	return fmt.Sprintf("Read file: %s\nType: file\nTruncated: false\n\nContent:\n%s\n", requestedPath, content)
}

func readPagedFile(requestedPath string, file io.Reader, offset, limit int) (string, error) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxReadBytes+maxLineLength+len(maxLineSuffix))

	lines := make([]string, 0, limit)
	lineNo := 0
	bytesUsed := 0
	truncated := false
	next := 0

	for scanner.Scan() {
		lineNo++
		if lineNo < offset {
			continue
		}
		if len(lines) >= limit || bytesUsed >= maxReadBytes {
			truncated = true
			next = lineNo
			break
		}
		line := scanner.Text()
		if !utf8.ValidString(line) {
			return "", fmt.Errorf("file is not valid UTF-8: %s", requestedPath)
		}
		line = truncateLine(line)
		size := len(line) + 1
		if bytesUsed+size > maxReadBytes {
			truncated = true
			next = lineNo
			break
		}
		lines = append(lines, fmt.Sprintf("%d: %s", lineNo, line))
		bytesUsed += size
	}
	if err := scanner.Err(); err != nil {
		if err == bufio.ErrTooLong {
			return "", fmt.Errorf("line too long in %s", requestedPath)
		}
		return "", fmt.Errorf("unable to read %s: %w", requestedPath, err)
	}
	if len(lines) == 0 && offset != 1 {
		return "", fmt.Errorf("offset %d is out of range", offset)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Read file: %s\nType: file\nOffset: %d\nTruncated: %t\n", requestedPath, offset, truncated)
	if truncated {
		fmt.Fprintf(&b, "Next: %d\n", next)
	}
	b.WriteString("\nContent:\n")
	b.WriteString(strings.Join(lines, "\n"))
	if len(lines) > 0 {
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func formatPagedFile(requestedPath, absPath string, offset, limit int, data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8: %s", absPath)
	}
	return readPagedFile(requestedPath, bytes.NewReader(data), offset, limit)
}

func truncateLine(line string) string {
	if len(line) <= maxLineLength {
		return line
	}
	return line[:maxLineLength] + maxLineSuffix
}

func isBinaryExtension(path string) bool {
	_, ok := binaryExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	nonPrintable := 0
	for _, b := range data {
		if b < 9 || (b > 13 && b < 32) {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(data)) > 0.3
}
