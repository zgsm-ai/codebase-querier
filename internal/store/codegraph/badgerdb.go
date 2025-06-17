package codegraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"path/filepath"
	"strings"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/badger/v4"
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

// QueryRelation 实现查询接口
func (b BadgerDBGraph) QueryRelation(ctx context.Context, opts *types.RelationRequest) ([]*types.GraphNode, error) {
	startTime := time.Now()
	defer func() {
		logx.WithContext(ctx).Infof("QueryRelation execution time: %d ms", time.Since(startTime).Milliseconds())
	}()

	// 1. 获取文档
	var doc codegraphpb.Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(opts.FilePath))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("document not found: %s", opts.FilePath)
			}
			return fmt.Errorf("failed to get document: %w", err)
		}
		return item.Value(func(val []byte) error {
			return DeserializeDocument(val, &doc)
		})
	})
	if err != nil {
		logx.WithContext(ctx).Errorf("Failed to get document: %v", err)
		return nil, err
	}

	var res []*types.GraphNode
	var foundSymbols []*codegraphpb.Symbol

	// Find root symbols based on query options
	if opts.SymbolName != "" {
		foundSymbols = b.querySymbolsByNameAndLine(&doc, opts)
		logx.WithContext(ctx).Debugf("Found %d symbols by name and line", len(foundSymbols))
	} else {
		foundSymbols = b.querySymbolsByPosition(&doc, opts)
		logx.WithContext(ctx).Debugf("Found %d symbols by position", len(foundSymbols))
	}

	// Check if any root symbols were found
	if len(foundSymbols) == 0 {
		err := fmt.Errorf("symbol not found: name %s startLine %d in document %s", opts.SymbolName, opts.StartLine, opts.FilePath)
		logx.WithContext(ctx).Errorf("%v", err)
		return nil, err
	}

	// root
	// 找定义节点，以定义节点为根节点进行深度遍历
	for _, s := range foundSymbols {
		// 如果当前Symbol 就是定义，加入
		if s.Role == codegraphpb.RelationType_RELATION_DEFINITION {
			res = append(res, &types.GraphNode{
				FilePath:   doc.Path,
				SymbolName: s.Name,
				Identifier: s.Identifier,
				Position:   types.ToPosition(s.Range),
				NodeType:   string(types.NodeTypeDefinition),
			})
			continue
		}
		// 不是定义节点，找它的relation中的定义节点
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

	logx.Debugf("Found %d root nodes", len(res))

	// Build the rest of the tree recursively
	// We need to build children for the root nodes found
	for _, rootNode := range res {
		// Pass the corresponding original symbol proto to the recursive function
		b.buildChildrenRecursive(rootNode, opts.MaxLayer)
	}
	return res, nil
}

// QueryDefinition 查询定义
func (b BadgerDBGraph) QueryDefinition(ctx context.Context, opts *types.DefinitionRequest) ([]*types.DefinitionNode, error) {
	startTime := time.Now()
	defer func() {
		logx.WithContext(ctx).Infof("QueryDefinition execution time: %d ms", time.Since(startTime).Milliseconds())
	}()

	// 1. 获取文档
	var doc codegraphpb.Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(opts.FilePath))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("document not found: %s", opts.FilePath)
			}
			return fmt.Errorf("failed to get document: %w", err)
		}
		return item.Value(func(val []byte) error {
			return DeserializeDocument(val, &doc)
		})
	})
	if err != nil {
		logx.WithContext(ctx).Errorf("Failed to get document: %v", err)
		return nil, err
	}
	queryStartLine := int32(opts.StartLine - 1)
	queryEndLine := int32(opts.EndLine - 1)
	var res []*types.DefinitionNode
	foundSymbols := b.findSymbolInDocByLineRange(&doc, queryStartLine, queryEndLine)
	// 去重, 如果range 出现过，则剔除
	existedDefs := make(map[string]bool)
	// 遍历，封装结果，取定义
	for _, s := range foundSymbols {
		// 本身是定义
		if s.Role == codegraphpb.RelationType_RELATION_DEFINITION {
			// 去掉本范围内的定义
			if len(s.Range) > 0 && isInLinesRange(s.Range[0], queryStartLine, queryEndLine) {
				continue
			}
			res = append(res, &types.DefinitionNode{
				FilePath: s.Path,
				Name:     s.Name,
				Position: types.ToPosition(s.Range),
			})

			continue
		} else if s.Role == codegraphpb.RelationType_RELATION_REFERENCE { // 引用
			for _, r := range s.Relations {
				// 引用的 relation 是定义
				if r.RelationType == codegraphpb.RelationType_RELATION_DEFINITION {
					// 如果已经访问过，就跳过
					if isSymbolExists(r.FilePath, r.Range, existedDefs) {
						continue
					}
					existedDefs[symbolMapKey(r.FilePath, r.Range)] = true
					// 引用节点，加入node的children
					res = append(res, &types.DefinitionNode{
						FilePath: r.FilePath,
						Name:     r.Name, //TODO r.Name 有时候是identifier，未处理
						Position: types.ToPosition(r.Range),
					})
				}
			}
		} else {
			logx.Errorf("QueryDefinition: unsupported symbol %s type %s ", s.Identifier, string(s.Role))
		}
	}
	//TODO  根据struct_doc 重新封装它的 position，变为范围。doc的range是单行。
	if err := b.refillDefinitionRange(res); err != nil {
		logx.Errorf("QueryDefinition:  refill definition range err:%v", utils.TruncateError(err))
	}

	return res, nil
}

