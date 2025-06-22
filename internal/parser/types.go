package parser

type ElementType int

const (
	ElementTypeNamespace ElementType = iota // 包/命名空间/
	ElementTypePackage                      // 包/命名空间/
	ElementTypeUndefined                    // 包/命名空间/
	ElementTypeImport                       // 导入语句

	ElementTypeClass     // 类
	ElementTypeInterface // 接口
	ElementTypeStruct    // 结构体
	ElementTypeEnum      // 枚举
	ElementTypeUnion     // 联合类型
	ElementTypeTrait     // 特性/特征（如Rust trait）
	ElementTypeTypeAlias // 类型别名

	ElementTypeFunction            // 函数
	ElementTypeFunctionCall        // 函数调用
	ElementTypeFunctionDeclaration // 函数声明no body
	ElementTypeMethod              // 类方法
	ElementTypeMethodCall          // 方法调用
	ElementTypeConstructor         // 构造函数
	ElementTypeDestructor          // 析构函数

	ElementTypeGlobalVariable // 全局变量
	ElementTypeLocalVariable  // 局部变量
	ElementTypeVariable       // 局部变量
	ElementTypeConstant       // 常量
	ElementTypeMacro          // 宏
	ElementTypeField          // 类字段/属性
	ElementTypeParameter      // 函数参数

	ElementTypeComment    // 注释
	ElementTypeDocComment // 文档注释

	ElementTypeAnnotation // 注解/属性
)

type ParsedSource struct {
	Path     string
	Package  string
	Imports  []Import
	Language Language
	Elements []*CodeElement
}

type Import struct {
	Name     string
	FullName string
	Source   string
	Alias    string
}

type CodeElement struct {
	Name       string
	Owner      string
	Type       ElementType
	Parameters []string
	Signature  string
	Range      []int32
	Content    []byte
	Parent     *CodeElement
	Children   []*CodeElement
}
