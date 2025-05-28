package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ SyncHistoryModel = (*customSyncHistoryModel)(nil)

type (
	// SyncHistoryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSyncHistoryModel.
	SyncHistoryModel interface {
		syncHistoryModel
		withSession(session sqlx.Session) SyncHistoryModel
	}

	customSyncHistoryModel struct {
		*defaultSyncHistoryModel
	}
)

// NewSyncHistoryModel returns a model for the database table.
func NewSyncHistoryModel(conn sqlx.SqlConn) SyncHistoryModel {
	return &customSyncHistoryModel{
		defaultSyncHistoryModel: newSyncHistoryModel(conn),
	}
}

func (m *customSyncHistoryModel) withSession(session sqlx.Session) SyncHistoryModel {
	return NewSyncHistoryModel(sqlx.NewSqlConnFromSession(session))
}
