package codebase

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

var codebaseLocal = "local"

func New(ctx context.Context, cfg config.CodeBaseStoreConf) (Store, error) {
	switch cfg.Type {
	case codebaseLocal:
		if cfg.Local.BasePath == types.EmptyString {
			return nil, errors.New("codebase local config is required for local type")
		}
		return newLocalCodeBase(ctx, cfg), nil
	default:
		return nil, fmt.Errorf("unsupported codebase type: %s", cfg.Type)
	}
}
