package codebase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	defaultLocalFileMode = 0644
	defaultLocalDirMode  = 0755
)

var _ Store = &localCodebase{}

type localCodebase struct {
	logger logx.Logger
	cfg    config.CodeBaseStoreConf
	mu     sync.RWMutex // 保护并发访问
}

func (l *localCodebase) Open(ctx context.Context, codebasePath string, filePath string) (io.ReadSeekCloser, error) {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("codebase path %s does not exist", codebasePath)
	}
	return os.Open(filepath.Join(codebasePath, filePath))
}

func (l *localCodebase) DeleteAll(ctx context.Context, codebasePath string) error {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}
	return os.RemoveAll(codebasePath)
}

func NewLocalCodebase(ctx context.Context, cfg config.CodeBaseStoreConf) (Store, error) {
	return &localCodebase{
		cfg:    cfg,
		logger: logx.WithContext(ctx),
	}, nil
}

// Init 初始化一个新的代码库
func (l *localCodebase) Init(ctx context.Context, clientId string, clientCodebasePath string) (*types.Codebase, error) {
	if clientId == types.EmptyString || clientCodebasePath == types.EmptyString {
		return nil, errors.New("clientId and clientCodebasePath cannot be empty")
	}

	// 生成唯一的路径
	uniquePath, err := generateCodebasePath(l.cfg.Local.BasePath, clientId, clientCodebasePath)
	if err != nil {
		return nil, err
	}
	codebasePath := filepath.Join(l.cfg.Local.BasePath, uniquePath)
	// 创建目录
	err = os.MkdirAll(codebasePath, defaultLocalDirMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	return &types.Codebase{FullPath: codebasePath}, nil
}

func (l *localCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	if codebasePath == types.EmptyString || target == types.EmptyString {
		return errors.New("codebasePath and target cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	// 构建完整路径
	fullPath := filepath.Join(codebasePath, target)

	// 确保目标目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, defaultLocalDirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 复制内容
	_, err = io.Copy(file, source)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (l *localCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	// Create a temporary file to store the zip content
	tmpFile, err := os.CreateTemp(types.EmptyString, "codebase-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, source); err != nil {
		return fmt.Errorf("failed to copy zip content: %w", err)
	}

	// Open the zip file
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipReader.Close()

	basePath := filepath.Join(codebasePath, target)

	// Extract each file
	for _, file := range zipReader.File {
		filePath := filepath.Join(basePath, file.Name)

		// Check for zip slip vulnerability
		if !strings.HasPrefix(filePath, basePath) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, defaultLocalDirMode); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), defaultLocalDirMode); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file entry: %w", err)
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultLocalFileMode)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			rc.Close()
			outFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		rc.Close()
		outFile.Close()
	}

	return nil
}

func (l *localCodebase) Delete(ctx context.Context, codebasePath string, path string) error {
	if codebasePath == types.EmptyString || path == types.EmptyString {
		return errors.New("codebasePath and path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, path)
	return os.RemoveAll(fullPath)
}

func (l *localCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	if codebasePath == types.EmptyString || path == types.EmptyString {
		return errors.New("codebasePath and path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, path)
	return os.MkdirAll(fullPath, defaultLocalDirMode)
}

func (l *localCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	if codebasePath == types.EmptyString {
		return false, errors.New("codebasePath cannot be empty")
	}

	fullPath := filepath.Join(codebasePath, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *localCodebase) Stat(ctx context.Context, codebasePath string, path string) (*types.FileInfo, error) {
	if codebasePath == types.EmptyString || path == types.EmptyString {
		return nil, errors.New("codebasePath and path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &types.FileInfo{
		Name:    info.Name(),
		Path:    fullPath,
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
	}, nil
}

func (l *localCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	if codebasePath == types.EmptyString {
		return nil, errors.New("codebasePath cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, dir)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []*types.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(entry.Name()) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(entry.Name()) {
			continue
		}

		files = append(files, &types.FileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
	}

	return files, nil
}

func (l *localCodebase) Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	if codebasePath == types.EmptyString {
		return nil, errors.New("codebasePath cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, dir)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var nodes []*types.TreeNode
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(entry.Name()) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(entry.Name()) {
			continue
		}

		node := &types.TreeNode{
			FileInfo: types.FileInfo{
				Name:    info.Name(),
				Path:    entry.Name(),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			},
			Children: make([]types.TreeNode, 0),
		}

		if info.IsDir() {
			subNodes, err := l.Tree(ctx, codebasePath, filepath.Join(dir, entry.Name()), option)
			if err != nil {
				continue
			}
			for _, subNode := range subNodes {
				node.Children = append(node.Children, *subNode)
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (l *localCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) ([]byte, error) {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (l *localCodebase) Walk(ctx context.Context, codebasePath string, dir string, walkFn WalkFunc) error {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	fullPath := filepath.Join(codebasePath, dir)
	return filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 构建 WalkContext
		walkCtx := &WalkContext{
			Path:         path,
			RelativePath: strings.TrimPrefix(path, fullPath),
			Info: &types.FileInfo{
				Name:    info.Name(),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			},
			ParentPath: filepath.Dir(path),
		}

		// 如果是目录，直接调用 walkFn，传入 nil reader
		if info.IsDir() {
			err := walkFn(walkCtx, nil)
			if errors.Is(err, SkipDir) {
				return filepath.SkipDir
			}
			return err
		}

		// 对于文件，打开并传入 reader
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		return walkFn(walkCtx, file)
	})
}

func (l *localCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("codebase path %s does not exist", codebasePath)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(paths))

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := l.Delete(ctx, codebasePath, p); err != nil {
				errChan <- fmt.Errorf("failed to delete %s: %w", p, err)
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	// 收集所有错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch delete errors: %v", errs)
	}
	return nil
}
