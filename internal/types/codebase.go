package types

import (
	"encoding/json"
	"io/fs"
	"time"
)

type Codebase struct {
	FullPath string
}

const CodebaseIndexDir = ".shenma"
const SyncMedataDir = ".shenma_sync"
const IndexFileName = "index.scip"

// SyncMetadataFile 元数据同步文件
type SyncMetadataFile struct {
	ClientID      string            `json:"clientId"`      // 客户端ID（可选）
	CodebasePath  string            `json:"codebasePath"`  // 项目根路径
	ExtraMetadata json.RawMessage   `json:"extraMetadata"` // 扩展元数据（保留原始JSON）
	FileList      map[string]string `json:"fileList"`      // 文件变更列表（路径→操作类型）
	Timestamp     int64             `json:"timestamp"`     // 时间戳（Unix毫秒）
}

type FileOp string

const (
	FileOpAdd    = "add"
	FileOpModify = "modify"
	FileOpDelete = "delete"
)

type SyncFile struct {
	Path string
	Op   FileOp
}

// TreeNode 表示目录树中的一个节点，可以是目录或文件
type TreeNode struct {
	FileInfo
	Children []*TreeNode `json:"children,omitempty"` // 子节点（仅目录有）
}

type FileInfo struct {
	Name    string    `json:"Language"`          // 节点名称
	Path    string    `json:"path"`              // 节点路径
	Size    int64     `json:"size,omitempty"`    // 文件大小（仅文件有）
	ModTime time.Time `json:"modTime,omitempty"` // 修改时间（可选）
	IsDir   bool      `json:"IsDir"`             // 是否是目录
	Mode    fs.FileMode
}
