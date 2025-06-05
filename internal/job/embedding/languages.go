package embedding

import (
	"errors"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Define language-specific ProcessMatchFunc implementations.
// These functions encapsulate the logic to process Tree-sitter matches
// and extract DefinitionNodeInfo based on each language's AST structure.

// processGoMatch processes a Tree-sitter query match for Go code
func processGoMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Go queries often capture the *name* node of a definition.
	// We need to find the parent definition node from the captured node.
	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node // Get the address of the Node struct

	// Traverse up to find the nearest definition node ancestor
	var definitionNode *sitter.Node // Use pointer
	curr := capturedNode            // Start traversal from the captured node (*sitter.Node)

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "method_declaration", "type_declaration":
			definitionNode = curr
			// Found a definition node, extract info and stop traversing up.
			goto extractGoInfo // Jump to extraction logic
		}
		curr = curr.Parent() // Move up to parent node (returns *sitter.Node)
	}

	// If we reached here, no relevant definition node was found by traversing up.
	return nil, nil

extractGoInfo:
	// Extract information from the found definitionNode
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processGoMatch: definition node not found after match or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""

	// Find the name node (common field name is "name")
	nameNode := definitionNode.ChildByFieldName("name")
	// For method declarations, the name is usually in the "name" field, but for type declarations
	// it might be in the `type_identifier` child of the `type_spec`.

	// Specific logic for different Go definition types
	switch definitionNode.Kind() {
	case "function_declaration":
		nameNode = definitionNode.ChildByFieldName("name")
	case "method_declaration":
		nameNode = definitionNode.ChildByFieldName("name")
		// For methods, find the receiver type to determine parent class/type
		receiverNode := definitionNode.ChildByFieldName("receiver")
		if receiverNode != nil && !receiverNode.IsMissing() {
			// The receiver node itself represents the type
			// Clean up the receiver text (e.g., remove *, [], or struct {})
			receiverText := receiverNode.Utf8Text(content)
			// Basic cleaning: remove pointer/slice/map indicators and spaces
			receiverText = strings.TrimLeft(receiverText, "*[]&")
			receiverText = strings.SplitN(receiverText, " ", 2)[0] // Get the type name before any variable name
			parentClass = receiverText
		}
	case "type_declaration":
		// Type declarations can be structs, interfaces, etc.
		// The name is typically in a `type_identifier` child.
		typeSpecNode := definitionNode.ChildByFieldName("type") // Child with field name "type"
		if typeSpecNode != nil && !typeSpecNode.IsMissing() {
			nameNode = typeSpecNode.ChildByFieldName("name") // Or similar identifier node
			// Also consider aliased types, which might have the name directly under type_declaration
			if nameNode == nil || nameNode.IsMissing() {
				nameNode = definitionNode.ChildByFieldName("name")
			}

		}
	}

	if nameNode != nil && !nameNode.IsMissing() {
		name = nameNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods
		// ParentFunc is typically not applicable for top-level definitions in Go
	})

	return defInfos, nil
}

// processPythonMatch processes a Tree-sitter query match for Python code
func processPythonMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Python queries often capture the *name* node of a definition or the definition node itself.
	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node // Get the address of the Node struct

	// Traverse up to find the nearest definition node ancestor
	var definitionNode *sitter.Node // Use pointer
	curr := capturedNode            // Start traversal from the captured node (*sitter.Node)

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_definition", "class_definition":
			definitionNode = curr
			// Found a definition node, extract info and stop traversing up.
			goto extractPythonInfo // Jump to extraction logic
		}
		curr = curr.Parent() // Move up to parent node (returns *sitter.Node)
	}

	// If we reached here, no relevant definition node was found by traversing up.
	return nil, nil

