package codebase

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	defaultFileMode = 0644
	defaultDirMode  = 0755
	maxRetries      = 3
	retryDelay      = time.Second
)

var _ Store = &minioCodebase{}

type minioCodebase struct {
	logger logx.Logger
	cfg    config.CodeBaseStoreConf
	client *minio.Client
	mu     sync.RWMutex // 保护并发访问
}

func NewMinioCodebase(ctx context.Context, cfg config.CodeBaseStoreConf) (Store, error) {
	client, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKeyID, cfg.Minio.SecretAccessKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Ensure bucket exists
	exists, err := client.BucketExists(ctx, cfg.Minio.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, cfg.Minio.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &minioCodebase{
		cfg:    cfg,
		logger: logx.WithContext(ctx),
		client: client,
	}, nil
}

// getObjectName 获取完整的对象名称
func (m *minioCodebase) getObjectName(clientId, codebasePath, target string) string {
	return getObjectName(clientId, codebasePath, target)
}

// Init 初始化一个新的代码库
func (m *minioCodebase) Init(ctx context.Context, clientId string, clientCodebasePath string) (*types.Codebase, error) {
	if clientId == "" || clientCodebasePath == "" {
		return nil, errors.New("clientId and clientCodebasePath cannot be empty")
	}

	// 生成唯一的路径
	dirPath := m.getObjectName(clientId, clientCodebasePath, "") + "/"

	// 在 MinIO 中创建目录（通过创建一个空的目录标记对象）
	_, err := m.client.PutObject(ctx, m.cfg.Minio.Bucket, dirPath, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	return &types.Codebase{
		ClientID: clientId,
		Path:     clientCodebasePath,
	}, nil
}

func (m *minioCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	objectName := m.getObjectName("", codebasePath, target)
	_, err := m.client.PutObject(ctx, m.cfg.Minio.Bucket, objectName, source, -1, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func (m *minioCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	// Create a temporary file to store the zip content
	tmpFile, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("failed to read zip content: %w", err)
	}

	zipReader, err := zip.NewReader(strings.NewReader(string(tmpFile)), int64(len(tmpFile)))
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}

	basePath := m.getObjectName("", codebasePath, target)

	// Extract each file
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		objectName := filepath.Join(basePath, file.Name)

		// Check for zip slip vulnerability
		if !strings.HasPrefix(objectName, basePath) {
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
	objectName := m.getObjectName("", codebasePath, path)
	err := m.client.RemoveObject(ctx, m.cfg.Minio.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (m *minioCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	// MinIO is object-based and doesn't have real directories
	// We'll create an empty object to simulate directory creation
	objectName := m.getObjectName("", codebasePath, path) + "/"
	_, err := m.client.PutObject(ctx, m.cfg.Minio.Bucket, objectName, strings.NewReader(""), 0, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (m *minioCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	objectName := m.getObjectName("", codebasePath, path)
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
	objectName := m.getObjectName("", codebasePath, path)
	info, err := m.client.StatObject(ctx, m.cfg.Minio.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &types.FileInfo{
		Name:    filepath.Base(objectName),
		Size:    info.Size,
		IsDir:   info.Size == 0 && strings.HasSuffix(objectName, "/"),
		ModTime: info.LastModified,
		Mode:    defaultFileMode,
	}, nil
}

func (m *minioCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	prefix := m.getObjectName("", codebasePath, dir)
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

		// Apply filters if specified
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(object.Key) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(object.Key) {
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
	prefix := m.getObjectName("", codebasePath, dir)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 获取所有对象
	var objects []minio.ObjectInfo
	objectCh := m.client.ListObjects(ctx, m.cfg.Minio.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		objects = append(objects, object)
	}

	// 构建路径到节点的映射
	nodeMap := make(map[string]*types.TreeNode)
	var rootNodes []*types.TreeNode

	// 首先创建所有目录节点
	for _, object := range objects {
		// 跳过根目录标记
		if object.Key == prefix {
			continue
		}

		// 获取相对路径
		relPath := strings.TrimPrefix(object.Key, prefix)
		if relPath == "" {
			continue
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(relPath) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(relPath) {
			continue
		}

		// 处理路径的每一级
		parts := strings.Split(relPath, "/")
		currentPath := ""

		for i, part := range parts {
			if i == 0 {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			// 如果这个路径的节点已经存在，跳过
			if _, exists := nodeMap[currentPath]; exists {
				continue
			}

			// 创建新节点
			node := &types.TreeNode{
				FileInfo: types.FileInfo{
					Name:    part,
					Path:    currentPath,
					IsDir:   i < len(parts)-1 || strings.HasSuffix(object.Key, "/"),
					Size:    object.Size,
					ModTime: object.LastModified,
					Mode:    defaultFileMode,
				},
				Children: make([]types.TreeNode, 0),
			}

			// 将节点添加到映射中
			nodeMap[currentPath] = node

			// 如果是根级节点，添加到rootNodes
			if i == 0 {
				rootNodes = append(rootNodes, node)
			} else {
				// 将节点添加到父节点的Children中
				parentPath := filepath.Dir(currentPath)
				if parent, exists := nodeMap[parentPath]; exists {
					parent.Children = append(parent.Children, *node)
				}
			}
		}
	}

	return rootNodes, nil
}

func (m *minioCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) (string, error) {
	objectName := m.getObjectName("", codebasePath, filePath)
	object, err := m.client.GetObject(ctx, m.cfg.Minio.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	content, err := io.ReadAll(object)
	if err != nil {
		return "", fmt.Errorf("failed to read object content: %w", err)
	}

	return string(content), nil
}

func (m *minioCodebase) Walk(ctx context.Context, codebasePath string, dir string, process func(io.ReadCloser) (bool, error)) error {
	prefix := m.getObjectName("", codebasePath, dir)
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

		// Skip directories
		if object.Size == 0 && strings.HasSuffix(object.Key, "/") {
			continue
		}

		obj, err := m.client.GetObject(ctx, m.cfg.Minio.Bucket, object.Key, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to get object %s: %w", object.Key, err)
		}

		stop, err := process(obj)
		obj.Close()
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}

	return nil
}

func (m *minioCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, path := range paths {
			objectName := m.getObjectName("", codebasePath, path)
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
