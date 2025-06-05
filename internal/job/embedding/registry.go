package embedding

import (
	"fmt"
	"os"

	"github.com/tiktoken-go/tokenizer"
)

const (
	GoLanguage         Language = "go"
	PythonLanguage     Language = "python"
	JavaLanguage       Language = "java"
	JavaScriptLanguage Language = "javascript"
)

// ParserRegistry manages Tree-sitter parsers for different languages.
type ParserRegistry struct {
	parsers map[Language]*GenericParser
	// You could add a map here for faster lookup by file extension or language type
	configMap map[string]*LanguageConfig // Map from file extension to config
}

// NewParserRegistry creates a new ParserRegistry and initializes parsers for supported languages.
func NewParserRegistry(tokenizer tokenizer.Codec, maxTokensPerChunk, overlapTokens int) (*ParserRegistry, error) {
	registry := &ParserRegistry{}
	registry.configMap = make(map[string]*LanguageConfig)
	registry.parsers = make(map[Language]*GenericParser)

	// Load query files and create parsers for supported languages.
	for i := range supportedLanguagesConfigs {
		config := &supportedLanguagesConfigs[i]
		queryFilePath := config.Query
		queryContent, err := os.ReadFile(queryFilePath)
		if err != nil {
			// If a query file is missing, we might want to log a warning or return an error
			// For now, let's return an error to ensure all necessary queries are present.
			return nil, fmt.Errorf("failed to read query file %s for %s: %w", queryFilePath, config.Language, err)
		}
		config.Query = string(queryContent) // Store the query content in the config

		// Create and add the parser to the registry
		parser, err := NewGenericParser(*config, tokenizer, maxTokensPerChunk, overlapTokens)
		if err != nil {
			return nil, fmt.Errorf("failed to create generic parser for %s: %w", config.Language, err)
		}
		registry.parsers[config.Language] = parser

		// Populate the configMap for faster lookup by extension
		for _, ext := range config.SupportedExts {
			registry.configMap[ext] = config
		}
	}

	return registry, nil
}

// GetParserByFileExtension retrieves a parser for the given file extension.
func (r *ParserRegistry) GetParserByFileExtension(ext string) (*GenericParser, bool) {
	config, ok := r.configMap[ext]
	if !ok {
		return nil, false
	}
	parser, ok := r.parsers[config.Language]
	if !ok {
		return nil, false
	}
	return parser, true
}

// Close releases resources held by all parsers in the registry.
func (r *ParserRegistry) Close() {
	for _, parser := range r.parsers {
		parser.Close()
	}
}
