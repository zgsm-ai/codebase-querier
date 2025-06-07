package parser

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
)

// Language represents a programming language.
type Language string

const (
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

// LanguageConfig holds the configuration for a language
type LanguageConfig struct {
	Language       Language
	SitterLanguage func() *sitter.Language
	SupportedExts  []string
}

// languageConfigs 定义了所有支持的语言配置
var languageConfigs = map[Language]*LanguageConfig{
	Go: {
		Language: Go,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittergo.Language())
		},
		SupportedExts: []string{".go"},
	},
	Java: {
		Language: Java,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjava.Language())
		},
		SupportedExts: []string{".java"},
	},
	Python: {
		Language: Python,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterpython.Language())
		},
		SupportedExts: []string{".py"},
	},
	JavaScript: {
		Language: JavaScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjavascript.Language())
		},
		SupportedExts: []string{".js", ".jsx"},
	},
	TypeScript: {
		Language: TypeScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittertypescript.LanguageTypescript())
		},
		SupportedExts: []string{".ts"},
	},
	TSX: {
		Language: TSX,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittertypescript.LanguageTSX())
		},
		SupportedExts: []string{".tsx"},
	},
	Rust: {
		Language: Rust,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterrust.Language())
		},
		SupportedExts: []string{".rs"},
	},
	C: {
		Language: C,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterc.Language())
		},
		SupportedExts: []string{".c", ".h"},
	},
	CPP: {
		Language: CPP,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittercpp.Language())
		},
		SupportedExts: []string{".cpp", ".cc", ".cxx", ".hpp"},
	},
	CSharp: {
		Language: CSharp,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittercsharp.Language())
		},
		SupportedExts: []string{".cs"},
	},
	Ruby: {
		Language: Ruby,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterruby.Language())
		},
		SupportedExts: []string{".rb"},
	},
	PHP: {
		Language: PHP,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterphp.LanguagePHP())
		},
		SupportedExts: []string{".php", ".phtml"},
	},
	Kotlin: {
		Language: Kotlin,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterkotlin.Language())
		},
		SupportedExts: []string{".kt", ".kts"},
	},
	Scala: {
		Language: Scala,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterscala.Language())
		},
		SupportedExts: []string{".scala"},
	},
}

// GetLanguageConfigs 获取所有语言配置
func GetLanguageConfigs() map[Language]*LanguageConfig {
	return languageConfigs
}

// GetLanguageConfigByExt 根据文件扩展名获取语言配置
func GetLanguageConfigByExt(ext string) *LanguageConfig {
	for _, config := range languageConfigs {
		for _, supportedExt := range config.SupportedExts {
			if supportedExt == ext {
				return config
			}
		}
	}
	return nil
}
