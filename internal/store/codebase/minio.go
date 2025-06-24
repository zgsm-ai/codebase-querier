package codebase

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase/wrapper"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	defaultFileMode = 0644
	defaultDirMode  = 0755
	maxRetries      = 3
	retryDelay      = time.Second
	filepathSlash   = "/"
)

var _ Store = &minioCodebase{}

type minioCodebase struct {
	cfg    config.CodeBaseStoreConf
	client wrapper.MinioClient
	mu     sync.RWMutex
}

func (m *minioCodebase) GetSyncFileListCollapse(ctx context.Context, codebasePath string) (*types.CollapseSyncMetaFile, error) {
	exists, err := m.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}
	// filepath -> mode(add delete modify)
	// 根据元数据获取代码文件列表
	// 递归目录，进行处理，并发
	// 获取代码文件列表
	list, err := m.List(ctx, codebasePath, types.SyncMedataDir, types.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("embeddingProcessor metadata dir is empty")
	}
	//TODO collapse list to fileList
	// 对目录下的文件按名字升序排序
	treeSet := utils.NewTimestampTreeSet()
	// sort
	for _, f := range list {
		treeSet.Add(f.Name)
	}
	var fileModeMap map[string]string
	var metaFileList []string
	it := treeSet.Iterator()
	for it.Next() {
		metadataFile := it.Value().(string)
		metaFileList = append(metaFileList, metadataFile)
		syncMetaData, err := m.Read(ctx, codebasePath, filepath.Join(types.SyncMedataDir, metadataFile), types.ReadOptions{})
		if err != nil {
			tracer.WithTrace(ctx).Errorf("read metadata file %v failed: %v", metadataFile, err)
			continue
		}
		if syncMetaData == nil {
			tracer.WithTrace(ctx).Errorf("sync file %s metadata is empty", metadataFile)
			continue
		}
		var syncMetaObj *types.SyncMetadataFile

		err = json.Unmarshal(syncMetaData, &syncMetaObj)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("failed to unmarshal metadata error: %v, raw: %s", err, syncMetaData)
		}
		files := syncMetaObj.FileList
		for k, v := range files {
			// add delete modify
			fileModeMap[k] = v
		}

	}
	return &types.CollapseSyncMetaFile{CodebasePath: codebasePath,
		FileModelMap: fileModeMap, MetaFilePaths: metaFileList}, nil
}

func (m *minioCodebase) Open(ctx context.Context, codebasePath string, filePath string) (io.ReadSeekCloser, error) {
	exists, err := m.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}
	objectName := filepath.Join(codebasePath, filePath)
	return m.client.GetObject(ctx, m.cfg.Minio.Bucket, objectName, minio.GetObjectOptions{})
}

func NewMinioCodebase(cfg config.CodeBaseStoreConf) (Store, error) {
	if cfg.Minio.Bucket == types.EmptyString {
		return nil, errors.New("minio bucket cannot be empty")
	}
	client, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKeyID, cfg.Minio.SecretAccessKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %v", err)
	}

	return &minioCodebase{
		cfg:    cfg,
		client: wrapper.NewMinioClientWrapper(client),
	}, nil
}

func (m *minioCodebase) DeleteAll(ctx context.Context, codebasePath string) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}

	if codebasePath == "*" {
		return fmt.Errorf("illegal codebasePath:%s", codebasePath)
	}
	tracer.WithTrace(ctx).Infof("start to delete codebasePath [%s]", codebasePath)
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		objectName := filepath.Join(codebasePath)
		objectsCh <- minio.ObjectInfo{Key: objectName}
	}()

	errCh := m.client.RemoveObjects(ctx, m.cfg.Minio.Bucket, objectsCh, minio.RemoveObjectsOptions{})
	var errs []error
	for err := range errCh {
		if err.Err != nil {
			errs = append(errs, fmt.Errorf("failed to delete object %s: %w", err.ObjectName, err.Err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch delete errors: %v", errs)
	}
	tracer.WithTrace(ctx).Infof("delete codebasePath [%s] successfully", codebasePath)
	return nil
}

// Init 初始化一个新的代码库
func (m *minioCodebase) Init(ctx context.Context, clientId string, clientPath string) (*types.Codebase, error) {
	if clientId == "" || clientPath == "" {
		return nil, errors.New("clientId and clientPath cannot be empty")
	}

	// 生成唯一的路径
	codebasePath, err := generateCodebasePath(m.cfg.Minio.Bucket, clientId, clientPath)
	if err != nil {
		return nil, err
	}
	// 在 MinIO 中创建目录（通过创建一个空的目录标记对象）
	_, err = m.client.PutObject(ctx, m.cfg.Minio.Bucket, codebasePath, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	return &types.Codebase{
		BasePath: codebasePath,
	}, nil
}

func (m *minioCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	if codebasePath == "" || target == "" {
		return errors.New("codebasePath and target cannot be empty")
	}
	objectName := filepath.Join(codebasePath, target)
	_, err := m.client.PutObject(ctx, m.cfg.Minio.Bucket, objectName, source, -1, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func (m *minioCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader) error {
	// Create a temporary file to store the zip content
	tmpFile, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("failed to read zip content: %w", err)
	}

	zipReader, err := zip.NewReader(strings.NewReader(string(tmpFile)), int64(len(tmpFile)))
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}

	// Extract each file
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		objectName := filepath.Join(codebasePath, file.Name)

		// Check for zip slip vulnerability
		if !strings.HasPrefix(objectName, codebasePath) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file entry: %w", err)
		}

		_, err = m.client.PutObject(ctx, m.cfg.Minio.Bucket, objectName, rc, file.FileInfo().Size(), minio.PutObjectOptions{})
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to upload file %s: %w", file.Name, err)
		}
	}

	return nil
}

