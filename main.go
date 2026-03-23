package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// tagsFileForContext resolves the tags file path given an optional context query param.
// If context is empty, "./tags" is used. Otherwise "<context>/tags" is used.
func tagsFileForContext(context string) string {
	if context == "" {
		return filepath.Join(".", "tags")
	}
	return filepath.Join(".", context, "tags")
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		// Extract tag name from URL path: /tags/<name>
		tagName := strings.TrimPrefix(r.URL.Path, "/tags/")
		if tagName == "" {
			http.Error(w, "tag name required", http.StatusBadRequest)
			return
		}

		// Resolve tags file path
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

		results := db.lookup(tagName)
		if results == nil {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
