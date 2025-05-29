package vector

import (
	"context"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type Embedder interface {
	embeddings.Embedder
	EmbedCodeChunks(ctx context.Context, chunks []types.CodeChunk) ([][]float32, error)
}

type embedder struct {
	*embeddings.EmbedderImpl
	logger logx.Logger
}

func (e *embedder) EmbedCodeChunks(ctx context.Context, chunks []types.CodeChunk) ([][]float32, error) {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}
	return e.EmbedDocuments(ctx, texts)
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
