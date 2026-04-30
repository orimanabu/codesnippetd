package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// readtagsAvailable is set at init time to indicate whether the readtags
// binary is found in PATH.
var readtagsAvailable bool

func init() {
	_, err := exec.LookPath("readtags")
	readtagsAvailable = err == nil
}

// parseLine parses a single line from a ctags file (extended format).
// Format: tagname TAB filename TAB address ;" TAB [fields...]
//
// The address field (column 3) may itself contain tab characters when the
// search pattern spans indented source lines. To handle this correctly we
// locate the field boundaries manually: tagname and filename are delimited
// by the first two literal tabs, and the address ends at the first `;"` that
// is immediately followed by a tab or the end of the line.
func parseLine(line string) (Tag, bool) {
	// Skip comment/metadata lines
	if strings.HasPrefix(line, "!_TAG_") || line == "" {
		return Tag{}, false
	}

	// Extract tagname (field 1): up to the first tab.
	tab1 := strings.Index(line, "\t")
	if tab1 == -1 {
		return Tag{}, false
	}
	tagName := line[:tab1]

	// Extract filename (field 2): between the first and second tab.
	rest := line[tab1+1:]
	tab2 := strings.Index(rest, "\t")
	if tab2 == -1 {
		return Tag{}, false
	}
	fileName := rest[:tab2]

	// The address (field 3) begins here and may contain embedded tabs
	// (e.g. a search pattern for an indented line).  It ends at `;"` which
	// is followed by either a tab (more extension fields) or end-of-string.
	addrAndRest := rest[tab2+1:]

	tag := Tag{
		Type:  "tag",
		Name:  tagName,
		Path:  fileName,
		Extra: make(map[string]string),
	}

	var addr, extensionFields string
	if sepIdx := strings.Index(addrAndRest, ";\"\t"); sepIdx != -1 {
		// `;"` followed by a tab: extension fields follow.
		addr = addrAndRest[:sepIdx]
		extensionFields = addrAndRest[sepIdx+3:] // skip `;"` + tab
	} else if strings.HasSuffix(addrAndRest, ";\"") {
		// `;"` at end of line: no extension fields.
		addr = addrAndRest[:len(addrAndRest)-2]
	} else {
		// No `;"` separator: the address is the next tab-delimited token (e.g. a
		// bare line number), and any remaining tokens are extension fields.
		if tabIdx := strings.Index(addrAndRest, "\t"); tabIdx != -1 {
			addr = addrAndRest[:tabIdx]
			extensionFields = addrAndRest[tabIdx+1:]
		} else {
			addr = addrAndRest
		}
	}
	addr = strings.TrimSpace(addr)

	// Determine if address is a line number or a search pattern.
	if n, err := strconv.Atoi(addr); err == nil {
		tag.Line = n
	} else {
		// Strip leading / and trailing / or ?...? delimiters used in ctags patterns.
		pattern := addr
		if (strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/")) ||
			(strings.HasPrefix(pattern, "?") && strings.HasSuffix(pattern, "?")) {
			pattern = pattern[1 : len(pattern)-1]
		}
		tag.Pattern = pattern
	}

	// Parse extension fields (tab-separated key:value pairs after the address field).
	for _, field := range strings.Split(extensionFields, "\t") {
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
	if readtagsAvailable {
		return lookupWithReadtags(tagsPath, tagName)
	}
	db, err := loadTagsFile(tagsPath)
	if err != nil {
		return nil, err
	}
	return db.lookup(tagName), nil
}
