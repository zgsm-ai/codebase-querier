package parser

import (
	"context"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"strings"
)

type ParsedSource struct {
	Path     string
	Package  *Package
	Imports  []*Import
	Language Language
	Elements []CodeElement
}

// CodeElement 定义所有代码元素的接口
type CodeElement interface {
	GetName() string
	GetType() ElementType
	GetRange() []int32
	GetParent() CodeElement
	SetParent(parent CodeElement)
	AddChild(child CodeElement)
	GetChildren() []CodeElement
	Update(ctx context.Context, captureName string,
		capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error
	SetContent(content []byte)
}

// BaseElement 提供接口的基础实现，其他类型嵌入该结构体
type BaseElement struct {
	Name     string
	Type     ElementType
	Content  []byte
	Range    []int32
	Parent   CodeElement
	Children []CodeElement
}

// Package 表示代码包
type Package struct {
	*BaseElement
}

// Import 表示导入语句
type Import struct {
	*BaseElement
	Source   string
	Alias    string
	FullName string
}

// Function 表示函数
type Function struct {
	*BaseElement
	Owner      string
	Parameters []string
	ReturnType string
}

// Method 表示方法
type Method struct {
	*BaseElement
	Owner      string
	Parameters []string
	ReturnType string
}

type Call struct {
	*BaseElement
	Owner     string
	Arguments []string
}

// Class 表示类
type Class struct {
	*BaseElement
	Fields  []*CodeElement
	Methods []*CodeElement
}

type Variable struct {
	*BaseElement
}

func (e *BaseElement) GetName() string              { return e.Name }
func (e *BaseElement) GetType() ElementType         { return e.Type }
func (e *BaseElement) GetRange() []int32            { return e.Range }
func (e *BaseElement) GetParent() CodeElement       { return e.Parent }
func (e *BaseElement) SetParent(parent CodeElement) { e.Parent = parent }
func (e *BaseElement) AddChild(child CodeElement) {
	e.Children = append(e.Children, child)
	child.SetParent(e)
}
func (e *BaseElement) GetChildren() []CodeElement { return e.Children }

func (e *BaseElement) SetContent(content []byte) {
	e.Content = content
}
func (e *BaseElement) Update(ctx context.Context, captureName string, capture *treesitter.QueryCapture,
	source []byte, opts ParseOptions) error {

	index := int(capture.Index)
	node := &capture.Node
	if index == 0 {
		// rootNode
		rootCaptureNode := node
		e.Range = []int32{
			int32(rootCaptureNode.StartPosition().Row),
			int32(rootCaptureNode.StartPosition().Column),
			int32(rootCaptureNode.StartPosition().Row),
			int32(rootCaptureNode.StartPosition().Column),
		}
		if opts.IncludeContent {
			content := source[node.StartByte():node.EndByte()]
			e.SetContent(content)
		}

	}

	if e.Name == types.EmptyString && isElementNameCapture(e.Type, captureName) {
		// 取root节点的name，比如definition.function.name
		// 获取名称 ,go import 带双引号
		name := strings.ReplaceAll(node.Utf8Text([]byte{}), types.SingleDoubleQuote, types.EmptyString)
		if name == types.EmptyString {
			tracer.WithTrace(ctx).Errorf("tree_sitter base_processor name_node %s %v name not found", captureName, e.Range)
		}
		e.Name = node.Utf8Text([]byte{})
	}

	return nil
}

func (f *Function) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error {

	if err := f.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}
	node := &capture.Node

	if len(f.Parameters) == 0 && isParametersCapture(captureName) {
		f.Parameters = strings.Split(node.Utf8Text(source), types.Comma)
	}

	if isOwnerCapture(captureName) && f.Owner == types.EmptyString {
		f.Owner = node.Utf8Text(source)
	}

	return nil
}

func (m *Method) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error {

	if err := m.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	if len(m.Parameters) == 0 && isParametersCapture(captureName) {
		m.Parameters = strings.Split(node.Utf8Text(source), types.Comma)
	}

	if isOwnerCapture(captureName) && m.Owner == types.EmptyString {
		m.Owner = node.Utf8Text(source)
	}

	return nil
}

func (c *Call) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error {

	if err := c.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}
	node := &capture.Node

	if len(c.Arguments) == 0 && isArgumentsCapture(captureName) {
		c.Arguments = strings.Split(node.Utf8Text(source), types.Comma)
	}

	if c.Owner == types.EmptyString && isOwnerCapture(captureName) {
		c.Owner = node.Utf8Text(source)
	}

	return nil
}

func (v *Variable) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error {

	if err := v.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	// TODO 局部变量不是很容易区分，存在多层嵌套。找到它的名字不太容易。存在一行返回多个局部变量的情况,当前只取了第一个
	if v.Name == types.EmptyString {
		nameNode := findIdentifier(node)
		if nameNode != nil {
			v.Name = nameNode.Utf8Text(source)
		}
	}
	return nil
}

func (v *Import) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts ParseOptions) error {

	if err := v.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	if v.Source == types.EmptyString && isSourceCapture(captureName) {
		v.Source = node.Utf8Text(source)
	}

	if v.Alias == types.EmptyString && isAliasCapture(captureName) {
		v.Alias = node.Utf8Text(source)
	}

	if v.FullName == types.EmptyString && isFullNameCapture(captureName) {
		v.FullName = node.Utf8Text(source)
	}

	return nil
}
