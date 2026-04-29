package main

import "strings"

// isCommentLine reports whether s (already trimmed of leading whitespace) is a
// single-line comment in a supported language.
func isCommentLine(s string) bool {
	return strings.HasPrefix(s, "//") || // Go, Rust, JS, TS, Kotlin, PHP, Java, C++
		strings.HasPrefix(s, "#") || // Python, Ruby, Shell
		strings.HasPrefix(s, "--") || // Haskell, SQL, Lua
		strings.HasPrefix(s, "* ") || // block comment continuation (e.g. /* ... */ bodies)
		strings.HasPrefix(s, "/*") || // block comment start
		strings.HasPrefix(s, "(*") // OCaml
}

// scanLeadingComments walks backward from startLine (1-based) through lines and
// returns the new start line extended to include any immediately preceding comment
// lines. Only contiguous comment lines with no blank line between them are included.
func scanLeadingComments(lines []string, startLine int) int {
	for startLine > 1 {
		prev := strings.TrimSpace(lines[startLine-2]) // lines is 0-indexed
		if isCommentLine(prev) {
			startLine--
		} else {
			break
		}
	}
	return startLine
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
