package types

import (
	"time"
)

// CodebaseSyncMessage 表示从 Redis 消息队列接收的代码库同步消息
type CodebaseSyncMessage struct {
	SyncID       int64     `json:"syncId"`       // 同步操作ID
	CodebaseID   int64     `json:"codebaseId"`   // 代码库ID
	CodebasePath string    `json:"codebasePath"` // 代码库路径
	CodebaseName string    `json:"codebaseName"` // 代码库名字
	SyncTime     time.Time `json:"syncTime"`     // 同步结束时间
}
