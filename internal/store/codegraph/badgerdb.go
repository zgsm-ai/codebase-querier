package codegraph

import (
	"context"
	"fmt"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "badger.db"

type badgerDBGraph struct {
	path string
	db   *badger.DB
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
	badgerOpts.ValueThreshold = 32        // 小于32字节的值直接存储在LSM树中

	// 内存表配置
	badgerOpts.NumMemtables = 4              // 增加内存表数量
	badgerOpts.NumLevelZeroTables = 50       // 增加L0层表数量
	badgerOpts.NumLevelZeroTablesStall = 100 // 增加L0层表数量阈值

	// 其他优化
	badgerOpts.BlockSize = 4 * 1024        // 4KB
	badgerOpts.BloomFalsePositive = 0.01   // 降低误判率
	badgerOpts.SyncWrites = false          // 异步写入
	badgerOpts.DetectConflicts = false     // 禁用冲突检测
	badgerOpts.Compression = options.ZSTD  // 使用ZSTD压缩
	badgerOpts.ZSTDCompressionLevel = 3    // 压缩级别
	badgerOpts.CompactL0OnClose = true     // 关闭时压缩L0层
	badgerOpts.VerifyValueChecksum = false // 禁用校验和验证

	// 设置索引缓存大小
	badgerOpts.IndexCacheSize = 256 << 20 // 256MB

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

// BatchWrite 批量写入文档和符号
func (b badgerDBGraph) BatchWrite(ctx context.Context, docs []*Document, symbols []*Symbol) error {
	wb := b.db.NewWriteBatch()
	defer wb.Cancel()

	// 写入文档
	for _, doc := range docs {
		docBytes, err := SerializeDocument(doc)
		if err != nil {
			return err
		}
		if err := wb.Set(DocKey(doc.Path), docBytes); err != nil {
			return err
		}
	}

	// 写入符号
	for _, symbol := range symbols {
		symbolBytes, err := SerializeSymbol(symbol)
		if err != nil {
			return err
		}
		if err := wb.Set(SymKey(symbol.Name), symbolBytes); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// Query 实现查询接口
func (b badgerDBGraph) Query(ctx context.Context, opts *types.RelationQueryOptions) ([]*types.GraphNode, error) {
	// 1. 获取文档
	var doc *Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(opts.FilePath))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			var err error
			doc, err = DeserializeDocument(val)
			return err
		})
	})
	if err != nil {
		return nil, err
	}

	var nodes []*types.GraphNode

	// 2. 根据查询条件构建树
	if opts.SymbolName != "" {
		// 按符号名查询
		var symbol *Symbol
		err := b.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(SymKey(opts.SymbolName))
			if err != nil {
				return err
			}
			return item.Value(func(val []byte) error {
				var err error
				symbol, err = DeserializeSymbol(val)
				return err
			})
		})
		if err != nil {
			return nil, err
		}

		// 构建符号树
		nodes = b.buildSymbolTree(symbol, opts.MaxLayer)
	} else {
		// 按位置查询
		nodes = b.buildPositionTree(doc, opts.StartLine, opts.EndLine, opts.MaxLayer)
	}

	// 3. 如果需要，添加代码内容
	if opts.IncludeContent == 1 {
		for _, node := range nodes {
			node.Content = doc.Content
		}
	}

	return nodes, nil
}

// buildSymbolTree 构建符号树
func (b badgerDBGraph) buildSymbolTree(symbol *Symbol, maxLayer int) []*types.GraphNode {
	if maxLayer <= 0 {
		return nil
	}

	var nodes []*types.GraphNode

	// 处理定义
	for _, def := range symbol.Definitions {
		node := &types.GraphNode{
			FilePath:   def.FilePath,
			SymbolName: symbol.Name,
			Position:   ToTypesPosition(def),
			NodeType:   def.NodeType,
			Children:   make([]*types.GraphNode, 0),
		}
		nodes = append(nodes, node)
	}

	// 处理引用
	for _, ref := range symbol.References {
		node := &types.GraphNode{
			FilePath:   ref.FilePath,
			SymbolName: symbol.Name,
			Position:   ToTypesPosition(ref),
			NodeType:   ref.NodeType,
			Children:   make([]*types.GraphNode, 0),
		}
		nodes = append(nodes, node)
	}

	// 处理实现
	for _, impl := range symbol.Implementations {
		node := &types.GraphNode{
			FilePath:   impl.FilePath,
			SymbolName: symbol.Name,
			Position:   ToTypesPosition(impl),
			NodeType:   impl.NodeType,
			Children:   make([]*types.GraphNode, 0),
		}
		nodes = append(nodes, node)
	}

	// 递归处理子节点
	if maxLayer > 1 {
		for _, node := range nodes {
			var childSymbol *Symbol
			err := b.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get(SymKey(node.SymbolName))
				if err != nil {
					return err
				}
				return item.Value(func(val []byte) error {
					var err error
					childSymbol, err = DeserializeSymbol(val)
					return err
				})
			})
			if err != nil {
				continue
			}
			node.Children = b.buildSymbolTree(childSymbol, maxLayer-1)
		}
	}

	return nodes
}

