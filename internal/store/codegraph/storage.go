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

//// Document 表示代码库中的一个文件
//type Document struct {
//	Path    string         `json:"path"`    // 文件路径
//	Symbols []*SymbolInDoc `json:"symbols"` // 文件中的符号列表（含位置信息等）
//}
//
//// SymbolInDoc 表示文档中的一个符号及其位置信息
//// 参考 SCIP Occurrence 结构
//type SymbolInDoc struct {
//	Name  string           `json:"name"`       // 符号名
//	Role  types.SymbolRole `json:"symbolType"` // 节点类型（定义/引用/实现），标识符号在该文件中的角色，
//	Range []int32          `json:"range"`      // [startLine, startCol, endLine, endCol]
//}
//
//// Symbol 表示代码库中的一个符号
//type Symbol struct {
//	Name        string                            `json:"name"`        // 符号名
//	Content     string                            `json:"content"`     // 符号内容（代码片段）
//	Occurrences map[types.SymbolRole][]Occurrence `json:"occurrences"` // 各类型出现位置
//}
//
//// Occurrence 表示符号在文件中的出现位置
//type Occurrence struct {
//	FilePath string           `json:"filePath"`   // 文件路径
//	Range    []int32          `json:"range"`      // 范围信息 [startLine,startCol,endLine,endCol]
//	NodeType types.SymbolRole `json:"symbolRole"` // 节点角色
//}

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
