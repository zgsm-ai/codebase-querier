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

func NewLocalCodebase(ctx context.Context, cfg config.CodeBaseStoreConf) Store {
	return &localCodebase{
		cfg:    cfg,
		logger: logx.WithContext(ctx),
	}
}

// getFullPath 获取完整的文件路径
func (l *localCodebase) getFullPath(clientId, codebasePath, target string) string {
	return getFullPath(l.cfg.Local.BasePath, clientId, codebasePath, target)
}

// Init 初始化一个新的代码库
func (l *localCodebase) Init(ctx context.Context, clientId string, codebasePath string) (*types.Codebase, error) {
	if clientId == "" || codebasePath == "" {
		return nil, errors.New("clientId and codebasePath cannot be empty")
	}

	// 生成唯一的路径
	targetPath := l.getFullPath(clientId, codebasePath, "")

	// 创建目录
	if err := os.MkdirAll(targetPath, defaultLocalDirMode); err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	return &types.Codebase{
		ClientID: clientId,
		Path:     codebasePath,
	}, nil
}

func (l *localCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	fullPath := l.getFullPath("", codebasePath, target)
	if err := os.MkdirAll(filepath.Dir(fullPath), defaultLocalDirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultLocalFileMode)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, source); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

func (l *localCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	// Create a temporary file to store the zip content
	tmpFile, err := os.CreateTemp("", "codebase-*.zip")
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

	basePath := l.getFullPath("", codebasePath, target)

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
	fullPath := l.getFullPath("", codebasePath, path)
	return os.RemoveAll(fullPath)
}

func (l *localCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	fullPath := l.getFullPath("", codebasePath, path)
	return os.MkdirAll(fullPath, defaultLocalDirMode)
}

func (l *localCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	fullPath := l.getFullPath("", codebasePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *localCodebase) Stat(ctx context.Context, codebasePath string, path string) (*types.FileInfo, error) {
	fullPath := l.getFullPath("", codebasePath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &types.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
	}, nil
}

func (l *localCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	fullPath := l.getFullPath("", codebasePath, dir)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []*types.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			l.logger.Errorf("failed to get file info for %s: %v", entry.Name(), err)
			continue
		}

		// Apply filters if specified
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(entry.Name()) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(entry.Name()) {
			continue
		}

		files = append(files, &types.FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
	}

	return files, nil
}

func (l *localCodebase) Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	fullPath := l.getFullPath("", codebasePath, dir)

	// 构建路径到节点的映射
	nodeMap := make(map[string]*types.TreeNode)
	var rootNodes []*types.TreeNode

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录
		if path == fullPath {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(fullPath, path)
		if err != nil {
			return err
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 处理路径的每一级
		parts := strings.Split(relPath, string(filepath.Separator))
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
					IsDir:   i < len(parts)-1 || info.IsDir(),
					Size:    info.Size(),
					ModTime: info.ModTime(),
					Mode:    info.Mode(),
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

		return nil
	})

	return rootNodes, err
}

func (l *localCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) (string, error) {
	fullPath := l.getFullPath("", codebasePath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (l *localCodebase) Walk(ctx context.Context, codebasePath string, dir string, process func(io.ReadCloser) (bool, error)) error {
	fullPath := l.getFullPath("", codebasePath, dir)
	return filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		stop, err := process(file)
		if err != nil {
			return err
		}
		if stop {
			return filepath.SkipAll
		}

		return nil
	})
}

func (l *localCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
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
