package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// writeTemp creates a temporary file with the given content and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "src*.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// ---- snippetForTag tests ----

func TestSnippetForTag_WithLineAndEndField(t *testing.T) {
	src := "package p\n\nfunc Greet() {\n\treturn\n}\n\nvar x = 1\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Greet",
		Path:  path,
		Line:  3,
		Extra: map[string]string{"end": "5"},
	}
	s, err := snippetForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Start != 3 {
		t.Errorf("Start: got %d, want 3", s.Start)
	}
	if s.End != 5 {
		t.Errorf("End: got %d, want 5", s.End)
	}
	if !strings.Contains(s.Code, "func Greet") {
		t.Errorf("Code should contain func Greet, got %q", s.Code)
	}
	if strings.Contains(s.Code, "var x") {
		t.Errorf("Code should not contain lines beyond end, got %q", s.Code)
	}
}

func TestSnippetForTag_WithPatternAndEndField(t *testing.T) {
	src := "package p\n\nfunc Hello() {\n}\n\nvar y = 2\n"
	path := writeTemp(t, src)

	// Line is 0, so pattern search is used; pattern uses ctags-style anchors
	tag := Tag{
		Name:    "Hello",
		Path:    path,
		Pattern: "^func Hello() {$",
		Extra:   map[string]string{"end": "4"},
	}
	s, err := snippetForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Start != 3 {
		t.Errorf("Start: got %d, want 3", s.Start)
	}
	if s.End != 4 {
		t.Errorf("End: got %d, want 4", s.End)
	}
}

func TestSnippetForTag_WithoutEndField_ReturnsZeroEnd(t *testing.T) {
	src := "line1\nfunc Foo() {\n\treturn\n}\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Foo",
		Path:  path,
		Line:  2,
		Extra: map[string]string{},
	}
	s, err := snippetForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Start != 2 {
		t.Errorf("Start: got %d, want 2", s.Start)
	}
	// End must be 0 when Extra["end"] is absent and tree-sitter is disabled
	if s.End != 0 {
		t.Errorf("End: got %d, want 0", s.End)
	}
}

func TestSnippetForTag_WithoutEndField_CodeIsSingleLine(t *testing.T) {
	src := "line1\nfunc Foo() {\n\treturn\n}\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Foo",
		Path:  path,
		Line:  2,
		Extra: map[string]string{},
	}
	s, err := snippetForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	// Code must contain only the single start line, not the full function body
	if s.Code != "func Foo() {" {
		t.Errorf("Code: got %q, want %q", s.Code, "func Foo() {")
	}
}

func TestSnippetForTag_FileNotFound(t *testing.T) {
	tag := Tag{
		Name:  "Foo",
		Path:  "/nonexistent/path/src.go",
		Line:  1,
		Extra: map[string]string{},
	}
	_, err := snippetForTag(context.Background(), tag, ".", false)
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

func TestSnippetForTag_PatternNotFoundInFile(t *testing.T) {
	src := "package p\n\nfunc Bar() {}\n"
	path := writeTemp(t, src)

	// Line is 0 and pattern doesn't match → error
	tag := Tag{
		Name:    "Foo",
		Path:    path,
		Pattern: "func Foo",
		Extra:   map[string]string{},
	}
	_, err := snippetForTag(context.Background(), tag, ".", false)
	if err == nil {
		t.Fatal("expected error when pattern not found")
	}
	if !strings.Contains(err.Error(), "cannot determine start line") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- lineRangeForTag tests ----

func TestLineRangeForTag_WithLineAndEndField(t *testing.T) {
	src := "package p\n\nfunc Greet() {\n\treturn\n}\n\nvar x = 1\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Greet",
		Path:  path,
		Line:  3,
		Extra: map[string]string{"end": "5"},
	}
	lr, err := lineRangeForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if lr.Start != 3 {
		t.Errorf("Start: got %d, want 3", lr.Start)
	}
	if lr.End != 5 {
		t.Errorf("End: got %d, want 5", lr.End)
	}
	if lr.Name != "Greet" {
		t.Errorf("Name: got %q, want Greet", lr.Name)
	}
	if lr.Path != path {
		t.Errorf("Path: got %q, want %q", lr.Path, path)
	}
}

func TestLineRangeForTag_WithPatternAndEndField(t *testing.T) {
	src := "package p\n\nfunc Hello() {\n}\n\nvar y = 2\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:    "Hello",
		Path:    path,
		Pattern: "^func Hello() {$",
		Extra:   map[string]string{"end": "4"},
	}
	lr, err := lineRangeForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if lr.Start != 3 {
		t.Errorf("Start: got %d, want 3", lr.Start)
	}
	if lr.End != 4 {
		t.Errorf("End: got %d, want 4", lr.End)
	}
}

func TestLineRangeForTag_WithoutEndField_ReturnsZeroEnd(t *testing.T) {
	src := "line1\nfunc Foo() {\n\treturn\n}\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Foo",
		Path:  path,
		Line:  2,
		Extra: map[string]string{},
	}
	lr, err := lineRangeForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if lr.Start != 2 {
		t.Errorf("Start: got %d, want 2", lr.Start)
	}
	// End must be 0 when Extra["end"] is absent and tree-sitter is disabled
	if lr.End != 0 {
		t.Errorf("End: got %d, want 0", lr.End)
	}
}

func TestLineRangeForTag_FileNotFound(t *testing.T) {
	tag := Tag{
		Name:    "Foo",
		Path:    "/nonexistent/path/src.go",
		Pattern: "^func Foo() {$",
		Extra:   map[string]string{},
	}
	_, err := lineRangeForTag(context.Background(), tag, ".", false)
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

func TestLineRangeForTag_NoCodeField(t *testing.T) {
	src := "package p\n\nfunc Bar() {\n\treturn\n}\n"
	path := writeTemp(t, src)

	tag := Tag{
		Name:  "Bar",
		Path:  path,
		Line:  3,
		Extra: map[string]string{"end": "5"},
	}
	lr, err := lineRangeForTag(context.Background(), tag, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	// LineRange must not contain a Code field
	b, _ := json.Marshal(lr)
	var m map[string]any
	json.Unmarshal(b, &m)
	if _, ok := m["code"]; ok {
		t.Error("LineRange JSON must not contain a 'code' field")
	}
}
