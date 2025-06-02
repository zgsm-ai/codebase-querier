package codegraph

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// Document represents a document in the codebase
type Document struct {
	Path    string   `json:"path"`    // 文档路径
	Symbols []string `json:"symbols"` // 文档中的符号列表
}

// Symbol represents a symbol in the codebase
type Symbol struct {
	Name            string     `json:"name"`            // 符号名
	Definitions     []Position `json:"definitions"`     // 定义位置
	References      []Position `json:"references"`      // 引用位置
	Implementations []Position `json:"implementations"` // 实现位置
}

// Position represents a position in a document
type Position struct {
	FilePath    string         `json:"file_path"`    // 文件路径
	StartLine   int            `json:"start_line"`   // 开始行
	StartColumn int            `json:"start_column"` // 开始列
	EndLine     int            `json:"end_line"`     // 结束行
	EndColumn   int            `json:"end_column"`   // 结束列
	NodeType    types.NodeType `json:"node_type"`    // 节点类型
}

// GraphStore defines the interface for graph storage
type GraphStore interface {
	// Document operations
	WriteDocument(ctx context.Context, doc *Document) error
	GetDocument(ctx context.Context, path string) (*Document, error)
	DeleteDocument(ctx context.Context, path string) error

	// Symbol operations
	WriteSymbol(ctx context.Context, symbol *Symbol) error
	GetSymbol(ctx context.Context, name string) (*Symbol, error)
	DeleteSymbol(ctx context.Context, name string) error

	// Position operations
	GetPositionsBySymbol(ctx context.Context, symbol string) ([]Position, error)
	GetPositionsByFile(ctx context.Context, filePath string) ([]Position, error)
	GetPositionsByRange(ctx context.Context, filePath string, startLine, endLine int) ([]Position, error)

	// Tree operations
	BuildSymbolTree(ctx context.Context, symbol string) (*types.GraphNode, error)
	GetSymbolReferences(ctx context.Context, symbol string) ([]*types.GraphNode, error)
	GetSymbolDefinitions(ctx context.Context, symbol string) ([]*types.GraphNode, error)

	// Transaction operations
	BeginWrite(ctx context.Context) error
	CommitWrite(ctx context.Context) error
	RollbackWrite(ctx context.Context) error

	// Database operations
	Close() error
	DeleteAll(ctx context.Context) error
}

// Key prefixes for different types of data
const (
	DocPrefix = "doc:" // 文档数据前缀
	SymPrefix = "sym:" // 符号数据前缀
	PosPrefix = "pos:" // 位置数据前缀
)

// Key generation functions
func DocKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", DocPrefix, path))
}

func SymKey(name string) []byte {
	return []byte(fmt.Sprintf("%s%s", SymPrefix, name))
}

func PosKey(filePath string, line, col int) []byte {
	return []byte(fmt.Sprintf("%s%s:%d:%d", PosPrefix, filePath, line, col))
}

// Helper functions for serialization
func SerializeDocument(doc *Document) ([]byte, error) {
	return json.Marshal(doc)
}

func DeserializeDocument(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func SerializeSymbol(symbol *Symbol) ([]byte, error) {
	return json.Marshal(symbol)
}

func DeserializeSymbol(data []byte) (*Symbol, error) {
	var symbol Symbol
	if err := json.Unmarshal(data, &symbol); err != nil {
		return nil, err
	}
	return &symbol, nil
}

// Helper function to convert Position to types.Position
func ToTypesPosition(pos Position) types.Position {
	return types.Position{
		StartLine:   pos.StartLine,
		StartColumn: pos.StartColumn,
		EndLine:     pos.EndLine,
		EndColumn:   pos.EndColumn,
	}
}

// Helper function to convert types.Position to Position
func FromTypesPosition(pos types.Position, filePath string, nodeType types.NodeType) Position {
	return Position{
		FilePath:    filePath,
		StartLine:   pos.StartLine,
		StartColumn: pos.StartColumn,
		EndLine:     pos.EndLine,
		EndColumn:   pos.EndColumn,
		NodeType:    nodeType,
	}
}
