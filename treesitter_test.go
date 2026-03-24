package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sample JavaScript source used across unit tests.
var jsSample = []byte(`function greet(name) {
  return "Hello, " + name;
}

const add = (a, b) => {
  return a + b;
};

class Point {
  constructor(x, y) {
    this.x = x;
    this.y = y;
  }
  distance() {
    return Math.sqrt(this.x ** 2 + this.y ** 2);
  }
}
`)

// sample Rust source used across unit tests.
var rustSample = []byte(`fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

fn add(a: i32, b: i32) -> i32 {
    a + b
}

struct RustPoint {
    x: f64,
    y: f64,
}

impl RustPoint {
    fn new(x: f64, y: f64) -> Self {
        RustPoint { x, y }
    }
}
`)

// ---- resolveEndWithTreeSitterRust tests ----

func TestResolveEndWithTreeSitterRust_TopLevelFunction(t *testing.T) {
	// fn greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterRust(rustSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterRust_SecondFunction(t *testing.T) {
	// fn add starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterRust(rustSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterRust_Struct(t *testing.T) {
	// struct RustPoint starts at line 9, ends at line 12
	end, err := resolveEndWithTreeSitterRust(rustSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 12 {
		t.Errorf("end: got %d, want 12", end)
	}
}

func TestResolveEndWithTreeSitterRust_ImplBlock(t *testing.T) {
	// impl RustPoint starts at line 14, ends at line 18
	end, err := resolveEndWithTreeSitterRust(rustSample, 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 18 {
		t.Errorf("end: got %d, want 18", end)
	}
}

func TestResolveEndWithTreeSitterRust_Method(t *testing.T) {
	// fn new inside impl starts at line 15, ends at line 17
	end, err := resolveEndWithTreeSitterRust(rustSample, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 17 {
		t.Errorf("end: got %d, want 17", end)
	}
}

func TestResolveEndWithTreeSitterRust_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterRust(rustSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// ---- resolveEndWithTreeSitterJS tests ----

func TestResolveEndWithTreeSitterJS_FunctionDeclaration(t *testing.T) {
	// function greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterJS(jsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterJS_ArrowFunction(t *testing.T) {
	// const add = (a, b) => { ... } starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterJS(jsSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterJS_Class(t *testing.T) {
	// class Point starts at line 9, ends at line 17
	end, err := resolveEndWithTreeSitterJS(jsSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 17 {
		t.Errorf("end: got %d, want 17", end)
	}
}

func TestResolveEndWithTreeSitterJS_Method(t *testing.T) {
	// constructor starts at line 10, ends at line 13
	end, err := resolveEndWithTreeSitterJS(jsSample, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 13 {
		t.Errorf("end: got %d, want 13", end)
	}
}

func TestResolveEndWithTreeSitterJS_SecondMethod(t *testing.T) {
	// distance() starts at line 14, ends at line 16
	end, err := resolveEndWithTreeSitterJS(jsSample, 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterJS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterJS(jsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample TypeScript source used across unit tests.
var tsSample = []byte(`function greet(name: string): string {
  return "Hello, " + name;
}

const add = (a: number, b: number): number => {
  return a + b;
};

interface Shape {
  area(): number;
}

type Point = {
  x: number;
  y: number;
};

class Circle implements Shape {
  constructor(private radius: number) {}
  area(): number {
    return Math.PI * this.radius ** 2;
  }
}

enum Direction {
  Up,
  Down,
  Left,
  Right,
}
`)

// ---- resolveEndWithTreeSitterTS tests ----

func TestResolveEndWithTreeSitterTS_FunctionDeclaration(t *testing.T) {
	// function greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterTS(tsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterTS_ArrowFunction(t *testing.T) {
	// const add = ... starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterTS(tsSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterTS_Interface(t *testing.T) {
	// interface Shape starts at line 9, ends at line 11
	end, err := resolveEndWithTreeSitterTS(tsSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 11 {
		t.Errorf("end: got %d, want 11", end)
	}
}

func TestResolveEndWithTreeSitterTS_TypeAlias(t *testing.T) {
	// type Point = { ... }; starts at line 13, ends at line 16
	end, err := resolveEndWithTreeSitterTS(tsSample, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterTS_Class(t *testing.T) {
	// class Circle starts at line 18, ends at line 23
	end, err := resolveEndWithTreeSitterTS(tsSample, 18)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 23 {
		t.Errorf("end: got %d, want 23", end)
	}
}

func TestResolveEndWithTreeSitterTS_Method(t *testing.T) {
	// area() starts at line 20, ends at line 22
	end, err := resolveEndWithTreeSitterTS(tsSample, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 22 {
		t.Errorf("end: got %d, want 22", end)
	}
}

func TestResolveEndWithTreeSitterTS_Enum(t *testing.T) {
	// enum Direction starts at line 25, ends at line 30
	end, err := resolveEndWithTreeSitterTS(tsSample, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 30 {
		t.Errorf("end: got %d, want 30", end)
	}
}

func TestResolveEndWithTreeSitterTS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterTS(tsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample Haskell source used across unit tests.
// Content must match testdata/hs/sample.hs exactly.
var hsSample = []byte(`greet :: String -> String
greet name =
  "Hello, " ++ name ++ "!"

add :: Int -> Int -> Int
add x y = x + y

data Color
  = Red
  | Green
  | Blue

data Point = Point
  { px :: Double
  , py :: Double
  }

class Shape a where
  area :: a -> Double
  perimeter :: a -> Double
`)

// ---- resolveEndWithTreeSitterHS tests ----

func TestResolveEndWithTreeSitterHS_FunctionMultiLine(t *testing.T) {
	// greet function starts at line 2, ends at line 3
	end, err := resolveEndWithTreeSitterHS(hsSample, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterHS_FunctionSingleLine(t *testing.T) {
	// add function starts at line 6, ends at line 6
	end, err := resolveEndWithTreeSitterHS(hsSample, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 6 {
		t.Errorf("end: got %d, want 6", end)
	}
}

func TestResolveEndWithTreeSitterHS_TypeSignature(t *testing.T) {
	// greet type signature starts at line 1, ends at line 1
	end, err := resolveEndWithTreeSitterHS(hsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 1 {
		t.Errorf("end: got %d, want 1", end)
	}
}

func TestResolveEndWithTreeSitterHS_DataType(t *testing.T) {
	// data Color starts at line 8, ends at line 11
	end, err := resolveEndWithTreeSitterHS(hsSample, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 11 {
		t.Errorf("end: got %d, want 11", end)
	}
}

func TestResolveEndWithTreeSitterHS_DataTypeRecord(t *testing.T) {
	// data Point starts at line 13, ends at line 16
	end, err := resolveEndWithTreeSitterHS(hsSample, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterHS_Class(t *testing.T) {
	// class Shape starts at line 18, ends at line 20
	end, err := resolveEndWithTreeSitterHS(hsSample, 18)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 20 {
		t.Errorf("end: got %d, want 20", end)
	}
}

func TestResolveEndWithTreeSitterHS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterHS(hsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// ---- isRustFile / isJSFile tests ----

func TestIsRustFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"main.rs", true},
		{"src/lib.rs", true},
		{"main.go", false},
		{"sample.js", false},
		{"README.md", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isRustFile(c.path); got != c.want {
			t.Errorf("isRustFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsJSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"app.js", true},
		{"src/index.js", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isJSFile(c.path); got != c.want {
			t.Errorf("isJSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsTSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"app.ts", true},
		{"src/index.ts", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.js", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isTSFile(c.path); got != c.want {
			t.Errorf("isTSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsHSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"Main.hs", true},
		{"src/Lib.hs", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isHSFile(c.path); got != c.want {
			t.Errorf("isHSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// ---- HTTP handler integration tests for Rust files ----

func TestSnippetHandler_RustFile_UsesTreeSitter(t *testing.T) {
	// The rust/tags file has no "end" field, so tree-sitter must supply it.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 {
			t.Errorf("Start: got %d, want 1", s.Start)
		}
		if s.End != 3 {
			t.Errorf("End: got %d, want 3 (tree-sitter should resolve this)", s.End)
		}
		if !strings.Contains(s.Code, "fn greet") {
			t.Errorf("Code should contain fn greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "fn add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestLinesHandler_RustFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/add?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		lr := ranges[0]
		if lr.Start != 5 {
			t.Errorf("Start: got %d, want 5", lr.Start)
		}
		if lr.End != 7 {
			t.Errorf("End: got %d, want 7 (tree-sitter should resolve this)", lr.End)
		}
	})
}

func TestSnippetHandler_RustFile_Method(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/new?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 15 || snippets[0].End != 17 {
			t.Errorf("Start/End: got %d/%d, want 15/17", snippets[0].Start, snippets[0].End)
		}
	})
}

// ---- HTTP handler integration tests for JavaScript files ----

func TestSnippetHandler_JSFile_FunctionDeclaration(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "function greet") {
			t.Errorf("Code should contain function greet, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_JSFile_ArrowFunction(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/add?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 5 || snippets[0].End != 7 {
			t.Errorf("Start/End: got %d/%d, want 5/7", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_JSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Point?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 9 || snippets[0].End != 17 {
			t.Errorf("Start/End: got %d/%d, want 9/17", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_JSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/distance?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 14 || ranges[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 14/16", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for TypeScript files ----

func TestSnippetHandler_TSFile_FunctionDeclaration(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "function greet") {
			t.Errorf("Code should contain function greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "const add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_TSFile_Interface(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Shape?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 9 || snippets[0].End != 11 {
			t.Errorf("Start/End: got %d/%d, want 9/11", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_TypeAlias(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Point?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 13 || snippets[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 13/16", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Circle?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 18 || snippets[0].End != 23 {
			t.Errorf("Start/End: got %d/%d, want 18/23", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_Enum(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Direction?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 25 || snippets[0].End != 30 {
			t.Errorf("Start/End: got %d/%d, want 25/30", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_TSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/add?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 5 || ranges[0].End != 7 {
			t.Errorf("Start/End: got %d/%d, want 5/7", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for Haskell files ----

func TestSnippetHandler_HSFile_FunctionMultiLine(t *testing.T) {
	// greet function spans lines 2-3; tree-sitter must resolve the end.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 2 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 2/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "greet name") {
			t.Errorf("Code should contain greet name, got %q", s.Code)
		}
		if strings.Contains(s.Code, "add x y") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_HSFile_FunctionSingleLine(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/add?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 6 || snippets[0].End != 6 {
			t.Errorf("Start/End: got %d/%d, want 6/6", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_HSFile_DataType(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Color?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 8 || snippets[0].End != 11 {
			t.Errorf("Start/End: got %d/%d, want 8/11", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_HSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Shape?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 18 || snippets[0].End != 20 {
			t.Errorf("Start/End: got %d/%d, want 18/20", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_HSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/Point?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 13 || ranges[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 13/16", ranges[0].Start, ranges[0].End)
		}
	})
}
