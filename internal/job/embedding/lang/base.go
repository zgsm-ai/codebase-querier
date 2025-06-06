package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const name = "name"

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured node is missing")
	ErrNoDefinition = errors.New("no Definition node found")
	ErrInvalidNode  = errors.New("invalid node")
)

type DefinitionType string

const (
	Function  DefinitionType = "function"
	Class     DefinitionType = "class"
	Struct    DefinitionType = "struct"
	Interface DefinitionType = "interface"
	Enum      DefinitionType = "enum"
	Variable  DefinitionType = "variable"
	TypeAlias DefinitionType = "type_alias"
)

type CodeFileStructure struct {
	Path        string        `json:"path"`        // 文件相对路径
	Language    string        `json:"language"`    // 编程语言
	Definitions []*Definition `json:"definitions"` // 定义列表
}

type Definition struct {
	Type      DefinitionType `json:"type"`
	Name      string         `json:"name"`
	Range     []int32        `json:"range"`     // [startLine, startColumn, endLine, endColumn] (0-based)
	Signature string         `json:"signature"` // 完整签名
}

// DefinitionNodeInfo holds information about a Definition node
type DefinitionNodeInfo struct {
	Node        *sitter.Node
	Kind        string
	Name        string
	ParentClass string
	ParentFunc  string
}

// LanguageProcessor defines the interface for language-specific AST processing
type LanguageProcessor interface {
	ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error)
	GetDefinitionKinds() []string
	FindEnclosingType(node *sitter.Node) *sitter.Node
	FindEnclosingFunction(node *sitter.Node) *sitter.Node
	ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error)
	GetStructureDefinitionKinds() []string
}

// BaseProcessor provides common functionality for all language processors
type BaseProcessor struct {
	definitionKinds []string
	structureKinds  []string
}

// NewBaseProcessor creates a new base processor with object pooling
func NewBaseProcessor(definitionKinds []string, structureKinds []string) *BaseProcessor {
	return &BaseProcessor{
		definitionKinds: definitionKinds,
		structureKinds:  structureKinds,
	}
}

// GetDefinitionKinds returns the list of Definition kinds for this language
func (p *BaseProcessor) GetDefinitionKinds() []string {
	return p.definitionKinds
}

// GetStructureDefinitionKinds returns the list of structure Definition kinds for this language
func (p *BaseProcessor) GetStructureDefinitionKinds() []string {
	return p.structureKinds
}

// CommonMatchProcessor provides shared functionality for processing matches
func (p *BaseProcessor) CommonMatchProcessor(
	match *sitter.QueryMatch,
	root *sitter.Node,
	content []byte,
	definitionKinds []string,
	findEnclosingType func(*sitter.Node) *sitter.Node,
	findEnclosingFunction func(*sitter.Node) *sitter.Node,
) ([]*DefinitionNodeInfo, error) {
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, ErrNoCaptures
	}
	capturedNode := &match.Captures[0].Node
	definitionNode := p.findDefinitionNode(capturedNode, definitionKinds)
	if definitionNode == nil {
		return nil, ErrNoDefinition
	}
	if definitionNode.IsMissing() {
		return nil, ErrMissingNode
	}

	nodeDefInfo := &DefinitionNodeInfo{
		Node: definitionNode,
		Kind: p.getDefinitionKindFromNodeKind(definitionNode.Kind()),
	}

	// Extract name
	if nameNode := definitionNode.ChildByFieldName(name); nameNode != nil && !nameNode.IsMissing() {
		nodeDefInfo.Name = nameNode.Utf8Text(content)
	} else {
		nodeDefInfo.Name = definitionNode.Utf8Text(content)
	}

	// Find parent context
	if enclosingType := findEnclosingType(definitionNode); enclosingType != nil {
		if nameNode := enclosingType.ChildByFieldName(name); nameNode != nil {
			nodeDefInfo.ParentClass = nameNode.Utf8Text(content)
		}
	}

	if findEnclosingFunction != nil {
		if enclosingFunc := findEnclosingFunction(definitionNode); enclosingFunc != nil {
			if nameNode := enclosingFunc.ChildByFieldName(name); nameNode != nil {
				nodeDefInfo.ParentFunc = nameNode.Utf8Text(content)
			}
		}
	}

	return []*DefinitionNodeInfo{nodeDefInfo}, nil
}

