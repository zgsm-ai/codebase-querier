package codegraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"google.golang.org/protobuf/proto"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// GraphStore 定义图存储接口
type GraphStore interface {
	// 批量写入接口
	BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error

	// 查询接口
	Query(ctx context.Context, opts *types.RelationQueryOptions) ([]*types.GraphNode, error)

	// 数据库操作
	Close() error
	DeleteAll(ctx context.Context) error
}

// 键前缀
const (
	DocPrefix = "doc:" // 文档数据前缀
	SymPrefix = "sym:" // 符号数据前缀
)

// 键生成函数
func DocKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", DocPrefix, path))
}

// SerializeDocument 序列化函数
func SerializeDocument(doc *codegraphpb.Document) ([]byte, error) {
	return proto.Marshal(doc)
}

func DeserializeDocument(data []byte) (*codegraphpb.Document, error) {
	var doc codegraphpb.Document
	if err := proto.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// toScipPosition 辅助函数：将 ranges 转换为 scip.Position
func toScipPosition(position []int32) (scip.Position, error) {
	if len(position) < 2 {
		return scip.Position{}, errors.New("invalid position params")
	}
	return scip.Position{Line: position[0], Character: position[1]}, nil
}
