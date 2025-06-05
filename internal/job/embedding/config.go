package embedding

import (
	sitterkotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittercsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	sitterc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	sitterjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	sitterphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	sitterruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	sitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	sitterscala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"path/filepath"
)

// Language represents a programming language.
type Language string

const (
	Unknown    Language = ""
	Java       Language = "java"
	Python     Language = "python"
	Go         Language = "go"
	JavaScript Language = "javascript"
	TypeScript Language = "typescript"
	TSX        Language = "tsx"
	Rust       Language = "rust"
	C          Language = "c"
	CPP        Language = "cpp"
	CSharp     Language = "csharp"
	Ruby       Language = "ruby"
	PHP        Language = "php"
	Kotlin     Language = "kotlin"
	Scala      Language = "scala"
)

const queryBaseDir = "queries"
const queryExt = ".scm"

// supportedLanguagesConfigs lists the configurations for all supported languages.
// The Query field will be populated in NewParserRegistry by loading the .scm files.
var supportedLanguagesConfigs = []LanguageConfig{ // This variable must be exported (capital S)
	{
		Name:           Go,
		sitterLanguage: sitter.NewLanguage(sittergo.Language()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Go)+queryExt),
		SupportedExts:  []string{".go"},
		ProcessMatch:   processGoMatch,
	},
	{
		Name:           Python,
		sitterLanguage: sitter.NewLanguage(sitterpython.Language()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Python)+queryExt),
		SupportedExts:  []string{".py"},
		ProcessMatch:   processPythonMatch,
	},
	{
		Name:           Java,
		sitterLanguage: sitter.NewLanguage(sitterjava.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Java)+queryExt), // Will be loaded from queries/java.scm
		SupportedExts:  []string{".java"},
		ProcessMatch:   processJavaMatch,
	},
	{
		Name:           JavaScript,
		sitterLanguage: sitter.NewLanguage(sitterjavascript.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(JavaScript)+queryExt), // Will be loaded from queries/javascript.scm
		SupportedExts:  []string{".js", ".jsx"},
		ProcessMatch:   processJavaScriptMatch,
	},
	{
		Name:           TypeScript,
		sitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTypescript()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(TypeScript)+queryExt),  // Will be loaded from queries/typescript.scm
		SupportedExts:  []string{".ts"},
		ProcessMatch:   processTypescriptMatch,
	},
	{
		Name:           TSX,
		sitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTSX()), // TSX uses the same language binding as TS, Type assertion
		Query:          filepath.Join(queryBaseDir, string(TSX)+queryExt),  // Will be loaded from queries/typescript_tsx.scm
		SupportedExts:  []string{".tsx"},
		ProcessMatch:   processTsxMatch,
	},
	{
		Name:           Rust,
		sitterLanguage: sitter.NewLanguage(sitterrust.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Rust)+queryExt), // Will be loaded from queries/rust.scm
		SupportedExts:  []string{".rs"},
		ProcessMatch:   processRustMatch,
	},
	{
		Name:           C,
		sitterLanguage: sitter.NewLanguage(sitterc.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(C)+queryExt), // Will be loaded from queries/c.scm
		SupportedExts:  []string{".c", ".h"},
		ProcessMatch:   processCMatch,
	},
	{
		Name:           CPP,
		sitterLanguage: sitter.NewLanguage(sittercpp.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(CPP)+queryExt), // Will be loaded from queries/cpp.scm
		SupportedExts:  []string{".cpp", ".cc", ".cxx", ".hpp"},
		ProcessMatch:   processCPPMatch,
	},
	{
		Name:           CSharp,
		sitterLanguage: sitter.NewLanguage(sittercsharp.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(CSharp)+queryExt), // Will be loaded from queries/csharp.scm
		SupportedExts:  []string{".cs"},
		ProcessMatch:   processCSharpMatch,
	},
	{
		Name:           Ruby,
		sitterLanguage: sitter.NewLanguage(sitterruby.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Ruby)+queryExt), // Will be loaded from queries/ruby.scm
		SupportedExts:  []string{".rb"},
		ProcessMatch:   processRubyMatch,
	},
	{
		Name:           PHP,
		sitterLanguage: sitter.NewLanguage(sitterphp.LanguagePHP()),       // Type assertion
		Query:          filepath.Join(queryBaseDir, string(PHP)+queryExt), // Will be loaded from queries/php.scm
		SupportedExts:  []string{".php", ".phtml"},
		ProcessMatch:   processPhpMatch,
	},
	{
		Name:           Kotlin,
		sitterLanguage: sitter.NewLanguage(sitterkotlin.Language()),          // Uncommented
		Query:          filepath.Join(queryBaseDir, string(Kotlin)+queryExt), // Will be loaded from queries/kotlin.scm
		SupportedExts:  []string{".kt", ".kts"},
		ProcessMatch:   processKotlinMatch,
	},
	{
		Name:           Scala,
		sitterLanguage: sitter.NewLanguage(sitterscala.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Scala)+queryExt), // Will be loaded from queries/scala.scm
		SupportedExts:  []string{".scala", ".sc"},
		ProcessMatch:   processScalaMatch,
	},
}

// getLanguageByExt finds the LanguageConfig for a given file extension.
func getLanguageByExt(ext string) (*LanguageConfig, bool) {
	for i := range supportedLanguagesConfigs {
		config := &supportedLanguagesConfigs[i]
		for _, supportedExt := range config.SupportedExts {
			if supportedExt == ext {
				return config, true
			}
		}
	}
	return nil, false
}
