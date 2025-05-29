package model

import (
	"context"
	"database/sql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"time"
)

var _ IndexHistoryModel = (*customIndexHistoryModel)(nil)

type (
	// IndexHistoryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customIndexHistoryModel.
	IndexHistoryModel interface {
		indexHistoryModel
		withSession(session sqlx.Session) IndexHistoryModel
		UpdateStatus(ctx context.Context, id int64, status string, errMessage string) error
	}

	customIndexHistoryModel struct {
		*defaultIndexHistoryModel
	}
)

func (m *customIndexHistoryModel) UpdateStatus(ctx context.Context, id int64, status string, errMessage string) error {
	one, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}
	one.Status = status
	one.ErrorMessage = sql.NullString{String: errMessage}
	one.EndTime = sql.NullTime{Time: time.Now()}
	return m.Update(ctx, one)
}

// NewIndexHistoryModel returns a model for the database table.
func NewIndexHistoryModel(conn sqlx.SqlConn) IndexHistoryModel {
	return &customIndexHistoryModel{
		defaultIndexHistoryModel: newIndexHistoryModel(conn),
	}
}

func (m *customIndexHistoryModel) withSession(session sqlx.Session) IndexHistoryModel {
	return NewIndexHistoryModel(sqlx.NewSqlConnFromSession(session))
}
