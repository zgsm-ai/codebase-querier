package definition

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured node is missing")
	ErrNoDefinition = errors.New("no QueryDefinition node found")
)

const name = "name"

// DefParser  用于解析代码结构
type DefParser struct {
}

type ParseOptions struct {
	IncludeContent bool
}

// NeDefinitionParser creates a new generic parser with the given config.
func NeDefinitionParser() (*DefParser, error) {
	return &DefParser{}, nil
}

// Parse 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func (s *DefParser) Parse(ctx context.Context, codeFile *types.SourceFile, opts ParseOptions) (*codegraphpb.CodeDefinition, error) {
	// Extract file extension
	langConf, err := parser.GetLangConfigByFilePath(codeFile.Path)
	if err != nil {
		return nil, err
	}
	queryScm, ok := parser.DefinitionQueries[langConf.Language]
	if !ok {
		return nil, parser.ErrQueryNotFound
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
		def, err := s.ProcessDefinitionNode(m, query, code, opts)
		if err != nil {
			continue // 跳过错误的匹配
		}
		definitions = append(definitions, def)
	}

	// 返回结构信息，包含处理后的定义
	return &codegraphpb.CodeDefinition{
		Definitions: definitions,
		Path:        codeFile.Path,
		Language:    string(langConf.Language),
	}, nil
}

// ProcessDefinitionNode provides shared functionality for processing structure matches
func (p *DefParser) ProcessDefinitionNode(match *sitter.QueryMatch, query *sitter.Query,
	source []byte, opts ParseOptions) (*codegraphpb.Definition, error) {
	if len(match.Captures) == 0 {
		return nil, ErrNoCaptures
	}

	// 获取定义节点、名称节点和其他必要节点
	var defNode *sitter.Node
	var nameNode *sitter.Node
	var defType string

	for _, capture := range match.Captures {
		captureName := query.CaptureNames()[capture.Index]
		if captureName == name {
			nameNode = &capture.Node
		} else if defNode == nil { // 使用第一个非 name 的捕获作为定义类型
			defNode = &capture.Node
			defType = captureName
		}
	}

	if defNode == nil || nameNode == nil {
		return nil, ErrMissingNode
	}
	// TODO range 有问题，golang  import (xxx xxx xxx) 捕获的是整体。
	// 获取名称
	nodeName := nameNode.Utf8Text(source)
	if nodeName == "" {
		return nil, fmt.Errorf("no name found for QueryDefinition")
	}

	// 获取范围
	startPoint := defNode.StartPosition()
	endPoint := defNode.EndPosition()
	startLine := startPoint.Row
	startColumn := startPoint.Column
	endLine := endPoint.Row
	endColumn := endPoint.Column

	var content []byte
	if opts.IncludeContent {
		content = source[defNode.StartByte():defNode.EndByte()]
	}

	return &codegraphpb.Definition{
		Type:    defType,
		Name:    nodeName,
		Range:   []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Content: content,
	}, nil
}
