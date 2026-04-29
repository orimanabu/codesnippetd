package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
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

// TagsDB holds parsed tags indexed by tag name.
type TagsDB struct {
	tags map[string][]Tag
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
