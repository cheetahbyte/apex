package files

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeReadFile(t *testing.T, args readFileArgs) (string, error) {
	t.Helper()
	input, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	result, err := (ReadFileTool{}).Execute(context.Background(), input)
	return result.Content, err
}

func TestReadFileToolReadsSmallTextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello\nworld"), 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := executeReadFile(t, readFileArgs{Path: path})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(content, "Read file:") || !strings.Contains(content, "hello\nworld") {
		t.Fatalf("unexpected content:\n%s", content)
	}
	if !strings.Contains(content, "Type: file") || !strings.Contains(content, "Truncated: false") {
		t.Fatalf("expected non-truncated content:\n%s", content)
	}
}

func TestReadFileToolPaginatesTextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := executeReadFile(t, readFileArgs{Path: path, Offset: 2, Limit: 2})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	for _, want := range []string{"2: two", "3: three", "Truncated: true", "Next: 4"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in content:\n%s", want, content)
		}
	}
	if strings.Contains(content, "1: one") || strings.Contains(content, "4: four") {
		t.Fatalf("unexpected paged lines:\n%s", content)
	}
}

func TestReadFileToolCapsLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "many.txt")
	var b strings.Builder
	for i := 0; i < maxReadLines+2; i++ {
		b.WriteString("x\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := executeReadFile(t, readFileArgs{Path: path, Limit: maxReadLines + 100})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if strings.Count(content, ": x") != maxReadLines {
		t.Fatalf("expected %d lines, got content:\n%s", maxReadLines, content)
	}
	if !strings.Contains(content, "Next: 2001") {
		t.Fatalf("expected capped next offset:\n%s", content)
	}
}

func TestReadFileToolTruncatesLongLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", maxLineLength+50)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := executeReadFile(t, readFileArgs{Path: path, Limit: 1})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(content, maxLineSuffix) {
		t.Fatalf("expected truncated line suffix:\n%s", content)
	}
}

func TestReadFileToolListsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := executeReadFile(t, readFileArgs{Path: dir})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	wantOrder := []string{"subdir" + string(os.PathSeparator), "a.txt", "b.txt"}
	last := -1
	for _, want := range wantOrder {
		idx := strings.Index(content, want)
		if idx == -1 {
			t.Fatalf("expected %q in content:\n%s", want, content)
		}
		if idx < last {
			t.Fatalf("directory entries out of order:\n%s", content)
		}
		last = idx
	}
}

func TestReadFileToolRejectsBinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(path, []byte{0x00, 0x01, 0x02}, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := executeReadFile(t, readFileArgs{Path: path})
	if err == nil || !strings.Contains(err.Error(), "cannot read binary file") {
		t.Fatalf("expected binary error, got %v", err)
	}
}

func TestReadFileToolRejectsInvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(path, []byte{0xff, 0xfe, '\n'}, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := executeReadFile(t, readFileArgs{Path: path})
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("expected UTF-8 error, got %v", err)
	}
}

func TestReadFileToolMissingPath(t *testing.T) {
	_, err := executeReadFile(t, readFileArgs{})
	if err == nil || !strings.Contains(err.Error(), "missing path") {
		t.Fatalf("expected missing path error, got %v", err)
	}
}

func TestReadFileToolOffsetOutOfRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "one.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := executeReadFile(t, readFileArgs{Path: path, Offset: 10})
	if err == nil || !strings.Contains(err.Error(), "offset 10 is out of range") {
		t.Fatalf("expected offset error, got %v", err)
	}
}

func TestReadFileToolSchemaRequiresPath(t *testing.T) {
	spec := (ReadFileTool{}).Spec()
	required, ok := spec.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("expected required []string, got %#v", spec.Parameters["required"])
	}
	if len(required) != 1 || required[0] != "path" {
		t.Fatalf("expected path required, got %#v", required)
	}
}