extractPythonInfo:
	// Extract information from the found definitionNode
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processPythonMatch: definition node not found after match or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""
	parentFunc := ""

	// Find the name node (common field name is "name")
	nameNode := definitionNode.ChildByFieldName("name")
	if nameNode != nil && !nameNode.IsMissing() {
		name = nameNode.Utf8Text(content)
	}

	// Determine if it's a method by checking for an enclosing class
	// or if it's a nested function by checking for an enclosing function.
	if definitionNode.Kind() == "function_definition" {
		// Check for enclosing class first
		enclosingClassNode := findEnclosingPythonClass(definitionNode) // Helper function expects *sitter.Node
		if enclosingClassNode != nil && !enclosingClassNode.IsMissing() {
			kind = MethodKind // Refine kind if it's a method
			// Get the name of the enclosing class
			classNameNode := enclosingClassNode.ChildByFieldName("name")
			if classNameNode != nil && !classNameNode.IsMissing() {
				parentClass = classNameNode.Utf8Text(content)
			}
		} else {
			// If not in a class, check for enclosing function (nested function)
			enclosingFuncNode := findEnclosingPythonFunction(definitionNode) // New helper function expects *sitter.Node
			if enclosingFuncNode != nil && !enclosingFuncNode.IsMissing() {
				// Get the name of the enclosing function
				funcNameNode := enclosingFuncNode.ChildByFieldName("name")
				if funcNameNode != nil && !funcNameNode.IsMissing() {
					parentFunc = funcNameNode.Utf8Text(content)
				}
			}
		}

	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods
		ParentFunc:  parentFunc,  // Populated for nested functions
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class definition node for a given node in Python AST.
// Works with *sitter.Node pointers.
func findEnclosingPythonClass(node *sitter.Node) *sitter.Node {
	// Start from the parent of the given node, as the node itself is inside the class.
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_definition" {
			return curr // Found enclosing class declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing class found
}

// Helper function to find the nearest enclosing function definition node for a given node in Python AST.
// Useful for identifying nested functions.
func findEnclosingPythonFunction(node *sitter.Node) *sitter.Node {
	// Start from the parent of the given node.
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_definition":
			return curr // Found enclosing function declaration
		}
		// Stop at class definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_definition" {
			return nil // Enclosed in a class, not a nested function in the top-level sense
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing function found
}

// processJavaMatch processes a Tree-sitter query match for Java code
func processJavaMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node // Get the address of the Node struct

	// Traverse up to find the nearest definition node ancestor (class, interface, enum, method, constructor)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration", "method_declaration", "constructor_declaration":
			definitionNode = curr
			goto extractJavaInfo // Jump to extraction logic
		}
		curr = curr.Parent() // Move up
	}

	return nil, nil // No relevant definition found

extractJavaInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processJavaMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "class_declaration", "interface_declaration", "enum_declaration":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_declaration", "constructor_declaration":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class/interface/enum for methods and constructors
		enclosingTypeNode := findEnclosingJavaType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			nameNode := enclosingTypeNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}
	}

	if name == "" {
		// Fallback: if name couldn't be extracted by field name, try the node content itself
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass,
		// ParentFunc is not typically applicable in Java like in Python nested functions
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class, interface, or enum declaration in Java AST.
func findEnclosingJavaType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// processJavaScriptMatch processes a Tree-sitter query match for JavaScript code
func processJavaScriptMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node // Get the address of the Node struct

	// Traverse up to find the nearest definition node ancestor (function, class, method)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "class_declaration", "method_definition":
			definitionNode = curr
			goto extractJavaScriptInfo // Jump to extraction logic
		}
		curr = curr.Parent() // Move up
	}

	return nil, nil // No relevant definition found

extractJavaScriptInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processJavaScriptMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""
	parentFunc := ""

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "function_declaration", "class_declaration":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_definition":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class for methods
		enclosingClassNode := findEnclosingJavaScriptClass(definitionNode) // Helper function
		if enclosingClassNode != nil && !enclosingClassNode.IsMissing() {
			nameNode := enclosingClassNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}
		// Check for enclosing function for nested functions
		enclosingFuncNode := findEnclosingJavaScriptFunction(definitionNode) // Helper function
		if enclosingFuncNode != nil && !enclosingFuncNode.IsMissing() {
			nameNode := enclosingFuncNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() {
				parentFunc = nameNode.Utf8Text(content)
			}
		}
	}

	if name == "" {
		// Fallback: if name couldn't be extracted by field name, try the node content itself
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass,
		ParentFunc:  parentFunc,
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class declaration in JavaScript AST.
func findEnclosingJavaScriptClass(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_declaration" {
			return curr // Found enclosing class declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing class found
}

// Helper function to find the nearest enclosing function declaration in JavaScript AST.
// Useful for identifying nested functions.
func findEnclosingJavaScriptFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration":
			return curr // Found enclosing function declaration
		}
		// Stop at class definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_declaration" {
			return nil // Enclosed in a class, not a nested function in the top-level sense
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing function found
}

