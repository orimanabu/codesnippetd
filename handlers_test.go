package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

// withCwd temporarily changes the working directory to dir for the duration of fn.
func withCwd(t *testing.T, dir string, fn func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	fn()
}

// newHandler returns an http.Handler with all routes registered (via registerHandlers)
// and wrapped in the access log middleware.
func newHandler(useTreeSitter bool) http.Handler {
	mux := http.NewServeMux()
	registerHandlers(mux, useTreeSitter)
	return accessLog(mux)
}

// captureLog redirects the default logger to a buffer for the duration of fn.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	fn()
	return buf.String()
}

// ---- /healthz handler tests ----

func TestHealthz_ReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	newHandler(false).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field: got %q, want %q", body["status"], "ok")
	}
}

// ---- accessLog middleware tests ----

func TestAccessLog_LogsMethodAndPath(t *testing.T) {
	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		newHandler(false).ServeHTTP(w, req)
	})
	if !strings.Contains(output, "GET") {
		t.Errorf("log output missing method GET: %q", output)
	}
	if !strings.Contains(output, "/healthz") {
		t.Errorf("log output missing path /healthz: %q", output)
	}
}

func TestAccessLog_LogsStatusCode(t *testing.T) {
	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		newHandler(false).ServeHTTP(w, req)
	})
	if !strings.Contains(output, "200") {
		t.Errorf("log output missing status 200: %q", output)
	}
}

func TestAccessLog_LogsErrorStatusCode(t *testing.T) {
	output := captureLog(t, func() {
		// /tags/{name} with empty name won't match the route → 404
		req := httptest.NewRequest(http.MethodGet, "/tags/NonExistentTagXYZ123", nil)
		w := httptest.NewRecorder()
		newHandler(false).ServeHTTP(w, req)
	})
	if !strings.Contains(output, "404") {
		t.Errorf("log output missing status 404: %q", output)
	}
}

func TestAccessLog_PassesThroughResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	newHandler(false).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// ---- GET /tags/{name} handler tests ----

func TestHandler_ReturnsTagJSON(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/MyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type: got %q, want application/json", ct)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) != 1 {
			t.Fatalf("expected 1 tag, got %d", len(tags))
		}
		if tags[0]["_type"] != "tag" {
			t.Errorf("_type: got %v, want tag", tags[0]["_type"])
		}
		if tags[0]["name"] != "MyStruct" {
			t.Errorf("name: got %v, want MyStruct", tags[0]["name"])
		}
		if tags[0]["kind"] != "type" {
			t.Errorf("kind: got %v, want type", tags[0]["kind"])
		}
		if tags[0]["language"] != "Go" {
			t.Errorf("language: got %v, want Go", tags[0]["language"])
		}
	})
}

func TestHandler_MultipleTagsSameName(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/overloaded")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(tags))
		}
	})
}

func TestHandler_TagNotFound(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/NonExistentTag")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestHandler_TagsFileNotFound(t *testing.T) {
	withCwd(t, t.TempDir(), func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/anything")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestHandler_ContextQueryParam(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/SubFunc?context=sub")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) != 1 {
			t.Fatalf("expected 1 tag, got %d", len(tags))
		}
		if tags[0]["name"] != "SubFunc" {
			t.Errorf("name: got %v, want SubFunc", tags[0]["name"])
		}
	})
}

func TestHandler_ExtraFieldsInlined(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/Run")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) != 1 {
			t.Fatalf("expected 1 tag, got %d", len(tags))
		}
		// typeref is an extension field and must be inlined at top level
		if tags[0]["typeref"] != "typename:error" {
			t.Errorf("typeref: got %v, want typename:error", tags[0]["typeref"])
		}
	})
}

// ---- GET /tags (list all) handler tests ----

func TestHandler_ListAllTags_ReturnsAllTags(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type: got %q, want application/json", ct)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// testdata/tags has 8 entries (overloaded appears twice, lineonly once, etc.)
		if len(tags) == 0 {
			t.Fatal("expected non-empty tag list")
		}
		// Every entry must have _type == "tag"
		for i, tag := range tags {
			if tag["_type"] != "tag" {
				t.Errorf("tags[%d]._type: got %v, want tag", i, tag["_type"])
			}
		}
	})
}