func isInLinesRange(current, start, end int32) bool {
	return current >= start-1 && current <= end-1
}

func isSymbolExists(filePath string, ranges []int32, state map[string]bool) bool {
	key := symbolMapKey(filePath, ranges)
	_, ok := state[key]
	return ok
}
func symbolMapKey(filePath string, ranges []int32) string {
	return filePath + "-" + utils.SliceToString(ranges)
}

// findSymbolInDocByIdentifier 在文档中查找指定名称的符号
func (b BadgerDBGraph) findSymbolInDocByIdentifier(doc *codegraphpb.Document, identifier string) *codegraphpb.Symbol {
	// TODO 使用Position 二分查找
	for _, sym := range doc.Symbols {
		if sym.Identifier == identifier {
			return sym
		}
	}
	return nil
}

// findSymbolInDocByIdentifier 在结构文件中符号在该范围的symbol
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

// buildChildrenRecursive recursively builds the child nodes for a given GraphNode and its corresponding Symbol.
func (b BadgerDBGraph) buildChildrenRecursive(node *types.GraphNode, maxLayer int) {
	if maxLayer <= 0 || node == nil {
		logx.Debugf("buildChildrenRecursive stopped: maxLayer=%d, node is nil=%v", maxLayer, node == nil)
		return
	}
	maxLayer-- // 防止死递归

	startTime := time.Now()
	defer func() {
		logx.Debugf("buildChildrenRecursive for node %s took %v", node.Identifier, time.Since(startTime))
	}()

	symbolPath := node.FilePath
	identifier := node.Identifier
	position := node.Position

	// 根据path和position，定义到 symbol，从而找到它的relation
	var document codegraphpb.Document
	err := b.findDocument(DocKey(symbolPath), &document)
	if err != nil {
		logx.Errorf("Failed to find document for path %s: %v", symbolPath, err)
		return
	}

	symbol := b.findSymbolInDocByIdentifier(&document, identifier)
	if symbol == nil {
		logx.Debugf("Symbol not found in document: path=%s, identifier=%s", symbolPath, identifier)
		return
	}

	var children []*types.GraphNode

	// 找到symbol 的relation. 只有定义的symbol 有reference，引用节点的relation是定义节点
	if len(symbol.Relations) > 0 {
		for _, r := range symbol.Relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_REFERENCE {
				// 引用节点，加入node的children
				children = append(children, &types.GraphNode{
					FilePath:   r.FilePath,
					SymbolName: r.Name,
					Identifier: r.Identifier,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeReference),
				})
			}
		}
		logx.Debugf("Found %d reference relations for symbol %s", len(children), identifier)
	}

	if len(children) == 0 {
		// 如果references 为空，说明当前 node 是引用节点， 找到它属于哪个函数/类/结构体，再找它的definition节点，再找引用
		var structFile codegraphpb.CodeStructure
		err = b.findDocument(StructKey(symbolPath), &structFile)
		if err != nil {
			logx.Errorf("Failed to find struct file for path %s: %v", symbolPath, err)
			return
		}
		if structFile.Path == types.EmptyString { // 没找到
			logx.Debugf("Cannot find symbol %s struct file by symbol path %s", identifier, symbolPath)
		}
		// 定义symbol
		foundDefSymbol := b.findSymbolInStruct(&structFile, position)
		if foundDefSymbol == nil {
			logx.Debugf("No definition found for symbol at position %+v", position)
			return
		}
		children = append(children, &types.GraphNode{
			FilePath:   foundDefSymbol.Path,
			SymbolName: foundDefSymbol.Name,
			Identifier: foundDefSymbol.Identifier,
			Position:   types.ToPosition(foundDefSymbol.Range),
			NodeType:   string(types.NodeTypeDefinition),
		})
		logx.Debugf("Found definition for reference node: %s", foundDefSymbol.Identifier)
	}

	//当前节点的子
	node.Children = children

	// 继续递归
	for _, ch := range children {
		b.buildChildrenRecursive(ch, maxLayer)
	}
}

