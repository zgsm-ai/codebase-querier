package codegraph

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/sourcegraph/scip/bindings/go/scip"
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

	badgerOpts = badgerOpts.WithLoggingLevel(badger.WARNING)

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
	if opts.SymbolName != "" {
		nodes = b.queryOccurrencesForSymbol(ctx, doc, opts)
	} else {
		nodes = b.queryOccurrencesForPosition(ctx, doc, opts)
	}

	b.buildChildrenForNodes(nodes, opts.MaxLayer)
	return nodes, nil
}

// buildChildrenForNodes 递归构建子节点
func (b BadgerDBGraph) buildChildrenForNodes(nodes []*types.GraphNode, maxLayer int) {
	if maxLayer <= 1 {
		return
	}
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
		if err != nil || childSymbol == nil {
			continue
		}
		node.Children = b.buildSymbolTree(childSymbol, maxLayer-1)
	}
}

// queryOccurrencesForSymbol 按 symbol 查询 occurrence
func (b BadgerDBGraph) queryOccurrencesForSymbol(ctx context.Context, doc *Document, opts *types.RelationQueryOptions) []*types.GraphNode {
	var nodes []*types.GraphNode
	for _, symbolInDoc := range doc.Symbols {
		if symbolInDoc.Name != opts.SymbolName {
			continue
		}
		symbol := b.getSymbolByName(ctx, symbolInDoc.Name)
		if symbol == nil {
			continue
		}
		nodes = append(nodes, b.handleOccurrencesForQuery(doc, symbol, opts)...)
	}
	return nodes
}

// queryOccurrencesForPosition 按位置查询 occurrence
func (b BadgerDBGraph) queryOccurrencesForPosition(ctx context.Context, doc *Document, opts *types.RelationQueryOptions) []*types.GraphNode {
	var nodes []*types.GraphNode
	for _, symbolInDoc := range doc.Symbols {
		symbol := b.getSymbolByName(ctx, symbolInDoc.Name)
		if symbol == nil {
			continue
		}
		nodes = append(nodes, b.handleOccurrencesForQuery(doc, symbol, opts)...)
	}
	return nodes
}

// handleOccurrencesForQuery 处理 occurrence 并构建 GraphNode
func (b BadgerDBGraph) handleOccurrencesForQuery(doc *Document, symbol *Symbol, opts *types.RelationQueryOptions) []*types.GraphNode {
	var nodes []*types.GraphNode
	for nodeType, occs := range symbol.Occurrences {
		if opts.StartLine > 0 && opts.StartColumn > 0 && opts.EndLine > 0 && opts.EndColumn > 0 {
			targetRange := []int32{
				int32(opts.StartLine - 1), int32(opts.StartColumn - 1),
				int32(opts.EndLine - 1), int32(opts.EndColumn - 1),
			}
			occ := findOccurrenceByRange(occs, targetRange)
			if occ != nil && occ.FilePath == doc.Path {
				node := &types.GraphNode{
					FilePath:   occ.FilePath,
					SymbolName: symbol.Name,
					Position:   ToTypesPosition(*occ),
					NodeType:   nodeType,
					Children:   make([]*types.GraphNode, 0),
				}
				nodes = append(nodes, node)
			}
			continue
		}
		for _, occ := range occs {
			if occ.FilePath != doc.Path {
				continue
			}
			if matchOccurrence(occ, opts) {
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
	}
	return nodes
}

// getSymbolByName 获取 symbol
func (b BadgerDBGraph) getSymbolByName(ctx context.Context, name string) *Symbol {
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
	if err != nil || symbol == nil {
		return nil
	}
	return symbol
}

// occurrence 匹配辅助函数
func matchOccurrence(occ Occurrence, opts *types.RelationQueryOptions) bool {
	if len(occ.Range) < 3 {
		return false
	}
	occR := scip.NewRangeUnchecked(occ.Range)
	// 1-based to 0-based
	startLine := opts.StartLine - 1
	startCol := opts.StartColumn - 1
	endLine := opts.EndLine - 1
	endCol := opts.EndColumn - 1

	// 优先级1：精确区间匹配
	if opts.StartLine > 0 && opts.StartColumn > 0 && opts.EndLine > 0 && opts.EndColumn > 0 {
		queryR := scip.Range{
			Start: scip.Position{Line: int32(startLine), Character: int32(startCol)},
			End:   scip.Position{Line: int32(endLine), Character: int32(endCol)},
		}
		return occR.CompareStrict(queryR) == 0
	}
	// 优先级2：点在区间内
	if opts.StartLine > 0 && opts.StartColumn > 0 {
		pos := scip.Position{Line: int32(startLine), Character: int32(startCol)}
		return occR.Contains(pos)
	}
	// 优先级3：行号范围重叠
	if opts.StartLine > 0 && opts.EndLine > 0 {
		queryR := scip.Range{
			Start: scip.Position{Line: int32(startLine), Character: 0},
			End:   scip.Position{Line: int32(endLine), Character: 0},
		}
		return occR.Intersects(queryR)
	}
	return false
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
	for _, occs := range symbol.Occurrences {
		sortOccurrencesByRange(occs)
	}
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
		NodeType:   types.SymbolRoleDefinition,
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
	for _, occ := range sym.Occurrences[types.SymbolRoleReference] {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   occ.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(occ),
			NodeType:   types.SymbolRoleReference,
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
	for _, occ := range sym.Occurrences[types.SymbolRoleDefinition] {
		nodes = append(nodes, &types.GraphNode{
			FilePath:   occ.FilePath,
			SymbolName: symbol,
			Position:   ToTypesPosition(occ),
			NodeType:   types.SymbolRoleDefinition,
		})
	}
	return nodes, nil
}

// DB returns the underlying BadgerDB instance
func (b BadgerDBGraph) DB() *badger.DB {
	return b.db
}

// sortOccurrencesByRange 对 occurrence 按 range 排序
func sortOccurrencesByRange(occurrences []Occurrence) {
	sort.Slice(occurrences, func(i, j int) bool {
		a, b := occurrences[i].Range, occurrences[j].Range
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
}

// findOccurrenceByRange 二分查找 occurrence
func findOccurrenceByRange(occurrences []Occurrence, target []int32) *Occurrence {
	low, high := 0, len(occurrences)-1
	for low <= high {
		mid := (low + high) / 2
		cmp := compareRange(occurrences[mid].Range, target)
		if cmp == 0 {
			return &occurrences[mid]
		} else if cmp < 0 {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return nil
}

func compareRange(a, b []int32) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return len(a) - len(b)
}
