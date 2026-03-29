package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Tag represents a single ctags entry in Universal Ctags JSON output format.
// See: https://docs.ctags.io/en/latest/man/ctags-client-tools.7.html
type Tag struct {
	Type     string `json:"_type"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Pattern  string `json:"pattern,omitempty"`
	Language string `json:"language,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Line     int    `json:"line,omitempty"`
	// Extension fields (e.g. scope, access, typeref, roles, extras, etc.)
	// stored as a flat map and inlined into the JSON output
	Extra map[string]string `json:"-"`
}

// MarshalJSON serializes Tag with extension fields inlined at the top level.
func (t Tag) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"_type": t.Type,
		"name":  t.Name,
		"path":  t.Path,
	}
	if t.Pattern != "" {
		m["pattern"] = t.Pattern
	}
	if t.Language != "" {
		m["language"] = t.Language
	}
	if t.Kind != "" {
		m["kind"] = t.Kind
	}
	if t.Line != 0 {
		m["line"] = t.Line
	}
	for k, v := range t.Extra {
		m[k] = v
	}
	return json.Marshal(m)
}

// pipe is the global in-memory buffer used by the /pipe endpoint.
var pipe struct {
	mu   sync.Mutex
	data []byte
}

// requestMetaKey is the context key for requestMeta values.
type requestMetaKey struct{}

// requestMeta carries per-request metadata that handlers can annotate and
// the access log middleware can read after the handler returns.
type requestMeta struct {
	usedTreeSitter bool
}

// markTreeSitterUsed records that tree-sitter resolved an end line for this request.
func markTreeSitterUsed(ctx context.Context) {
	if m, ok := ctx.Value(requestMetaKey{}).(*requestMeta); ok {
		m.usedTreeSitter = true
	}
}

// responseRecorder wraps http.ResponseWriter to capture the written status code.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// corsMiddleware sets Access-Control-Allow-Origin on every response and handles
// preflight OPTIONS requests so that browser clients (e.g. canvas.html loaded
// from a file:// origin) are not blocked by CORS policy.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// accessLog is middleware that logs each request's method, path, status code, and duration.
// It also logs whether tree-sitter was used to resolve an end line.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		meta := &requestMeta{}
		r = r.WithContext(context.WithValue(r.Context(), requestMetaKey{}, meta))
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.URL.Path == "/pipe" {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
		} else {
			parser := "ctags"
			if meta.usedTreeSitter {
				parser = "tree-sitter"
			}
			log.Printf("%s %s %d %s parser=%s", r.Method, r.URL.Path, rec.status, time.Since(start), parser)
		}
	})
}

// TagsDB holds parsed tags indexed by tag name.
type TagsDB struct {
	tags map[string][]Tag
}

// parseLine parses a single line from a ctags file (extended format).
// Format: tagname TAB filename TAB address ;" TAB [fields...]
func parseLine(line string) (Tag, bool) {
	// Skip comment/metadata lines
	if strings.HasPrefix(line, "!_TAG_") || line == "" {
		return Tag{}, false
	}

	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return Tag{}, false
	}

	tag := Tag{
		Type:  "tag",
		Name:  parts[0],
		Path:  parts[1],
		Extra: make(map[string]string),
	}

	// parts[2] is the address (pattern or line number), optionally followed by ;"
	addr := parts[2]
	if idx := strings.Index(addr, ";\""); idx != -1 {
		addr = addr[:idx]
	}
	addr = strings.TrimSpace(addr)

	// Determine if address is a line number or a search pattern
	if n, err := strconv.Atoi(addr); err == nil {
		tag.Line = n
	} else {
		// Strip leading / and trailing / or ?...? delimiters used in ctags patterns
		pattern := addr
		if (strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/")) ||
			(strings.HasPrefix(pattern, "?") && strings.HasSuffix(pattern, "?")) {
			pattern = pattern[1 : len(pattern)-1]
		}
		tag.Pattern = pattern
	}

	// Parse extension fields (tab-separated key:value pairs after the address field)
	for _, field := range parts[3:] {
		if field == "" {
			continue
		}
		colonIdx := strings.Index(field, ":")
		if colonIdx == -1 {
			// kind shorthand (single character or word with no colon)
			tag.Kind = field
			continue
		}
		key := field[:colonIdx]
		val := field[colonIdx+1:]
		switch key {
		case "kind":
			// "kind:f" or "kind:function"
			tag.Kind = val
		case "line":
			if n, err := strconv.Atoi(val); err == nil {
				tag.Line = n
			}
		case "language":
			tag.Language = val
		default:
			tag.Extra[key] = val
		}
	}

	return tag, true
}

