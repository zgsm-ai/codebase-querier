package codegraph

import (
	"encoding/json"
	"fmt"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// Key prefixes for different types of data
const (
	// Document related prefixes
	prefixDocument = "doc:"  // Document metadata and content
	prefixDocPath  = "path:" // Document path lookup

	// Symbol related prefixes
	prefixSymbol = "sym:"  // Symbol information
	prefixDef    = "def:"  // Symbol definitions
	prefixRef    = "ref:"  // Symbol references
	prefixImpl   = "impl:" // Symbol implementations
	prefixType   = "type:" // Symbol type definitions

	// Position related prefix
	prefixPosition = "pos:" // Position information
)

// StorageKey defines the interface for storage keys
type StorageKey interface {
	Encode() []byte
}

// StorageValue defines the interface for storage values
type StorageValue interface {
	Encode() ([]byte, error)
	Decode([]byte) error
}

// DocumentKey represents a key for storing document information
// Format: doc:{documentPath}
type DocumentKey struct {
	DocumentPath string
}

func (k DocumentKey) Encode() []byte {
	return []byte(fmt.Sprintf("%s%s", prefixDocument, k.DocumentPath))
}

// SymbolKey represents a key for storing symbol information
// Format: sym:{symbolName}
type SymbolKey struct {
	SymbolName string
}

func (k SymbolKey) Encode() []byte {
	return []byte(fmt.Sprintf("%s%s", prefixSymbol, k.SymbolName))
}

// PositionKey represents a key for storing position information
// Format: pos:{documentPath}:{startLine}:{startColumn}
type PositionKey struct {
	DocumentPath string
	StartLine    int
	StartColumn  int
}

func (k PositionKey) Encode() []byte {
	return []byte(fmt.Sprintf("%s%s:%d:%d", prefixPosition, k.DocumentPath, k.StartLine, k.StartColumn))
}

// DocumentValue represents the value stored for a document
// Contains the document content and metadata
type DocumentValue struct {
	Content       string `json:"content"`        // Document content
	SchemaVersion int    `json:"schema_version"` // Schema version for future compatibility
}

func (v DocumentValue) Encode() ([]byte, error) {
	return json.Marshal(v)
}

func (v *DocumentValue) Decode(data []byte) error {
	return json.Unmarshal(data, v)
}

// SymbolValue represents the value stored for a symbol
// Contains symbol information and its relationships
type SymbolValue struct {
	Definitions     []PositionInfo `json:"definitions"`                // Symbol definitions
	References      []PositionInfo `json:"references"`                 // Symbol references
	Implementations []PositionInfo `json:"implementations,omitempty"`  // Symbol implementations
	TypeDefinitions []PositionInfo `json:"type_definitions,omitempty"` // Type definitions
}

func (v SymbolValue) Encode() ([]byte, error) {
	return json.Marshal(v)
}

func (v *SymbolValue) Decode(data []byte) error {
	return json.Unmarshal(data, v)
}

// PositionInfo represents position information for a symbol occurrence
type PositionInfo struct {
	FilePath   string         `json:"file_path"`             // File path
	Position   types.Position `json:"position"`              // Position in file
	NodeType   types.NodeType `json:"node_type"`             // Type of node (definition, reference, etc.)
	Content    string         `json:"content,omitempty"`     // Code content at this position
	SymbolKind string         `json:"symbol_kind,omitempty"` // Kind of symbol (function, class, etc.)
	IsExternal bool           `json:"is_external,omitempty"` // Whether this is an external reference
}

// PositionValue represents the value stored for a position
// Contains position-specific information
type PositionValue struct {
	SymbolName string         `json:"symbol_name"` // Symbol name at this position
	NodeType   types.NodeType `json:"node_type"`   // Type of node
}

func (v PositionValue) Encode() ([]byte, error) {
	return json.Marshal(v)
}

func (v *PositionValue) Decode(data []byte) error {
	return json.Unmarshal(data, v)
}
