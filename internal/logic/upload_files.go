package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/model"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"

	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSyncFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncFilesLogic {
	return &SyncFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncFilesLogic) SyncFiles(req *types.FileUploadRequest, r *http.Request) error {
	clientId := req.ClientId
	clientPath := req.CodebasePath
	codebaseName := req.CodebaseName
	metadata := req.ExtraMetadata
	l.Logger.Debugf("uploadFiles request: %s, %s, %s", clientId, clientPath, codebaseName)
	userUid := utils.ParseJWTUserInfo(r, l.svcCtx.Config.Auth.UserInfoHeader)
	// 判断代码库数据库记录、存储路径是否存在
	codebase, err := l.initCodebaseIfNotExists(userUid, clientId, clientPath, codebaseName, metadata)
	if err != nil {
		return err
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}
	defer r.MultipartForm.RemoveAll()

	// Get the ZIP file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		return fmt.Errorf("failed to get file from form: %w", err)
	}
	defer file.Close()

	// Verify file is a ZIP
	if !strings.HasSuffix(header.Filename, ".zip") {
		return fmt.Errorf("uploaded file must be a ZIP file, got: %s", header.Filename)
	}

	// Unzip the file
	err = l.svcCtx.CodebaseStore.Unzip(l.ctx, codebase.Path, file)
	if err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	// 查找待删除的文件，进行处理
	metaFiles, err := l.svcCtx.CodebaseStore.GetSyncFileListCollapse(ctx, codebase.Path)
	if err != nil {
		l.Logger.Errorf("get sync file list error: %v", err)
		return err
	}

	var deleteList []string
	for f, m := range metaFiles.FileModelMap {
		if m == types.FileOpDelete {
			deleteList = append(deleteList, f)
		}
	}

	if len(deleteList) > 0 {
		err = l.svcCtx.CodebaseStore.BatchDelete(ctx, codebase.Path, deleteList)
		if err != nil {
			l.Logger.Errorf("delete files error: %v", err)
			return err
		}
	}

	// Create sync history
	var publishStatus = model.PublishStatusPending

	syncHistory := &model.SyncHistory{
		CodebaseID:    codebase.ID,
		PublishStatus: string(publishStatus),
		PublishTime:   utils.CurrentTime(),
	}
	if err = l.svcCtx.Querier.SyncHistory.WithContext(ctx).Save(syncHistory); err != nil {
		l.Logger.Errorf("insert sync history %v error: %v", syncHistory, err)
		return err
	}

	l.Logger.Debugf("uploadFiles successfully: %s, %s, %s", clientId, clientPath, codebaseName)
	syncId := syncHistory.ID
	if syncId == 0 {
		syncId = int32(time.Now().Unix())
	}

	// 发送文件消息
	msg := &types.CodebaseSyncMessage{
		SyncID:       syncId,
		CodebaseID:   codebase.ID,
		CodebasePath: codebase.Path,
		CodebaseName: codebase.Name,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		l.Logger.Errorf("marshal message error: %v", err)
		publishStatus = model.PublishStatusFailed
	} else {
		// 更新最新版本
		err := l.svcCtx.Cache.AddVersion(ctx, types.SyncVersionKey(codebase.ID), int64(syncHistory.ID), time.Hour*24)
		if err != nil {
			l.Logger.Errorf("set sync version error: %v", err)
		}
		// 发送消息
		err = l.svcCtx.MessageQueue.Produce(ctx, l.svcCtx.Config.IndexTask.Topic, bytes, types.ProduceOptions{})
		if err != nil {
			l.Logger.Errorf("produce message error: %v", err)
			publishStatus = model.PublishStatusFailed
		} else {
			publishStatus = model.PublishStatusSuccess
		}
	}

	// Update sync history
	syncHistory.PublishStatus = string(publishStatus)
	syncHistory.PublishTime = utils.CurrentTime()
	messageStr := string(bytes)
	syncHistory.Message = &messageStr
	if _, err = l.svcCtx.Querier.SyncHistory.WithContext(ctx).
		Where(l.svcCtx.Querier.SyncHistory.ID.Eq(syncHistory.ID)).
		Updates(&syncHistory); err != nil {
		l.Logger.Errorf("update sync history %v error: %v", syncHistory, err)
		return err
	}

	return nil
}

func (l *SyncFilesLogic) initCodebaseIfNotExists(userUid, clientId, clientPath, codebaseName, metadata string) (*model.Codebase, error) {
	var codebase *model.Codebase
	var err error
	// 判断数据库记录是否存在 ，状态为 active
	codebase, err = l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 存在 则 直接插入， 不存在则需要先init
	// 数据库唯一索引
	// 连数据库记录都不存在，那存储理论上也不存在
	if errors.Is(err, gorm.ErrRecordNotFound) {
		storeCodebase, err := l.svcCtx.CodebaseStore.Init(l.ctx, clientId, clientPath)
		if err != nil {
			logx.Errorf("init codebase store err:%v", err)
			return nil, fmt.Errorf("init codebase err:%w", err)
		}
		codebase, err = l.saveCodebase(clientId, clientPath, storeCodebase.BasePath, userUid, codebaseName, metadata)
		if err != nil {
			return nil, err
		}
	}
	// 检查存储路径是否存在
	exists, existsErr := l.svcCtx.CodebaseStore.Exists(l.ctx, codebase.Path, types.EmptyString)
	if existsErr != nil {
		logx.Errorf("check codebase exists err:%v", existsErr)
	}
	if existsErr != nil || !exists {
		if _, err := l.svcCtx.CodebaseStore.Init(l.ctx, clientId, clientPath); err != nil {
			logx.Errorf("init codebase store err:%v", err)
			return nil, fmt.Errorf("init codebase err:%w", err)
		}
	}
	if codebase == nil {
		logx.Errorf("init codebase failed, clientId:%s, clientPath:%s, codebase model is nil",
			clientId, clientPath)
		return nil, fmt.Errorf("init codebase failed")
	}

	return codebase, nil
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
func (l *SyncFilesLogic) saveCodebase(clientId, clientPath, codebasePath, userUId,
	codebaseName, metadata string) (*model.Codebase, error) {
	// 不存在则插入
	// clientId + codebasepath 为联合唯一索引
	// 保存到数据库
	codebaseModel := &model.Codebase{
		ClientID:      clientId,
		UserID:        userUId,
		Name:          codebaseName,
		ClientPath:    clientPath,
		Status:        string(model.CodebaseStatusActive),
		Path:          codebasePath,
		ExtraMetadata: &metadata,
	}
	err := l.svcCtx.Querier.Codebase.WithContext(l.ctx).Save(codebaseModel)
	if err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
		// 不是 唯一索引冲突
		return nil, err
	}
	return codebaseModel, nil
}
