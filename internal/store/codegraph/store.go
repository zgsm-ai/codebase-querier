package codegraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"google.golang.org/protobuf/proto"
	"strings"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// GraphStore 定义图存储接口
type GraphStore interface {
	// BatchWrite 批量写入接口
	BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error

	// BatchWriteCodeStructures BatchWrite 批量写入接口
	BatchWriteCodeStructures(ctx context.Context, docs []*codegraphpb.CodeDefinition) error

	BatchWriteDefSymbolKeysMap(ctx context.Context, symbolKeysMap map[string]*codegraphpb.KeySet) error

	// QueryRelation 查询接口
	QueryRelations(ctx context.Context, opts *types.RelationRequest) ([]*types.GraphNode, error)

	// QueryDefinitions 查询定义
	QueryDefinitions(ctx context.Context, opts *types.DefinitionRequest) ([]*types.DefinitionNode, error)

	// Close 数据库操作
	Close() error
	DeleteAll(ctx context.Context) error
	Delete(ctx context.Context, files []string) error
	DeleteByCodebase(ctx context.Context, codebaseId int32, codebasePath string) error
	GetIndexSummary(ctx context.Context, codebaseId int32, codebasePath string) (*types.CodeGraphSummary, error)
}

// 键前缀
const (
	docPrefix      = "doc:"       // 代码文件数据前缀
	structPrefix   = "sct:"       // 代码文件定义结构数据前缀
	symPrefix      = "sym:"       // 符号数据前缀
	symIndexPrefix = "sym_index:" // 符号->doc_key数据前缀
)

// DocKey 键生成函数  unix path
func DocKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", docPrefix, utils.ToUnixPath(path)))
}

// StructKey 键生成函数 unix path
func StructKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", structPrefix, utils.ToUnixPath(path)))
}

// SymbolKey 键生成函数 unix path
func SymbolKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", symPrefix, utils.ToUnixPath(path)))
}

// SymbolIndexKey 键生成函数 unix path
func SymbolIndexKey(symbolName string) []byte {
	return []byte(fmt.Sprintf("%s%s", symIndexPrefix, symbolName))
}

func isDocKey(key []byte) bool {
	return strings.HasPrefix(string(key), docPrefix)
}

func isStructKey(key []byte) bool {
	return strings.HasPrefix(string(key), structPrefix)
}
func isSymbolIndexKey(key []byte) bool {
	return strings.HasPrefix(string(key), symIndexPrefix)
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
