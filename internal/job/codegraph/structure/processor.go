package structure

import (
	"errors"
	"fmt"

	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured node is missing")
	ErrNoDefinition = errors.New("no Definition node found")
)

const name = "name"

// ProcessDefinitionNode provides shared functionality for processing structure matches
func (p *Parser) ProcessDefinitionNode(
	match *sitter.QueryMatch,
	query *sitter.Query,
	root *sitter.Node,
	code []byte,
	opts ParseOptions,
) (*codegraphpb.Definition, error) {
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
	nodeName := nameNode.Utf8Text(code)
	if nodeName == "" {
		return nil, fmt.Errorf("no name found for Definition")
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
		content = code[defNode.StartByte():defNode.EndByte()]
	}

	return &codegraphpb.Definition{
		Type:    defType,
		Name:    nodeName,
		Range:   []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Content: content,
	}, nil
}