func TestHandler_ListAllTags_CountMatchesFile(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// testdata/tags has 9 non-metadata lines
		if len(tags) != 9 {
			t.Errorf("expected 9 tags, got %d", len(tags))
		}
	})
}

func TestHandler_ListAllTags_ContextQueryParam(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags?context=sub")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) == 0 {
			t.Fatal("expected non-empty tag list from sub context")
		}
		names := make(map[string]bool)
		for _, tag := range tags {
			if n, ok := tag["name"].(string); ok {
				names[n] = true
			}
		}
		if !names["SubFunc"] {
			t.Errorf("expected SubFunc in sub context tags, got names: %v", names)
		}
	})
}

func TestHandler_ListAllTags_TagsFileNotFound(t *testing.T) {
	withCwd(t, t.TempDir(), func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ---- GET /snippets/{name} handler tests ----

func TestSnippetHandler_ReturnsJSON(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/NewMyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type: got %q, want application/json", ct)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
	})
}

func TestSnippetHandler_SnippetFields(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/NewMyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		s := snippets[0]

		if s.Name != "NewMyStruct" {
			t.Errorf("Name: got %q, want NewMyStruct", s.Name)
		}
		if s.Start != 11 {
			t.Errorf("Start: got %d, want 11", s.Start)
		}
		if s.End != 13 {
			t.Errorf("End: got %d, want 13", s.End)
		}
		if !strings.Contains(s.Code, "func NewMyStruct") {
			t.Errorf("Code should contain func NewMyStruct, got %q", s.Code)
		}
		if strings.Contains(s.Code, "func (m *MyStruct)") {
			t.Errorf("Code should not extend beyond end line, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_MultipleSnippets(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/overloaded")
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
		if len(snippets) != 2 {
			t.Fatalf("expected 2 snippets for overloaded, got %d", len(snippets))
		}
		paths := map[string]bool{}
		for _, s := range snippets {
			paths[s.Path] = true
		}
		if !paths["sample.go"] || !paths["other.go"] {
			t.Errorf("expected snippets from sample.go and other.go, got paths: %v", paths)
		}
	})
}

func TestSnippetHandler_TagNotFound(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/NonExistent")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestSnippetHandler_TagsFileNotFound(t *testing.T) {
	withCwd(t, t.TempDir(), func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/anything")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestSnippetHandler_ContextQueryParam(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/SubFunc?context=sub")
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
		if snippets[0].Name != "SubFunc" {
			t.Errorf("Name: got %q, want SubFunc", snippets[0].Name)
		}
		if !strings.Contains(snippets[0].Code, "func SubFunc") {
			t.Errorf("Code should contain func SubFunc, got %q", snippets[0].Code)
		}
	})
}

func TestSnippetHandler_LineOnlyTag(t *testing.T) {
	// readtags silently skips line-number addressed tags; only loadTagsFile handles them.
	if _, err := exec.LookPath("readtags"); err == nil {
		t.Skip("readtags does not return line-number addressed tags")
	}
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/lineonly")
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
		if snippets[0].Start != 42 {
			t.Errorf("Start: got %d, want 42", snippets[0].Start)
		}
		// lineonly tag has no end field: End must be 0 and Code must be a single line
		if snippets[0].End != 0 {
			t.Errorf("End: got %d, want 0", snippets[0].End)
		}
		if !strings.Contains(snippets[0].Code, "var lineonly") {
			t.Errorf("Code should contain var lineonly, got %q", snippets[0].Code)
		}
		if strings.Contains(snippets[0].Code, "\n") {
			t.Errorf("Code should be a single line when end is unknown, got %q", snippets[0].Code)
		}
	})
}

func TestSnippetHandler_CodeBoundaries(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Run")
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
		s := snippets[0]
		if s.Start != 17 || s.End != 22 {
			t.Errorf("Start/End: got %d/%d, want 17/22", s.Start, s.End)
		}
		// Must include the function body
		if !strings.Contains(s.Code, "func (m *MyStruct) Run()") {
			t.Errorf("Code missing function signature, got %q", s.Code)
		}
		// Must not include helperFunc which starts at line 24
		if strings.Contains(s.Code, "func helperFunc") {
			t.Errorf("Code should not extend past end line, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_NoEndField_ReturnsZeroEndAndSingleLine(t *testing.T) {
	// noendVar tag in testdata/tags has no "end" field.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/noendVar")
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
		if s.End != 0 {
			t.Errorf("End: got %d, want 0 when end field is absent", s.End)
		}
		if strings.Contains(s.Code, "\n") {
			t.Errorf("Code should be a single line when end is unknown, got %q", s.Code)
		}
		if !strings.Contains(s.Code, "noendVar") {
			t.Errorf("Code should contain noendVar, got %q", s.Code)
		}
	})
}

