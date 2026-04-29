package main

import (
	"fmt"
	"net/http"
	"os/user"
	"path/filepath"
	"strings"
)

// tagsFileForContext resolves the tags file path given an optional context query param.
// If context is empty, "./tags" is used. If context is an absolute path, "<context>/tags"
// is used directly. Otherwise "./<context>/tags" is used.
func tagsFileForContext(context string) string {
	if context == "" {
		return filepath.Join(".", "tags")
	}
	if filepath.IsAbs(context) {
		return filepath.Join(context, "tags")
	}
	return filepath.Join(".", context, "tags")
}

// resolveTagsPath returns the tags file path to use for a request.
// If tagsParam is non-empty it is used directly (as provided by the "tags" query parameter).
// Otherwise tagsFileForContext(context) is used.
func resolveTagsPath(context, tagsParam string) string {
	if tagsParam != "" {
		return tagsParam
	}
	return tagsFileForContext(context)
}

// expandTilde replaces a leading "~" in path with the current user's home directory.
// Paths that do not start with "~" are returned unchanged.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("expanding ~: %w", err)
	}
	return u.HomeDir + path[1:], nil
}

// queryTagsPath reads the "context" and "tags" query parameters from r, expands
// any leading "~" to the current user's home directory, and returns the resolved
// tags file path.
func queryTagsPath(r *http.Request) (string, error) {
	context, err := expandTilde(r.URL.Query().Get("context"))
	if err != nil {
		return "", err
	}
	tagsParam, err := expandTilde(r.URL.Query().Get("tags"))
	if err != nil {
		return "", err
	}
	return resolveTagsPath(context, tagsParam), nil
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
