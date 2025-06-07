package parser

import (
	"embed"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:embed queries/**/*.scm

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
const chunkSubDir = "chunk"
const structureSubDir = "structure"
const queryExt = ".scm"

// LanguageConfig holds the configuration for a language
type LanguageConfig struct {
	Language           Language
	SitterLanguage     *sitter.Language
	structureQueryPath string
	StructureQuery     string
	SupportedExts      []string
	Processor          LanguageProcessor
}

var configs []*LanguageConfig

// GetLanguageConfigs returns all supported language configurations
func GetLanguageConfigs() ([]*LanguageConfig, error) {
	if configs != nil {
		return configs, nil
	}
	var err error
	configs = make([]*LanguageConfig, 0)

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
	//TODO 校验scm文件的语法
	for _, config := range configs {
		structureQueryContent, err := scmFS.ReadFile(config.structureQueryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read structure query file %s for %s: %w", config.structureQueryPath, config.Language, err)
		}
		config.StructureQuery = string(structureQueryContent)
	}
	return configs, err
}

// Helper function to create a query path
func makeChunkQueryPath(lang Language) string {
	return queryBaseDir + "/" + chunkSubDir + "/" + string(lang) + queryExt
}

func makeStructureQueryPath(lang Language) string {
	return queryBaseDir + "/" + structureSubDir + "/" + string(lang) + queryExt
}

// GetLanguageConfigByExt 根据文件扩展名获取语言配置
func GetLanguageConfigByExt(configs []*LanguageConfig, ext string) *LanguageConfig {
	for _, config := range configs {
		for _, supportedExt := range config.SupportedExts {
			if supportedExt == ext {
				return config
			}
		}
	}
	return nil
}
