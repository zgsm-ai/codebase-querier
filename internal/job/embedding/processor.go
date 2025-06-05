package embedding

import (
	"errors"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const name = "name"

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured node is missing")
	ErrNoDefinition = errors.New("no definition node found")
	ErrInvalidNode  = errors.New("invalid node")
)

// LanguageProcessor defines the interface for language-specific AST processing
type LanguageProcessor interface {
	ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error)
	GetDefinitionKinds() []string
	FindEnclosingType(node *sitter.Node) *sitter.Node
	FindEnclosingFunction(node *sitter.Node) *sitter.Node
}

// BaseProcessor provides common functionality for all language processors
type BaseProcessor struct {
	definitionKinds []string
	pool            *sync.Pool
}

// NewBaseProcessor creates a new base processor with object pooling
func NewBaseProcessor(definitionKinds []string) *BaseProcessor {
	return &BaseProcessor{
		definitionKinds: definitionKinds,
		pool: &sync.Pool{
			New: func() interface{} {
				return &DefinitionNodeInfo{}
			},
		},
	}
}

// GetDefinitionKinds returns the list of definition kinds for this language
func (p *BaseProcessor) GetDefinitionKinds() []string {
	return p.definitionKinds
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
		Kind: getDefinitionKindFromNodeKind(definitionNode.Kind()),
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

// findDefinitionNode traverses up the AST to find a definition node of the specified kinds
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

// GoProcessor implements LanguageProcessor for Go code
type GoProcessor struct {
	*BaseProcessor
}

// NewGoProcessor creates a new Go language processor
func NewGoProcessor() *GoProcessor {
	return &GoProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_declaration",
			"method_declaration",
			"type_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Go
func (p *GoProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Go doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for Go
func (p *GoProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "type_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Go
func (p *GoProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Go doesn't support nested functions
	return nil
}

// PythonProcessor implements LanguageProcessor for Python code
type PythonProcessor struct {
	*BaseProcessor
}

// NewPythonProcessor creates a new Python language processor
func NewPythonProcessor() *PythonProcessor {
	return &PythonProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"class_definition",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Python
func (p *PythonProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Python
func (p *PythonProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Python
func (p *PythonProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" {
			return curr
		}
		// Stop at class definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_definition" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// JavaProcessor implements LanguageProcessor for Java code
type JavaProcessor struct {
	*BaseProcessor
}

// NewJavaProcessor creates a new Java language processor
func NewJavaProcessor() *JavaProcessor {
	return &JavaProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"interface_declaration",
			"enum_declaration",
			"method_declaration",
			"constructor_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Java
func (p *JavaProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Java doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for Java
func (p *JavaProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Java
func (p *JavaProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Java doesn't support nested functions
	return nil
}

// JavaScriptProcessor implements LanguageProcessor for JavaScript/TypeScript code
type JavaScriptProcessor struct {
	*BaseProcessor
}

// NewJavaScriptProcessor creates a new JavaScript/TypeScript language processor
func NewJavaScriptProcessor() *JavaScriptProcessor {
	return &JavaScriptProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_declaration",
			"class_declaration",
			"method_definition",
			"interface_declaration",
			"enum_declaration",
			"type_alias_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "arrow_function":
			return curr
		}
		// Stop at class/interface definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// RustProcessor implements LanguageProcessor for Rust code
type RustProcessor struct {
	*BaseProcessor
}

// NewRustProcessor creates a new Rust language processor
func NewRustProcessor() *RustProcessor {
	return &RustProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_item",
			"struct_item",
			"enum_item",
			"trait_item",
			"impl_item",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Rust
func (p *RustProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Rust doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for Rust
func (p *RustProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "struct_item", "enum_item", "trait_item", "impl_item":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Rust
func (p *RustProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Rust doesn't support nested functions in the same way as Python
	return nil
}

// CProcessor implements LanguageProcessor for C code
type CProcessor struct {
	*BaseProcessor
}

// NewCProcessor creates a new C language processor
func NewCProcessor() *CProcessor {
	return &CProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"struct_specifier",
			"enum_specifier",
			"union_specifier",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for C
func (p *CProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for C
func (p *CProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "struct_specifier", "enum_specifier", "union_specifier":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C
func (p *CProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C doesn't support nested functions
	return nil
}

// CppProcessor implements LanguageProcessor for C++ code
type CppProcessor struct {
	*BaseProcessor
}

// NewCppProcessor creates a new C++ language processor
func NewCppProcessor() *CppProcessor {
	return &CppProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"class_specifier",
			"struct_specifier",
			"method_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for C++
func (p *CppProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C++ doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for C++
func (p *CppProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_specifier", "struct_specifier":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C++
func (p *CppProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C++ doesn't support nested functions in the same way as Python
	return nil
}

// CSharpProcessor implements LanguageProcessor for C# code
type CSharpProcessor struct {
	*BaseProcessor
}

// NewCSharpProcessor creates a new C# language processor
func NewCSharpProcessor() *CSharpProcessor {
	return &CSharpProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"struct_declaration",
			"interface_declaration",
			"enum_declaration",
			"method_declaration",
			"property_declaration",
			"field_declaration",
			"event_declaration",
			"delegate_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for C#
func (p *CSharpProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C# doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for C#
func (p *CSharpProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C#
func (p *CSharpProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C# doesn't support nested functions in the same way as Python
	return nil
}

// RubyProcessor implements LanguageProcessor for Ruby code
type RubyProcessor struct {
	*BaseProcessor
}

// NewRubyProcessor creates a new Ruby language processor
func NewRubyProcessor() *RubyProcessor {
	return &RubyProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"method_declaration",
			"class_declaration",
			"module_declaration",
			"singleton_method",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Ruby
func (p *RubyProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Ruby doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for Ruby
func (p *RubyProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "module_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Ruby
func (p *RubyProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Ruby doesn't support nested functions in the same way as Python
	return nil
}

// PhpProcessor implements LanguageProcessor for PHP code
type PhpProcessor struct {
	*BaseProcessor
}

// NewPhpProcessor creates a new PHP language processor
func NewPhpProcessor() *PhpProcessor {
	return &PhpProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"interface_declaration",
			"trait_declaration",
			"function_definition",
			"method_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for PHP
func (p *PhpProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // PHP doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for PHP
func (p *PhpProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for PHP
func (p *PhpProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// PHP doesn't support nested functions in the same way as Python
	return nil
}

// KotlinProcessor implements LanguageProcessor for Kotlin code
type KotlinProcessor struct {
	*BaseProcessor
}

// NewKotlinProcessor creates a new Kotlin language processor
func NewKotlinProcessor() *KotlinProcessor {
	return &KotlinProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"object_declaration",
			"interface_declaration",
			"function_declaration",
			"property_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "interface_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_declaration" {
			return curr
		}
		// Stop at class/object/interface definition
		if curr.Kind() == "class_declaration" || curr.Kind() == "object_declaration" || curr.Kind() == "interface_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// ScalaProcessor implements LanguageProcessor for Scala code
type ScalaProcessor struct {
	*BaseProcessor
}

// NewScalaProcessor creates a new Scala language processor
func NewScalaProcessor() *ScalaProcessor {
	return &ScalaProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"object_declaration",
			"trait_declaration",
			"method_declaration",
			"type_alias",
			"enum_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Scala
func (p *ScalaProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Scala
func (p *ScalaProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "trait_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Scala
func (p *ScalaProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "method_declaration" {
			return curr
		}
		// Stop at class/object/trait definition
		if curr.Kind() == "class_declaration" || curr.Kind() == "object_declaration" || curr.Kind() == "trait_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}
