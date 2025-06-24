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
	NodeTypeUnknown        NodeType = "unknown"        // 未知
	NodeTypeReference      NodeType = "reference"      // 引用关系
	NodeTypeImplementation NodeType = "implementation" // 实现关系（类 -> 接口）

)

type GraphNode struct {
	FilePath   string       `json:"FilePaths"`
	SymbolName string       `json:"symbolName"`
	Identifier string       `json:"-"`
	Position   Position     `json:"position"`
	Content    string       `json:"content"`
	NodeType   string       `json:"nodeType"`
	Children   []*GraphNode `json:"children"`
	Caller     *GraphNode   `json:"caller,omitempty"`
}

type Position struct {
	StartLine   int `json:"startLine"`   // 开始行（从1开始）
	StartColumn int `json:"startColumn"` // 开始列（从1开始）
	EndLine     int `json:"endLine"`     // 结束行（从1开始）
	EndColumn   int `json:"endColumn"`   // 结束列（从1开始）
}

// ToPosition 辅助函数：将 ranges 转换为 types.Position
func ToPosition(ranges []int32) Position {
	if len(ranges) != 3 && len(ranges) != 4 {
		return Position{}
	}
	if len(ranges) == 3 {
		return Position{
			StartLine:   int(ranges[0]) + 1,
			StartColumn: int(ranges[1]) + 1,
			EndLine:     int(ranges[0]) + 1,
			EndColumn:   int(ranges[2]) + 1,
		}
	} else {
		return Position{
			StartLine:   int(ranges[0]) + 1,
			StartColumn: int(ranges[1]) + 1,
			EndLine:     int(ranges[2]) + 1,
			EndColumn:   int(ranges[3]) + 1,
		}
	}

}
