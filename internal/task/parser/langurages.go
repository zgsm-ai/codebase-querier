package parser

import (
	"github.com/zgsm-ai/codebase-indexer/internal/task/parser/lang"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

var supportedLanguages = []struct {
	lang          types.Language
	new           func() (lang.CodeParser, error)
	supportedExts []string
}{
	{types.Java, lang.NewJavaParser, []string{".java"}},
	{types.Python, lang.NewPythonParser, []string{".py"}},
	{types.Go, lang.NewGoParser, []string{".go"}},
	{types.JavaScript, lang.NewJavaScriptParser, []string{".js", ".jsx"}},
	{types.TypeScript, lang.NewTypeScriptTSParser, []string{".ts"}},
	{types.TSX, lang.NewTypeScriptTSXParser, []string{".tsx"}},
	{types.Rust, lang.NewRustParser, []string{".rs"}},
	{types.C, lang.NewCParser, []string{".c", ".h"}},
	{types.CPP, lang.NewCPPParser, []string{".cpp", ".cc", ".cxx", ".hpp"}},
	{types.CSharp, lang.NewCSharpParser, []string{".cs"}},
	{types.Ruby, lang.NewRubyParser, []string{".rb"}},
	{types.PHP, lang.NewPhpParser, []string{".php", ".phtml"}},
	{types.Kotlin, lang.NewKotlinParser, []string{".kt", ".kts"}},
	{types.Scala, lang.NewScalaParser, []string{".scala", ".sc"}},
}