// processRustMatch processes a Tree-sitter query match for Rust code
func processRustMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_item", "struct_item", "enum_item", "trait_item", "impl_item":
			definitionNode = curr
			goto extractRustInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractRustInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processRustMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // In Rust, parent is usually a Trait or struct in an impl block

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "function_item":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Check if it's a method within an impl block
		enclosingImplNode := findEnclosingRustImpl(definitionNode) // Helper function
		if enclosingImplNode != nil && !enclosingImplNode.IsMissing() {
			// In an impl block, the parent is the type being implemented for or the trait
			// Look for the `type` or `trait` child node
			parentTypeNode := enclosingImplNode.ChildByFieldName("type")
			if parentTypeNode == nil || parentTypeNode.IsMissing() {
				parentTypeNode = enclosingImplNode.ChildByFieldName("trait")
			}

			if parentTypeNode != nil && !parentTypeNode.IsMissing() {
				parentClass = parentTypeNode.Utf8Text(content)
			}
		}

	case "struct_item", "enum_item", "trait_item":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
		// Impl blocks themselves can be considered definitions, but they don't have names in the same way.
		// We are primarily interested in the items *within* impl blocks (like functions).
	case "impl_item":
		// Impl blocks don't have a simple "name" field that represents the implemented type/trait.
		// The name is derived from the `type` or `trait` child node.
		// However, we only want to create a definition for the impl block itself if it's a top-level item
		// and we can extract a meaningful name (e.g., the type it implements for).
		// For now, we will focus on definitions *within* impl blocks captured by other queries (like function_item).
		return nil, nil // Skip creating a definition for the impl_item itself for now.
	}

	if name == "" {
		// Fallback: if name couldn't be extracted by field name, try the node content itself
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods within impl blocks
		// ParentFunc is not typically applicable in Rust
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing impl block in Rust AST.
func findEnclosingRustImpl(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "impl_item" {
			return curr // Found enclosing impl block
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing impl block found
}

// processCMatch processes a Tree-sitter query match for C code
func processCMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (function, struct, enum, union)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_definition", "struct_specifier", "enum_specifier", "union_specifier":
			definitionNode = curr
			goto extractCInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractCInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processCMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "function_definition":
		// Function name is often in the `declarator` child, which might have a `identifier` child
		declaratorNode := definitionNode.ChildByFieldName("declarator")
		if declaratorNode != nil && !declaratorNode.IsMissing() {
			nameNode := declaratorNode.ChildByFieldName("identifier")
			if nameNode != nil && !nameNode.IsMissing() {
				name = nameNode.Utf8Text(content)
			}
		}
	case "struct_specifier", "enum_specifier", "union_specifier":
		// Name is usually a `type_identifier` child
		nameNode := definitionNode.ChildByFieldName("name") // Try "name" first
		if nameNode == nil || nameNode.IsMissing() {
			nameNode = definitionNode.ChildByFieldName("type_identifier") // Then try "type_identifier"
		}
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	}

	if name == "" {
		// Fallback: if name couldn't be extracted by field name, try the node content itself
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node: definitionNode,
		Kind: kind,
		Name: name,
		// ParentClass and ParentFunc are not typically applicable for C
	})

	return defInfos, nil
}

// processCPPMatch processes a Tree-sitter query match for C++ code
func processCPPMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (function, class, struct, method)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_definition", "class_specifier", "struct_specifier", "method_declaration":
			definitionNode = curr
			goto extractCppInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractCppInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processCPPMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "function_definition", "method_declaration":
		// Function or method name is often in the `declarator` child, which might have a `identifier` child
		declaratorNode := definitionNode.ChildByFieldName("declarator")
		if declaratorNode != nil && !declaratorNode.IsMissing() {
			// The declarator can be complex, try to find the identifier within it
			nameNode := findIdentifierInDeclarator(declaratorNode) // Helper function
			if nameNode != nil && !nameNode.IsMissing() {
				name = nameNode.Utf8Text(content)
			}
		}

		// Find enclosing class/struct for methods
		if definitionNode.Kind() == "method_declaration" {
			enclosingTypeNode := findEnclosingCppType(definitionNode) // Helper function
			if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
				nameNode := enclosingTypeNode.ChildByFieldName("name") // Class/struct name
				if nameNode != nil && !nameNode.IsMissing() {
					parentClass = nameNode.Utf8Text(content)
				}
			}
		}

	case "class_specifier", "struct_specifier":
		// Name is usually a `type_identifier` child
		nameNode := definitionNode.ChildByFieldName("name") // Try "name" first
		if nameNode == nil || nameNode.IsMissing() {
			nameNode = definitionNode.ChildByFieldName("type_identifier") // Then try "type_identifier"
		}
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	}

	// Fallback: if name couldn't be extracted by field name, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods
		// ParentFunc is not typically applicable in C++ like in Python nested functions
	})

	return defInfos, nil
}

