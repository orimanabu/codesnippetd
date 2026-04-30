package main

import (
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

// langEntry describes a single tree-sitter language configuration.
type langEntry struct {
	name            string
	lang            func() *sitter.Language
	definitionTypes map[string]bool
	// Optional overrides for languages with custom logic (e.g. Lua).
	// If nil, the generic resolveStartWithTreeSitter/resolveEndWithTreeSitter is used.
	resolveEnd   func(content []byte, startLine int) (int, error)
	resolveStart func(content []byte, funcLine int) (int, error)
}

// langRegistry maps file extensions to language configurations.
var langRegistry = map[string]*langEntry{
	".go": {
		name:            "Go",
		lang:            golang.GetLanguage,
		definitionTypes: goDefinitionTypes,
	},
	".py": {
		name:            "Python",
		lang:            python.GetLanguage,
		definitionTypes: pythonDefinitionTypes,
	},
	".rb": {
		name:            "Ruby",
		lang:            ruby.GetLanguage,
		definitionTypes: rubyDefinitionTypes,
	},
	".java": {
		name:            "Java",
		lang:            java.GetLanguage,
		definitionTypes: javaDefinitionTypes,
	},
	".rs": {
		name:            "Rust",
		lang:            rust.GetLanguage,
		definitionTypes: rustDefinitionTypes,
	},
	".js": {
		name:            "JavaScript",
		lang:            javascript.GetLanguage,
		definitionTypes: jsDefinitionTypes,
	},
	".ts": {
		name:            "TypeScript",
		lang:            typescript.GetLanguage,
		definitionTypes: tsDefinitionTypes,
	},
	".hs": {
		name:            "Haskell",
		lang:            func() *sitter.Language { return sitter.NewLanguage(haskell.Language()) },
		definitionTypes: hsDefinitionTypes,
	},
	".kt": {
		name:            "Kotlin",
		lang:            kotlin.GetLanguage,
		definitionTypes: ktDefinitionTypes,
	},
	".php": {
		name:            "PHP",
		lang:            php.GetLanguage,
		definitionTypes: phpDefinitionTypes,
	},
	".ml": {
		name:            "OCaml",
		lang:            func() *sitter.Language { return sitter.NewLanguage(ocaml.LanguageOCaml()) },
		definitionTypes: ocamlMLDefinitionTypes,
	},
	".mli": {
		name:            "OCaml Interface",
		lang:            func() *sitter.Language { return sitter.NewLanguage(ocaml.LanguageOCamlInterface()) },
		definitionTypes: ocamlMLIDefinitionTypes,
	},
	".c": {
		name:            "C",
		lang:            c.GetLanguage,
		definitionTypes: cDefinitionTypes,
	},
	".h": {
		name:            "C",
		lang:            c.GetLanguage,
		definitionTypes: cDefinitionTypes,
	},
	".cc": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".cpp": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".cxx": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".hh": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".hpp": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".hxx": {
		name:            "C++",
		lang:            cpp.GetLanguage,
		definitionTypes: cppDefinitionTypes,
	},
	".lua": {
		name:            "Lua",
		lang:            lua.GetLanguage,
		definitionTypes: luaDefinitionTypes,
		resolveEnd:      resolveEndWithTreeSitterLua,
		resolveStart:    resolveStartWithTreeSitterLua,
	},
}

// lookupLang returns the langEntry for the given file path, or nil if the
// file extension is not supported.
func lookupLang(path string) *langEntry {
	ext := filepath.Ext(path)
	return langRegistry[ext]
}

// resolveStartForLang resolves the start line (including leading comments) for
// a definition at funcLine using tree-sitter. If the entry has a custom
// resolveStart function, it is used; otherwise the generic
// resolveStartWithTreeSitter is called.
func (e *langEntry) resolveStartForLang(content []byte, funcLine int) (int, error) {
	if e.resolveStart != nil {
		return e.resolveStart(content, funcLine)
	}
	return resolveStartWithTreeSitter(e.lang(), content, funcLine)
}

// resolveEndForLang resolves the end line for a definition starting at
// startLine using tree-sitter. If the entry has a custom resolveEnd function,
// it is used; otherwise the generic resolveEndWithTreeSitter is called.
func (e *langEntry) resolveEndForLang(content []byte, startLine int) (int, error) {
	if e.resolveEnd != nil {
		return e.resolveEnd(content, startLine)
	}
	return resolveEndWithTreeSitter(e.lang(), e.definitionTypes, content, startLine)
}
