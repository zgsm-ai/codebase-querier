package parser

import (
	"fmt"
	"os"

	sitter "github.com/tree-sitter/go-tree-sitter"
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

// Registry manages the registration and retrieval of language parsers.
type Registry struct {
	parsers          map[types.Language]CodeParser
	maxTokenPerBlock int
	overlapTokens    int
}

// Register registers a CodeParser for a specific language with the registry.
func (r *Registry) Register(lang types.Language, parser CodeParser) {
	r.parsers[lang] = parser
}

// Get retrieves the CodeParser for the given language from the registry.
func (r *Registry) Get(lang types.Language) (CodeParser, bool) {
	parser, ok := r.parsers[lang]
	return parser, ok
}

// GetAllParsers returns the map of all registered parsers.
// Note: Returning the map directly allows external access and modification.
// A more robust approach might be to return a copy or an iterator.
func (r *Registry) GetAllParsers() map[types.Language]CodeParser {
	return r.parsers
}

type RegistryFunc func(registry *Registry)

func WithMaxTokensPerBlock(maxTokensPerBlock int) RegistryFunc {
	return func(registry *Registry) {
		registry.maxTokenPerBlock = maxTokensPerBlock
	}
}

func WithOverlapTokens(overlapTokens int) RegistryFunc {
	return func(registry *Registry) {
		registry.overlapTokens = overlapTokens
	}
}

// NewParserRegistry initializes and registers all known language parsers.
func NewParserRegistry(opts ...RegistryFunc) (*Registry, error) {
	registry := &Registry{
		parsers: make(map[types.Language]CodeParser),
	}
	for _, opt := range opts {
		opt(registry)
	}

	parsersToRegister := []struct {
		lang    types.Language
		newFunc func(maxTokensPerBlock, overlapTokens int) (CodeParser, error)
	}{
		{types.Java, NewJavaParser},
		{types.Python, NewPythonParser},
		{types.Go, NewGoParser},
		{types.JavaScript, NewJavaScriptParser},
		{types.TypeScript, NewTypeScriptTSParser},
		{types.TSX, NewTypeScriptTSXParser},
		{types.Rust, NewRustParser},
		{types.C, NewCParser},
		{types.CPP, NewCPPParser},
		{types.CSharp, NewCSharpParser},
		{types.Ruby, NewRubyParser},
		{types.PHP, NewPhpParser},
		{types.Kotlin, NewKotlinParser},
		{types.Scala, NewScalaParser},
	}

	for _, p := range parsersToRegister {
		parser, err := p.newFunc(registry.maxTokenPerBlock, registry.overlapTokens)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize %s parser: %w", p.lang, err)
		}
		registry.Register(p.lang, parser)
	}

	return registry, nil
}

func loadQuery(lang *sitter.Language, queryFilePath string) (string, error) {
	// Read the query file
	queryContent, err := os.ReadFile(queryFilePath)
	if err != nil {
		return "", err
	}
	queryStr := string(queryContent)
	// Validate query early
	_, queryErr := sitter.NewQuery(lang, queryStr)
	if queryErr != nil {
		return "", err
	}
	return queryStr, nil
}
