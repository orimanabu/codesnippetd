package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeReadtags(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "readtags")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake readtags: %v", err)
	}
	t.Setenv("PATH", dir)
}

// ---- parseLine tests ----

func TestParseLine_SkipsMetadataLines(t *testing.T) {
	lines := []string{
		"!_TAG_FILE_FORMAT\t2\t/extended format/",
		"!_TAG_FILE_SORTED\t1\t/0=unsorted/",
		"",
	}
	for _, line := range lines {
		if _, ok := parseLine(line); ok {
			t.Errorf("expected parseLine(%q) to be skipped", line)
		}
	}
}

func TestParseLine_BasicFunction(t *testing.T) {
	line := "MyFunc\tsample.go\t/^func MyFunc() {$/;\"\tkind:function\tline:10\tlanguage:Go"
	tag, ok := parseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if tag.Type != "tag" {
		t.Errorf("Type: got %q, want %q", tag.Type, "tag")
	}
	if tag.Name != "MyFunc" {
		t.Errorf("Name: got %q, want %q", tag.Name, "MyFunc")
	}
	if tag.Path != "sample.go" {
		t.Errorf("Path: got %q, want %q", tag.Path, "sample.go")
	}
	if tag.Pattern != "^func MyFunc() {$" {
		t.Errorf("Pattern: got %q, want %q", tag.Pattern, "^func MyFunc() {$")
	}
	if tag.Kind != "function" {
		t.Errorf("Kind: got %q, want %q", tag.Kind, "function")
	}
	if tag.Line != 10 {
		t.Errorf("Line: got %d, want %d", tag.Line, 10)
	}
	if tag.Language != "Go" {
		t.Errorf("Language: got %q, want %q", tag.Language, "Go")
	}
}

func TestParseLine_LineNumberAddress(t *testing.T) {
	// Some ctags formats use a plain line number as the address
	line := "myVar\tsample.go\t42\tkind:variable\tlanguage:Go"
	tag, ok := parseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if tag.Line != 42 {
		t.Errorf("Line: got %d, want 42", tag.Line)
	}
	if tag.Pattern != "" {
		t.Errorf("Pattern should be empty when address is a line number, got %q", tag.Pattern)
	}
}

func TestParseLine_QuestionMarkPattern(t *testing.T) {
	line := "backSearch\tsample.go\t?^func backSearch?;\"\tkind:function\tline:5\tlanguage:Go"
	tag, ok := parseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if tag.Pattern != "^func backSearch" {
		t.Errorf("Pattern: got %q, want %q", tag.Pattern, "^func backSearch")
	}
}

func TestParseLine_ExtraFields(t *testing.T) {
	line := "Run\tsample.go\t/^func (m \\*MyStruct) Run() error {$/;\"\tkind:method\tline:17\tlanguage:Go\ttyperef:typename:error"
	tag, ok := parseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if tag.Extra["typeref"] != "typename:error" {
		t.Errorf("Extra[typeref]: got %q, want %q", tag.Extra["typeref"], "typename:error")
	}
}

func TestParseLine_TooFewFields(t *testing.T) {
	if _, ok := parseLine("onlyone"); ok {
		t.Error("expected ok=false for line with too few fields")
	}
	if _, ok := parseLine("two\tfields"); ok {
		t.Error("expected ok=false for line with only two fields")
	}
}

func TestParseLine_PatternWithEmbeddedTab(t *testing.T) {
	// Pattern field contains a literal tab (e.g. an indented method signature).
	// Real-world example from a Go interface tags file:
	//   run\tpkg/machine/e2e/config_test.go\t/^	run() (*machineSession, error)$/;"\tn\tinterface:...
	line := "run\tpkg/machine/e2e/config_test.go\t/^\trun() (*machineSession, error)$/;\"\tn\tinterface:e2e_test.MachineTestBuilder\ttyperef:typename:(*machineSession, error)"
	tag, ok := parseLine(line)
	if !ok {
		t.Fatal("expected ok=true for pattern with embedded tab")
	}
	if tag.Name != "run" {
		t.Errorf("Name: got %q, want %q", tag.Name, "run")
	}
	if tag.Path != "pkg/machine/e2e/config_test.go" {
		t.Errorf("Path: got %q, want %q", tag.Path, "pkg/machine/e2e/config_test.go")
	}
	wantPattern := "^\trun() (*machineSession, error)$"
	if tag.Pattern != wantPattern {
		t.Errorf("Pattern: got %q, want %q", tag.Pattern, wantPattern)
	}
	if tag.Kind != "n" {
		t.Errorf("Kind: got %q, want %q", tag.Kind, "n")
	}
	if tag.Extra["interface"] != "e2e_test.MachineTestBuilder" {
		t.Errorf("Extra[interface]: got %q, want %q", tag.Extra["interface"], "e2e_test.MachineTestBuilder")
	}
	if tag.Extra["typeref"] != "typename:(*machineSession, error)" {
		t.Errorf("Extra[typeref]: got %q, want %q", tag.Extra["typeref"], "typename:(*machineSession, error)")
	}
}

