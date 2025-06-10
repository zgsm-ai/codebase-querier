package vector

import (
	"context"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type Embedder interface {
	EmbedCodeChunks(ctx context.Context, chunks []*types.CodeChunk) ([]*CodeChunkEmbedding, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

type CodeChunkEmbedding struct {
	*types.CodeChunk
	Embedding []float32
}

type embedder struct {
	config           config.EmbedderConf
	embeddingService *openai.EmbeddingService
}

func NewEmbedder(cfg config.EmbedderConf) (Embedder, error) {

	embeddingService := openai.NewEmbeddingService(option.WithBaseURL(cfg.APIBase), option.WithAPIKey(cfg.APIKey))

	return &embedder{
		embeddingService: &embeddingService,
		config:           cfg,
	}, nil

}

func (e *embedder) EmbedCodeChunks(ctx context.Context, chunks []*types.CodeChunk) ([]*CodeChunkEmbedding, error) {
	embeds := make([]*CodeChunkEmbedding, 0, len(chunks))
	for _, chunk := range chunks {
		embeddings, err := e.EmbedQuery(ctx, string(chunk.Content))
		if err != nil {
			return nil, err
		}
		embeds = append(embeds, &CodeChunkEmbedding{
			CodeChunk: chunk,
			Embedding: embeddings,
		})
	}

	return embeds, nil
}

func (e *embedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(query),
		},
		Model:          e.config.Model,
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	}

	res, err := e.embeddingService.New(ctx, params,
		option.WithMaxRetries(e.config.MaxRetries),
		option.WithRequestTimeout(e.config.Timeout),
	)
	if err != nil {
		return nil, err
	}
	vectors := make([]float32, 0)
	for _, d := range res.Data {
		embeddings := make([]float32, 0)
		for _, v := range d.Embedding {
			embeddings = append(embeddings, float32(v))
		}
		vectors = append(vectors, embeddings...)
	}
	return vectors, nil
}
