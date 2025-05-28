package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ IndexHistoryModel = (*customIndexHistoryModel)(nil)

type (
	// IndexHistoryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customIndexHistoryModel.
	IndexHistoryModel interface {
		indexHistoryModel
		withSession(session sqlx.Session) IndexHistoryModel
	}

	customIndexHistoryModel struct {
		*defaultIndexHistoryModel
	}
)

// NewIndexHistoryModel returns a model for the database table.
func NewIndexHistoryModel(conn sqlx.SqlConn) IndexHistoryModel {
	return &customIndexHistoryModel{
		defaultIndexHistoryModel: newIndexHistoryModel(conn),
	}
}

func (m *customIndexHistoryModel) withSession(session sqlx.Session) IndexHistoryModel {
	return NewIndexHistoryModel(sqlx.NewSqlConnFromSession(session))
}
