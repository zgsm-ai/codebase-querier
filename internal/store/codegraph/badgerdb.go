package codegraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"google.golang.org/protobuf/proto"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "badger.db"

const filePath = "filePath"

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
func (b BadgerDBGraph) BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error {
	wb := b.db.NewWriteBatch()
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
	return wb.Flush()
}

func (b BadgerDBGraph) BatchWriteCodeStructures(ctx context.Context, docs []*codegraphpb.CodeStructure) error {
	wb := b.db.NewWriteBatch()
	// 写入文档
	for _, doc := range docs {
		docBytes, err := SerializeDocument(doc)
		if err != nil {
			return err
		}
		if err := wb.Set(StructKey(doc.Path), docBytes); err != nil {
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
	if opts.FilePath == types.EmptyString {
		return nil, errs.NewMissingParamError(filePath)
	}
	// 1. 获取文档
	var doc *codegraphpb.Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(opts.FilePath))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return DeserializeDocument(val, doc)
		})
	})
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", opts.FilePath)
	}

	var res []*types.GraphNode
	var foundSymbols []*codegraphpb.Symbol

	// Find root symbols based on query options
	if opts.SymbolName != "" {
		foundSymbols = b.querySymbolsByNameAndLine(doc, opts)
	} else {
		foundSymbols = b.querySymbolsByPosition(doc, opts)
	}

	// Check if any root symbols were found
	if len(foundSymbols) == 0 {
		return nil, fmt.Errorf("symbol not found: name %s startLine %d in document %s", opts.SymbolName, opts.StartLine, opts.FilePath)
	}
	// root
	// 找定义节点，以定义节点为根节点进行深度遍历
	for _, s := range foundSymbols {
		relations := s.Relations
		if len(relations) == 0 {
			continue
		}
		for _, r := range relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_DEFINITION {
				// 定义节点，加入root
				res = append(res, &types.GraphNode{
					FilePath:   r.FilePath,
					SymbolName: r.Name,
					Identifier: r.Identifier,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeDefinition),
				})
			}
		}
	}

	// Convert root symbols to GraphNodes and add to result
	//for _, sym := range foundSymbols {
	//	graphNode := b.convertSymbolToGraphNode(doc.Path, sym)
	//	if graphNode != nil {
	//		res = append(res, graphNode)
	//	}
	//}

	// Build the rest of the tree recursively
	// We need to build children for the root nodes found
	for _, rootNode := range res {
		// Pass the corresponding original symbol proto to the recursive function
		b.buildChildrenRecursive(rootNode, opts.MaxLayer)
	}

	return res, nil
}

// convertSymbolToGraphNode converts a codegraphpb.Symbol to a types.GraphNode
func (b BadgerDBGraph) convertSymbolToGraphNode(filePath string, symbol *codegraphpb.Symbol) *types.GraphNode {
	if symbol == nil {
		return nil
	}

	// Determine NodeType based on Symbol Role
	nodeType := types.NodeTypeUnknown
	switch symbol.Role {
	case codegraphpb.RelationType_RELATION_DEFINITION:
		nodeType = types.NodeTypeDefinition
	case codegraphpb.RelationType_RELATION_REFERENCE:
		nodeType = types.NodeTypeReference
	// Add other cases if needed, e.g., implementation, type definition
	case codegraphpb.RelationType_RELATION_IMPLEMENTATION:
		nodeType = types.NodeTypeImplementation
	case codegraphpb.RelationType_RELATION_TYPE_DEFINITION:
		nodeType = types.NodeTypeDefinition // Map type definition relation to NodeTypeDefinition
	default:
		nodeType = types.NodeTypeReference // Defaulting to reference or unknown
	}

	graphNode := &types.GraphNode{
		FilePath:   filePath,
		SymbolName: symbol.Identifier,
		Position:   types.ToPosition(symbol.Range), // Use the helper function from types
		Content:    symbol.Content,
		NodeType:   string(nodeType),
		Children:   []*types.GraphNode{}, // Initialize children slice
		Parent:     nil,                  // Parent will be set by the caller when building the tree
	}

	return graphNode
}

// findSymbolInDoc 在文档中查找指定名称的符号
func (b BadgerDBGraph) findSymbolInDoc(doc *codegraphpb.Document, identifier string) *codegraphpb.Symbol {
	// TODO 使用Position 二分查找
	for _, sym := range doc.Symbols {
		if sym.Identifier == identifier {
			return sym
		}
	}
	return nil
}