// loadTagsFile reads and parses the ctags file at path.
func loadTagsFile(path string) (*TagsDB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading tags file: %w", err)
	}

	db := &TagsDB{tags: make(map[string][]Tag)}
	for _, line := range strings.Split(string(data), "\n") {
		tag, ok := parseLine(line)
		if !ok {
			continue
		}
		db.tags[tag.Name] = append(db.tags[tag.Name], tag)
	}
	return db, nil
}

// lookup returns all tags matching name, or nil if not found.
func (db *TagsDB) lookup(name string) []Tag {
	return db.tags[name]
}

// lookupWithReadtags runs the readtags command to find tags matching tagName in tagsPath.
// Returns os.ErrNotExist if the tags file does not exist.
func lookupWithReadtags(tagsPath, tagName string) ([]Tag, error) {
	if _, err := os.Stat(tagsPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("tags file not found: %w", os.ErrNotExist)
		}
		return nil, fmt.Errorf("stat tags file: %w", err)
	}

	cmd := exec.Command("readtags", "-t", tagsPath, "-e", tagName)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("readtags: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("running readtags: %w", err)
	}

	var tags []Tag
	for _, line := range strings.Split(string(out), "\n") {
		tag, ok := parseLine(line)
		if !ok {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// lookupTag searches for tags by name. It uses readtags if available, otherwise
// falls back to in-memory parsing via loadTagsFile.
func lookupTag(tagsPath, tagName string) ([]Tag, error) {
	if _, err := exec.LookPath("readtags"); err == nil {
		return lookupWithReadtags(tagsPath, tagName)
	}
	db, err := loadTagsFile(tagsPath)
	if err != nil {
		return nil, err
	}
	return db.lookup(tagName), nil
}

// tagsFileForContext resolves the tags file path given an optional context query param.
// If context is empty, "./tags" is used. Otherwise "<context>/tags" is used.
func tagsFileForContext(context string) string {
	if context == "" {
		return filepath.Join(".", "tags")
	}
	return filepath.Join(".", context, "tags")
}

// Snippet represents a code snippet extracted from a source file for a given tag.
type Snippet struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Code  string `json:"code"`
}

// LineRange represents the start and end line numbers of a tag in its source file.
type LineRange struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// normalizeTagPattern strips ctags regex anchors (^ prefix, $ suffix) and
// unescapes common regex metacharacters so the result can be used with
// strings.Contains for line matching.
func normalizeTagPattern(pattern string) string {
	p := strings.TrimPrefix(pattern, "^")
	p = strings.TrimSuffix(p, "$")
	p = strings.NewReplacer(`\*`, "*", `\.`, ".", `\/`, "/", `\\`, `\`).Replace(p)
	return p
}

// findPatternLine returns the 1-based line number of the first line containing pattern,
// or -1 if not found. The pattern may include ctags-style anchors (^/$) and escapes.
func findPatternLine(lines []string, pattern string) int {
	search := normalizeTagPattern(pattern)
	for i, line := range lines {
		if strings.Contains(line, search) {
			return i + 1
		}
	}
	return -1
}

// extractLines returns the joined content of lines[start-1 : end] (1-based, inclusive).
func extractLines(lines []string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n")
}

// resolveFilePath returns the path to the source file for a tag.
// If tagPath is already absolute it is returned unchanged; otherwise contextDir
// (the directory that contains the tags file) is prepended.
func resolveFilePath(contextDir, tagPath string) string {
	if filepath.IsAbs(tagPath) {
		return tagPath
	}
	return filepath.Join(contextDir, tagPath)
}

// resolveStartEnd returns the start and end line numbers for a Tag.
// contextDir is the directory containing the tags file; it is prepended to
// tag.Path (which is relative to the tags file) when reading source files.
// The source file is read only when pattern matching is needed (tag.Line == 0).
// If the "end" extension field is absent and useTreeSitter is true and the file
// is a supported language, tree-sitter is used to determine the end line.
// If neither source provides an end line, endLine is returned as 0.
// When tree-sitter successfully resolves the end line, markTreeSitterUsed is
// called on ctx so that the access log middleware can record the fact.
func resolveStartEnd(ctx context.Context, tag Tag, contextDir string, useTreeSitter bool) (startLine, endLine int, err error) {
	filePath := resolveFilePath(contextDir, tag.Path)
	needFile := tag.Line == 0 && tag.Pattern != ""

	var lines []string
	if needFile {
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return 0, 0, fmt.Errorf("reading file %s: %w", filePath, readErr)
		}
		lines = strings.Split(string(data), "\n")
	}

	startLine = tag.Line
	if startLine == 0 && tag.Pattern != "" {
		startLine = findPatternLine(lines, tag.Pattern)
	}
	if startLine <= 0 {
		return 0, 0, fmt.Errorf("cannot determine start line for tag %q in %s", tag.Name, filePath)
	}

	if endStr, ok := tag.Extra["end"]; ok {
		if n, err := strconv.Atoi(endStr); err == nil {
			endLine = n
			return startLine, endLine, nil
		}
	}

	// end field absent: read the file (if not already read) and try tree-sitter.
	var data []byte
	if lines == nil {
		var readErr error
		data, readErr = os.ReadFile(filePath)
		if readErr != nil {
			return 0, 0, fmt.Errorf("reading file %s: %w", filePath, readErr)
		}
		lines = strings.Split(string(data), "\n")
	} else {
		data = []byte(strings.Join(lines, "\n"))
	}

	if useTreeSitter {
		var tsEnd int
		var tsErr error
		switch {
		case isRustFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterRust(data, startLine)
		case isJSFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterJS(data, startLine)
		case isTSFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterTS(data, startLine)
		case isHSFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterHS(data, startLine)
		case isKtFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterKotlin(data, startLine)
		case isPHPFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterPHP(data, startLine)
		case isMLFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterOCaml(data, startLine)
		case isMLIFile(tag.Path):
			tsEnd, tsErr = resolveEndWithTreeSitterOCamlInterface(data, startLine)
		}
		if tsErr == nil && tsEnd > 0 {
			markTreeSitterUsed(ctx)
			return startLine, tsEnd, nil
		}
	}

	return startLine, 0, nil
}

// snippetForTag resolves a Snippet from a Tag by reading the source file.
// contextDir is the directory containing the tags file.
func snippetForTag(ctx context.Context, tag Tag, contextDir string, useTreeSitter bool) (Snippet, error) {
	startLine, endLine, err := resolveStartEnd(ctx, tag, contextDir, useTreeSitter)
	if err != nil {
		return Snippet{}, err
	}

	filePath := resolveFilePath(contextDir, tag.Path)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Snippet{}, fmt.Errorf("reading file %s: %w", filePath, err)
	}
	lines := strings.Split(string(data), "\n")

	extractEnd := endLine
	if extractEnd == 0 {
		extractEnd = startLine
	}

	return Snippet{
		Name:  tag.Name,
		Path:  tag.Path,
		Start: startLine,
		End:   endLine,
		Code:  extractLines(lines, startLine, extractEnd),
	}, nil
}

// lineRangeForTag resolves the start and end line numbers for a Tag without reading
// the full file content (the file is read only when pattern matching is needed).
// contextDir is the directory containing the tags file.
func lineRangeForTag(ctx context.Context, tag Tag, contextDir string, useTreeSitter bool) (LineRange, error) {
	startLine, endLine, err := resolveStartEnd(ctx, tag, contextDir, useTreeSitter)
	if err != nil {
		return LineRange{}, err
	}
	return LineRange{
		Name:  tag.Name,
		Path:  tag.Path,
		Start: startLine,
		End:   endLine,
	}, nil
}

func main() {
	addr := flag.String("listen", ":8999", "listen address (host:port)")
	flag.StringVar(addr, "l", ":8999", "listen address (shorthand for -listen)")
	port := flag.Int("port", 0, "port number to listen on; overrides -addr when set")
	flag.IntVar(port, "p", 0, "port number to listen on (shorthand for -port)")
	useTreeSitter := flag.Bool("tree-sitter", false, "use tree-sitter to resolve end lines when ctags does not provide them")
	flag.Parse()

	listenAddr := *addr
	if *port != 0 {
		listenAddr = fmt.Sprintf(":%d", *port)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("POST /pipe", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusInternalServerError)
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
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	})

	mux.HandleFunc("GET /tags", func(w http.ResponseWriter, r *http.Request) {
		context := r.URL.Query().Get("context")
		tagsPath := tagsFileForContext(context)

		db, err := loadTagsFile(tagsPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("failed to load tags file: %v", err), http.StatusInternalServerError)
			}
			return
		}

		// Collect all tags from the database
		var all []Tag
		for _, tags := range db.tags {
			all = append(all, tags...)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(all); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /tags/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		context := r.URL.Query().Get("context")
		tagsPath := tagsFileForContext(context)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /snippets/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		context := r.URL.Query().Get("context")
		tagsPath := tagsFileForContext(context)
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		var snippets []Snippet
		for _, tag := range results {
			s, err := snippetForTag(r.Context(), tag, contextDir, *useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			snippets = append(snippets, s)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snippets); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /lines/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		context := r.URL.Query().Get("context")
		tagsPath := tagsFileForContext(context)
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		var ranges []LineRange
		for _, tag := range results {
			lr, err := lineRangeForTag(r.Context(), tag, contextDir, *useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ranges = append(ranges, lr)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ranges); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	log.Printf("listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, corsMiddleware(accessLog(mux))); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
