package parser

import (
	"context"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"strings"
)

type Parser struct{}

type ParseOptions struct {
	IncludeContent bool
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(ctx context.Context, sourceFile *types.SourceFile, opts ParseOptions) (*ParsedSource, error) {
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
		element, err := p.processNode(ctx, m, query, content, opts)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("tree_sitter base processor processNode error: %v", err)
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

func (p *Parser) processNode(ctx context.Context,
	match *sitter.QueryMatch,
	query *sitter.Query,
	source []byte,
	opts ParseOptions) (*CodeElement, error) {
	if len(match.Captures) == 0 {
		return nil, ErrNoCaptures
	}

	var captureRoot *sitter.Node
	var captureRootNameNode *sitter.Node
	var parameterNode *sitter.Node
	var ownerNode *sitter.Node // a.method  b.function
	var rootCaptureName string
	for i, capture := range match.Captures {
		captureName := query.CaptureNames()[capture.Index]
		if i == 0 {
			captureRoot = &capture.Node
			rootCaptureName = captureName
			continue
		}

		if isNodeNameCapture(rootCaptureName, captureName) { // *.name 可能存在多个.name
			captureRootNameNode = &capture.Node
		} else if isParameterCapture(captureName) {
			parameterNode = &capture.Node
		} else if isOwnerCapture(captureName) {
			ownerNode = &capture.Node
		} else {
			// TODO full_name（import）、 find identifier recur (variable)、parameters/arguments
			tracer.WithTrace(ctx).Debugf("unknown capture: %s", captureName)
		}
	}
	// TODO 局部变量不是很容易区分，存在多层嵌套。找到它的名字不太容器。存在一行返回多个局部变量的情况,当前只取了第一个
	if captureRootNameNode == nil && isElementType(rootCaptureName, ElementTypeVariable) {
		captureRootNameNode = findIdentifier(captureRoot)
	}

	if captureRoot == nil || captureRootNameNode == nil {
		return nil, ErrMissingNode
	}

	// 获取名称 ,go import 带双引号
	name := strings.ReplaceAll(captureRootNameNode.Utf8Text(source), types.SingleDoubleQuote, types.EmptyString)
	if name == types.EmptyString {
		return nil, fmt.Errorf("tree_sitter base_processor no name found")
	}
	// 获取参数
	var parameters []string
	if parameterNode != nil {
		parameters = append(parameters, strings.Split(parameterNode.Utf8Text(source), types.Comma)...)
	}
	// 获取owner 方法所属的类，函数所属的包/模块等。
	var ownerName string
	if ownerNode != nil {
		ownerName = ownerNode.Utf8Text(source)
	}

	// 获取范围
	startPoint := captureRoot.StartPosition()
	endPoint := captureRoot.EndPosition()
	startLine := startPoint.Row
	startColumn := startPoint.Column
	endLine := endPoint.Row
	endColumn := endPoint.Column

	var content []byte
	if opts.IncludeContent {
		content = source[captureRoot.StartByte():captureRoot.EndByte()]
	}

	return &CodeElement{
		Type:       toElementType(rootCaptureName),
		Name:       name,
		Owner:      ownerName,
		Parameters: parameters,
		Range:      []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Content:    content,
	}, nil
}

// findIdentifier 递归遍历语法树节点，查找类型为"identifier"的节点
func findIdentifier(node *sitter.Node) *sitter.Node {
	// 检查当前节点是否为identifier类型
	if node.Kind() == identifier {
		return node
	}

	// 遍历所有子节点
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		// 递归查找子节点中的identifier
		result := findIdentifier(child)
		if result != nil {
			return result // 找到则立即返回
		}
	}

	// 未找到identifier节点
	return nil
}
