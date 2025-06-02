package codegraph

import (
	"context"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "codegraph.db"

type badgerDBGraph struct {
	path string
	db   *badger.DB
}

func (b badgerDBGraph) Close() error {
	return b.db.Close()
}

func NewBadgerDBGraph(opts ...GraphOption) (GraphStore, error) {
	b := &badgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}
	// Open database
	badgerDB, err := badger.Open(badger.DefaultOptions(filepath.Join(b.path, dbName)))
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

// Save saves the SCIP index data to BadgerDB
func (b badgerDBGraph) Save(ctx context.Context, codebaseId int64, codebasePath string, nodes []*types.GraphNode) error {
	return b.db.Update(func(txn *badger.Txn) error {
		// 按文档路径分组节点
		nodesByPath := make(map[string][]*types.GraphNode)
		for _, node := range nodes {
			nodesByPath[node.FilePath] = append(nodesByPath[node.FilePath], node)
		}

		// 处理每个文档的节点
		for path, docNodes := range nodesByPath {
			// 1. 保存文档内容
			docKey := DocumentKey{
				DocumentPath: path,
			}
			docValue := DocumentValue{
				Content:       "", // 文档内容暂不存储
				SchemaVersion: 1,
			}
			docBytes, err := docValue.Encode()
			if err != nil {
				return err
			}
			if err := txn.Set(docKey.Encode(), docBytes); err != nil {
				return err
			}

			// 2. 保存符号信息
			for _, node := range docNodes {
				symKey := SymbolKey{
					SymbolName: node.SymbolName,
				}
				item, err := txn.Get(symKey.Encode())
				var symValue SymbolValue
				if err == nil {
					// 更新现有符号
					if err := item.Value(func(val []byte) error {
						return symValue.Decode(val)
					}); err != nil {
						return err
					}
				} else if err == badger.ErrKeyNotFound {
					// 创建新符号
					symValue = SymbolValue{
						Definitions:     make([]PositionInfo, 0),
						References:      make([]PositionInfo, 0),
						Implementations: make([]PositionInfo, 0),
						TypeDefinitions: make([]PositionInfo, 0),
					}
				} else {
					return err
				}

				// 创建位置信息
				posInfo := PositionInfo{
					FilePath: node.FilePath,
					Position: node.Position,
					NodeType: node.NodeType,
					Content:  node.Content,
				}

				// 根据节点类型添加到相应列表
				switch node.NodeType {
				case types.NodeTypeDefinition:
					symValue.Definitions = append(symValue.Definitions, posInfo)
					symValue.TypeDefinitions = append(symValue.TypeDefinitions, posInfo)
				case types.NodeTypeReference:
					symValue.References = append(symValue.References, posInfo)
				case types.NodeTypeImplementation:
					symValue.Implementations = append(symValue.Implementations, posInfo)
				}

				symBytes, err := symValue.Encode()
				if err != nil {
					return err
				}
				if err := txn.Set(symKey.Encode(), symBytes); err != nil {
					return err
				}

				// 3. 保存位置索引
				if node.Position.StartLine > 0 {
					posKey := PositionKey{
						DocumentPath: node.FilePath,
						StartLine:    node.Position.StartLine,
						StartColumn:  node.Position.StartColumn,
					}
					posValue := PositionValue{
						SymbolName: node.SymbolName,
						NodeType:   node.NodeType,
					}
					posBytes, err := posValue.Encode()
					if err != nil {
						return err
					}
					if err := txn.Set(posKey.Encode(), posBytes); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// Query queries the code graph based on the given options
func (b badgerDBGraph) Query(ctx context.Context, req *types.RelationQueryOptions) ([]*types.GraphNode, error) {
	var results []*types.GraphNode

	err := b.db.View(func(txn *badger.Txn) error {
		// If symbol name is provided, query by symbol
		if req.SymbolName != "" {
			return b.queryBySymbol(txn, req, &results)
		}

		// If file path and position are provided, query by position
		if req.FilePath != "" && req.StartLine > 0 {
			return b.queryByPosition(txn, req, &results)
		}

		return nil
	})

	return results, err
}

// queryBySymbol queries the graph by symbol name
func (b badgerDBGraph) queryBySymbol(txn *badger.Txn, req *types.RelationQueryOptions, results *[]*types.GraphNode) error {
	symKey := SymbolKey{
		SymbolName: req.SymbolName,
	}
	item, err := txn.Get(symKey.Encode())
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	}

	var symValue SymbolValue
	if err := item.Value(func(val []byte) error {
		return symValue.Decode(val)
	}); err != nil {
		return err
	}

	// Create nodes for definitions
	for _, def := range symValue.Definitions {
		node := b.createGraphNode(req.SymbolName, def, types.NodeTypeDefinition)
		*results = append(*results, node)
	}

	// Add references as children
	for _, ref := range symValue.References {
		child := b.createGraphNode(req.SymbolName, ref, types.NodeTypeReference)
		if len(*results) > 0 {
			child.Parent = (*results)[0]
			(*results)[0].Children = append((*results)[0].Children, child)
		}
	}

	// Add implementations if requested
	if req.MaxLayer > 1 {
		for _, impl := range symValue.Implementations {
			child := b.createGraphNode(req.SymbolName, impl, types.NodeTypeImplementation)
			if len(*results) > 0 {
				child.Parent = (*results)[0]
				(*results)[0].Children = append((*results)[0].Children, child)
			}
		}
	}

	return nil
}

// queryByPosition queries the graph by file position
func (b badgerDBGraph) queryByPosition(txn *badger.Txn, req *types.RelationQueryOptions, results *[]*types.GraphNode) error {
	posKey := PositionKey{
		DocumentPath: req.FilePath,
		StartLine:    req.StartLine,
		StartColumn:  req.StartColumn,
	}
	item, err := txn.Get(posKey.Encode())
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	}

	var posValue PositionValue
	if err := item.Value(func(val []byte) error {
		return posValue.Decode(val)
	}); err != nil {
		return err
	}

	// Query symbol information
	return b.queryBySymbol(txn, &types.RelationQueryOptions{
		SymbolName: posValue.SymbolName,
		MaxLayer:   req.MaxLayer,
	}, results)
}

// createGraphNode creates a GraphNode from PositionInfo
func (b badgerDBGraph) createGraphNode(symbolName string, posInfo PositionInfo, nodeType types.NodeType) *types.GraphNode {
	node := &types.GraphNode{
		FilePath:   posInfo.FilePath,
		SymbolName: symbolName,
		Position:   posInfo.Position,
		NodeType:   nodeType,
		// SymbolKind is not supported in GraphNode yet
	}
	if posInfo.Content != "" {
		node.Content = posInfo.Content
	}
	return node
}

// DeleteAll deletes all data for a codebase
func (b badgerDBGraph) DeleteAll(ctx context.Context, codebaseId int64, codebasePath string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		// Delete all keys
		prefixes := []string{
			prefixDocument,
			prefixSymbol,
			prefixPosition,
		}

		for _, prefix := range prefixes {
			iter := txn.NewIterator(badger.DefaultIteratorOptions)
			defer iter.Close()

			for iter.Seek([]byte(prefix)); iter.ValidForPrefix([]byte(prefix)); iter.Next() {
				if err := txn.Delete(iter.Item().Key()); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
