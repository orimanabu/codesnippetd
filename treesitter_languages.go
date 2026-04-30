package main

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/lua"
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

// javaDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Java source files.
var javaDefinitionTypes = map[string]bool{
	"class_declaration":           true,
	"interface_declaration":       true,
	"method_declaration":          true,
	"constructor_declaration":     true,
	"enum_declaration":            true,
	"annotation_type_declaration": true,
	"record_declaration":          true,
}

// isJavaFile reports whether path is a Java source file.
func isJavaFile(path string) bool {
	return filepath.Ext(path) == ".java"
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

// isRustFile reports whether path is a Rust source file.
func isRustFile(path string) bool {
	return filepath.Ext(path) == ".rs"
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

// isJSFile reports whether path is a JavaScript source file.
func isJSFile(path string) bool {
	return filepath.Ext(path) == ".js"
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
	"interface_declaration":      true,
	"type_alias_declaration":     true,
	"enum_declaration":           true,
	"abstract_class_declaration": true,
	"internal_module":            true,
}

// isTSFile reports whether path is a TypeScript source file.
func isTSFile(path string) bool {
	return filepath.Ext(path) == ".ts"
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

// isHSFile reports whether path is a Haskell source file.
func isHSFile(path string) bool {
	return filepath.Ext(path) == ".hs"
}

// ktDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Kotlin source files. Kotlin interfaces and enums are
// represented as class_declaration in the tree-sitter grammar.
var ktDefinitionTypes = map[string]bool{
	"function_declaration": true,
	"class_declaration":    true,
	"object_declaration":   true,
}

// isKtFile reports whether path is a Kotlin source file.
func isKtFile(path string) bool {
	return filepath.Ext(path) == ".kt"
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

// isPHPFile reports whether path is a PHP source file.
func isPHPFile(path string) bool {
	return filepath.Ext(path) == ".php"
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

// isMLFile reports whether path is an OCaml implementation file.
func isMLFile(path string) bool {
	return filepath.Ext(path) == ".ml"
}

// isMLIFile reports whether path is an OCaml interface file.
func isMLIFile(path string) bool {
	return filepath.Ext(path) == ".mli"
}

// cDefinitionTypes is the set of tree-sitter node types treated as definitions
// in C source files.
var cDefinitionTypes = map[string]bool{
	"function_definition": true,
	"struct_specifier":    true,
	"enum_specifier":      true,
	"union_specifier":     true,
	"type_definition":     true,
}

// isCFile reports whether path is a C source or header file.
func isCFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".c" || ext == ".h"
}

// cppDefinitionTypes is the set of tree-sitter node types treated as
// definitions in C++ source files. It extends cDefinitionTypes with
// C++-specific constructs.
var cppDefinitionTypes = map[string]bool{
	// shared with C
	"function_definition": true,
	"struct_specifier":    true,
	"enum_specifier":      true,
	"union_specifier":     true,
	"type_definition":     true,
	// C++-specific
	"class_specifier":      true,
	"template_declaration": true,
	"namespace_definition": true,
}

// isCppFile reports whether path is a C++ source or header file.
func isCppFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".cc" || ext == ".cpp" || ext == ".cxx" || ext == ".hh" || ext == ".hpp" || ext == ".hxx"
}

// luaDefinitionTypes is the set of tree-sitter node types treated as
// definitions in Lua source files.
var luaDefinitionTypes = map[string]bool{
	"function_statement": true,
}

// isLuaFile reports whether path is a Lua source file.
func isLuaFile(path string) bool {
	return filepath.Ext(path) == ".lua"
}

// resolveEndWithTreeSitterLua returns the 1-based end line of the Lua function
// definition starting at startLine (1-based). It uses a Lua-specific node
// search because the Lua tree-sitter grammar attaches leading newlines to
// function_statement nodes, making their StartPoint().Row differ from the
// actual "function" keyword line reported by ctags.
func resolveEndWithTreeSitterLua(content []byte, startLine int) (int, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lua.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return 0, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()
	targetRow := uint32(startLine - 1)
	node := findLuaFunctionStatementByFunctionStartRow(tree.RootNode(), targetRow)
	if node == nil {
		return 0, fmt.Errorf("no Lua function found at line %d", startLine)
	}
	return int(node.EndPoint().Row) + 1, nil
}

// resolveStartWithTreeSitterLua returns the 1-based start line (including any
// leading comment block) for the Lua function whose first line is funcLine.
// It uses end-row matching for comments to work around the Lua grammar's
// leading-newline quirk.
func resolveStartWithTreeSitterLua(content []byte, funcLine int) (int, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lua.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return funcLine, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()
	startRow := findCommentStartRowLua(tree.RootNode(), uint32(funcLine-1))
	return int(startRow) + 1, nil
}
