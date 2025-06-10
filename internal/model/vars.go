package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var ErrNotFound = sqlx.ErrNotFound

// TODO pgsql  unique index conflict error
var UniqueIndexConflictErr error

const CodebaseStatusActive = "active"
const CodebaseStatusExpired = "expired"

type PublishStatus string

const (
	PublishStatusPending PublishStatus = "pending"
	PublishStatusSuccess PublishStatus = "success"
	PublishStatusFailed  PublishStatus = "failed"
)
