package vector

import (
	"context"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"strings"
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
	if len(chunks) == 0 {
		return []*CodeChunkEmbedding{}, nil
	}

	embeds := make([]*CodeChunkEmbedding, 0, len(chunks))
	batchSize := e.config.BatchSize

	for start := 0; start < len(chunks); start += batchSize {
		end := start + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		// 准备当前批次的内容
		batch := make([][]byte, end-start)
		for i := 0; i < end-start; i++ {
			batch[i] = chunks[start+i].Content
		}

		// 执行嵌入
		embeddings, err := e.doEmbeddings(ctx, batch)
		if err != nil {
			return nil, err
		}

		// 将嵌入结果与原始块关联
		for i, em := range embeddings {
			embeds = append(embeds, &CodeChunkEmbedding{
				CodeChunk: chunks[start+i],
				Embedding: em,
			})
		}
	}

	return embeds, nil
}

func (e *embedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if e.config.StripNewLines {
		query = strings.ReplaceAll(query, "\n", " ")
	}
	vectors, err := e.doEmbeddings(ctx, [][]byte{[]byte(query)})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, ErrEmptyResponse
	}
	return vectors[0], nil
}

func (e *embedder) doEmbeddings(ctx context.Context, textsByte [][]byte) ([][]float32, error) {
	texts := make([]string, len(textsByte))
	for _, b := range textsByte {
		texts = append(texts, string(b))
	}

	// 批量处理
	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
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
	vectors := make([][]float32, len(textsByte), len(textsByte))
	for _, d := range res.Data {
		transferredVector := make([]float32, 0)
		for _, v := range d.Embedding {
			transferredVector = append(transferredVector, float32(v))
		}
		vectors[d.Index] = transferredVector
	}
	return vectors, nil
}
