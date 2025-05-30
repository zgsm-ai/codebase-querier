package types

// NodeType 节点类型枚举
type NodeType string

const (
	NodeTypeDefinition     NodeType = "definition"     // 定义节点（根节点）
	NodeTypeReference      NodeType = "reference"      // 引用关系
	NodeTypeInheritance    NodeType = "inheritance"    // 继承关系（子类 -> 父类）
	NodeTypeImplementation NodeType = "implementation" // 实现关系（类 -> 接口）
)

type GraphNode struct {
	FilePath   string       `json:"filePath"`
	SymbolName string       `json:"symbolName"`
	Position   Position     `json:"position"`
	Content    string       `json:"content"`
	NodeType   NodeType     `json:"nodeType"`
	Children   []*GraphNode `json:"children"`
	Parent     *GraphNode   `json:"parent"`
}

type Position struct {
	StartLine   int `json:"startLine"`   // 开始行（从1开始）
	StartColumn int `json:"startColumn"` // 开始列（从1开始）
	EndLine     int `json:"endLine"`     // 结束行（从1开始）
	EndColumn   int `json:"endColumn"`   // 结束列（从1开始）
}
