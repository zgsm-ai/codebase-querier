package codegraph

import (
	"context"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
)

type scipBuilder struct {
	svcCtx *svc.ServiceContext
}

func NewScipBuilder(svcCtx *svc.ServiceContext) GraphBuilder {
	return &scipBuilder{
		svcCtx: svcCtx,
	}
}

func (s scipBuilder) Build(ctx context.Context, codebasePath string) ([]*types.GraphNode, error) {
	// generate index.scip

	file, err := os.OpenFile(codebasePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	visitor := scip.IndexVisitor{}

	err = visitor.ParseStreaming(file)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
