package structure

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

var ErrQueryNotFound = errors.New("query not found")

// Parser  用于解析代码结构
type Parser struct {
}

type ParseOptions struct {
	IncludeContent bool
}

// NewStructureParser creates a new generic parser with the given config.
func NewStructureParser() (*Parser, error) {
	return &Parser{}, nil
}

// Parse 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func (s *Parser) Parse(ctx context.Context, codeFile *types.CodeFile, opts ParseOptions) (*codegraphpb.CodeStructure, error) {
	// Extract file extension
	langConf, err := parser.GetLangConfigByFilePath(codeFile.Path)
	if err != nil {
		return nil, err
	}
	queryScm, ok := languageQueryConfig[langConf.Language]
	if !ok {
		return nil, ErrQueryNotFound
	}

	sitterParser := sitter.NewParser()
	sitterLanguage := langConf.SitterLanguage()
	if err := sitterParser.SetLanguage(sitterLanguage); err != nil {
		return nil, err
	}
	code := codeFile.Content
	tree := sitterParser.Parse(code, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	query, err := sitter.NewQuery(sitterLanguage, queryScm)
	if err != nil && parser.IsRealQueryErr(err) {
		return nil, err
	}
	defer query.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), code)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	definitions := make([]*codegraphpb.Definition, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		def, err := s.ProcessDefinitionNode(m, query, tree.RootNode(), code, opts)
		if err != nil {
			continue // 跳过错误的匹配
		}
		definitions = append(definitions, def)
	}

	// 返回结构信息，包含处理后的定义
	return &codegraphpb.CodeStructure{
		Definitions: definitions,
		Path:        codeFile.Path,
		Language:    string(langConf.Language),
	}, nil
}