func (m *minioCodebase) Delete(ctx context.Context, codebasePath string, path string) error {
	if codebasePath == "" || path == "" {
		return errors.New("codebasePath and path cannot be empty")
	}
	objectName := filepath.Join(codebasePath, path)
	err := m.client.RemoveObject(ctx, m.cfg.Minio.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (m *minioCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	// MinIO is object-based and doesn't have real directories
	// We'll create an empty object to simulate directory creation
	objectName := filepath.Join(codebasePath, path) + "/"
	_, err := m.client.PutObject(ctx, m.cfg.Minio.Bucket, objectName, strings.NewReader(""), 0, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (m *minioCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	objectName := filepath.Join(codebasePath, path)
	_, err := m.client.StatObject(ctx, m.cfg.Minio.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return true, nil
}

func (m *minioCodebase) Stat(ctx context.Context, codebasePath string, path string) (*types.FileInfo, error) {
	objectName := filepath.Join(codebasePath, path)
	info, err := m.client.StatObject(ctx, m.cfg.Minio.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &types.FileInfo{
		Name:    filepath.Base(objectName),
		Path:    objectName,
		Size:    info.Size,
		IsDir:   info.Size == 0 && strings.HasSuffix(objectName, "/"),
		ModTime: info.LastModified,
		Mode:    defaultFileMode,
	}, nil
}

func (m *minioCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	if codebasePath == "" {
		return nil, errors.New("codebasePath cannot be empty")
	}
	prefix := filepath.Join(codebasePath, dir)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var files []*types.FileInfo
	objectCh := m.client.ListObjects(ctx, m.cfg.Minio.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// Skip the directory marker itself
		if object.Key == prefix {
			continue
		}

		// Get the relative path
		relPath := strings.TrimPrefix(object.Key, prefix)
		if relPath == "" {
			continue
		}

		// Apply filters if specified
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(relPath) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(relPath) {
			continue
		}

		files = append(files, &types.FileInfo{
			Name:    filepath.Base(object.Key),
			Size:    object.Size,
			IsDir:   object.Size == 0 && strings.HasSuffix(object.Key, "/"),
			ModTime: object.LastModified,
			Mode:    defaultFileMode,
		})
	}

	return files, nil
}

func (m *minioCodebase) Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	if codebasePath == "" {
		return nil, errors.New("codebasePath cannot be empty")
	}

	prefix := filepath.Join(codebasePath, dir)
	// 只有当不是根目录时才添加斜杠
	if dir != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 使用 map 来构建目录树
	nodeMap := make(map[string]*types.TreeNode)

	objectCh := m.client.ListObjects(ctx, m.cfg.Minio.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// Skip empty objects
		if object.Key == "" {
			continue
		}

		// Get the relative path
		relPath := strings.TrimPrefix(object.Key, prefix)
		if relPath == "" {
			continue
		}

		// Skip hidden files
		if strings.HasPrefix(filepath.Base(relPath), ".") {
			continue
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(relPath) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(relPath) {
			continue
		}

		// 检查深度限制
		if option.MaxDepth > 0 {
			depth := len(strings.Split(relPath, "/"))
			if depth > option.MaxDepth {
				continue
			}
		}

		var currentPath string
		var parts []string

		// 如果是根目录下的文件或目录
		if !strings.Contains(relPath, "/") {
			currentPath = relPath
			parts = []string{relPath}
		} else {
			// 处理子目录中的文件和目录
			parts = strings.Split(relPath, "/")
			currentPath = parts[0]
		}

		// 处理路径中的每一级
		for i, part := range parts {
			if part == "" {
				continue
			}

			if i > 0 {
				currentPath = filepath.Join(currentPath, part)
			}

			// 如果节点已存在，跳过
			if _, exists := nodeMap[currentPath]; exists {
				continue
			}

			// 创建新节点
			isLast := i == len(parts)-1
			isDir := !isLast || strings.HasSuffix(object.Key, "/")
			var size int64
			if isLast && !isDir {
				size = object.Size
			}

			node := &types.TreeNode{
				FileInfo: types.FileInfo{
					Name:    part,
					Path:    currentPath,
					IsDir:   isDir,
					Size:    size,
					ModTime: object.LastModified,
					Mode:    defaultFileMode,
				},
				Children: make([]*types.TreeNode, 0),
			}

			// 将节点添加到 map
			nodeMap[currentPath] = node

			// 如果不是根级节点，添加到父节点的子节点列表
			if i > 0 {
				parentPath := filepath.Dir(currentPath)
				if parent, exists := nodeMap[parentPath]; exists {
					parent.Children = append(parent.Children, node)
				}
			}
		}
	}

	// 构建根节点列表
	var rootNodes []*types.TreeNode
	for path, node := range nodeMap {
		if !strings.Contains(path, "/") {
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes, nil
}

func (m *minioCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) ([]byte, error) {
	objectName := filepath.Join(codebasePath, filePath)
	object, err := m.client.GetObject(ctx, m.cfg.Minio.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// 如果StartLine <= 0，设置为1
	if option.StartLine <= 0 {
		option.StartLine = 1
	}

	// 创建reader来读取文件
	reader := bufio.NewReader(object)
	var lines []string
	lineNum := 1

	// 读取行
	for {
		// 读取一行，允许超过默认缓冲区大小
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read object content: %w", err)
		}

		// 处理可能被截断的行
		var lineBuffer []byte
		lineBuffer = append(lineBuffer, line...)
		for isPrefix {
			line, isPrefix, err = reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("failed to read object content: %w", err)
			}
			lineBuffer = append(lineBuffer, line...)
		}

		// 转换为字符串
		lineStr := string(lineBuffer)

		// 如果当前行号大于等于StartLine，则添加到结果中
		if lineNum >= option.StartLine {
			// 如果EndLine > 0 且当前行号大于EndLine，则退出
			if option.EndLine > 0 && lineNum > option.EndLine {
				break
			}
			lines = append(lines, lineStr)
		}
		lineNum++
	}

	// 将结果转换为字节数组
	return []byte(strings.Join(lines, "\n")), nil
}

func (m *minioCodebase) Walk(ctx context.Context, codebasePath string, dir string, walkFn WalkFunc, walkOpts WalkOptions) error {
	prefix := filepath.Join(codebasePath, dir)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objectCh := m.client.ListObjects(ctx, m.cfg.Minio.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %w", object.Err)
		}

		if slices.Contains(walkOpts.ExcludeExts, filepath.Ext(object.Key)) {
			return nil
		}

		// Skip empty paths
		relPath := strings.TrimPrefix(object.Key, prefix)
		if relPath == "" {
			continue
		}
		// TODO 应用过滤策略

		// 构建 WalkContext，使用相对路径
		walkCtx := &WalkContext{
			Path:         relPath,
			RelativePath: relPath,
			Info: &types.FileInfo{
				Name:    filepath.Base(relPath),
				Size:    object.Size,
				IsDir:   object.Size == 0 && strings.HasSuffix(object.Key, "/"),
				ModTime: object.LastModified,
				Mode:    defaultFileMode,
			},
			ParentPath: filepath.Dir(relPath),
		}

		// 如果是目录，直接调用 walkFn，传入 nil reader
		if walkCtx.Info.IsDir {
			err := walkFn(walkCtx, nil)
			if errors.Is(err, SkipDir) {
				continue
			}
			if err != nil && !walkOpts.IgnoreError {
				return err
			}
			continue
		}

		// 对于文件，获取对象并传入 reader
		obj, err := m.client.GetObject(ctx, m.cfg.Minio.Bucket, object.Key, minio.GetObjectOptions{})
		if err != nil && !walkOpts.IgnoreError {
			return fmt.Errorf("failed to get object %s: %w", object.Key, err)
		}

		err = walkFn(walkCtx, obj)
		obj.Close()
		if err != nil && !walkOpts.IgnoreError {
			return err
		}
	}

	return nil
}

func (m *minioCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	if len(paths) == 0 {
		return errors.New("batch delete paths cannot be empty")
	}

	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, path := range paths {
			objectName := filepath.Join(codebasePath, path)
			objectsCh <- minio.ObjectInfo{Key: objectName}
		}
	}()

	errCh := m.client.RemoveObjects(ctx, m.cfg.Minio.Bucket, objectsCh, minio.RemoveObjectsOptions{})
	var errs []error
	for err := range errCh {
		if err.Err != nil {
			errs = append(errs, fmt.Errorf("failed to delete object %s: %w", err.ObjectName, err.Err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch delete errors: %v", errs)
	}
	return nil
}

func (m *minioCodebase) ResolveSourceRoot(ctx context.Context, codebasePath string, language parser.Language) (string, error) {
	// SourceRootResolver 默认 SourceRoot 配置（按优先级排序）
	sourceRootResolver := getSourceRootResolver(ctx, codebasePath, m)
	resolver, ok := sourceRootResolver[language]
	if !ok {

		return types.EmptyString, ErrSourceRootResolverNotFound
	}
	return resolver()

}

func (m *minioCodebase) InferLanguage(ctx context.Context, codebasePath string) (string, error) {
	panic("implement me")
}
