package main

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// isCommentNodeType reports whether nodeType represents a comment node in any
// of the tree-sitter grammars used by this program.
func isCommentNodeType(nodeType string) bool {
	switch nodeType {
	case "comment",          // Go, JS, TS, PHP, OCaml
		"line_comment",      // Rust, Kotlin
		"block_comment",     // Rust, Kotlin
		"multiline_comment", // some grammars
		"shell_comment",     // PHP (#!)
		"doc_comment":       // some grammars
		return true
	}
	return false
}

// rowHasCommentNode reports whether any node in the subtree rooted at n covers
// row (0-based) and is a comment node type. Block comments that span multiple
// rows are detected correctly because the check uses the node's full span.
func rowHasCommentNode(n *sitter.Node, row uint32) bool {
	if isCommentNodeType(n.Type()) && n.StartPoint().Row <= row && n.EndPoint().Row >= row {
		return true
	}
	for i := range int(n.ChildCount()) {
		child := n.Child(i)
		if child.StartPoint().Row > row {
			break
		}
		if child.EndPoint().Row < row {
			continue
		}
		if rowHasCommentNode(child, row) {
			return true
		}
	}
	return false
}

// findCommentStartRow walks backward from funcRow (0-based) and returns the
// earliest row that belongs to a contiguous block of comment lines immediately
// preceding funcRow (no blank lines between the comment block and the
// definition). Returns funcRow if there are no preceding comment lines.
func findCommentStartRow(root *sitter.Node, funcRow uint32) uint32 {
	row := funcRow
	for row > 0 && rowHasCommentNode(root, row-1) {
		row--
	}
	return row
}

// resolveStartWithTreeSitter parses content with lang and returns the 1-based
// start line including any leading comment block immediately before funcLine
// (1-based). Returns funcLine unchanged on error.
func resolveStartWithTreeSitter(lang *sitter.Language, content []byte, funcLine int) (int, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return funcLine, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()
	startRow := findCommentStartRow(tree.RootNode(), uint32(funcLine-1))
	return int(startRow) + 1, nil
}

// findDefinitionNodeAtRow returns the outermost node of one of the given
// definition types whose start row matches row (0-indexed). Falls back to
// any named node at that row if no definition-type node is found.
func findDefinitionNodeAtRow(n *sitter.Node, row uint32, definitionTypes map[string]bool) *sitter.Node {
	if n.StartPoint().Row == row && n.IsNamed() && definitionTypes[n.Type()] {
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
		if result := findDefinitionNodeAtRow(child, row, definitionTypes); result != nil {
			return result
		}
		if child.StartPoint().Row == row && child.IsNamed() && fallback == nil {
			fallback = child
		}
	}
	return fallback
}

// resolveEndWithTreeSitter parses content using lang and returns the 1-based
// end line number of the definition that starts at startLine (1-based).
func resolveEndWithTreeSitter(lang *sitter.Language, definitionTypes map[string]bool, content []byte, startLine int) (int, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return 0, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()

	targetRow := uint32(startLine - 1) // tree-sitter uses 0-based rows
	node := findDefinitionNodeAtRow(tree.RootNode(), targetRow, definitionTypes)
	if node == nil {
		return 0, fmt.Errorf("no definition found at line %d", startLine)
	}
	return int(node.EndPoint().Row) + 1, nil // convert back to 1-based
}

// rowEndsWithCommentNode reports whether any comment node in the subtree
// rooted at n has its last row equal to row (0-based). This is used for Lua
// because the Lua grammar attaches preceding newlines to comment nodes,
// causing StartPoint().Row to precede the actual comment text. EndPoint().Row
// always corresponds to the line containing the comment text.
func rowEndsWithCommentNode(n *sitter.Node, row uint32) bool {
	if isCommentNodeType(n.Type()) && n.EndPoint().Row == row {
		return true
	}
	for i := range int(n.ChildCount()) {
		child := n.Child(i)
		if child.EndPoint().Row < row || child.StartPoint().Row > row {
			continue
		}
		if rowEndsWithCommentNode(child, row) {
			return true
		}
	}
	return false
}

// findCommentStartRowLua walks backward from funcRow (0-based) and returns the
// earliest row that is immediately preceded by a comment whose text ends on
// that row (using end-row matching to handle the Lua grammar's leading-newline
// quirk). Returns funcRow if no preceding comment lines exist.
func findCommentStartRowLua(root *sitter.Node, funcRow uint32) uint32 {
	row := funcRow
	for row > 0 && rowEndsWithCommentNode(root, row-1) {
		row--
	}
	return row
}

// findLuaFunctionStatementByFunctionStartRow searches the subtree rooted at n
// for a function_statement node whose function_start child has its EndPoint on
// the given row (0-based). The Lua grammar attaches leading newlines to the
// function_statement and function_start nodes, so StartPoint().Row of those
// nodes does not match the actual "function" keyword line; EndPoint().Row of
// function_start does.
func findLuaFunctionStatementByFunctionStartRow(n *sitter.Node, row uint32) *sitter.Node {
	if n.Type() == "function_statement" {
		for i := range int(n.ChildCount()) {
			child := n.Child(i)
			if child.Type() == "function_start" && child.EndPoint().Row == row {
				return n
			}
		}
	}
	for i := range int(n.ChildCount()) {
		child := n.Child(i)
		if child.StartPoint().Row > row || child.EndPoint().Row < row {
			continue
		}
		if result := findLuaFunctionStatementByFunctionStartRow(child, row); result != nil {
			return result
		}
	}
	return nil
}
