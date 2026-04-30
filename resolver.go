package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// resolveStartEnd returns the start and end line numbers for a Tag, along with
// the file content as lines.
// contextDir is the directory containing the tags file; it is prepended to
// tag.Path (which is relative to the tags file) when reading source files.
// The source file is read only when pattern matching is needed (tag.Line == 0).
// If the "end" extension field is absent and useTreeSitter is true and the file
// is a supported language, tree-sitter is used to determine the end line.
// If neither source provides an end line, endLine is returned as 0.
// When tree-sitter successfully resolves the end line, markTreeSitterUsed is
// called on ctx so that the access log middleware can record the fact.
func resolveStartEnd(ctx context.Context, tag Tag, contextDir string, useTreeSitter bool) (startLine, endLine int, fileLines []string, err error) {
	filePath := resolveFilePath(contextDir, tag.Path)

	var lines []string
	var data []byte

	if tag.Line == 0 && tag.Pattern != "" {
		data, err = os.ReadFile(filePath)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("reading file %s: %w", filePath, err)
		}
		lines = strings.Split(string(data), "\n")
	}

	// funcLine is the line where the definition itself begins (used for tree-sitter).
	funcLine := tag.Line
	if funcLine == 0 && tag.Pattern != "" {
		funcLine = findPatternLine(lines, tag.Pattern)
	}
	if funcLine <= 0 {
		return 0, 0, nil, fmt.Errorf("cannot determine start line for tag %q in %s", tag.Name, filePath)
	}

	// Ensure file is loaded so we can scan for leading comment lines.
	if lines == nil {
		data, err = os.ReadFile(filePath)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("reading file %s: %w", filePath, err)
		}
		lines = strings.Split(string(data), "\n")
	} else if data == nil {
		data = []byte(strings.Join(lines, "\n"))
	}

	// Extend start upward to include any immediately preceding comment block.
	// For tree-sitter-supported languages use the AST (handles block comments
	// and is language-aware); fall back to heuristic string matching otherwise.
	startLine = funcLine
	if useTreeSitter {
		if entry := lookupLang(tag.Path); entry != nil {
			tsStart, tsErr := entry.resolveStartForLang(data, funcLine)
			if tsErr == nil && tsStart > 0 {
				startLine = tsStart
			}
		} else {
			startLine = scanLeadingComments(lines, funcLine)
		}
	} else {
		startLine = scanLeadingComments(lines, funcLine)
	}

	if endStr, ok := tag.Extra["end"]; ok {
		if n, parseErr := strconv.Atoi(endStr); parseErr == nil {
			return startLine, n, lines, nil
		}
	}

	// end field absent: try tree-sitter using funcLine (the definition start, not the comment).
	if useTreeSitter {
		if entry := lookupLang(tag.Path); entry != nil {
			tsEnd, tsErr := entry.resolveEndForLang(data, funcLine)
			if tsErr == nil && tsEnd > 0 {
				markTreeSitterUsed(ctx)
				return startLine, tsEnd, lines, nil
			}
		}
	}

	return startLine, 0, lines, nil
}

// snippetForTag resolves a Snippet from a Tag by reading the source file.
// contextDir is the directory containing the tags file.
func snippetForTag(ctx context.Context, tag Tag, contextDir string, useTreeSitter bool) (Snippet, error) {
	startLine, endLine, lines, err := resolveStartEnd(ctx, tag, contextDir, useTreeSitter)
	if err != nil {
		return Snippet{}, err
	}

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
	startLine, endLine, _, err := resolveStartEnd(ctx, tag, contextDir, useTreeSitter)
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
