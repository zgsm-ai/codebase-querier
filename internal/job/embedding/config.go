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

// Language constants
const (
	Unknown    Language = "unknown"
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
		Language:       Go,
		sitterLanguage: sitter.NewLanguage(sittergo.Language()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Go)+queryExt),
		SupportedExts:  []string{".go"},
		Processor:      NewGoProcessor(),
	},
	{
		Language:       Python,
		sitterLanguage: sitter.NewLanguage(sitterpython.Language()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Python)+queryExt),
		SupportedExts:  []string{".py"},
		Processor:      NewPythonProcessor(),
	},
	{
		Language:       Java,
		sitterLanguage: sitter.NewLanguage(sitterjava.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Java)+queryExt), // Will be loaded from queries/java.scm
		SupportedExts:  []string{".java"},
		Processor:      NewJavaProcessor(),
	},
	{
		Language:       JavaScript,
		sitterLanguage: sitter.NewLanguage(sitterjavascript.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(JavaScript)+queryExt), // Will be loaded from queries/javascript.scm
		SupportedExts:  []string{".js", ".jsx"},
		Processor:      NewJavaScriptProcessor(),
	},
	{
		Language:       TypeScript,
		sitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTypescript()), // Type assertion
		Query:          filepath.Join(queryBaseDir, string(TypeScript)+queryExt),  // Will be loaded from queries/typescript.scm
		SupportedExts:  []string{".ts"},
		Processor:      NewJavaScriptProcessor(),
	},
	{
		Language:       TSX,
		sitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTSX()), // TSX uses the same language binding as TS, Type assertion
		Query:          filepath.Join(queryBaseDir, string(TSX)+queryExt),  // Will be loaded from queries/typescript_tsx.scm
		SupportedExts:  []string{".tsx"},
		Processor:      NewJavaScriptProcessor(),
	},
	{
		Language:       Rust,
		sitterLanguage: sitter.NewLanguage(sitterrust.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Rust)+queryExt), // Will be loaded from queries/rust.scm
		SupportedExts:  []string{".rs"},
		Processor:      NewRustProcessor(),
	},
	{
		Language:       C,
		sitterLanguage: sitter.NewLanguage(sitterc.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(C)+queryExt), // Will be loaded from queries/c.scm
		SupportedExts:  []string{".c", ".h"},
		Processor:      NewCProcessor(),
	},
	{
		Language:       CPP,
		sitterLanguage: sitter.NewLanguage(sittercpp.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(CPP)+queryExt), // Will be loaded from queries/cpp.scm
		SupportedExts:  []string{".cpp", ".cc", ".cxx", ".hpp"},
		Processor:      NewCppProcessor(),
	},
	{
		Language:       CSharp,
		sitterLanguage: sitter.NewLanguage(sittercsharp.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(CSharp)+queryExt), // Will be loaded from queries/csharp.scm
		SupportedExts:  []string{".cs"},
		Processor:      NewCSharpProcessor(),
	},
	{
		Language:       Ruby,
		sitterLanguage: sitter.NewLanguage(sitterruby.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Ruby)+queryExt), // Will be loaded from queries/ruby.scm
		SupportedExts:  []string{".rb"},
		Processor:      NewRubyProcessor(),
	},
	{
		Language:       PHP,
		sitterLanguage: sitter.NewLanguage(sitterphp.LanguagePHP()),       // Type assertion
		Query:          filepath.Join(queryBaseDir, string(PHP)+queryExt), // Will be loaded from queries/php.scm
		SupportedExts:  []string{".php", ".phtml"},
		Processor:      NewPhpProcessor(),
	},
	{
		Language:       Kotlin,
		sitterLanguage: sitter.NewLanguage(sitterkotlin.Language()),          // Uncommented
		Query:          filepath.Join(queryBaseDir, string(Kotlin)+queryExt), // Will be loaded from queries/kotlin.scm
		SupportedExts:  []string{".kt", ".kts"},
		Processor:      NewKotlinProcessor(),
	},
	{
		Language:       Scala,
		sitterLanguage: sitter.NewLanguage(sitterscala.Language()),          // Type assertion
		Query:          filepath.Join(queryBaseDir, string(Scala)+queryExt), // Will be loaded from queries/scala.scm
		SupportedExts:  []string{".scala", ".sc"},
		Processor:      NewScalaProcessor(),
	},
}
