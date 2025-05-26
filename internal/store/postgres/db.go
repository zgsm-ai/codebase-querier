package postgres

import (
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"sync"

	"github.com/zeromicro/go-zero/core/stores/postgres" // 引入 go-zero 自带 postgres 包
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	db   sqlx.SqlConn
	once sync.Once
)

func New(c config.Config) sqlx.SqlConn {
	once.Do(func() {
		// 初始化主库（使用 go-zero 的 postgres.New 方法）
		db = postgres.New(
			c.DBPrimary.DataSource,
		)
	})
	return db
}
