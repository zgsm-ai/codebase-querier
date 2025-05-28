package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ CodebaseModel = (*customCodebaseModel)(nil)

type (
	// CodebaseModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCodebaseModel.
	CodebaseModel interface {
		codebaseModel
		withSession(session sqlx.Session) CodebaseModel
	}

	customCodebaseModel struct {
		*defaultCodebaseModel
	}
)

// NewCodebaseModel returns a model for the database table.
func NewCodebaseModel(conn sqlx.SqlConn) CodebaseModel {
	return &customCodebaseModel{
		defaultCodebaseModel: newCodebaseModel(conn),
	}
}

func (m *customCodebaseModel) withSession(session sqlx.Session) CodebaseModel {
	return NewCodebaseModel(sqlx.NewSqlConnFromSession(session))
}
