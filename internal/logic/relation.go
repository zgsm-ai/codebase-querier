package logic

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const fillContentLayerLimit = 2
const fillContentLayerNodeLimit = 10
const maxLayerLimit = 5

type RelationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRelationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RelationLogic {
	return &RelationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RelationLogic) Relation(req *types.RelationQueryOptions) (resp *types.RelationResponseData, err error) {

	// 参数验证
	if req == nil {
		return nil, errs.NewMissingParamError("opts")
	}
	if req.MaxLayer <= 0 {
		req.MaxLayer = 1
	}

	if req.MaxLayer > maxLayerLimit {
		return nil, errs.NewInvalidParamErr("maxLayer", fmt.Sprintf("参数maxLayer非法，最大值为%d", maxLayerLimit))
	}

	if req.FilePath == types.EmptyString {
		return nil, errs.NewMissingParamError(types.FilePath)
	}

	clientId := req.ClientId
	clientCodebasePath := req.CodebasePath
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path

	graphStore, err := codegraph.NewBadgerDBGraph(l.ctx, codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}
	nodes, err := graphStore.Query(l.ctx, req)
	if err != nil {
		return nil, err
	}
	// 填充content，控制层数和节点数
	if err = l.fillContent(l.ctx, nodes, codebasePath, fillContentLayerLimit, fillContentLayerNodeLimit); err != nil {
		logx.Errorf("fill graph query contents err:%v", err)
	}

	return &types.RelationResponseData{
		List: nodes,
	}, nil
}

func (l *RelationLogic) fillContent(ctx context.Context, nodes []*types.GraphNode, codebasePath string, layerLimit, layerNodeLimit int) error {
	if len(nodes) == 0 {
		return nil
	}
	// 处理当前层的节点
	for i, node := range nodes {
		// 如果超过每层节点限制，跳过剩余节点
		if i >= layerNodeLimit {
			break
		}

		// 读取文件内容
		content, err := l.svcCtx.CodebaseStore.Read(ctx, codebasePath, node.FilePath, types.ReadOptions{
			StartLine: node.Position.StartLine,
			EndLine:   node.Position.EndLine,
		})

		if err != nil {
			l.Logger.Errorf("read file content failed: %v", err)
			continue
		}

		// 设置节点内容
		node.Content = string(content)

		// 如果还没有达到层级限制且有子节点，递归处理子节点
		if layerLimit > 1 && len(node.Children) > 0 {
			if err := l.fillContent(ctx, node.Children, codebasePath, layerLimit-1, layerNodeLimit); err != nil {
				l.Logger.Errorf("fill children content failed: %v", err)
			}
		}
	}

	return nil
}
