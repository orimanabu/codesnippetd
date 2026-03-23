package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /tags", func(w http.ResponseWriter, r *http.Request) {
		context := r.URL.Query().Get("context")
		tagsPath := tagsFileForContext(context)
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
		tagsPath := tagsFileForContext(context)
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
	return accessLog(mux)
}

func TestHandler_ReturnsTagJSON(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
	newHandler().ServeHTTP(w, req)

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
		newHandler().ServeHTTP(w, req)
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
		newHandler().ServeHTTP(w, req)
	})
	if !strings.Contains(output, "200") {
		t.Errorf("log output missing status 200: %q", output)
	}
}

func TestAccessLog_LogsErrorStatusCode(t *testing.T) {
	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/tags/", nil)
		w := httptest.NewRecorder()
		newHandler().ServeHTTP(w, req)
	})
	if !strings.Contains(output, "400") {
		t.Errorf("log output missing status 400: %q", output)
	}
}

func TestAccessLog_PassesThroughResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	newHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// ---- GET /tags (list all) handler tests ----

func TestHandler_ListAllTags_ReturnsAllTags(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
		// testdata/tags has 8 non-metadata lines
		if len(tags) != 8 {
			t.Errorf("expected 8 tags, got %d", len(tags))
		}
	})
}

func TestHandler_ListAllTags_ContextQueryParam(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler())
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
		srv := httptest.NewServer(newHandler())
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
