package main

import "testing"

func TestLookupLang_AllExtensions(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".go", "Go"},
		{".py", "Python"},
		{".rb", "Ruby"},
		{".java", "Java"},
		{".rs", "Rust"},
		{".js", "JavaScript"},
		{".ts", "TypeScript"},
		{".hs", "Haskell"},
		{".kt", "Kotlin"},
		{".php", "PHP"},
		{".ml", "OCaml"},
		{".mli", "OCaml Interface"},
		{".c", "C"},
		{".h", "C"},
		{".cc", "C++"},
		{".cpp", "C++"},
		{".cxx", "C++"},
		{".hh", "C++"},
		{".hpp", "C++"},
		{".hxx", "C++"},
		{".lua", "Lua"},
	}
	for _, tt := range tests {
		entry := lookupLang("test" + tt.ext)
		if entry == nil {
			t.Errorf("lookupLang(%q) returned nil", tt.ext)
			continue
		}
		if entry.name != tt.want {
			t.Errorf("lookupLang(%q).name = %q, want %q", tt.ext, entry.name, tt.want)
		}
	}
}

func TestLookupLang_UnknownExtension(t *testing.T) {
	tests := []string{".txt", ".md", ".unknown", ""}
	for _, ext := range tests {
		entry := lookupLang("file" + ext)
		if entry != nil {
			t.Errorf("lookupLang(%q) = %v, want nil", ext, entry)
		}
	}
}

func TestResolveEndForLang_UsesCustomOverrideForLua(t *testing.T) {
	luaSample := []byte(`-- comment
function foo()
    print("hello")
end
`)
	entry := lookupLang("test.lua")
	if entry == nil {
		t.Fatal("lookupLang returned nil for .lua")
	}
	if entry.resolveEnd == nil {
		t.Error("expected Lua entry to have custom resolveEnd, got nil")
	}
	end, err := entry.resolveEndForLang(luaSample, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 4 {
		t.Errorf("got end line %d, want 4", end)
	}
}

func TestResolveStartForLang_UsesCustomOverrideForLua(t *testing.T) {
	luaSample := []byte(`-- comment
function foo()
    print("hello")
end
`)
	entry := lookupLang("test.lua")
	if entry == nil {
		t.Fatal("lookupLang returned nil for .lua")
	}
	if entry.resolveStart == nil {
		t.Error("expected Lua entry to have custom resolveStart, got nil")
	}
	start, err := entry.resolveStartForLang(luaSample, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 1 {
		t.Errorf("got start line %d, want 1", start)
	}
}

func TestResolveEndForLang_StandardLanguage(t *testing.T) {
	goSample := []byte(`package main

// MyFunc does something.
func MyFunc() {
	println("hello")
}
`)
	entry := lookupLang("test.go")
	if entry == nil {
		t.Fatal("lookupLang returned nil for .go")
	}
	if entry.resolveEnd != nil {
		t.Error("expected Go entry to use generic resolveEnd, but has custom override")
	}
	end, err := entry.resolveEndForLang(goSample, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 6 {
		t.Errorf("got end line %d, want 6", end)
	}
}
