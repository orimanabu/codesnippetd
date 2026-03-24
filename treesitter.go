package main

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	haskell "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	ocaml "github.com/tree-sitter/tree-sitter-ocaml/bindings/go"
)

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