// buildPositionTree 构建位置树
func (b badgerDBGraph) buildPositionTree(doc *Document, startLine, endLine, maxLayer int) []*types.GraphNode {
	var nodes []*types.GraphNode

	// 遍历文件中的符号
	for _, symbolName := range doc.Symbols {
		var symbol *Symbol
		err := b.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(SymKey(symbolName))
			if err != nil {
				return err
			}
			return item.Value(func(val []byte) error {
				var err error
				symbol, err = DeserializeSymbol(val)
				return err
			})
		})
		if err != nil {
			continue
		}

		// 检查符号是否在指定范围内
		for _, def := range symbol.Definitions {
			if def.FilePath == doc.Path {
				pos := ToTypesPosition(def)
				if pos.StartLine >= startLine && pos.EndLine <= endLine {
					node := &types.GraphNode{
						FilePath:   def.FilePath,
						SymbolName: symbol.Name,
						Position:   pos,
						NodeType:   def.NodeType,
						Children:   make([]*types.GraphNode, 0),
					}
					nodes = append(nodes, node)
				}
			}
		}
	}

	// 递归处理子节点
	if maxLayer > 1 {
		for _, node := range nodes {
			var childSymbol *Symbol
			err := b.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get(SymKey(node.SymbolName))
				if err != nil {
					return err
				}
				return item.Value(func(val []byte) error {
					var err error
					childSymbol, err = DeserializeSymbol(val)
					return err
				})
			})
			if err != nil {
				continue
			}
			node.Children = b.buildSymbolTree(childSymbol, maxLayer-1)
		}
	}

	return nodes
}

// Close 关闭数据库连接
func (b badgerDBGraph) Close() error {
	if err := b.db.RunValueLogGC(0.5); err != nil {
		_ = fmt.Errorf("failed to run value log GC, err:%v", err)
	}
	return b.db.Close()
}

// DeleteAll 删除所有数据并执行一次清理
func (b badgerDBGraph) DeleteAll(ctx context.Context) error {
	// 删除所有数据
	if err := b.db.DropAll(); err != nil {
		return err
	}

	// 执行一次压缩
	return b.db.RunValueLogGC(0.1)
}

// Document operations
func (b badgerDBGraph) WriteDocument(ctx context.Context, doc *Document) error {
	docBytes, err := SerializeDocument(doc)
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(DocKey(doc.Path), docBytes)
	})
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
	symbolBytes, err := SerializeSymbol(symbol)
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(SymKey(symbol.Name), symbolBytes)
	})
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
func (b badgerDBGraph) GetPositionsBySymbol(ctx context.Context, symbol string) ([]Occurrence, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var positions []Occurrence
	positions = append(positions, sym.Definitions...)
	positions = append(positions, sym.References...)
	positions = append(positions, sym.Implementations...)
	return positions, nil
}

func (b badgerDBGraph) GetPositionsByFile(ctx context.Context, filePath string) ([]Occurrence, error) {
	doc, err := b.GetDocument(ctx, filePath)
	if err != nil {
		return nil, err
	}
	var positions []Occurrence
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

func (b badgerDBGraph) GetPositionsByRange(ctx context.Context, filePath string, startLine, endLine int) ([]Occurrence, error) {
	positions, err := b.GetPositionsByFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	var filtered []Occurrence
	for _, pos := range positions {
		if len(pos.Range) >= 4 {
			start := int(pos.Range[0]) + 1
			end := int(pos.Range[2]) + 1
			if start >= startLine && end <= endLine {
				filtered = append(filtered, pos)
			}
		}
	}
	return filtered, nil
}

// BuildSymbolTree Tree operations
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
