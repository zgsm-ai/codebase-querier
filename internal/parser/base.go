package parser

import (
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type Parser struct{}

type ParseOptions struct {
	IncludeContent bool
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(sourceFile *types.SourceFile, opts ParseOptions) (*ParsedSource, error) {
	// Extract file extension
	langConf, err := GetLangConfigByFilePath(sourceFile.Path)
	if err != nil {
		return nil, err
	}
	queryScm, ok := BaseQueries[langConf.Language]
	if !ok {
		return nil, ErrQueryNotFound
	}

	sitterParser := sitter.NewParser()
	sitterLanguage := langConf.SitterLanguage()
	if err := sitterParser.SetLanguage(sitterLanguage); err != nil {
		return nil, err
	}
	content := sourceFile.Content
	tree := sitterParser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: %s", sourceFile.Path)
	}

	defer tree.Close()

	query, err := sitter.NewQuery(sitterLanguage, queryScm)
	if err != nil && IsRealQueryErr(err) {
		return nil, err
	}
	defer query.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	elements := make([]*CodeElement, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		element, err := p.ProcessNode(m, query, content, opts)
		if err != nil {
			continue // 跳过错误的匹配
		}
		elements = append(elements, element)
	}

	// 返回结构信息，包含处理后的定义
	return &ParsedSource{
		Path:     sourceFile.Path,
		Language: langConf.Language,
		Elements: elements,
	}, nil
}

func (p *Parser) ProcessNode(match *sitter.QueryMatch, query *sitter.Query, source []byte, opts ParseOptions) (*CodeElement, error) {

	return nil, nil
}
