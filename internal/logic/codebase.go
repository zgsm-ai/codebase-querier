package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	codebase_store "github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"crypto/sha256"
	"encoding/hex"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"strings"
)

type CompareCodebasesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompareCodebaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompareCodebasesLogic {
	return &CompareCodebasesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CompareCodebasesLogic) CompareCodebase(req *types.CodebaseComparisonRequest) (resp *types.ComparisonResponseData, err error) {
	clientCodebasePath := req.CodebasePath
	clientId := req.ClientId
	codebase, err := l.svcCtx.CodebaseModel.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, model.ErrNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path
	if utils.IsBlank(codebasePath) {
		return nil, errors.New("codebase path is empty")
	}

	// 构建响应数据
	resp = &types.ComparisonResponseData{
		CodebaseTree: make([]types.CodebaseTreeItem, 0),
	}

	// 使用 Walk 方法递归遍历目录树
	err = l.svcCtx.CodebaseStore.Walk(l.ctx, codebasePath, "", func(walkCtx *codebase_store.WalkContext, reader io.ReadCloser) error {
		// 跳过目录和隐藏文件
		if walkCtx.Info.IsDir || strings.HasPrefix(walkCtx.Info.Name, ".") || walkCtx.Info.Name == types.SyncMedataDir {
			if walkCtx.Info.IsDir {
				return codebase_store.SkipDir
			}
			return nil
		}

		// 读取文件内容
		content, err := io.ReadAll(reader)
		if err != nil {
			l.Logger.Errorf("failed to read file %s: %v", walkCtx.RelativePath, err)
			return nil
		}

		// 计算文件内容的 SHA-256 哈希
		hash := sha256.Sum256(content)
		hashStr := hex.EncodeToString(hash[:])

		// 添加到响应数据中
		resp.CodebaseTree = append(resp.CodebaseTree, types.CodebaseTreeItem{
			Path: walkCtx.RelativePath,
			Hash: hashStr,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return resp, nil
}