// Helper function to find the identifier node within a complex C++ declarator.
func findIdentifierInDeclarator(node *sitter.Node) *sitter.Node {
	// Declarators can be nested (e.g., pointers, arrays, function pointers).
	// Traverse down until we find an identifier.
	curr := node

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "identifier":
			return curr // Found the identifier
		case "pointer_declarator", "array_declarator", "function_declarator":
			// Continue searching in the child declarator
			curr = curr.ChildByFieldName("declarator")
		default:
			return nil // Unknown declarator type, stop searching
		}
	}

	return nil // No identifier found in the declarator
}

// Helper function to find the nearest enclosing class or struct specifier in C++ AST.
func findEnclosingCppType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_specifier", "struct_specifier":
			return curr // Found enclosing type specifier
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// processCSharpMatch processes a Tree-sitter query match for C# code
func processCSharpMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (class, struct, interface, enum, method, property, field, event, delegate)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration", "method_declaration", "property_declaration", "field_declaration", "event_declaration", "delegate_declaration":
			definitionNode = curr
			goto extractCSharpInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractCSharpInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processCSharpMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := ""

	// Extract name and find parent class
	switch definitionNode.Kind() {
	case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_declaration", "property_declaration", "field_declaration", "event_declaration", "delegate_declaration":
		// For members, the name is often an identifier child
		nameNode := definitionNode.ChildByFieldName("name") // Try "name" field first
		if nameNode == nil || nameNode.IsMissing() {
			// Fallback for nodes where name is a direct child identifier
			for i := 0; i < int(definitionNode.ChildCount()); i++ {
				child := definitionNode.Child(uint(i))
				if child != nil && !child.IsMissing() && child.Kind() == "identifier" {
					nameNode = child
					break
				}
			}
		}
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing type (class, struct, interface, enum)
		enclosingTypeNode := findEnclosingCSharpType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			nameNode := enclosingTypeNode.ChildByFieldName("name") // Enclosing type name
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}
	}

	// Fallback: if name couldn't be extracted, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for members within types
		// ParentFunc is not typically applicable in C#
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing type declaration in C# AST.
func findEnclosingCSharpType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// processRubyMatch processes a Tree-sitter query match for Ruby code
func processRubyMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (method, class, module, singleton_method)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "method_declaration", "class_declaration", "module_declaration", "singleton_method":
			definitionNode = curr
			goto extractRubyInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractRubyInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processRubyMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // In Ruby, this could be a class or module name

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "method_declaration", "singleton_method":
		// Method name is usually an identifier child
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class or module
		enclosingTypeNode := findEnclosingRubyType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			// The name of the enclosing class/module is typically an identifier child
			nameNode := enclosingTypeNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}

	case "class_declaration", "module_declaration":
		// Class or module name is usually an constant or identifier child
		nameNode := definitionNode.ChildByFieldName("name") // Try "name" field first
		if nameNode == nil || nameNode.IsMissing() {
			// Fallback for nodes where name is a direct child (e.g., constant)
			for i := 0; i < int(definitionNode.ChildCount()); i++ {
				child := definitionNode.Child(uint(i))
				if child != nil && !child.IsMissing() && (child.Kind() == "constant" || child.Kind() == "identifier") {
					nameNode = child
					break
				}
			}
		}
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	}

	// Fallback: if name couldn't be extracted, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods within classes/modules
		// ParentFunc is not typically applicable in Ruby for nested methods in the same way as Python
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class or module declaration in Ruby AST.
func findEnclosingRubyType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "module_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// processScalaMatch processes a Tree-sitter query match for Scala code
func processScalaMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (class, object, trait, method, type alias, enum)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "trait_declaration", "method_declaration", "type_alias", "enum_declaration":
			definitionNode = curr
			goto extractScalaInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractScalaInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processScalaMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // In Scala, this could be a class, object, or trait name
	parentFunc := ""  // Scala supports nested functions

	// Extract name and find parent context
	switch definitionNode.Kind() {
	case "class_declaration", "object_declaration", "trait_declaration", "type_alias", "enum_declaration":
		// Name is often an identifier
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_declaration":
		// Method name is an identifier
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class, object, trait, or function
		enclosingNode := findEnclosingScalaContext(definitionNode) // Helper function
		if enclosingNode != nil && !enclosingNode.IsMissing() {
			switch enclosingNode.Kind() {
			case "class_declaration", "object_declaration", "trait_declaration":
				// Enclosed in a type
				nameNode := enclosingNode.ChildByFieldName("name")
				if nameNode != nil && !nameNode.IsMissing() {
					parentClass = nameNode.Utf8Text(content)
				}
			case "method_declaration": // Nested method within another method
				// Enclosed in a function/method
				nameNode := enclosingNode.ChildByFieldName("name")
				if nameNode != nil && !nameNode.IsMissing() {
					parentFunc = nameNode.Utf8Text(content)
				}
			}
		}
	}

	// Fallback: if name couldn't be extracted, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for members within types
		ParentFunc:  parentFunc,  // Populated for nested methods
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class, object, trait, or method declaration in Scala AST.
func findEnclosingScalaContext(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "trait_declaration", "method_declaration":
			return curr // Found enclosing context
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing context found
}

