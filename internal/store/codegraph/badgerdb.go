package codegraph

import (
	"context"
	"fmt"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "codegraph.db"

type badgerDBGraph struct {
	path string
	db   *badger.DB
	txn  *badger.Txn // 当前事务
}

func (b badgerDBGraph) Close() error {
	return b.db.Close()
}

func NewBadgerDBGraph(opts ...GraphOption) (GraphStore, error) {
	b := &badgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}

	// 自定义 BadgerDB 配置
	badgerOpts := badger.DefaultOptions(filepath.Join(b.path, dbName))

	// 值日志配置
	badgerOpts.ValueLogFileSize = 1 << 30 // 1GB
	badgerOpts.ValueLogMaxEntries = 1000000
	badgerOpts.ValueThreshold = 32 // 小于32字节的值直接存储在LSM树中

	// 内存表配置
	badgerOpts.NumMemtables = 1
	badgerOpts.MaxLevels = 7 // 使用默认的7层
	badgerOpts.NumLevelZeroTables = 5
	badgerOpts.NumLevelZeroTablesStall = 10

	// 其他优化
	badgerOpts.BlockSize = 4 * 1024 // 4KB
	badgerOpts.BloomFalsePositive = 0.1
	badgerOpts.SyncWrites = false
	badgerOpts.DetectConflicts = false
	badgerOpts.Compression = options.None
	badgerOpts.CompactL0OnClose = true
	badgerOpts.VerifyValueChecksum = false

	// 设置索引缓存大小
	badgerOpts.IndexCacheSize = 100 << 20 // 100MB

	// Open database
	badgerDB, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	b.db = badgerDB
	return b, nil
}

type GraphOption func(*badgerDBGraph)

func WithPath(basePath string) GraphOption {
	return func(b *badgerDBGraph) {
		b.path = basePath
	}
}

// Document operations
func (b badgerDBGraph) WriteDocument(ctx context.Context, doc *Document) error {
	if b.txn == nil {
		return fmt.Errorf("no active transaction")
	}
	docBytes, err := SerializeDocument(doc)
	if err != nil {
		return err
	}
	return b.txn.Set(DocKey(doc.Path), docBytes)
}

func (b badgerDBGraph) GetDocument(ctx context.Context, path string) (*Document, error) {
	var doc *Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(path))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			var err error
			doc, err = DeserializeDocument(val)
			return err
		})
	})
	return doc, err
}

func (b badgerDBGraph) DeleteDocument(ctx context.Context, path string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(DocKey(path))
	})
}

// Symbol operations
func (b badgerDBGraph) WriteSymbol(ctx context.Context, symbol *Symbol) error {
	if b.txn == nil {
		return fmt.Errorf("no active transaction")
	}
	symbolBytes, err := SerializeSymbol(symbol)
	if err != nil {
		return err
	}
	return b.txn.Set(SymKey(symbol.Name), symbolBytes)
}

func (b badgerDBGraph) GetSymbol(ctx context.Context, name string) (*Symbol, error) {
	var symbol *Symbol
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(SymKey(name))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			var err error
			symbol, err = DeserializeSymbol(val)
			return err
		})
	})
	return symbol, err
}

func (b badgerDBGraph) DeleteSymbol(ctx context.Context, name string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(SymKey(name))
	})
}

// Position operations
func (b badgerDBGraph) GetPositionsBySymbol(ctx context.Context, symbol string) ([]Position, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var positions []Position
	positions = append(positions, sym.Definitions...)
	positions = append(positions, sym.References...)
	positions = append(positions, sym.Implementations...)
	return positions, nil
}

func (b badgerDBGraph) GetPositionsByFile(ctx context.Context, filePath string) ([]Position, error) {
	doc, err := b.GetDocument(ctx, filePath)
	if err != nil {
		return nil, err
	}
	var positions []Position
	for _, symbol := range doc.Symbols {
		sym, err := b.GetSymbol(ctx, symbol)
		if err != nil {
			continue
		}
		for _, pos := range sym.Definitions {
			if pos.FilePath == filePath {
				positions = append(positions, pos)
			}
		}
		for _, pos := range sym.References {
			if pos.FilePath == filePath {
				positions = append(positions, pos)
			}
		}
		for _, pos := range sym.Implementations {
			if pos.FilePath == filePath {
				positions = append(positions, pos)
			}
		}
	}
	return positions, nil
}

func (b badgerDBGraph) GetPositionsByRange(ctx context.Context, filePath string, startLine, endLine int) ([]Position, error) {
	positions, err := b.GetPositionsByFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	var filtered []Position
	for _, pos := range positions {
		if pos.StartLine >= startLine && pos.EndLine <= endLine {
			filtered = append(filtered, pos)
		}
	}
	return filtered, nil
}

// Tree operations
func (b badgerDBGraph) BuildSymbolTree(ctx context.Context, symbol string) (*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	root := &types.GraphNode{
		SymbolName: symbol,
		NodeType:   types.NodeTypeDefinition,
	}

	// Add definitions
	for _, def := range sym.Definitions {
		root.Children = append(root.Children, &types.GraphNode{
			FilePath:   def.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(def),
			NodeType:   types.NodeTypeDefinition,
		})
	}

	// Add references
	for _, ref := range sym.References {
		root.Children = append(root.Children, &types.GraphNode{
			FilePath:   ref.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(ref),
			NodeType:   types.NodeTypeReference,
		})
	}

	// Add implementations
	for _, impl := range sym.Implementations {
		root.Children = append(root.Children, &types.GraphNode{
			FilePath:   impl.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(impl),
			NodeType:   types.NodeTypeImplementation,
		})
	}

	return root, nil
}

func (b badgerDBGraph) GetSymbolReferences(ctx context.Context, symbol string) ([]*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var nodes []*types.GraphNode
	for _, ref := range sym.References {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   ref.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(ref),
			NodeType:   types.NodeTypeReference,
		})
	}
	return nodes, nil
}

func (b badgerDBGraph) GetSymbolDefinitions(ctx context.Context, symbol string) ([]*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var nodes []*types.GraphNode
	for _, def := range sym.Definitions {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   def.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(def),
			NodeType:   types.NodeTypeDefinition,
		})
	}
	return nodes, nil
}

// Transaction operations
func (b *badgerDBGraph) BeginWrite(ctx context.Context) error {
	if b.txn != nil {
		return fmt.Errorf("transaction already exists")
	}
	b.txn = b.db.NewTransaction(true)
	return nil
}

func (b *badgerDBGraph) CommitWrite(ctx context.Context) error {
	if b.txn == nil {
		return fmt.Errorf("no active transaction")
	}
	err := b.txn.Commit()
	b.txn = nil
	return err
}

func (b *badgerDBGraph) RollbackWrite(ctx context.Context) error {
	if b.txn == nil {
		return fmt.Errorf("no active transaction")
	}
	b.txn.Discard()
	b.txn = nil
	return nil
}

// Database operations
func (b badgerDBGraph) DeleteAll(ctx context.Context) error {
	if err := b.db.DropAll(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	return nil
}
