package vector

import (
	"context"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

type Embedder interface {
	embeddings.Embedder
}

type embedder struct {
	*embeddings.EmbedderImpl
	logger logx.Logger
}

func NewEmbedder(ctx context.Context, cfg config.EmbedderConf) (Embedder, error) {
	embeddingClient, err := openai.New(
		openai.WithBaseURL(cfg.APIBase),
		openai.WithEmbeddingModel(cfg.Model),
		openai.WithToken(cfg.APIKey),
	)
	embedderImpl, err := embeddings.NewEmbedder(
		embeddingClient,
		embeddings.WithBatchSize(cfg.BatchSize),
		embeddings.WithStripNewLines(cfg.StripNewLines),
	)
	if err != nil {
		return nil, err
	}
	return &embedder{
		EmbedderImpl: embedderImpl,
		logger:       logx.WithContext(ctx),
	}, nil

}
