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
	// BatchWrite 批量写入接口
	BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error

	// BatchWriteCodeStructures BatchWrite 批量写入接口
	BatchWriteCodeStructures(ctx context.Context, docs []*codegraphpb.CodeStructure) error

	// Query 查询接口
	Query(ctx context.Context, opts *types.RelationQueryOptions) ([]*types.GraphNode, error)

	// Close 数据库操作
	Close() error
	DeleteAll(ctx context.Context) error
}

// 键前缀
const (
	DocPrefix    = "doc:" // 文档数据前缀
	StructPrefix = "sct:" // 结构数据前缀
	SymPrefix    = "sym:" // 符号数据前缀
)

// DocKey 键生成函数
func DocKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", DocPrefix, path))
}

// StructKey 键生成函数
func StructKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", StructPrefix, path))
}

// SerializeDocument 序列化函数
func SerializeDocument(doc proto.Message) ([]byte, error) {
	return proto.Marshal(doc)
}

func DeserializeDocument(data []byte, doc proto.Message) error {
	if err := proto.Unmarshal(data, doc); err != nil {
		return err
	}
	return nil
}

// toScipPosition 辅助函数：将 ranges 转换为 scip.Position
func toScipPosition(position []int32) (scip.Position, error) {
	if len(position) < 2 {
		return scip.Position{}, errors.New("invalid position params")
	}
	return scip.Position{Line: position[0], Character: position[1]}, nil
}
