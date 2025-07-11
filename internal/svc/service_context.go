package svc

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

type ServiceContext struct {
	Config        config.Config
	serverContext context.Context
}

// Close closes the shared Redis client and database connection
func (s *ServiceContext) Close() {
	var errs []error
	if len(errs) > 0 {
		logx.Errorf("service_context close err:%v", errs)
	} else {
		logx.Infof("service_context close successfully.")
	}
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	var err error
	svcCtx := &ServiceContext{
		Config:        c,
		serverContext: ctx,
	}

	return svcCtx, err
}
