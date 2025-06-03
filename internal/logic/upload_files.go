package logic

import (
	"context"
	"database/sql"
	"errors"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"net/http"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFilesLogic {
	return &UploadFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadFilesLogic) UploadFiles(req *types.FileUploadRequest, r *http.Request) error {
	// TODO 删除的处理
	clientId := req.ClientId
	clientPath := req.CodebasePath
	codebaseName := req.CodebaseName
	metadata := req.ExtraMetadata
	l.Logger.Debugf("uploadFiles successfully: %s, %s, %s", clientId, clientPath, codebaseName)

	// 判断是否存在
	codebase, err := l.svcCtx.CodebaseModel.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}

	userUid := utils.ParseJWTUserInfo(r, l.svcCtx.Config.Auth.UserInfoHeader)
	// 存在 则 直接插入， 不存在则需要先init
	// 数据库唯一索引
	if errors.Is(err, model.ErrNotFound) {
		codebase, err = l.initCodebase(clientId, clientPath, userUid, codebaseName, metadata)
		if err != nil {
			return err
		}
	}
	body, err := r.GetBody()
	defer body.Close()
	if err != nil {
		return err
	}
	err = l.svcCtx.CodebaseStore.Unzip(l.ctx, codebase.Path, body, codebase.Path)
	if err != nil {
		return err
	}
	l.Logger.Debugf("uploadFiles successfully: %s, %s, %s", clientId, clientPath, codebaseName)
	return nil
}

/**
 * @Description: 初始化 codebase
 * @receiver l
 * @param clientId
 * @param clientPath
 * @param r
 * @param codebaseName
 * @param metadata
 * @return error
 * @return bool
 */
func (l *UploadFilesLogic) initCodebase(clientId string, clientPath string, userUId string,
	codebaseName string, metadata string) (*model.Codebase, error) {
	// 不存在则插入
	// clientId + codebasepath 为联合唯一索引
	// 保存到数据库
	codebaseStore, err := l.svcCtx.CodebaseStore.Init(l.ctx, clientId, clientPath)
	if err != nil {
		return nil, err
	}
	codebaseModel := &model.Codebase{
		ClientId:   clientId,
		UserId:     userUId,
		Name:       codebaseName,
		ClientPath: clientPath,
		Path:       codebaseStore.FullPath,
		ExtraMetadata: sql.NullString{
			String: metadata,
		},
	}
	_, err = l.svcCtx.CodebaseModel.Insert(l.ctx, codebaseModel)
	// 不是 唯一索引冲突 TODO define this error
	if err != nil && !errors.Is(err, model.UniqueIndexConflictErr) {
		return nil, err
	}
	return codebaseModel, nil
}