// ---- loadTagsFile tests ----

func TestLoadTagsFile_NotFound(t *testing.T) {
	_, err := loadTagsFile("/nonexistent/path/tags")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("expected file-not-found error, got: %v", err)
	}
}

func TestLoadTagsFile_ParsesTagsFile(t *testing.T) {
	db, err := loadTagsFile(filepath.Join("testdata", "tags"))
	if err != nil {
		t.Fatalf("loadTagsFile: %v", err)
	}

	tags := db.lookup("MyStruct")
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag for MyStruct, got %d", len(tags))
	}
	if tags[0].Kind != "type" {
		t.Errorf("Kind: got %q, want %q", tags[0].Kind, "type")
	}
	if tags[0].Language != "Go" {
		t.Errorf("Language: got %q, want %q", tags[0].Language, "Go")
	}
}

func TestLoadTagsFile_MultipleTagsSameName(t *testing.T) {
	db, err := loadTagsFile(filepath.Join("testdata", "tags"))
	if err != nil {
		t.Fatalf("loadTagsFile: %v", err)
	}

	tags := db.lookup("overloaded")
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags for overloaded, got %d", len(tags))
	}
}

func TestLookupTag_FallsBackWhenReadtagsUnavailable(t *testing.T) {
	// Temporarily disable readtags to test fallback behavior
	originalReadtagsAvailable := readtagsAvailable
	readtagsAvailable = false
	t.Cleanup(func() { readtagsAvailable = originalReadtagsAvailable })

	tags, err := lookupTag(filepath.Join("testdata", "tags"), "MyStruct")
	if err != nil {
		t.Fatalf("lookupTag: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "MyStruct" {
		t.Errorf("Name: got %q, want %q", tags[0].Name, "MyStruct")
	}
}

func TestLookupTag_UsesReadtagsWhenAvailable(t *testing.T) {
	// Temporarily enable readtags and install fake binary
	originalReadtagsAvailable := readtagsAvailable
	readtagsAvailable = true
	t.Cleanup(func() { readtagsAvailable = originalReadtagsAvailable })

	writeFakeReadtags(t, "#!/bin/sh\nprintf '%s\\n' 'SubFunc\tsub/sub.go\t/^func SubFunc() {$/;\"\tkind:function\tline:3\tlanguage:Go'\n")

	tags, err := lookupTag(filepath.Join("testdata", "sub", "tags"), "SubFunc")
	if err != nil {
		t.Fatalf("lookupTag: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Path != "sub/sub.go" {
		t.Errorf("Path: got %q, want %q", tags[0].Path, "sub/sub.go")
	}
}

func TestLookupWithReadtags_ReturnsCommandError(t *testing.T) {
	writeFakeReadtags(t, "#!/bin/sh\necho 'boom' >&2\nexit 1\n")

	_, err := lookupWithReadtags(filepath.Join("testdata", "tags"), "MyStruct")
	if err == nil {
		t.Fatal("expected error from readtags command")
	}
	if !strings.Contains(err.Error(), "readtags: boom") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLookupWithReadtags_ParsesOutput(t *testing.T) {
	writeFakeReadtags(t, "#!/bin/sh\nprintf '%s\\n' 'Run\tsample.go\t/^func (m \\*MyStruct) Run() error {$/;\"\tkind:method\tline:17\tlanguage:Go\ttyperef:typename:error'\n")

	tags, err := lookupWithReadtags(filepath.Join("testdata", "tags"), "Run")
	if err != nil {
		t.Fatalf("lookupWithReadtags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Extra["typeref"] != "typename:error" {
		t.Errorf("Extra[typeref]: got %q, want %q", tags[0].Extra["typeref"], "typename:error")
	}
}