// processTypescriptMatch processes a Tree-sitter query match for TypeScript code.
// Since TSX is a superset of TypeScript, we can often reuse the TypeScript parsing logic.
func processTypescriptMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node // Get the address of the Node struct

	// Traverse up to find the nearest definition node ancestor (function, class, method, interface, enum, type alias)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "class_declaration", "method_definition", "interface_declaration", "enum_declaration", "type_alias_declaration":
			definitionNode = curr
			goto extractTypescriptInfo // Jump to extraction logic
		}
		curr = curr.Parent() // Move up
	}

	return nil, nil // No relevant definition found

extractTypescriptInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processTypescriptMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // Parent class or interface
	parentFunc := ""  // Enclosing function for nested functions/classes

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "function_declaration", "class_declaration", "interface_declaration", "enum_declaration", "type_alias_declaration":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_definition":
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class or interface for methods
		enclosingTypeNode := findEnclosingTypescriptType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			nameNode := enclosingTypeNode.ChildByFieldName("name") // Enclosing type name
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}
		// Check for enclosing function for nested functions or classes within functions
		enclosingFuncNode := findEnclosingTypescriptFunction(definitionNode) // Helper function
		if enclosingFuncNode != nil && !enclosingFuncNode.IsMissing() {
			nameNode := enclosingFuncNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() { // Fix: use nameNode
				parentFunc = nameNode.Utf8Text(content)
			}
		}
	}

	if name == "" {
		// Fallback: if name couldn't be extracted by field name, try the node content itself
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass,
		ParentFunc:  parentFunc,
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class or interface declaration in TypeScript AST.
func findEnclosingTypescriptType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// Helper function to find the nearest enclosing function declaration in TypeScript AST.
// Useful for identifying nested functions or classes within functions.
func findEnclosingTypescriptFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "arrow_function": // Include arrow functions
			return curr // Found enclosing function declaration
		}
		// Stop at class/interface definition to correctly identify methods/nested types vs nested functions
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" {
			return nil // Enclosed in a type, not a nested function in the top-level sense
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing function found
}

// processTsxMatch processes a Tree-sitter query match for TSX code.
func processTsxMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	// TSX is a superset of TypeScript, so we can largely reuse the TypeScript logic.
	// Additional logic might be needed in the future to handle JSX-specific nodes if necessary.
	return processTypescriptMatch(match, root, content)
}

