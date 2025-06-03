package codegraph

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "badger.db"

// BadgerDBGraph implements GraphStore using BadgerDB
type BadgerDBGraph struct {
	basePath string
	db       *badger.DB
	logger   logx.Logger
}

func NewBadgerDBGraph(ctx context.Context, opts ...GraphOption) (GraphStore, error) {
	b := &BadgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}

	// 自定义 BadgerDB 配置
	badgerOpts := badger.DefaultOptions(filepath.Join(b.basePath, dbName))

	// 值日志配置
	badgerOpts.ValueLogFileSize = 1 << 30 // 512 << 20 512MB // 1 << 30 1GB
	badgerOpts.ValueThreshold = 128       // 小于128字节的值直接存储在LSM树中

	// 内存表配置
	badgerOpts.NumMemtables = 2              // 增加内存表数量
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
	b.logger = logx.WithContext(ctx)
	return b, nil
}

type GraphOption func(*BadgerDBGraph)

func WithPath(basePath string) GraphOption {
	return func(b *BadgerDBGraph) {
		b.basePath = basePath
	}
}

// BatchWrite 批量写入文档和符号
func (b BadgerDBGraph) BatchWrite(ctx context.Context, docs []*Document, symbols []*Symbol) error {
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
func (b BadgerDBGraph) Query(ctx context.Context, opts *types.RelationQueryOptions) ([]*types.GraphNode, error) {
	if opts.MaxLayer <= 0 {
		opts.MaxLayer = 1
	}
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
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", opts.FilePath)
	}

	var nodes []*types.GraphNode

	// 2. 根据查询条件构建树
	if opts.SymbolName != "" {
		var fullSymbolName string
		for _, sy := range doc.Symbols {
			if strings.Contains(sy.Name, opts.SymbolName) {
				fullSymbolName = sy.Name
				break
			}
		}
		if fullSymbolName == "" {
			return nil, fmt.Errorf("symbol not found: %s", opts.SymbolName)
		}

		// 按符号名查询
		var symbol *Symbol
		err := b.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(SymKey(fullSymbolName))
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

	return nodes, nil
}

// buildSymbolTree 构建符号树
func (b BadgerDBGraph) buildSymbolTree(symbol *Symbol, maxLayer int) []*types.GraphNode {
	if maxLayer <= 0 {
		return nil
	}

	var nodes []*types.GraphNode

	for nodeType, occs := range symbol.Occurrences {
		for _, occ := range occs {
			node := &types.GraphNode{
				FilePath:   occ.FilePath,
				SymbolName: symbol.Name,
				Position:   ToTypesPosition(occ),
				NodeType:   nodeType,
				Children:   make([]*types.GraphNode, 0),
			}
			nodes = append(nodes, node)
		}
	}

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
func (b BadgerDBGraph) buildPositionTree(doc *Document, startLine, endLine, maxLayer int) []*types.GraphNode {
	var nodes []*types.GraphNode

	for _, symbolInDoc := range doc.Symbols {
		symbolName := symbolInDoc.Name
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

		for nodeType, occs := range symbol.Occurrences {
			for _, occ := range occs {
				if occ.FilePath == doc.Path {
					pos := ToTypesPosition(occ)
					if pos.StartLine >= startLine && pos.EndLine <= endLine {
						node := &types.GraphNode{
							FilePath:   occ.FilePath,
							SymbolName: symbol.Name,
							Position:   pos,
							NodeType:   nodeType,
							Children:   make([]*types.GraphNode, 0),
						}
						nodes = append(nodes, node)
					}
				}
			}
		}
	}

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
func (b BadgerDBGraph) Close() error {
	start := time.Now()
	if err := b.db.RunValueLogGC(0.5); err != nil {
		_ = fmt.Errorf("failed to run value log GC, err:%v", err)
	}
	b.logger.Debugf("badger db %s GC took %d ms", b.basePath, time.Since(start).Milliseconds())
	return b.db.Close()
}

// DeleteAll 删除所有数据并执行一次清理
func (b BadgerDBGraph) DeleteAll(ctx context.Context) error {
	// 删除所有数据
	if err := b.db.DropAll(); err != nil {
		return err
	}

	// 执行一次压缩
	return b.db.RunValueLogGC(0.1)
}

// WriteDocument Document operations
func (b BadgerDBGraph) WriteDocument(ctx context.Context, doc *Document) error {
	docBytes, err := SerializeDocument(doc)
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(DocKey(doc.Path), docBytes)
	})
}

func (b BadgerDBGraph) GetDocument(ctx context.Context, path string) (*Document, error) {
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

func (b BadgerDBGraph) DeleteDocument(ctx context.Context, path string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(DocKey(path))
	})
}

