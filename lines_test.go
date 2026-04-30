package main

import "testing"

// ---- isCommentLine tests ----

func TestIsCommentLine_GoStyleComment(t *testing.T) {
	if !isCommentLine("// comment") {
		t.Error("expected // to be recognized as comment")
	}
}

func TestIsCommentLine_PythonHash(t *testing.T) {
	if !isCommentLine("# comment") {
		t.Error("expected # to be recognized as comment")
	}
}

func TestIsCommentLine_HaskellLuaDash(t *testing.T) {
	if !isCommentLine("-- comment") {
		t.Error("expected -- to be recognized as comment")
	}
}

func TestIsCommentLine_BlockCommentStar(t *testing.T) {
	if !isCommentLine("* continuation") {
		t.Error("expected * to be recognized as comment")
	}
}

func TestIsCommentLine_BlockCommentStart(t *testing.T) {
	if !isCommentLine("/* start") {
		t.Error("expected /* to be recognized as comment")
	}
}

func TestIsCommentLine_OCamlComment(t *testing.T) {
	if !isCommentLine("(* comment") {
		t.Error("expected (* to be recognized as comment")
	}
}

func TestIsCommentLine_NotAComment(t *testing.T) {
	if isCommentLine("regular code") {
		t.Error("expected regular code to not be recognized as comment")
	}
}

func TestIsCommentLine_EmptyString(t *testing.T) {
	if isCommentLine("") {
		t.Error("expected empty string to not be recognized as comment")
	}
}

// ---- scanLeadingComments tests ----

func TestScanLeadingComments_NoComments(t *testing.T) {
	lines := []string{"package main", "func foo() {", "}"}
	got := scanLeadingComments(lines, 2)
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestScanLeadingComments_SingleComment(t *testing.T) {
	lines := []string{"package main", "// comment", "func foo() {", "}"}
	got := scanLeadingComments(lines, 3)
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestScanLeadingComments_MultipleComments(t *testing.T) {
	lines := []string{"package main", "// first", "// second", "// third", "func foo() {", "}"}
	got := scanLeadingComments(lines, 5)
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestScanLeadingComments_BlankLineBetween(t *testing.T) {
	lines := []string{"package main", "// comment", "", "func foo() {", "}"}
	got := scanLeadingComments(lines, 4)
	if got != 4 {
		t.Errorf("got %d, want 4 (should stop at blank line)", got)
	}
}

func TestScanLeadingComments_AtLineOne(t *testing.T) {
	lines := []string{"func foo() {", "}"}
	got := scanLeadingComments(lines, 1)
	if got != 1 {
		t.Errorf("got %d, want 1 (cannot go below line 1)", got)
	}
}

// ---- normalizeTagPattern tests ----

func TestNormalizeTagPattern_StripsAnchors(t *testing.T) {
	got := normalizeTagPattern("^func MyFunc() {$")
	want := "func MyFunc() {"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeTagPattern_UnescapesAsterisk(t *testing.T) {
	got := normalizeTagPattern(`^func (m \*MyStruct) Run() {$`)
	want := "func (m *MyStruct) Run() {"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeTagPattern_NoAnchors(t *testing.T) {
	got := normalizeTagPattern("func plain")
	if got != "func plain" {
		t.Errorf("got %q, want %q", got, "func plain")
	}
}

// ---- findPatternLine tests ----

func TestFindPatternLine_Found(t *testing.T) {
	lines := []string{"package main", "", "func MyFunc() {", "\treturn", "}"}
	// ctags-style pattern with anchors
	got := findPatternLine(lines, "^func MyFunc() {$")
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestFindPatternLine_NotFound(t *testing.T) {
	lines := []string{"foo", "bar", "baz"}
	got := findPatternLine(lines, "^nothere$")
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestFindPatternLine_FirstMatchWins(t *testing.T) {
	lines := []string{"func foo() {", "// func foo is here", "func foo() {"}
	got := findPatternLine(lines, "^func foo() {$")
	if got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestFindPatternLine_EmptyLines(t *testing.T) {
	got := findPatternLine([]string{}, "^anything$")
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestFindPatternLine_UnescapedPattern(t *testing.T) {
	lines := []string{"package p", "", `func (m *MyStruct) Run() error {`}
	got := findPatternLine(lines, `^func (m \*MyStruct) Run() error {$`)
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

// ---- extractLines tests ----

func TestExtractLines_Basic(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := extractLines(lines, 2, 4)
	want := "b\nc\nd"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractLines_SingleLine(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got := extractLines(lines, 2, 2)
	if got != "b" {
		t.Errorf("got %q, want %q", got, "b")
	}
}

func TestExtractLines_ClampsEndBeyondEOF(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got := extractLines(lines, 2, 100)
	want := "b\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractLines_ClampsStartBelowOne(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got := extractLines(lines, 0, 2)
	want := "a\nb"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractLines_FullFile(t *testing.T) {
	lines := []string{"x", "y", "z"}
	got := extractLines(lines, 1, 3)
	want := "x\ny\nz"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
