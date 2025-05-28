package codebase

import "time"

// TreeNode 表示目录树中的一个节点，可以是目录或文件
type TreeNode struct {
	FileInfo
	Children []TreeNode `json:"children,omitempty"` // 子节点（仅目录有）
}

type FileInfo struct {
	Name    string    `json:"name"`              // 节点名称
	Type    string    `json:"type"`              // 节点类型："directory" 或 "file"
	Path    string    `json:"path"`              // 节点路径
	Size    int64     `json:"size,omitempty"`    // 文件大小（仅文件有）
	ModTime time.Time `json:"modTime,omitempty"` // 修改时间（可选）
	IsDir   bool      `json:"isDir"`             // 是否是目录
}

// ListOption 定义List方法的可选参数
type ListOption func(*ListOptions)

// ListOptions 包含List方法的可选参数
type ListOptions struct {
	Recursive bool   // 是否递归列出子目录
	Filter    string // 文件名过滤模式
	Limit     int    // 返回结果数量限制
	Offset    int    // 结果偏移量
}

// TreeOption 定义Tree方法的可选参数
type TreeOption func(*TreeOptions)

// TreeOptions 包含Tree方法的可选参数
type TreeOptions struct {
	MaxDepth int    // 最大递归深度
	Filter   string // 文件名过滤模式
}
