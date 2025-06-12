package vector

import (
	"context"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/zeromicro/go-zero/core/logx"
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

type customEmbedder struct {
	config           config.EmbedderConf
	embeddingService *openai.EmbeddingService
}

func NewEmbedder(cfg config.EmbedderConf) (Embedder, error) {

	embeddingService := openai.NewEmbeddingService(option.WithBaseURL(cfg.APIBase), option.WithAPIKey(cfg.APIKey))

	return &customEmbedder{
		embeddingService: &embeddingService,
		config:           cfg,
	}, nil

}

func (e *customEmbedder) EmbedCodeChunks(ctx context.Context, chunks []*types.CodeChunk) ([]*CodeChunkEmbedding, error) {
	if len(chunks) == 0 {
		return []*CodeChunkEmbedding{}, nil
	}

	embeds := make([]*CodeChunkEmbedding, 0, len(chunks))
	batchSize := e.config.BatchSize
	logx.Infof("start to embedding %d chunks for codebase:%s, batchSize: %d ", len(chunks), chunks[0].CodebasePath, batchSize)

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
	logx.Infof("embedding %d chunks for codebase:%s successfully", len(chunks), chunks[0].CodebasePath)

	return embeds, nil
}

func (e *customEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if e.config.StripNewLines {
		query = strings.ReplaceAll(query, "\n", " ")
	}
	logx.Info("start to embed query")
	vectors, err := e.doEmbeddings(ctx, [][]byte{[]byte(query)})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, ErrEmptyResponse
	}
	logx.Info("embed query successfully")
	return vectors[0], nil
}

func (e *customEmbedder) doEmbeddings(ctx context.Context, textsByte [][]byte) ([][]float32, error) {
	texts := make([]string, len(textsByte))
	for i, b := range textsByte {
		texts[i] = string(b)
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
		transferredVector := make([]float32, 0, 768) //768维
		for _, v := range d.Embedding {
			transferredVector = append(transferredVector, float32(v))
		}
		vectors[d.Index] = transferredVector
	}
	return vectors, nil
}
