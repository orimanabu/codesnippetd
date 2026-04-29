package main

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	haskell "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	ocaml "github.com/tree-sitter/tree-sitter-ocaml/bindings/go"
)

// goDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Go source files.
var goDefinitionTypes = map[string]bool{
	"function_declaration": true,
	"method_declaration":   true,
	"type_declaration":     true,
	"const_declaration":    true,
	"var_declaration":      true,
}

// isGoFile reports whether path is a Go source file.
func isGoFile(path string) bool {
	return filepath.Ext(path) == ".go"
}

// resolveEndWithTreeSitterGo returns the 1-based end line of the Go
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterGo(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(golang.GetLanguage(), goDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterGo returns the 1-based start line (including any
// leading comment block) for the Go definition whose first line is funcLine.
func resolveStartWithTreeSitterGo(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(golang.GetLanguage(), content, funcLine)
}

// pythonDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Python source files.
var pythonDefinitionTypes = map[string]bool{
	"function_definition":  true,
	"class_definition":     true,
	"decorated_definition": true,
}

// isPyFile reports whether path is a Python source file.
func isPyFile(path string) bool {
	return filepath.Ext(path) == ".py"
}

// resolveEndWithTreeSitterPython returns the 1-based end line of the Python
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterPython(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(python.GetLanguage(), pythonDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterPython returns the 1-based start line (including
// any leading comment block) for the Python definition whose first line is
// funcLine.
func resolveStartWithTreeSitterPython(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(python.GetLanguage(), content, funcLine)
}

// rubyDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Ruby source files.
var rubyDefinitionTypes = map[string]bool{
	"method":           true,
	"singleton_method": true,
	"class":            true,
	"module":           true,
	"singleton_class":  true,
}

// isRbFile reports whether path is a Ruby source file.
func isRbFile(path string) bool {
	return filepath.Ext(path) == ".rb"
}

// resolveEndWithTreeSitterRuby returns the 1-based end line of the Ruby
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterRuby(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(ruby.GetLanguage(), rubyDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterRuby returns the 1-based start line (including
// any leading comment block) for the Ruby definition whose first line is
// funcLine.
func resolveStartWithTreeSitterRuby(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(ruby.GetLanguage(), content, funcLine)
}

// rustDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Rust source files.
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

// jsDefinitionTypes is the set of tree-sitter node types treated as
// definitions in JavaScript source files.
var jsDefinitionTypes = map[string]bool{
	"function_declaration":           true,
	"generator_function_declaration": true,
	"class_declaration":              true,
	"method_definition":              true,
	"lexical_declaration":            true,
	"variable_declaration":           true,
	"export_statement":               true,
}

// tsDefinitionTypes is the set of tree-sitter node types treated as
// definitions in TypeScript source files. It includes all JS definition types
// plus TypeScript-specific constructs.
var tsDefinitionTypes = map[string]bool{
	// shared with JavaScript
	"function_declaration":           true,
	"generator_function_declaration": true,
	"class_declaration":              true,
	"method_definition":              true,
	"lexical_declaration":            true,
	"variable_declaration":           true,
	"export_statement":               true,
	// TypeScript-specific
	"interface_declaration":   true,
	"type_alias_declaration":  true,
	"enum_declaration":        true,
	"abstract_class_declaration": true,
	"internal_module":         true,
}

// isCommentNodeType reports whether nodeType represents a comment node in any
// of the tree-sitter grammars used by this program.
func isCommentNodeType(nodeType string) bool {
	switch nodeType {
	case "comment",           // Go, JS, TS, PHP, OCaml
		"line_comment",       // Rust, Kotlin
		"block_comment",      // Rust, Kotlin
		"multiline_comment",  // some grammars
		"shell_comment",      // PHP (#!)
		"doc_comment":        // some grammars
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

// resolveStartWithTreeSitterRust returns the 1-based start line (including any
// leading comment block) for the Rust definition whose first line is funcLine.
func resolveStartWithTreeSitterRust(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(rust.GetLanguage(), content, funcLine)
}

// resolveStartWithTreeSitterJS returns the 1-based start line (including any
// leading comment block) for the JavaScript definition whose first line is funcLine.
func resolveStartWithTreeSitterJS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(javascript.GetLanguage(), content, funcLine)
}

// resolveStartWithTreeSitterTS returns the 1-based start line (including any
// leading comment block) for the TypeScript definition whose first line is funcLine.
func resolveStartWithTreeSitterTS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(typescript.GetLanguage(), content, funcLine)
}

// resolveStartWithTreeSitterHS returns the 1-based start line (including any
// leading comment block) for the Haskell definition whose first line is funcLine.
func resolveStartWithTreeSitterHS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(haskell.Language()), content, funcLine)
}

// resolveStartWithTreeSitterKotlin returns the 1-based start line (including any
// leading comment block) for the Kotlin definition whose first line is funcLine.
func resolveStartWithTreeSitterKotlin(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(kotlin.GetLanguage(), content, funcLine)
}

// resolveStartWithTreeSitterPHP returns the 1-based start line (including any
// leading comment block) for the PHP definition whose first line is funcLine.
func resolveStartWithTreeSitterPHP(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(php.GetLanguage(), content, funcLine)
}

// resolveStartWithTreeSitterOCaml returns the 1-based start line (including any
// leading comment block) for the OCaml (.ml) definition whose first line is funcLine.
func resolveStartWithTreeSitterOCaml(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCaml()), content, funcLine)
}

// resolveStartWithTreeSitterOCamlInterface returns the 1-based start line
// (including any leading comment block) for the OCaml (.mli) definition whose
// first line is funcLine.
func resolveStartWithTreeSitterOCamlInterface(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCamlInterface()), content, funcLine)
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

// resolveEndWithTreeSitterRust returns the 1-based end line of the Rust
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterRust(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(rust.GetLanguage(), rustDefinitionTypes, content, startLine)
}

// resolveEndWithTreeSitterJS returns the 1-based end line of the JavaScript
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterJS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(javascript.GetLanguage(), jsDefinitionTypes, content, startLine)
}

// resolveEndWithTreeSitterTS returns the 1-based end line of the TypeScript
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterTS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(typescript.GetLanguage(), tsDefinitionTypes, content, startLine)
}

// isRustFile reports whether path is a Rust source file.
func isRustFile(path string) bool {
	return filepath.Ext(path) == ".rs"
}

// isJSFile reports whether path is a JavaScript source file.
func isJSFile(path string) bool {
	return filepath.Ext(path) == ".js"
}

// isTSFile reports whether path is a TypeScript source file.
func isTSFile(path string) bool {
	return filepath.Ext(path) == ".ts"
}

// isHSFile reports whether path is a Haskell source file.
func isHSFile(path string) bool {
	return filepath.Ext(path) == ".hs"
}

// hsDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Haskell source files.
// Note: "type_synomym" is the spelling used by the tree-sitter-haskell grammar.
var hsDefinitionTypes = map[string]bool{
	"function":     true,
	"signature":    true,
	"data_type":    true,
	"class":        true,
	"instance":     true,
	"newtype":      true,
	"type_synomym": true,
}

// resolveEndWithTreeSitterHS returns the 1-based end line of the Haskell
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterHS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(haskell.Language()), hsDefinitionTypes, content, startLine)
}

