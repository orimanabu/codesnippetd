package main

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

// rustDefinitionTypes is the set of tree-sitter node types treated as
// top-level definitions in Rust source files.
var rustDefinitionTypes = map[string]bool{
	"function_item":           true,
	"function_signature_item": true,
	"struct_item":             true,
	"enum_item":               true,
	"trait_item":              true,
	"impl_item":               true,
	"type_item":               true,
	"const_item":              true,
	"static_item":             true,
	"mod_item":                true,
}

// findDefinitionNodeAtRow returns the outermost definition-type node whose
// start row matches row (0-indexed). Falls back to any named node if no
// definition node is found.
func findDefinitionNodeAtRow(n *sitter.Node, row uint32) *sitter.Node {
	if n.StartPoint().Row == row && n.IsNamed() && rustDefinitionTypes[n.Type()] {
		return n
	}
	var fallback *sitter.Node
	for i := range int(n.ChildCount()) {
		child := n.Child(i)
		if child.StartPoint().Row > row {
			break
		}
		if child.EndPoint().Row < row {
			continue
		}
		if result := findDefinitionNodeAtRow(child, row); result != nil {
			return result
		}
		if child.StartPoint().Row == row && child.IsNamed() && fallback == nil {
			fallback = child
		}
	}
	return fallback
}

// resolveEndWithTreeSitterRust parses a Rust source file and returns the
// 1-based end line number of the definition that starts at startLine (1-based).
func resolveEndWithTreeSitterRust(content []byte, startLine int) (int, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(rust.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return 0, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()

	targetRow := uint32(startLine - 1) // tree-sitter uses 0-based rows
	node := findDefinitionNodeAtRow(tree.RootNode(), targetRow)
	if node == nil {
		return 0, fmt.Errorf("no definition found at line %d", startLine)
	}
	return int(node.EndPoint().Row) + 1, nil // convert back to 1-based
}

// isRustFile reports whether path is a Rust source file.
func isRustFile(path string) bool {
	return filepath.Ext(path) == ".rs"
}
