package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"net/http"
	"time"

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
	// 查找待删除的文件，进行处理
	fileModeMap, _, err := l.svcCtx.CodebaseStore.GetSyncFileListCollapse(l.ctx, codebase.Path)
	var deleteList []string
	for f, m := range fileModeMap {
		if m == types.FileOpDelete {
			deleteList = append(deleteList, f)
		}
	}
	err = l.svcCtx.CodebaseStore.BatchDelete(l.ctx, codebase.Path, deleteList)
	if err != nil {
		l.Logger.Errorf("delete files error", err)
		return err
	}

	if err != nil {
		return err
	}
	var publishStatus = model.PublishStatusPending
	syncHistory := model.SyncHistory{
		CodebaseId:    codebase.Id,
		PublishStatus: string(publishStatus),
		PublishTime:   sql.NullTime{Time: time.Now()},
	}
	if _, err = l.svcCtx.SyncHistoryModel.Insert(l.ctx, &syncHistory); err != nil {
		logx.Errorf("insert sync history %v error:%v", syncHistory, err)
		return nil
	}

	l.Logger.Debugf("uploadFiles successfully: %s, %s, %s", clientId, clientPath, codebaseName)
	syncId := syncHistory.Id
	if syncHistory.Id == 0 {
		syncId = time.Now().Unix()
	}
	// 发送文件消息
	msg := &types.CodebaseSyncMessage{
		SyncID:       syncId,
		CodebaseID:   codebase.Id,
		CodebasePath: codebase.Path,
		CodebaseName: codebase.Name,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("marshal message error:%v", err)
		publishStatus = model.PublishStatusFailed
	} else {
		//TODO 发送失败，本地记录，重发。
		err = l.svcCtx.MessageQueue.Produce(l.ctx, l.svcCtx.Config.IndexTask.Topic, bytes, types.ProduceOptions{})
		if err != nil {
			logx.Errorf("produce message error:%v", err)
			publishStatus = model.PublishStatusFailed
		}
		publishStatus = model.PublishStatusSuccess
	}

	// 更新
	syncHistory.PublishStatus = string(publishStatus)
	syncHistory.PublishTime = sql.NullTime{Time: time.Now()}
	syncHistory.Message = string(bytes)
	if _, err = l.svcCtx.SyncHistoryModel.Insert(l.ctx, &syncHistory); err != nil {
		logx.Errorf("update sync history %v error:%v", syncHistory, err)
		return nil
	}

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