// findSymbolInDoc 在结构文件中符号在该范围的symbol
func (b BadgerDBGraph) findSymbolInStruct(doc *codegraphpb.CodeStructure, position types.Position) *codegraphpb.Symbol {
	line := position.StartLine
	column := position.StartColumn
	if line == 0 && column == 0 {
		logx.Errorf("findSymbolInStruct invalid position :%v, length less than 2", position)
		return nil
	}
	// TODO 二分查找
	var foundDef *codegraphpb.Definition
	for _, d := range doc.Definitions {
		parsedRange, err := scip.NewRange(d.Range)
		if err != nil {
			logx.Errorf("findSymbolInStruct parse range error:%w", err)
			return nil
		}
		if parsedRange.Contains(scip.Position{Line: int32(line), Character: int32(column)}) {
			foundDef = d
			break
		}
	}

	if foundDef == nil {
		logx.Debugf("findSymbolInStruct definition not found by position %v in doc: %s", position, doc.Path)
		return nil
	}
	// 找到了def, 下一步根据def 的path、 range，找 symbol
	var document codegraphpb.Document
	err := b.findDocument(DocKey(doc.Path), &document)
	if err != nil {
		logx.Debugf("findSymbolInStruct document not found by path %v in doc: %s", position, doc.Path)
		return nil
	}
	return b.findSymbolInDocByRange(&document, foundDef.Range)

}

// buildChildrenForNodes (Deprecated) - Replaced by buildChildrenRecursive
func (b BadgerDBGraph) buildChildrenForNodes(nodes []*types.GraphNode, maxLayer int) {
	// This function is no longer used.
}

// buildChildrenRecursive recursively builds the child nodes for a given GraphNode and its corresponding Symbol.
// node: The current GraphNode to build children for.
// symbol: The codegraphpb.Symbol corresponding to the node, containing the Relations.
// maxLayer: Maximum depth to build the tree from this node downwards.
func (b BadgerDBGraph) buildChildrenRecursive(node *types.GraphNode, maxLayer int) {
	if maxLayer <= 0 || node == nil {
		return
	}
	maxLayer-- // 防止死递归

	symbolPath := node.FilePath
	identifier := node.Identifier
	position := node.Position
	// 根据path和position，定义到 symbol，从而找到它的relation
	var document codegraphpb.Document
	err := b.findDocument(DocKey(symbolPath), &document)
	if err != nil {
		return
	}
	symbol := b.findSymbolInDoc(&document, identifier)
	if symbol == nil {
		return
	}

	var children []*types.GraphNode

	// 找到symbol 的relation. 只有定义的symbol 有reference，
	if len(symbol.Relations) > 0 {
		for _, r := range symbol.Relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_REFERENCE {
				// 引用节点，加入node的children
				children = append(children, &types.GraphNode{
					FilePath:   r.FilePath,
					SymbolName: r.Name,
					Identifier: r.Identifier,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeDefinition),
				})
			}
		}
	}

	if len(children) == 0 {
		// 如果references 为空，说明当前 node 是引用节点， 找到它属于哪个函数/类/结构体，再找它的definition节点，再找引用
		var structFile codegraphpb.CodeStructure
		err = b.findDocument(StructKey(symbolPath), &structFile)
		if err != nil {
			return
		}
		// 定义symbol
		foundDefSymbol := b.findSymbolInStruct(&structFile, position)
		children = append(children, &types.GraphNode{
			FilePath:   foundDefSymbol.Path,
			SymbolName: foundDefSymbol.Name,
			Identifier: foundDefSymbol.Identifier,
			Position:   types.ToPosition(foundDefSymbol.Range),
			NodeType:   string(types.NodeTypeDefinition),
		})
	}
	//当前节点的子
	node.Children = children
	// 继续递归
	for _, ch := range children {
		b.buildChildrenRecursive(ch, maxLayer)
	}

}

func (b BadgerDBGraph) findDocument(key []byte, message proto.Message) error {
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil // Not finding the doc is not an error we should stop for
			}
			return err // Propagate other errors
		}
		return item.Value(func(val []byte) error {
			err = DeserializeDocument(val, message)
			if err != nil {
				return err
			}
			return nil
		})
	})
	return err
}