// ---- GET /lines/{name} handler tests ----

func TestLinesHandler_ReturnsJSON(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/NewMyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type: got %q, want application/json", ct)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(ranges))
		}
	})
}

func TestLinesHandler_LineRangeFields(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/NewMyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		lr := ranges[0]

		if lr.Name != "NewMyStruct" {
			t.Errorf("Name: got %q, want NewMyStruct", lr.Name)
		}
		if lr.Start != 11 {
			t.Errorf("Start: got %d, want 11", lr.Start)
		}
		if lr.End != 13 {
			t.Errorf("End: got %d, want 13", lr.End)
		}
	})
}

func TestLinesHandler_NoCodeField(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/NewMyStruct")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var raw []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := raw[0]["code"]; ok {
			t.Error("response must not contain a 'code' field")
		}
	})
}

func TestLinesHandler_MultipleRanges(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/overloaded")
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
		if len(ranges) != 2 {
			t.Fatalf("expected 2 entries for overloaded, got %d", len(ranges))
		}
		paths := map[string]bool{}
		for _, lr := range ranges {
			paths[lr.Path] = true
		}
		if !paths["sample.go"] || !paths["other.go"] {
			t.Errorf("expected entries from sample.go and other.go, got: %v", paths)
		}
	})
}

func TestLinesHandler_TagNotFound(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/NonExistent")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestLinesHandler_TagsFileNotFound(t *testing.T) {
	withCwd(t, t.TempDir(), func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/anything")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestLinesHandler_ContextQueryParam(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/SubFunc?context=sub")
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
			t.Fatalf("expected 1 entry, got %d", len(ranges))
		}
		if ranges[0].Start != 3 || ranges[0].End != 4 {
			t.Errorf("Start/End: got %d/%d, want 3/4", ranges[0].Start, ranges[0].End)
		}
	})
}

func TestLinesHandler_NoEndField_ReturnsZeroEnd(t *testing.T) {
	// noendVar tag in testdata/tags has no "end" field.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/noendVar")
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
			t.Fatalf("expected 1 entry, got %d", len(ranges))
		}
		if ranges[0].End != 0 {
			t.Errorf("End: got %d, want 0 when end field is absent", ranges[0].End)
		}
	})
}

// ---- tags= query parameter handler tests ----

// TestHandler_TagsQueryParam_OverridesDefault verifies that tags= pointing to
// testdata/sub/tags returns SubFunc even when cwd is testdata (default tags file
// does not contain SubFunc).
func TestHandler_TagsQueryParam_OverridesDefault(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		absTagsPath, err := filepath.Abs("sub/tags")
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(srv.URL + "/tags/SubFunc?tags=" + absTagsPath)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) != 1 || tags[0]["name"] != "SubFunc" {
			t.Errorf("expected SubFunc tag, got %v", tags)
		}
	})
}

// TestHandler_TagsQueryParam_ListAll verifies that GET /tags?tags= returns tags
// from the explicitly specified tags file.
func TestHandler_TagsQueryParam_ListAll(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		absTagsPath, err := filepath.Abs("sub/tags")
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(srv.URL + "/tags?tags=" + absTagsPath)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		names := map[string]bool{}
		for _, tag := range tags {
			if n, ok := tag["name"].(string); ok {
				names[n] = true
			}
		}
		if !names["SubFunc"] {
			t.Errorf("expected SubFunc in tags from sub/tags, got names: %v", names)
		}
	})
}

// TestHandler_TagsQueryParam_Snippets verifies that GET /snippets/{name}?tags=
// returns a snippet using the explicitly specified tags file.
func TestHandler_TagsQueryParam_Snippets(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		absTagsPath, err := filepath.Abs("sub/tags")
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(srv.URL + "/snippets/SubFunc?tags=" + absTagsPath)
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
		if snippets[0].Name != "SubFunc" {
			t.Errorf("Name: got %q, want SubFunc", snippets[0].Name)
		}
		if !strings.Contains(snippets[0].Code, "func SubFunc") {
			t.Errorf("Code should contain func SubFunc, got %q", snippets[0].Code)
		}
	})
}