// findDefinitionNode traverses up the AST to find a Definition node of the specified kinds
func (p *BaseProcessor) findDefinitionNode(node *sitter.Node, kinds []string) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		for _, kind := range kinds {
			if curr.Kind() == kind {
				return curr
			}
		}
		curr = curr.Parent()
	}
	return nil
}

// CommonStructureProcessor provides shared functionality for processing structure matches
func (p *BaseProcessor) CommonStructureProcessor(
	match *sitter.QueryMatch,
	query *sitter.Query,
	root *sitter.Node,
	content []byte,
) (*Definition, error) {
	if len(match.Captures) == 0 {
		return nil, ErrNoCaptures
	}

	// 获取定义节点（使用第一个捕获的节点）
	defNode := &match.Captures[0].Node
	if defNode.IsMissing() {
		return nil, ErrMissingNode
	}

	// 根据 capture 名判断类型
	var defType DefinitionType
	captureName := ""
	if query != nil && int(match.Captures[0].Index) < len(query.CaptureNames()) {
		captureName = query.CaptureNames()[int(match.Captures[0].Index)]
	}

	switch captureName {
	case "function":
		defType = Function
	case "struct", "class":
		defType = Struct
	case "interface":
		defType = Interface
	case "type_alias":
		defType = TypeAlias
	case "variable":
		defType = Variable
	case "enum":
		defType = Enum
	default:
		// fallback: 根据节点类型判断
		kind := defNode.Kind()
		switch kind {
		case "function_declaration", "method_declaration":
			defType = Function
		case "type_declaration", "class_declaration", "struct_declaration":
			defType = Struct
		case "interface_declaration":
			defType = Interface
		case "const_declaration", "var_declaration":
			defType = Variable
		case "enum_declaration":
			defType = Enum
		default:
			defType = DefinitionType(kind)
		}
	}

	if !isValidDefinitionType(defType) {
		return nil, fmt.Errorf("invalid Definition type: %s", defType)
	}

	// 获取名称
	var name string
	for _, capture := range match.Captures {
		if capture.Index == 0 { // 第一个捕获是名称
			name = capture.Node.Utf8Text(content)
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("no name found for Definition")
	}

	// 获取范围
	startPoint := defNode.StartPosition()
	endPoint := defNode.EndPosition()
	startLine := startPoint.Row
	startColumn := startPoint.Column
	endLine := endPoint.Row
	endColumn := endPoint.Column

	// 构建签名
	signature := buildSignature(defNode, content)

	return &Definition{
		Type:      defType,
		Name:      name,
		Range:     []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Signature: signature,
	}, nil
}

// getDefinitionKindFromNodeKind 将 tree-sitter 节点类型映射到定义类型
func (p *BaseProcessor) getDefinitionKindFromNodeKind(kind string) string {
	// 函数相关节点类型
	switch kind {
	// 函数定义
	case "function_declaration",
		"function_definition",
		"method_declaration",
		"method_definition",
		"arrow_function",
		"function",
		"lambda",
		"lambda_expression":
		return string(Function)

	// 类定义
	case "class_declaration",
		"class_definition",
		"class",
		"class_body",
		"class_constructor":
		return string(Class)

	// 结构体定义
	case "struct_declaration",
		"struct_definition",
		"struct",
		"struct_item",
		"record_declaration":
		return string(Struct)

	// 接口定义
	case "interface_declaration",
		"interface_definition",
		"interface",
		"protocol_declaration",
		"trait_declaration":
		return string(Interface)

	// 枚举定义
	case "enum_declaration",
		"enum_definition",
		"enum",
		"enum_item",
		"enum_class":
		return string(Enum)

	// 变量定义
	case "variable_declaration",
		"variable_definition",
		"var_declaration",
		"let_declaration",
		"const_declaration",
		"field_declaration",
		"property_declaration",
		"local_variable_declaration",
		"global_variable_declaration":
		return string(Variable)

	// 类型别名
	case "type_alias_declaration",
		"type_definition",
		"typedef_declaration",
		"type_declaration",
		"type_alias",
		"using_declaration":
		return string(TypeAlias)

	// 默认返回原始类型
	default:
		// 检查是否在预定义的类型列表中
		for _, defKind := range p.definitionKinds {
			if kind == defKind {
				return kind
			}
		}
		// 如果不在预定义列表中，返回原始类型
		return kind
	}
}

// buildSignature 构建定义的签名
func buildSignature(node *sitter.Node, content []byte) string {
	// 对于函数和方法，包含参数和返回值
	if node.Kind() == "function_declaration" || node.Kind() == "method_declaration" {
		// 获取参数列表
		params := node.ChildByFieldName("parameters")
		if params != nil {
			return params.Utf8Text(content)
		}
	}
	// 对于其他类型，返回完整定义
	return node.Utf8Text(content)
}

// isValidDefinitionType 检查定义类型是否有效
func isValidDefinitionType(t DefinitionType) bool {
	switch t {
	case Function, Class, Struct, Interface, Enum, Variable, TypeAlias, DefinitionType("function_declaration"), DefinitionType("type_declaration"), DefinitionType("method_declaration"), DefinitionType("const_declaration"), DefinitionType("var_declaration"):
		return true
	default:
		return false
	}
}

// ProcessStructureMatch 处理结构查询的匹配结果
func ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	if len(match.Captures) == 0 {
		return nil, ErrNoCaptures
	}

	// 获取定义节点（使用第一个捕获的节点）
	defNode := &match.Captures[0].Node
	if defNode.IsMissing() {
		return nil, ErrMissingNode
	}

	// 新增：根据 capture 名判断类型
	var defType DefinitionType
	captureName := ""
	if query != nil && int(match.Captures[0].Index) < len(query.CaptureNames()) {
		captureName = query.CaptureNames()[int(match.Captures[0].Index)]
	}
	switch captureName {
	case "function":
		defType = Function
	case "struct":
		defType = Struct
	case "interface":
		defType = Interface
	case "type_alias":
		defType = TypeAlias
	case "variable":
		defType = Variable
	default:
		// fallback: 兼容旧逻辑
		kind := defNode.Kind()
		println("ProcessStructureMatch (base.go) fallback defNode.Kind():", kind)
		switch kind {
		case "function_declaration":
			defType = Function
		case "type_declaration":
			defType = Struct
		case "method_declaration":
			defType = Function
		case "const_declaration":
			defType = Variable
		case "var_declaration":
			defType = Variable
		default:
			defType = DefinitionType(kind)
		}
	}

	if !isValidDefinitionType(defType) {
		return nil, fmt.Errorf("invalid Definition type: %s", defType)
	}

	// 获取名称
	var name string
	for _, capture := range match.Captures {
		if capture.Index == 0 { // 第一个捕获是名称
			name = capture.Node.Utf8Text(content)
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("no name found for Definition")
	}

	// 获取范围
	startPoint := defNode.StartPosition()
	endPoint := defNode.EndPosition()
	startLine := startPoint.Row
	startColumn := startPoint.Column
	endLine := endPoint.Row
	endColumn := endPoint.Column

	// 构建签名
	signature := buildSignature(defNode, content)

	return &Definition{
		Type:      defType,
		Name:      name,
		Range:     []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Signature: signature,
	}, nil
}
