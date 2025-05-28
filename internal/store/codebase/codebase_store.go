package codebase

import (
	"context"
	"io"
)

// CodebaseStore 存储代码仓库文件
type CodebaseStore interface {
	// Add 将代码文件添加到目标路径
	Add(ctx context.Context, source io.Reader, target string) error

	// Unzip 将zip文件解压到目标路径
	Unzip(ctx context.Context, source io.Reader, target string) error

	// Delete 删除文件或目录
	Delete(ctx context.Context, path string) error

	// Exists 检查路径是否存在
	Exists(ctx context.Context, path string) (bool, error)

	// Stat 获取文件或目录的元信息
	Stat(ctx context.Context, path string) (FileInfo, error)

	// List 列出目录内容
	List(ctx context.Context, dir string, opts ...ListOption) ([]*FileInfo, error)

	// Tree 构建目录树结构
	Tree(ctx context.Context, dir string, opts ...TreeOption) ([]*TreeNode, error)

	// Read 读取文件内容
	Read(ctx context.Context, filePath string) (io.ReadCloser, error)

	// Walk 递归遍历目录并处理每个文件
	Walk(ctx context.Context, dir string, process func(io.ReadCloser) error) error

	// BatchDelete 批量删除文件或目录
	BatchDelete(ctx context.Context, paths []string) error
}