// findDocument 查找并反序列化文档
func (b BadgerDBGraph) findDocument(key []byte, message proto.Message) error {
	startTime := time.Now()
	defer func() {
		logx.Debugf("findDocument took %v", time.Since(startTime))
	}()

	if len(key) == 0 {
		return fmt.Errorf("invalid input: key is empty")
	}
	if message == nil {
		return fmt.Errorf("invalid input: message is nil")
	}

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil // Not finding the doc is not an error we should stop for
			}
			return fmt.Errorf("failed to get document: %w", err)
		}
		return item.Value(func(val []byte) error {
			if len(val) == 0 {
				return fmt.Errorf("empty document value")
			}
			if err := DeserializeDocument(val, message); err != nil {
				return fmt.Errorf("failed to deserialize document: %w", err)
			}
			return nil
		})
	})

	if err != nil {
		logx.Errorf("findDocument error for key %s: %v", string(key), err)
	}
	return err
}

// querySymbolsByNameAndLine 通过 symbolName + startLine
func (b BadgerDBGraph) querySymbolsByNameAndLine(doc *codegraphpb.Document, opts *types.RelationRequest) []*codegraphpb.Symbol {
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
func (b BadgerDBGraph) querySymbolsByPosition(doc *codegraphpb.Document, opts *types.RelationRequest) []*codegraphpb.Symbol {
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

	// 执行一次gc
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
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] == symbolRange[0] {
			return s
		}
	}
	return nil
}

func (b BadgerDBGraph) Delete(ctx context.Context, files []string) error {
	logx.Debugf("start to delete docs:%v", files)
	if len(files) == 0 {
		return nil
	}
	var docKeys [][]byte
	for _, v := range files {
		if v == types.EmptyString {
			logx.Errorf("Delete docs, file path is empty")
			continue
		}
		docKeys = append(docKeys, DocKey(v))
		docKeys = append(docKeys, StructKey(v))
	}

	err := b.db.DropPrefix(docKeys...)
	logx.Debugf("docs delete end:%v", docKeys)
	return err
}

func (b BadgerDBGraph) findSymbolInDocByLineRange(doc *codegraphpb.Document, startLine int32, endLine int32) []*codegraphpb.Symbol {
	var res []*codegraphpb.Symbol
	for _, s := range doc.Symbols {
		// s
		// 开始行
		if len(s.Range) < 2 {
			logx.Debugf("findSymbolInDocByLineRange invalid range in doc:%s, less than 2: %v", s.Identifier, s.Range)
			continue
		}
		if s.Range[0] > endLine {
			break
		}
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] >= startLine && s.Range[0] <= endLine {
			res = append(res, s)
		}
	}
	return res
}

func (b BadgerDBGraph) refillDefinitionRange(nodes []*types.DefinitionNode) error {
	var errs []error
	for _, n := range nodes {
		line := n.Position.StartLine - 1
		if line < 0 {
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s invalid line :%d, length less than 0", n.FilePath, n.Name, line))
			continue
		}
		var structFile codegraphpb.CodeStructure
		err := b.findDocument(StructKey(n.FilePath), &structFile)
		if err != nil {
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s ,failed to find struct file for path %s: %v", n.FilePath, n.Name, line, err))
			continue
		}
		if structFile.Path == types.EmptyString { // 没找到
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s ,struct file not found", n.FilePath, n.Name))
			continue
		}

		// TODO 二分查找
		var foundDef *codegraphpb.Definition
		for _, d := range structFile.Definitions {
			if len(d.Range) == 0 {
				errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s , struct doc range length is 0", n.FilePath, n.Name))
				continue
			}
			if d.Range[0] == int32(line) {
				foundDef = d
				break
			}
		}
		if foundDef == nil { // 目前发现变量的定义会找不到
			errs = append(errs, fmt.Errorf("refillDefinitionRange definition not found by line %d in doc: %s", line, structFile.Path))
			continue
		}
		n.Position = types.ToPosition(foundDef.Range)
		n.NodeType = foundDef.Type
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