// TestHandler_TagsQueryParam_Lines verifies that GET /lines/{name}?tags= returns
// line ranges using the explicitly specified tags file.
func TestHandler_TagsQueryParam_Lines(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		absTagsPath, err := filepath.Abs("sub/tags")
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.Get(srv.URL + "/lines/SubFunc?tags=" + absTagsPath)
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
			t.Fatalf("expected 1 entry, got %d", len(ranges))
		}
		if ranges[0].Name != "SubFunc" {
			t.Errorf("Name: got %q, want SubFunc", ranges[0].Name)
		}
	})
}

// TestHandler_TagsQueryParam_NotFound verifies that tags= pointing to a
// nonexistent file returns 404.
func TestHandler_TagsQueryParam_NotFound(t *testing.T) {
	endpoints := []string{
		"/tags",
		"/tags/SomeFunc",
		"/snippets/SomeFunc",
		"/lines/SomeFunc",
	}
	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	for _, ep := range endpoints {
		resp, err := http.Get(srv.URL + ep + "?tags=/nonexistent/path/tags")
		if err != nil {
			t.Fatalf("%s: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status got %d, want %d", ep, resp.StatusCode, http.StatusNotFound)
		}
	}
}

// TestHandler_ContextNotFound checks that all four endpoints return 404 when
// the context directory (and therefore the tags file) does not exist.
func TestHandler_ContextNotFound(t *testing.T) {
	endpoints := []string{
		"/tags",
		"/tags/SomeFunc",
		"/snippets/SomeFunc",
		"/lines/SomeFunc",
	}
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		for _, ep := range endpoints {
			url := srv.URL + ep + "?context=nonexistent"
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("%s: %v", ep, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("%s: status got %d, want %d", ep, resp.StatusCode, http.StatusNotFound)
			}
		}
	})
}

// TestHandler_ContextEmpty verifies that omitting context= (empty string)
// falls back to the default tags file in the current directory.
func TestHandler_ContextEmpty(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		// /tags without context= should return tags from ./tags (the default).
		resp, err := http.Get(srv.URL + "/tags")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var tags []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tags) == 0 {
			t.Error("expected at least one tag from default tags file")
		}
	})
}

// TestHandler_ContextIsolation verifies that two different context values
// return results from their respective tags files and do not bleed into
// each other. SubFunc only exists in the "sub" context, not in the default.
func TestHandler_ContextIsolation(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		// SubFunc should be found with ?context=sub
		resp, err := http.Get(srv.URL + "/tags/SubFunc?context=sub")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("context=sub: status got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// SubFunc should NOT be found without context= (default tags file).
		// The server returns 404 when the tag does not exist.
		resp2, err := http.Get(srv.URL + "/tags/SubFunc")
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusNotFound {
			t.Errorf("SubFunc should not exist in default context: status got %d, want %d", resp2.StatusCode, http.StatusNotFound)
		}
	})
}

// ---- /pipe endpoint tests ----

// TestPipe_InitiallyEmpty verifies that GET /pipe/status reports empty before any POST.
func TestPipe_InitiallyEmpty(t *testing.T) {
	// Reset global buffer before the test.
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pipe/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var got map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["empty"] != true {
		t.Errorf("empty: got %v, want true", got["empty"])
	}
}

// TestPipe_PostAndGet verifies that POST /pipe stores content and GET /pipe retrieves it.
func TestPipe_PostAndGet(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	content := "hello, pipe!"

	// POST content into the pipe.
	resp, err := http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// GET the content back.
	resp2, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET status: got %d, want %d", resp2.StatusCode, http.StatusOK)
	}
	if ct := resp2.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != content {
		t.Errorf("body: got %q, want %q", string(body), content)
	}
}

// TestPipe_StatusNonEmpty verifies that GET /pipe/status reports non-empty after POST.
func TestPipe_StatusNonEmpty(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp2, err := http.Get(srv.URL + "/pipe/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	var got map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["empty"] != false {
		t.Errorf("empty: got %v, want false", got["empty"])
	}
}

// TestPipe_OverwriteOnPost verifies that a second POST replaces the buffer content.
func TestPipe_OverwriteOnPost(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader("first"))
	http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader("second"))

	resp, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "second" {
		t.Errorf("body: got %q, want %q", string(body), "second")
	}
}

