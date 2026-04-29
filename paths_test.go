package main

import (
	"os/user"
	"path/filepath"
	"testing"
)

// ---- tagsFileForContext tests ----

func TestTagsFileForContext_Empty(t *testing.T) {
	got := tagsFileForContext("")
	want := filepath.Join(".", "tags")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTagsFileForContext_WithPath(t *testing.T) {
	got := tagsFileForContext("sub/project")
	want := filepath.Join(".", "sub", "project", "tags")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestTagsFileForContext_Slashes verifies that context values with slashes are
// joined correctly so there is no double-separator in the resulting path.
func TestTagsFileForContext_Slashes(t *testing.T) {
	got := tagsFileForContext("a/b")
	want := filepath.Join(".", "a", "b", "tags")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestTagsFileForContext_AbsolutePath verifies that an absolute path context is used
// directly without prepending ".".
func TestTagsFileForContext_AbsolutePath(t *testing.T) {
	got := tagsFileForContext("/home/user/myproject")
	want := "/home/user/myproject/tags"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---- resolveTagsPath tests ----

func TestResolveTagsPath_TagsParamTakesPrecedence(t *testing.T) {
	got := resolveTagsPath("some/context", "/explicit/path/tags")
	want := "/explicit/path/tags"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTagsPath_FallsBackToContext(t *testing.T) {
	got := resolveTagsPath("sub/project", "")
	want := filepath.Join(".", "sub", "project", "tags")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTagsPath_BothEmpty(t *testing.T) {
	got := resolveTagsPath("", "")
	want := filepath.Join(".", "tags")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveTagsPath_RelativeTagsParam(t *testing.T) {
	got := resolveTagsPath("", "custom/tags")
	if got != "custom/tags" {
		t.Errorf("got %q, want %q", got, "custom/tags")
	}
}

// ---- expandTilde tests ----

func TestExpandTilde_NoTilde(t *testing.T) {
	got, err := expandTilde("/absolute/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/absolute/path" {
		t.Errorf("got %q, want %q", got, "/absolute/path")
	}
}

func TestExpandTilde_EmptyString(t *testing.T) {
	got, err := expandTilde("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestExpandTilde_TildeAlone(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	got, err := expandTilde("~")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != u.HomeDir {
		t.Errorf("got %q, want %q", got, u.HomeDir)
	}
}

func TestExpandTilde_TildeWithPath(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	got, err := expandTilde("~/projects/myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := u.HomeDir + "/projects/myrepo"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandTilde_TildeInMiddle(t *testing.T) {
	// ~ not at the start must not be expanded
	got, err := expandTilde("/path/to/~/file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/path/to/~/file" {
		t.Errorf("got %q, want %q", got, "/path/to/~/file")
	}
}
