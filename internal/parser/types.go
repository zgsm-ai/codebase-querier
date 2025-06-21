package parser

type ElementType string

const (
	// 模块、包级别元素
	ElementTypeNamespace ElementType = "namespace" // 包/命名空间/
	ElementTypeImport                = "import"    // 导入语句

	// 类型定义
	ElementTypeClass     = "class"      // 类
	ElementTypeInterface = "interface"  // 接口
	ElementTypeStruct    = "struct"     // 结构体
	ElementTypeEnum      = "enum"       // 枚举
	ElementTypeUnion     = "union"      // 联合类型
	ElementTypeTrait     = "trait"      // 特性/特征（如Rust trait）
	ElementTypeTypeAlias = "type_alias" // 类型别名

	// 函数、方法
	ElementTypeFunction    = "function"    // 函数
	ElementTypeMethod      = "method"      // 类方法
	ElementTypeConstructor = "constructor" // 构造函数
	ElementTypeDestructor  = "destructor"  // 析构函数

	// 变量、常量
	ElementTypeVariable  = "variable"  // 变量
	ElementTypeConstant  = "constant"  // 常量
	ElementTypeMacro     = "macro"     // 宏
	ElementTypeField     = "field"     // 类字段/属性
	ElementTypeParameter = "parameter" // 函数参数

	// 表达式、语句
	ElementTypeExpression = "expression" // 表达式
	ElementTypeStatement  = "statement"  // 语句
	ElementTypeAssignment = "assignment" // 赋值语句

	// 注释、文档
	ElementTypeComment    = "comment"     // 注释
	ElementTypeDocComment = "doc_comment" // 文档注释

	// 其他
	ElementTypeFile       = "file"       // 文件
	ElementTypeAnnotation = "annotation" // 注解/属性
	ElementTypeGeneric    = "generic"    // 泛型参数/类型
)

type ParsedSource struct {
	Path      string
	Namespace string
	Imports   []Import
	Language  Language
	Elements  []*CodeElement
}

type Import struct {
	Path  string
	Alias string
}

type CodeElement struct {
	Name      string
	Type      ElementType
	Signature string
	Range     []int32
	Content   []byte
	Parent    *CodeElement
	Children  []*CodeElement
}
