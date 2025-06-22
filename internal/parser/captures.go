package parser

import (
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"strings"
)

const (
	dotName       = ".name"
	dotArguments  = ".arguments"
	dotParameters = ".parameters"
	dotOwner      = ".owner"
)

// 类型映射表 - captureName -> ElementType
var typeMappings = map[string]ElementType{
	"package":                ElementTypePackage,
	"namespace":              ElementTypeNamespace,
	"import":                 ElementTypeImport,
	"declaration.function":   ElementTypeFunctionDeclaration,
	"definition.method":      ElementTypeMethod,
	"call.method":            ElementTypeMethodCall,
	"definition.function":    ElementTypeFunction,
	"call.function":          ElementTypeFunctionCall,
	"definition.class":       ElementTypeClass,
	"definition.interface":   ElementTypeInterface,
	"definition.struct":      ElementTypeStruct,
	"definition.enum":        ElementTypeEnum,
	"definition.union":       ElementTypeUnion,
	"definition.trait":       ElementTypeTrait,
	"definition.type_alias":  ElementTypeTypeAlias,
	"definition.constructor": ElementTypeConstructor,
	"definition.destructor":  ElementTypeDestructor,
	"global_variable":        ElementTypeGlobalVariable,
	"local_variable":         ElementTypeLocalVariable,
	"variable":               ElementTypeVariable,
	"constant":               ElementTypeConstant,
	"macro":                  ElementTypeMacro,
	"definition.field":       ElementTypeField,
	"definition.parameter":   ElementTypeParameter,
	"comment":                ElementTypeComment,
	"doc_comment":            ElementTypeDocComment,
	"annotation":             ElementTypeAnnotation,
	"undefined":              ElementTypeUndefined,
}

// toElementType 将字符串映射为ElementType
func toElementType(captureName string) ElementType {
	if captureName == types.EmptyString {
		return ElementTypeUndefined
	}
	if et, exists := typeMappings[captureName]; exists {
		return et
	}
	return ElementTypeUndefined
}

// isElementType 字符串是否是ElementType
func isElementType(captureName string, elementType ElementType) bool {
	return toElementType(captureName) == elementType
}

func isNameCapture(captureName string) bool {
	return strings.HasSuffix(captureName, dotName)
}

func isNodeNameCapture(nodeCaptureName string, captureName string) bool {
	if !isNameCapture(captureName) {
		return false
	}
	return captureName == nodeCaptureName+dotName
}

func isParameterCapture(captureName string) bool {
	return strings.HasSuffix(captureName, dotArguments) ||
		strings.HasSuffix(captureName, dotParameters)
}

func isOwnerCapture(captureName string) bool {
	return strings.HasSuffix(captureName, dotOwner)
}