// processPhpMatch processes a Tree-sitter query match for PHP code.
func processPhpMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (class, interface, trait, function, method)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration", "function_definition", "method_declaration":
			definitionNode = curr
			goto extractPhpInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractPhpInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processPhpMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // In PHP, this could be a class, interface, or trait name
	// PHP supports nested functions (closures, anonymous functions), but defining named functions inside other named functions is less common for top-level definitions.

	// Extract name based on node kind
	switch definitionNode.Kind() {
	case "class_declaration", "interface_declaration", "trait_declaration", "function_definition":
		nameNode := definitionNode.ChildByFieldName("name") // Often an identifier
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}
	case "method_declaration":
		nameNode := definitionNode.ChildByFieldName("name") // Often an identifier
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class, interface, or trait for methods
		enclosingTypeNode := findEnclosingPhpType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			nameNode := enclosingTypeNode.ChildByFieldName("name") // Enclosing type name
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}
	}

	// Fallback: if name couldn't be extracted, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for methods within types
		// ParentFunc could be populated for nested functions if needed
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing class, interface, or trait declaration in PHP AST.
func findEnclosingPhpType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// processKotlinMatch processes a Tree-sitter query match for Kotlin code.
func processKotlinMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	var defInfos []*DefinitionNodeInfo

	// Ensure the match has at least one capture and the node is not missing.
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, nil // No valid captured node, skip
	}

	// Assuming the first capture is the most relevant node (e.g., name or definition)
	capturedNode := &match.Captures[0].Node

	// Traverse up to find the nearest definition node ancestor (class, object, interface, function, property)
	var definitionNode *sitter.Node
	curr := capturedNode

	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "interface_declaration", "function_declaration", "property_declaration":
			definitionNode = curr
			goto extractKotlinInfo // Jump to extraction logic
		}
		curr = curr.Parent()
	}

	return nil, nil // No relevant definition found

extractKotlinInfo:
	if definitionNode == nil || definitionNode.IsMissing() {
		return nil, errors.New("processKotlinMatch: definition node not found or is missing")
	}

	kind := GetDefinitionKindFromNodeKind(definitionNode.Kind())
	name := ""
	parentClass := "" // In Kotlin, this could be a class, object, or interface name
	parentFunc := ""  // Kotlin supports local functions (nested functions)

	// Extract name and find parent context
	switch definitionNode.Kind() {
	case "class_declaration", "object_declaration", "interface_declaration", "property_declaration":
		// Name is often an identifier
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class, object, or interface
		enclosingTypeNode := findEnclosingKotlinType(definitionNode) // Helper function
		if enclosingTypeNode != nil && !enclosingTypeNode.IsMissing() {
			nameNode := enclosingTypeNode.ChildByFieldName("name")
			if nameNode != nil && !nameNode.IsMissing() {
				parentClass = nameNode.Utf8Text(content)
			}
		}

	case "function_declaration":
		// Function name is an identifier
		nameNode := definitionNode.ChildByFieldName("name")
		if nameNode != nil && !nameNode.IsMissing() {
			name = nameNode.Utf8Text(content)
		}

		// Find enclosing class, object, interface, or function
		enclosingNode := findEnclosingKotlinContext(definitionNode) // Helper function
		if enclosingNode != nil && !enclosingNode.IsMissing() {
			switch enclosingNode.Kind() {
			case "class_declaration", "object_declaration", "interface_declaration":
				// Enclosed in a type
				nameNode := enclosingNode.ChildByFieldName("name")
				if nameNode != nil && !nameNode.IsMissing() {
					parentClass = nameNode.Utf8Text(content)
				}
			case "function_declaration": // Nested function
				// Enclosed in a function
				nameNode := enclosingNode.ChildByFieldName("name")
				if nameNode != nil && !nameNode.IsMissing() {
					parentFunc = nameNode.Utf8Text(content)
				}
			}
		}
	}

	// Fallback: if name couldn't be extracted, try the node content itself
	if name == "" {
		name = definitionNode.Utf8Text(content)
	}

	defInfos = append(defInfos, &DefinitionNodeInfo{
		Node:        definitionNode,
		Kind:        kind,
		Name:        name,
		ParentClass: parentClass, // Populated for members within types
		ParentFunc:  parentFunc,  // Populated for nested functions
	})

	return defInfos, nil
}

// Helper function to find the nearest enclosing type declaration (class, object, interface) in Kotlin AST.
func findEnclosingKotlinType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "interface_declaration":
			return curr // Found enclosing type declaration
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing type found
}

// Helper function to find the nearest enclosing context (class, object, interface, or function) in Kotlin AST.
func findEnclosingKotlinContext(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "interface_declaration", "function_declaration":
			return curr // Found enclosing context
		}
		curr = curr.Parent() // Move up
	}
	return nil // No enclosing context found
}
