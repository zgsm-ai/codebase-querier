package parser

import (
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/task/parser/lang"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
)

// Registry manages the registration and retrieval of types.Language parsers.
type Registry struct {
	parsers             map[types.Language]lang.CodeParser
	extensionToLanguage map[string]types.Language
}

// NewParserRegistry initializes and registers all known types.Language parsers.
func NewParserRegistry(opts ...RegistryFunc) (*Registry, error) {
	registry := &Registry{
		parsers:             make(map[types.Language]lang.CodeParser),
		extensionToLanguage: make(map[string]types.Language),
	}

	// opts are no longer used to configure the registry fields directly.
	// They could potentially be used for other registry-level configuration in the future.
	for _, opt := range opts {
		opt(registry)
	}

	for _, p := range supportedLanguages {
		// New functions no longer take maxTokensPerChunk and overlapTokens
		parser, err := p.new()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize %s parser: %w", p.lang, err)
		}
		registry.Register(p.lang, parser)
		// Populate the extensionToLanguage map
		for _, ext := range p.supportedExts {
			registry.extensionToLanguage[ext] = p.lang
		}
	}

	return registry, nil
}

// Register registers a CodeParser for a specific types.Language with the registry.
func (r *Registry) Register(lang types.Language, parser lang.CodeParser) {
	r.parsers[lang] = parser
}

// Close releases resources held by the CodeSplitter and its parsers.
func (s *Registry) Close() {
	for _, p := range s.parsers {
		p.Close()
	}
}

// Get retrieves the CodeParser for the given types.Language from the registry.
func (r *Registry) Get(lang types.Language) (lang.CodeParser, bool) {
	parser, ok := r.parsers[lang]
	return parser, ok
}

// InferLanguage infers the language of a code file based on its file extension.
func (r *Registry) InferLanguage(codeFile *types.CodeFile) (types.Language, bool) {
	ext := filepath.Ext(codeFile.Path)
	lang, ok := r.extensionToLanguage[ext]
	return lang, ok
}

func (r *Registry) TryGet(file *types.CodeFile) (lang.CodeParser, error) {
	if file.Language != types.Unknown {
		codeParser, ok := r.parsers[file.Language]
		if !ok {
			return nil, fmt.Errorf("unknown language for file %s", file.Path)
		}
		return codeParser, nil
	}

	// Use the registry's InferLanguage method
	inferredLang, ok := r.InferLanguage(file)
	if ok {
		codeParser, ok := r.parsers[inferredLang]
		if !ok {
			// This case should ideally not happen if registration is correct
			return nil, fmt.Errorf("inferred language %s has no registered parser for file %s", inferredLang, file.Path)
		}
		return codeParser, nil
	}

	return nil, fmt.Errorf("no parser found for file %s", file.Path)
}

type RegistryFunc func(registry *Registry)
