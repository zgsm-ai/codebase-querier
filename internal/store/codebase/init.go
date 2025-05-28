package codebase

import (
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

var codebaseLocal = "local"

func New(cfg config.CodeBaseStoreConf) (CodebaseStore, error) {
	switch cfg.Type {
	case codebaseLocal:
		if cfg.Local == nil {
			return nil, errors.New("codebase local config is required for local type")
		}
		return newLocalCodeBase(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported codebase type: %s", cfg.Type)
	}
}
