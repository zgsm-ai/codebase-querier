package structure

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/job/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

var ErrExtNotFound = errors.New("file extension not found")
var ErrLangConfNotFound = errors.New("langConf not found")
var ErrQueryNotFound = errors.New("query not found")

// Parser  用于解析代码结构
type Parser struct {
}

// NewStructureParser creates a new generic parser with the given config.
func NewStructureParser() (*Parser, error) {
	return &Parser{}, nil
}

// Parse 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func (s Parser) Parse(codeFile *types.CodeFile) (*codegraphpb.CodeStructure, error) {
	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		return nil, ErrExtNotFound
	}
	langConf := parser.GetLanguageConfigByExt(ext)
	if langConf == nil {
		return nil, ErrLangConfNotFound
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
	content := codeFile.Content
	tree := sitterParser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	query, err := sitter.NewQuery(sitterLanguage, queryScm)
	if err != nil && parser.IsRealErr(err) {
		return nil, err
	}
	defer query.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	processor := newDefinitionProcessor()
	definitions := make([]*codegraphpb.Definition, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		def, err := processor.ProcessDefinitionNode(m, query, tree.RootNode(), content)
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
