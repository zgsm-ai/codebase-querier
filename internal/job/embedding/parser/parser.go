package parser

import (
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// CodeParser defines the interface for language-specific code parsing.
type CodeParser interface {
	// Parse takes the code content and returns a slice of CodeBlocks based on syntax (e.g., functions, classes).
	Parse(code string, filePath string) ([]types.CodeBlock, error)
	// InferLanguage infers if this parser can handle the given file based on its path/extension.
	InferLanguage(filePath string) bool
	// Close releases any resources used by the parser.
	Close()
}

// RegisteredParsers stores the registered language parsers.
var RegisteredParsers = make(map[types.Language]CodeParser)

// registerParser registers a CodeParser for a specific language.
func registerParser(lang types.Language, parser CodeParser) {
	RegisteredParsers[lang] = parser
}

// GetParser retrieves the CodeParser for the given language.
func GetParser(lang types.Language) (CodeParser, bool) {
	parser, ok := RegisteredParsers[lang]
	return parser, ok
}

// InitRegisteredParsers initializes and registers all known language parsers.
func InitRegisteredParsers(maxTokensPerBlock, overlapTokens int) {
	// Call the New functions for each parser. These functions should handle their own registration.
	NewJavaParser(maxTokensPerBlock, overlapTokens)
	NewPythonParser(maxTokensPerBlock, overlapTokens)
	NewGoParser(maxTokensPerBlock, overlapTokens)
	NewJavaScriptParser(maxTokensPerBlock, overlapTokens)
	NewTypeScriptTSParser(maxTokensPerBlock, overlapTokens)
	NewTypeScriptTSXParser(maxTokensPerBlock, overlapTokens)
	NewRustParser(maxTokensPerBlock, overlapTokens)
	NewCParser(maxTokensPerBlock, overlapTokens)
	NewCPPParser(maxTokensPerBlock, overlapTokens)
	NewCSharpParser(maxTokensPerBlock, overlapTokens)
	NewRubyParser(maxTokensPerBlock, overlapTokens)
	NewPhpParser(maxTokensPerBlock, overlapTokens)
	NewKotlinParser(maxTokensPerBlock, overlapTokens)
	NewScalaParser(maxTokensPerBlock, overlapTokens)
	// Add calls for any new parsers here
}