// Symbol operations
func (b BadgerDBGraph) WriteSymbol(ctx context.Context, symbol *Symbol) error {
	symbolBytes, err := SerializeSymbol(symbol)
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(SymKey(symbol.Name), symbolBytes)
	})
}

func (b BadgerDBGraph) GetSymbol(ctx context.Context, name string) (*Symbol, error) {
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

func (b BadgerDBGraph) DeleteSymbol(ctx context.Context, name string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(SymKey(name))
	})
}

// Position operations
func (b BadgerDBGraph) GetPositionsBySymbol(ctx context.Context, symbol string) ([]Occurrence, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var positions []Occurrence
	for _, occs := range sym.Occurrences {
		for _, pos := range occs {
			positions = append(positions, pos)
		}
	}
	return positions, nil
}

func (b BadgerDBGraph) GetPositionsByFile(ctx context.Context, filePath string) ([]Occurrence, error) {
	doc, err := b.GetDocument(ctx, filePath)
	if err != nil {
		return nil, err
	}
	var positions []Occurrence
	for _, symbolInDoc := range doc.Symbols {
		symbolName := symbolInDoc.Name
		sym, err := b.GetSymbol(ctx, symbolName)
		if err != nil {
			continue
		}
		for _, occs := range sym.Occurrences {
			for _, pos := range occs {
				if pos.FilePath == filePath {
					positions = append(positions, pos)
				}
			}
		}
	}
	return positions, nil
}

func (b BadgerDBGraph) GetPositionsByRange(ctx context.Context, filePath string, startLine, endLine int) ([]Occurrence, error) {
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
func (b BadgerDBGraph) BuildSymbolTree(ctx context.Context, symbol string) (*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	root := &types.GraphNode{
		SymbolName: symbol,
		NodeType:   types.NodeTypeDefinition,
	}

	// Add all occurrences as children
	for nodeType, occs := range sym.Occurrences {
		for _, occ := range occs {
			root.Children = append(root.Children, &types.GraphNode{
				FilePath:   occ.FilePath,
				SymbolName: symbol,
				Position:   ToTypesPosition(occ),
				NodeType:   nodeType,
			})
		}
	}

	return root, nil
}

func (b BadgerDBGraph) GetSymbolReferences(ctx context.Context, symbol string) ([]*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var nodes []*types.GraphNode
	for _, occ := range sym.Occurrences[types.NodeTypeReference] {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   occ.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(occ),
			NodeType:   types.NodeTypeReference,
		})
	}
	return nodes, nil
}

func (b BadgerDBGraph) GetSymbolDefinitions(ctx context.Context, symbol string) ([]*types.GraphNode, error) {
	sym, err := b.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var nodes []*types.GraphNode
	for _, occ := range sym.Occurrences[types.NodeTypeDefinition] {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   occ.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(occ),
			NodeType:   types.NodeTypeDefinition,
		})
	}
	return nodes, nil
}

// DB returns the underlying BadgerDB instance
func (b BadgerDBGraph) DB() *badger.DB {
	return b.db
}
