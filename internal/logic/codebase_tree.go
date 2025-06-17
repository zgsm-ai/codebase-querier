package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CodebaseTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCodebaseTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CodebaseTreeLogic {
	return &CodebaseTreeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CodebaseTreeLogic) CodebaseTree(req *types.CodebaseTreeRequest) (resp *types.CodebaseTreeResponseData, err error) {
	// 1. 从数据库查询 codebase 信息
	clientCodebasePath := req.CodebasePath
	clientId := req.ClientId

	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}

	codebasePath := codebase.Path
	if utils.IsBlank(codebasePath) {
		return nil, errors.New("codebase path is empty")
	}

	// 2. 获取目录树
	store := l.svcCtx.CodebaseStore
	treeOpts := types.TreeOptions{
		MaxDepth: req.Depth,
	}

	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	nodes, err := store.Tree(ctx, codebasePath, req.SubDir, treeOpts)
	if err != nil {
		l.Logger.Errorf("failed to get directory tree: %v", err)
		return nil, err
	}

	// 3. 计算文件统计信息
	var totalFiles int
	var totalSize int64
	if len(nodes) > 0 {
		countFilesAndSize(nodes, &totalFiles, &totalSize, req.IncludeFiles == 1)
	}

	resp = &types.CodebaseTreeResponseData{
		CodebaseId:    codebase.ID,
		Name:          codebase.Name,
		RootPath:      codebasePath,
		TotalFiles:    totalFiles,
		TotalSize:     totalSize,
		DirectoryTree: nodes,
	}

	return resp, nil
}

// countFilesAndSize 统计文件数量和总大小
func countFilesAndSize(nodes []*types.TreeNode, totalFiles *int, totalSize *int64, includeFiles bool) {
	if len(nodes) == 0 {
		return
	}

	for _, node := range nodes {
		if node == nil {
			continue
		}

		if !node.IsDir {
			if includeFiles {
				*totalFiles++
				*totalSize += node.Size
			}
			continue
		}

		// 递归处理子节点
		countFilesAndSize(node.Children, totalFiles, totalSize, includeFiles)
	}
}