// TestPipe_AppendMode verifies that POST /pipe?mode=append appends to the buffer.
func TestPipe_AppendMode(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader("hello"))
	http.Post(srv.URL+"/pipe?mode=append", "text/plain", strings.NewReader(", world"))

	resp, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello, world" {
		t.Errorf("body: got %q, want %q", string(body), "hello, world")
	}
}

// TestPipe_AppendModeMultiple verifies that multiple appends accumulate correctly.
func TestPipe_AppendModeMultiple(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	for _, chunk := range []string{"a", "b", "c"} {
		http.Post(srv.URL+"/pipe?mode=append", "text/plain", strings.NewReader(chunk))
	}

	resp, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "abc" {
		t.Errorf("body: got %q, want %q", string(body), "abc")
	}
}

// TestPipe_DefaultModeOverwritesAfterAppend verifies that a plain POST (no mode) replaces
// content even if the buffer was previously built up with appends.
func TestPipe_DefaultModeOverwritesAfterAppend(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	http.Post(srv.URL+"/pipe?mode=append", "text/plain", strings.NewReader("old"))
	http.Post(srv.URL+"/pipe", "text/plain", strings.NewReader("new"))

	resp, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new" {
		t.Errorf("body: got %q, want %q", string(body), "new")
	}
}

// TestPipe_GetEmptyBuffer verifies that GET /pipe on an empty buffer returns 200 with empty body.
func TestPipe_GetEmptyBuffer(t *testing.T) {
	pipe.mu.Lock()
	pipe.data = nil
	pipe.mu.Unlock()

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", string(body))
	}
}

// ---- queryTagsPath / tilde expansion in handlers tests ----

func TestHandler_ContextTildeExpansion(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}

	// Create a temporary directory under the real home dir to serve as the context.
	tmpDir, err := os.MkdirTemp(u.HomeDir, "codesnippetd-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy testdata/tags into the temp dir so the handler can find it.
	tagsData, err := os.ReadFile(filepath.Join("testdata", "tags"))
	if err != nil {
		t.Fatalf("reading testdata/tags: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "tags"), tagsData, 0o644); err != nil {
		t.Fatalf("writing tags: %v", err)
	}

	// Also copy the source files that the tags reference.
	for _, name := range []string{"sample.go", "other.go"} {
		src, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, name), src, 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	// Build a context value using ~ so it starts with the home dir.
	rel, err := filepath.Rel(u.HomeDir, tmpDir)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	tildeContext := "~/" + rel

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/tags?context=" + tildeContext)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var tags []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tags) == 0 {
		t.Error("expected non-empty tag list with tilde context")
	}
}

func TestHandler_TagsParamTildeExpansion(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}

	tmpDir, err := os.MkdirTemp(u.HomeDir, "codesnippetd-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tagsData, err := os.ReadFile(filepath.Join("testdata", "tags"))
	if err != nil {
		t.Fatalf("reading testdata/tags: %v", err)
	}
	tagsFile := filepath.Join(tmpDir, "tags")
	if err := os.WriteFile(tagsFile, tagsData, 0o644); err != nil {
		t.Fatalf("writing tags: %v", err)
	}

	rel, err := filepath.Rel(u.HomeDir, tagsFile)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	tildeTags := "~/" + rel

	srv := httptest.NewServer(newHandler(false))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/tags?tags=" + tildeTags)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var tags []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tags) == 0 {
		t.Error("expected non-empty tag list with tilde tags param")
	}
}


func TestPipe_ConcurrentAccess(t *testing.T) {
	srv := httptest.NewServer(newHandler(true))
	defer srv.Close()

	const numGoroutines = 10
	const numIterations = 5

	done := make(chan bool)

	// Concurrent POSTs
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numIterations; j++ {
				data := []byte("data" + string(rune('0'+id)))
				resp, err := http.Post(srv.URL+"/pipe", "text/plain", bytes.NewReader(data))
				if err != nil {
					t.Errorf("POST error: %v", err)
					return
				}
				resp.Body.Close()
			}
			done <- true
		}(i)
	}

	// Concurrent GETs
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIterations; j++ {
				resp, err := http.Get(srv.URL + "/pipe")
				if err != nil {
					t.Errorf("GET error: %v", err)
					return
				}
				resp.Body.Close()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}
