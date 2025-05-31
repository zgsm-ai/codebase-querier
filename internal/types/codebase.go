package types

import "encoding/json"

type Codebase struct {
	ClientID string
	Path     string
}

const SyncMedataDir = ".sync_metadata"
const CodebaseIndexDir = ".codebase_index"

// SyncMetadata 代码变更事件结构体
type SyncMetadata struct {
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