// func (b BadgerDBGraph) findDocument(relation any, relatedDoc *codegraphpb.Document,
//
//		relatedSymbol *codegraphpb.Symbol) (error, *codegraphpb.Symbol) {
//		err := b.db.View(func(txn *badger.Txn) error {
//			item, err := txn.Get(DocKey(relation.FilePath))
//			if err != nil {
//				if errors.Is(err, badger.ErrKeyNotFound) {
//					return nil // Not finding the doc is not an error we should stop for
//				}
//				return err // Propagate other errors
//			}
//			return item.Value(func(val []byte) error {
//				err = DeserializeDocument(val, relatedDoc)
//				if err != nil {
//					return err
//				}
//				// Document found, now try to find the specific symbol within it
//				relatedSymbol = b.findSymbolInDoc(relatedDoc, relation.Name)
//				// If relatedSymbol is still nil after finding the doc, that's unexpected but handled below
//				return nil
//			})
//		})
//		return err, relatedSymbol
//	}
//
// Helper to map codegraphpb.RelationType to types.NodeType (string)
func getGraphNodeTypeFromRelationType(relationType codegraphpb.RelationType) string {
	switch relationType {
	case codegraphpb.RelationType_RELATION_DEFINITION:
		return string(types.NodeTypeDefinition)
	case codegraphpb.RelationType_RELATION_TYPE_DEFINITION:
		return string(types.NodeTypeDefinition) // Map type definition relation to NodeTypeDefinition
	case codegraphpb.RelationType_RELATION_IMPLEMENTATION:
		return string(types.NodeTypeImplementation)
	case codegraphpb.RelationType_RELATION_REFERENCE:
		return string(types.NodeTypeReference)
	default:
		return string(types.NodeTypeUnknown)
	}
}

// querySymbolsByNameAndLine 通过 symbolName + startLine
func (b BadgerDBGraph) querySymbolsByNameAndLine(doc *codegraphpb.Document, opts *types.RelationQueryOptions) []*codegraphpb.Symbol {
	var nodes []*codegraphpb.Symbol
	queryName := opts.SymbolName
	// 根据名字和 行号， 找到symbol
	for _, s := range doc.Symbols {
		// symbol 名字 模糊匹配
		if strings.Contains(s.Identifier, queryName) {
			symbolRange := s.Range
			if symbolRange != nil && len(symbolRange) > 0 {
				if symbolRange[0] == int32(opts.StartLine-1) {
					nodes = append(nodes, s)
				}
			}
		}
	}
	return nodes
}

// querySymbolsByPosition 按位置查询 occurrence
func (b BadgerDBGraph) querySymbolsByPosition(doc *codegraphpb.Document, opts *types.RelationQueryOptions) []*codegraphpb.Symbol {
	var nodes []*codegraphpb.Symbol
	scipPosition, err := toScipPosition([]int32{int32(opts.StartLine), int32(opts.StartColumn)})
	if err != nil {
		logx.Errorf("toScipPosition error: %v", err)
		return nodes
	}
	for _, s := range doc.Symbols {
		// symbol 名字 模糊匹配
		if len(s.Range) == 0 {
			continue
		}
		sRange, err := scip.NewRange(s.Range)
		if err != nil {
			logx.Errorf("parse doc range error: %v, range:%v", err, s.Range)
			continue
		}
		if sRange.Contains(scipPosition) {
			nodes = append(nodes, s)
		}

	}
	return nodes
}

// Close 关闭数据库连接
func (b BadgerDBGraph) Close() error {
	if err := b.db.RunValueLogGC(0.5); err != nil {
		_ = fmt.Errorf("failed to run value log GC, err:%v", err)
	}
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

func (b BadgerDBGraph) DB() *badger.DB {
	return b.db
}

// TODO 这个symbol 得和scip统一，要找到 name 的position
func (b BadgerDBGraph) findSymbolInDocByRange(document *codegraphpb.Document, symbolRange []int32) *codegraphpb.Symbol {
	//TODO 二分查找
	for _, s := range document.Symbols {
		// s
		// 开始行
		if len(s.Range) < 2 {
			logx.Debugf("findSymbolInDocByRange invalid range in doc:%s, less than 2: %v", s.Identifier, s.Range)
			continue
		}
		// 开始行、列一致
		if s.Range[0] == symbolRange[0] && s.Range[1] == symbolRange[1] {
			return s
		}
	}
	return nil
}
