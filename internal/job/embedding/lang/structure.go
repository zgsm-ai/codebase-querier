package lang

import "fmt"
import sitter "github.com/tree-sitter/go-tree-sitter"

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
	Definitions []*definition `json:"definitions"` // 定义列表
}

type definition struct {
	Type      DefinitionType `json:"type"`
	Name      string         `json:"name"`
	Range     []int32        `json:"range"`     // [startLine, startColumn, endLine, endColumn] (0-based)
	Signature string         `json:"signature"` // 完整签名
}

// NewFileStructureProcessor 创建一个新的文件结构处理器
func NewFileStructureProcessor() *FileStructureProcessor {
	return &FileStructureProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function",
			"class",
			"struct",
			"interface",
			"type_alias",
			"variable",
		}),
	}
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
func (p *FileStructureProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*definition, error) {
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
		return nil, fmt.Errorf("invalid definition type: %s", defType)
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
		return nil, fmt.Errorf("no name found for definition")
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

	return &definition{
		Type:      defType,
		Name:      name,
		Range:     []int32{int32(startLine), int32(startColumn), int32(endLine), int32(endColumn)},
		Signature: signature,
	}, nil
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

// ParseFileStructure 解析文件结构，返回结构信息（例如函数、结构体、接口、变量、常量等）
func ParseFileStructure(content []byte, config *LanguageConfig) (*CodeFileStructure, error) {
	parser := sitter.NewParser()
	if err := parser.SetLanguage(config.SitterLanguage); err != nil {
		return nil, err
	}
	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	query, err := sitter.NewQuery(config.SitterLanguage, config.StructureQuery)
	if err != nil {
		// 打印 query 解析的错误信息，便于排查 tree-sitter 解析问题
		println("ParseFileStructure (base.go) query parse error:", err.Error())
		return nil, err
	}
	defer query.Close()

	// 执行 query，并处理匹配结果
	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	// 消费 matches，并调用 ProcessStructureMatch 处理匹配结果
	processor := NewFileStructureProcessor()
	definitions := make([]*definition, 0)
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		// 打印 query 执行后的匹配结果，便于排查"解析不出"问题
		def, err := processor.ProcessStructureMatch(m, query, tree.RootNode(), content)
		if err != nil {
			continue // 跳过错误的匹配
		}
		definitions = append(definitions, def)
	}

	// 返回结构信息，包含处理后的定义
	return &CodeFileStructure{Definitions: definitions}, nil
}
