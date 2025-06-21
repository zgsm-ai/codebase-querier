package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"gorm.io/gorm"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxLineLimit = 500
const definitionFillContentNodeLimit = 100

type DefinitionQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDefinitionQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DefinitionQueryLogic {
	return &DefinitionQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DefinitionQueryLogic) QueryDefinition(req *types.DefinitionRequest) (resp *types.DefinitionResponseData, err error) {
	// 参数验证
	if req.ClientId == types.EmptyString {
		return nil, errs.NewMissingParamError(types.ClientId)
	}
	if req.CodebasePath == types.EmptyString {
		return nil, errs.NewMissingParamError(types.CodebasePath)
	}
	if req.StartLine <= 0 {
		req.StartLine = 1
	}

	if req.EndLine <= 0 {
		req.EndLine = 1
	}
	if req.EndLine < req.StartLine {
		req.EndLine = req.StartLine
	}

	if req.EndLine-req.StartLine > maxLineLimit {
		req.EndLine = req.StartLine + maxLineLimit
	}

	if req.FilePath == types.EmptyString {
		return nil, errs.NewMissingParamError(types.FilePath)
	}

	clientId := req.ClientId
	clientPath := req.CodebasePath
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path
	// todo concurrency test
	graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}
	defer graphStore.Close()

	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))
	nodes, err := graphStore.QueryDefinition(ctx, req)
	if err != nil {
		return nil, err
	}
	// 填充content，控制层数和节点数
	if err = l.fillContent(ctx, nodes, codebasePath, definitionFillContentNodeLimit); err != nil {
		logx.Errorf("fill definition query contents err:%v", err)
	}

	return &types.DefinitionResponseData{
		List: nodes,
	}, nil
}

func (l *DefinitionQueryLogic) fillContent(ctx context.Context, nodes []*types.DefinitionNode, codebasePath string, nodeLimit int) error {
	if len(nodes) == 0 {
		return nil
	}
	// 处理当前层的节点
	for i, node := range nodes {
		// 如果超过节点限制，跳过剩余节点
		if i >= nodeLimit {
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
	}

	return nil
}