// isKtFile reports whether path is a Kotlin source file.
func isKtFile(path string) bool {
	return filepath.Ext(path) == ".kt"
}

// ktDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Kotlin source files. Kotlin interfaces and enums are
// represented as class_declaration in the tree-sitter grammar.
var ktDefinitionTypes = map[string]bool{
	"function_declaration": true,
	"class_declaration":    true,
	"object_declaration":   true,
}

// resolveEndWithTreeSitterKotlin returns the 1-based end line of the Kotlin
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterKotlin(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(kotlin.GetLanguage(), ktDefinitionTypes, content, startLine)
}

// isPHPFile reports whether path is a PHP source file.
func isPHPFile(path string) bool {
	return filepath.Ext(path) == ".php"
}

// phpDefinitionTypes is the set of tree-sitter node types treated as
// definitions in PHP source files.
var phpDefinitionTypes = map[string]bool{
	"function_definition":   true,
	"method_declaration":    true,
	"class_declaration":     true,
	"interface_declaration": true,
	"trait_declaration":     true,
}

// resolveEndWithTreeSitterPHP returns the 1-based end line of the PHP
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterPHP(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(php.GetLanguage(), phpDefinitionTypes, content, startLine)
}

// isMLFile reports whether path is an OCaml implementation file.
func isMLFile(path string) bool {
	return filepath.Ext(path) == ".ml"
}

// isMLIFile reports whether path is an OCaml interface file.
func isMLIFile(path string) bool {
	return filepath.Ext(path) == ".mli"
}

// ocamlMLDefinitionTypes is the set of tree-sitter node types treated as
// definitions in OCaml implementation (.ml) source files.
var ocamlMLDefinitionTypes = map[string]bool{
	"value_definition":       true,
	"type_definition":        true,
	"module_definition":      true,
	"module_type_definition": true,
	"class_definition":       true,
}

// ocamlMLIDefinitionTypes is the set of tree-sitter node types treated as
// definitions in OCaml interface (.mli) source files.
var ocamlMLIDefinitionTypes = map[string]bool{
	"value_specification":    true,
	"type_definition":        true,
	"module_type_definition": true,
}

// resolveEndWithTreeSitterOCaml returns the 1-based end line of the OCaml
// implementation (.ml) definition starting at startLine (1-based).
func resolveEndWithTreeSitterOCaml(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCaml()), ocamlMLDefinitionTypes, content, startLine)
}

// resolveEndWithTreeSitterOCamlInterface returns the 1-based end line of the
// OCaml interface (.mli) definition starting at startLine (1-based).
func resolveEndWithTreeSitterOCamlInterface(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCamlInterface()), ocamlMLIDefinitionTypes, content, startLine)
}
