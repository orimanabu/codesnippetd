package main

import (
	"context"
	"fmt"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
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

// resolveEndWithTreeSitterJava returns the 1-based end line of the Java
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterJava(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(java.GetLanguage(), javaDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterJava returns the 1-based start line (including
// any leading comment block) for the Java definition whose first line is
// funcLine.
func resolveStartWithTreeSitterJava(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(java.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterRust returns the 1-based end line of the Rust
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterRust(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(rust.GetLanguage(), rustDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterRust returns the 1-based start line (including any
// leading comment block) for the Rust definition whose first line is funcLine.
func resolveStartWithTreeSitterRust(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(rust.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterJS returns the 1-based end line of the JavaScript
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterJS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(javascript.GetLanguage(), jsDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterJS returns the 1-based start line (including any
// leading comment block) for the JavaScript definition whose first line is funcLine.
func resolveStartWithTreeSitterJS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(javascript.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterTS returns the 1-based end line of the TypeScript
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterTS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(typescript.GetLanguage(), tsDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterTS returns the 1-based start line (including any
// leading comment block) for the TypeScript definition whose first line is funcLine.
func resolveStartWithTreeSitterTS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(typescript.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterHS returns the 1-based end line of the Haskell
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterHS(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(haskell.Language()), hsDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterHS returns the 1-based start line (including any
// leading comment block) for the Haskell definition whose first line is funcLine.
func resolveStartWithTreeSitterHS(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(haskell.Language()), content, funcLine)
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

// resolveEndWithTreeSitterKotlin returns the 1-based end line of the Kotlin
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterKotlin(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(kotlin.GetLanguage(), ktDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterKotlin returns the 1-based start line (including any
// leading comment block) for the Kotlin definition whose first line is funcLine.
func resolveStartWithTreeSitterKotlin(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(kotlin.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterPHP returns the 1-based end line of the PHP
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterPHP(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(php.GetLanguage(), phpDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterPHP returns the 1-based start line (including any
// leading comment block) for the PHP definition whose first line is funcLine.
func resolveStartWithTreeSitterPHP(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(php.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterOCaml returns the 1-based end line of the OCaml
// implementation (.ml) definition starting at startLine (1-based).
func resolveEndWithTreeSitterOCaml(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCaml()), ocamlMLDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterOCaml returns the 1-based start line (including any
// leading comment block) for the OCaml (.ml) definition whose first line is funcLine.
func resolveStartWithTreeSitterOCaml(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCaml()), content, funcLine)
}

// resolveEndWithTreeSitterOCamlInterface returns the 1-based end line of the
// OCaml interface (.mli) definition starting at startLine (1-based).
func resolveEndWithTreeSitterOCamlInterface(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCamlInterface()), ocamlMLIDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterOCamlInterface returns the 1-based start line
// (including any leading comment block) for the OCaml (.mli) definition whose
// first line is funcLine.
func resolveStartWithTreeSitterOCamlInterface(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(sitter.NewLanguage(ocaml.LanguageOCamlInterface()), content, funcLine)
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

// resolveEndWithTreeSitterC returns the 1-based end line of the C definition
// starting at startLine (1-based).
func resolveEndWithTreeSitterC(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(c.GetLanguage(), cDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterC returns the 1-based start line (including any
// leading comment block) for the C definition whose first line is funcLine.
func resolveStartWithTreeSitterC(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(c.GetLanguage(), content, funcLine)
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

// resolveEndWithTreeSitterCpp returns the 1-based end line of the C++
// definition starting at startLine (1-based).
func resolveEndWithTreeSitterCpp(content []byte, startLine int) (int, error) {
	return resolveEndWithTreeSitter(cpp.GetLanguage(), cppDefinitionTypes, content, startLine)
}

// resolveStartWithTreeSitterCpp returns the 1-based start line (including any
// leading comment block) for the C++ definition whose first line is funcLine.
func resolveStartWithTreeSitterCpp(content []byte, funcLine int) (int, error) {
	return resolveStartWithTreeSitter(cpp.GetLanguage(), content, funcLine)
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
