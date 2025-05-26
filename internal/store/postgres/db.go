package postgres

import (
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"sync"

	"github.com/zeromicro/go-zero/core/stores/postgres" // 引入 go-zero 自带 postgres 包
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	primaryDB  sqlx.SqlConn   // 主库连接（使用 go-zero 的 sqlx.SqlConn）
	replicasDB []sqlx.SqlConn // 复制库连接列表
	once       sync.Once
	mu         sync.RWMutex
	replicaIdx int // 轮询索引
)

// New 初始化数据库连接（单节点/主从模式兼容）
func New(c config.Config) error {
	once.Do(func() {
		// 初始化主库（使用 go-zero 的 postgres.New 方法）
		primaryDB = postgres.New(
			c.DBPrimary.DataSource,
		)

		// 初始化复制库
		replicasDB = make([]sqlx.SqlConn, 0, len(c.DBReplicas))
		for _, replicaConf := range c.DBReplicas {
			conn := postgres.New(
				replicaConf.DataSource,
			)
			replicasDB = append(replicasDB, conn)
		}
	})
	return nil
}

// getReadDB 获取读连接（轮询复制库或主库）
func getReadDB() sqlx.SqlConn {
	mu.RLock()
	defer mu.RUnlock()

	if len(replicasDB) > 0 {
		db := replicasDB[replicaIdx]
		replicaIdx = (replicaIdx + 1) % len(replicasDB) // 轮询逻辑
		return db
	}
	return primaryDB // 无复制库时使用主库
}

// getWriteDB 获取写连接（始终使用主库）
func getWriteDB() sqlx.SqlConn {
	return primaryDB
}
