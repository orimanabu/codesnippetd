package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ---- HTTP handler tests ----

// newTestServer returns an httptest.Server rooted at dir, so that tags file
// resolution (which uses relative paths from cwd) works correctly.
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

func newHandler(useTreeSitter bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		u, err := user.Current()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get current user: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"user":    u.Username,
			"homedir": u.HomeDir,
		}); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})
	mux.HandleFunc("GET /tags", func(w http.ResponseWriter, r *http.Request) {
		context := r.URL.Query().Get("context")
		tagsPath := resolveTagsPath(context, r.URL.Query().Get("tags"))
		db, err := loadTagsFile(tagsPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "tags file not found: "+tagsPath, http.StatusNotFound)
			} else {
				http.Error(w, "failed to load tags file: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		var all []Tag
		for _, tags := range db.tags {
			all = append(all, tags...)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(all)
	})
	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		tagName := strings.TrimPrefix(r.URL.Path, "/tags/")
		if tagName == "" {
			http.Error(w, "tag name required", http.StatusBadRequest)
			return
		}
		context := r.URL.Query().Get("context")
		tagsPath := resolveTagsPath(context, r.URL.Query().Get("tags"))
		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "tags file not found: "+tagsPath, http.StatusNotFound)
			} else {
				http.Error(w, "readtags error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if len(results) == 0 {
			http.Error(w, "tag not found: "+tagName, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
	mux.HandleFunc("GET /snippets/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		context := r.URL.Query().Get("context")
		tagsPath := resolveTagsPath(context, r.URL.Query().Get("tags"))
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "tags file not found: "+tagsPath, http.StatusNotFound)
			} else {
				http.Error(w, "readtags error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if len(results) == 0 {
			http.Error(w, "tag not found: "+tagName, http.StatusNotFound)
			return
		}

		var snippets []Snippet
		for _, tag := range results {
			s, err := snippetForTag(r.Context(), tag, contextDir, useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			snippets = append(snippets, s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snippets)
	})
	mux.HandleFunc("GET /lines/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		context := r.URL.Query().Get("context")
		tagsPath := resolveTagsPath(context, r.URL.Query().Get("tags"))
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "tags file not found: "+tagsPath, http.StatusNotFound)
			} else {
				http.Error(w, "readtags error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if len(results) == 0 {
			http.Error(w, "tag not found: "+tagName, http.StatusNotFound)
			return
		}

		var ranges []LineRange
		for _, tag := range results {
			lr, err := lineRangeForTag(r.Context(), tag, contextDir, useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ranges = append(ranges, lr)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ranges)
	})

	mux.HandleFunc("POST /pipe", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		pipe.mu.Lock()
		if r.URL.Query().Get("mode") == "append" {
			pipe.data = append(pipe.data, body...)
		} else {
			pipe.data = body
		}
		pipe.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /pipe/status", func(w http.ResponseWriter, r *http.Request) {
		pipe.mu.Lock()
		empty := len(pipe.data) == 0
		pipe.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if empty {
			fmt.Fprintln(w, `{"empty":true}`)
		} else {
			fmt.Fprintln(w, `{"empty":false}`)
		}
	})

	mux.HandleFunc("GET /pipe", func(w http.ResponseWriter, r *http.Request) {
		pipe.mu.Lock()
		data := pipe.data
		pipe.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	return accessLog(mux)
}

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

func TestHandler_MissingTagName(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(false))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/tags/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
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

// captureLog redirects the default logger to a buffer for the duration of fn.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	fn()
	return buf.String()
}

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
		req := httptest.NewRequest(http.MethodGet, "/tags/", nil)
		w := httptest.NewRecorder()
		newHandler(false).ServeHTTP(w, req)
	})
	if !strings.Contains(output, "400") {
		t.Errorf("log output missing status 400: %q", output)
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

// ---- snippetForTag tests ----

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

// ---- context= query parameter edge-case tests ----

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

// ---- GET /user handler tests ----

func TestUserHandler_StatusAndContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()
	newHandler(false).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
}

func TestUserHandler_ReturnsUserAndHomedir(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()
	newHandler(false).ServeHTTP(w, req)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["user"] != u.Username {
		t.Errorf("user: got %q, want %q", body["user"], u.Username)
	}
	if body["homedir"] != u.HomeDir {
		t.Errorf("homedir: got %q, want %q", body["homedir"], u.HomeDir)
	}
}

func TestUserHandler_OnlyExpectedFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()
	newHandler(false).ServeHTTP(w, req)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["user"]; !ok {
		t.Error("response missing 'user' field")
	}
	if _, ok := body["homedir"]; !ok {
		t.Error("response missing 'homedir' field")
	}
	if len(body) != 2 {
		t.Errorf("expected exactly 2 fields, got %d: %v", len(body), body)
	}
}

// ---- MarshalJSON tests ----

func TestMarshalJSON_OmitsEmptyOptionalFields(t *testing.T) {
	tag := Tag{
		Type:  "tag",
		Name:  "Foo",
		Path:  "foo.go",
		Extra: map[string]string{},
	}
	b, err := json.Marshal(tag)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pattern", "language", "kind", "line"} {
		if _, exists := m[key]; exists {
			t.Errorf("expected %q to be absent when zero/empty, but found in JSON", key)
		}
	}
}
