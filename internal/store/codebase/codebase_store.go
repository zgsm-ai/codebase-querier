package codebase

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
)

// Store 存储代码仓库文件
type Store interface {
	// Init 初始化代码仓库
	Init(ctx context.Context, codebase types.Codebase) error

	// Add 将代码文件添加到目标路径
	Add(ctx context.Context, codebasePath string, source io.Reader, target string) error

	// Unzip 将zip文件解压到目标路径
	Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error

	// Delete 删除文件或目录
	Delete(ctx context.Context, codebasePath string, path string) error

	// MkDirs 创建目录
	MkDirs(ctx context.Context, codebasePath string, path string) error

	// Exists 检查路径是否存在
	Exists(ctx context.Context, codebasePath string, path string) (bool, error)

	// Stat 获取文件或目录的元信息
	Stat(ctx context.Context, codebasePath string, path string) (types.FileInfo, error)

	// List 列出目录内容
	List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error)

	// Tree 构建目录树结构
	Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error)

	// Read 读取文件内容
	Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) (string, error)

	// Walk 递归遍历目录并处理每个文件
	Walk(ctx context.Context, codebasePath string, dir string, process func(io.ReadCloser) (bool, error)) error

	// BatchDelete 批量删除文件或目录
	BatchDelete(ctx context.Context, codebasePath string, paths []string) error
}
