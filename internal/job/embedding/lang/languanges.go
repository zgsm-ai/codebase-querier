package lang

import (
	"embed"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:embed queries/*.scm

var scmFS embed.FS

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

// LanguageConfig holds the configuration for a language
type LanguageConfig struct {
	Language       Language
	SitterLanguage *sitter.Language
	queryPath      string
	Query          string
	SupportedExts  []string
	Processor      LanguageProcessor
}

// GetLanguageConfigs returns all supported language configurations
func GetLanguageConfigs() ([]*LanguageConfig, error) {
	configs := make([]*LanguageConfig, 0)

	// Add configurations from each language file
	configs = append(configs, GetGoConfig())
	configs = append(configs, GetPythonConfig())
	configs = append(configs, GetJavaConfig())
	configs = append(configs, GetJavaScriptConfig())
	configs = append(configs, GetTypeScriptConfig())
	configs = append(configs, GetTSXConfig())
	configs = append(configs, GetRustConfig())
	configs = append(configs, GetCConfig())
	configs = append(configs, GetCppConfig())
	configs = append(configs, GetCSharpConfig())
	configs = append(configs, GetRubyConfig())
	configs = append(configs, GetPhpConfig())
	configs = append(configs, GetKotlinConfig())
	configs = append(configs, GetScalaConfig())

	for _, config := range configs {
		queryFilePath := config.queryPath
		queryContent, err := scmFS.ReadFile(queryFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read query file %s for %s: %w", queryFilePath, config.Language, err)
		}
		config.Query = string(queryContent)
	}

	return configs, nil
}

// Helper function to create a query path
func makeQueryPath(lang Language) string {
	return queryBaseDir + "/" + string(lang) + queryExt
}
