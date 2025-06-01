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
	filepathSlash   = "/"
)

var _ Store = &minioCodebase{}

type minioCodebase struct {
	cfg    config.CodeBaseStoreConf
	client MinioClient
	logger logx.Logger
	mu     sync.RWMutex
}

func NewMinioCodebase(ctx context.Context, cfg config.CodeBaseStoreConf) Store {
	client, err := minio.New(cfg.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.AccessKeyID, cfg.Minio.SecretAccessKey, ""),
		Secure: cfg.Minio.UseSSL,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create minio client: %v", err))
	}

	return &minioCodebase{
		cfg:    cfg,
		client: NewMinioClientWrapper(client),
		logger: logx.WithContext(ctx),
	}
}

func (m *minioCodebase) DeleteAll(ctx context.Context, codebasePath string) error {
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
	return nil
}

// Init 初始化一个新的代码库
func (m *minioCodebase) Init(ctx context.Context, clientId string, clientCodebasePath string) (*types.Codebase, error) {
	if clientId == "" || clientCodebasePath == "" {
		return nil, errors.New("clientId and clientCodebasePath cannot be empty")
	}

	// 生成唯一的路径
	codebasePath, err := generateCodebasePath(m.cfg.Minio.Bucket, clientId, clientCodebasePath)
	if err != nil {
		return nil, err
	}
	// 在 MinIO 中创建目录（通过创建一个空的目录标记对象）
	_, err = m.client.PutObject(ctx, m.cfg.Minio.Bucket, codebasePath, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	return &types.Codebase{
		FullPath: codebasePath,
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

	basePath := filepath.Join(codebasePath, target)

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
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Build path to node mapping
	nodeMap := make(map[string]*types.TreeNode)
	var rootNodes []*types.TreeNode

	objectCh := m.client.ListObjects(ctx, m.cfg.Minio.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
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

		// Apply filters
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(relPath) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(relPath) {
			continue
		}

		// Process each level of the path
		parts := strings.Split(relPath, "/")
		currentPath := ""

		for i, part := range parts {
			if i == 0 {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			// Skip if node already exists
			if _, exists := nodeMap[currentPath]; exists {
				continue
			}

			// Create new node
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

			// Add node to map
			nodeMap[currentPath] = node

			// Add to root nodes if it's a root level node
			if i == 0 {
				rootNodes = append(rootNodes, node)
			} else {
				// Add to parent's children
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
	objectName := filepath.Join(codebasePath, filePath)
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

func (m *minioCodebase) Walk(ctx context.Context, codebasePath string, dir string, walkFn WalkFunc) error {
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

		// 构建 WalkContext
		walkCtx := &WalkContext{
			Path:         object.Key,
			RelativePath: strings.TrimPrefix(object.Key, prefix),
			Info: &types.FileInfo{
				Name:    filepath.Base(object.Key),
				Size:    object.Size,
				IsDir:   object.Size == 0 && strings.HasSuffix(object.Key, "/"),
				ModTime: object.LastModified,
				Mode:    defaultFileMode,
			},
			ParentPath: filepath.Dir(object.Key),
		}

		// 如果是目录，直接调用 walkFn，传入 nil reader
		if walkCtx.Info.IsDir {
			err := walkFn(walkCtx, nil)
			if errors.Is(err, SkipDir) {
				continue
			}
			if err != nil {
				return err
			}
			continue
		}

		// 对于文件，获取对象并传入 reader
		obj, err := m.client.GetObject(ctx, m.cfg.Minio.Bucket, object.Key, minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to get object %s: %w", object.Key, err)
		}

		err = walkFn(walkCtx, obj)
		obj.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *minioCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
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
