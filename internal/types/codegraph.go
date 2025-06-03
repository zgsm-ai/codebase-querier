package types

// NodeType 节点类型枚举
type NodeType string

type SymbolRole string

type SymbolType string

const (
	SymbolRoleDefinition     SymbolRole = "definition"      // 在代码文件中属于定义
	SymbolRoleReference      SymbolRole = "reference"       // 在代码文件中属于引用
	SymbolRoleImport         SymbolRole = "import"          // 在代码文件中属于导入
	SymbolRoleImplementation SymbolRole = "implementation"  // 属于实现关系
	SymbolRoleTypeDefinition SymbolRole = "type_definition" // 属于类型定义关系
)

const (
	SymbolTypeFunction SymbolType = "function" // 函数
	SymbolTypeClass    SymbolType = "class"    // 类
	SymbolTypePackage  SymbolType = "package"  // 包
	SymbolTypeVariable SymbolType = "variable" // 变量
)

const ( //
	NodeTypeDefinition     NodeType = "definition"     // 定义节点（根节点）
	NodeTypeReference      NodeType = "reference"      // 引用关系
	NodeTypeInheritance    NodeType = "inheritance"    // 继承关系（子类 -> 父类）
	NodeTypeImplementation NodeType = "implementation" // 实现关系（类 -> 接口）
	NodeTypeImport         NodeType = "import"         // 导入包
)

type GraphNode struct {
	FilePath   string       `json:"filePath"`
	SymbolName string       `json:"symbolName"`
	Position   Position     `json:"position"`
	Content    string       `json:"content"`
	NodeType   SymbolRole   `json:"nodeType"`
	Children   []*GraphNode `json:"children"`
	Parent     *GraphNode   `json:"parent"`
}

type Position struct {
	StartLine   int `json:"startLine"`   // 开始行（从1开始）
	StartColumn int `json:"startColumn"` // 开始列（从1开始）
	EndLine     int `json:"endLine"`     // 结束行（从1开始）
	EndColumn   int `json:"endColumn"`   // 结束列（从1开始）
}
