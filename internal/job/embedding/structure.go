package embedding

import (
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/lang"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type StructureParser struct {
	languages []*lang.LanguageConfig // Language-specific configuration
}

// NewStructureParser creates a new generic parser with the given config.
func NewStructureParser() (*StructureParser, error) {
	languages, err := lang.GetLanguageConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to get languages config: %w", err)
	}

	return &StructureParser{
		languages: languages,
	}, nil
}

// Parse 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func (s StructureParser) Parse(codeFile *types.CodeFile) (*lang.CodeFileStructure, error) {
	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		return nil, fmt.Errorf("file %s has no extension, cannot determine language", codeFile.Path)
	}
	language := lang.GetLanguageConfigByExt(s.languages, ext)
	if language == nil {
		return nil, fmt.Errorf("cannot find language config by ext %s", ext)
	}

	parser := sitter.NewParser()
	if err := parser.SetLanguage(language.SitterLanguage); err != nil {
		return nil, err
	}
	content := codeFile.Content
	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	query, err := sitter.NewQuery(language.SitterLanguage, language.StructureQuery)
	if err != nil {
		// 打印 query 解析的错误信息，便于排查 tree-sitter 解析问题
		println("Parse (base.go) query parse error:", err.Error())
		return nil, err
	}
	defer query.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	processor := language.Processor
	definitions := make([]*lang.Definition, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		def, err := processor.ProcessStructureMatch(m, query, tree.RootNode(), content)
		if err != nil {
			continue // 跳过错误的匹配
		}
		definitions = append(definitions, def)
	}

	// 返回结构信息，包含处理后的定义
	return &lang.CodeFileStructure{Definitions: definitions}, nil
}
