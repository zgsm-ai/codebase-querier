package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"net/http"
	"sort"
	"time"
)

const (
	rerankQuery     = "query"
	rerankDocuments = "documents"
	rerankScores    = "scores"
)

type Reranker interface {
	Rerank(ctx context.Context, query string, docs []*types.SemanticFileItem) ([]*types.SemanticFileItem, error)
}

type rerank struct {
	config config.RerankerConf
}

func (r *rerank) Rerank(ctx context.Context, query string, docs []*types.SemanticFileItem) ([]*types.SemanticFileItem, error) {
	if len(docs) == 0 {
		return docs, nil
	}

	requestBody := map[string]any{
		rerankQuery: query,
		rerankDocuments: func() []string {
			contents := make([]string, len(docs))
			for i, doc := range docs {
				contents[i] = doc.Content
			}
			return contents
		}(),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rerank request body: %w", err)
	}

	rerankEndpoint := r.config.APIBase

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rerankEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create rerank request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send rerank request to %s: %w", rerankEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody := new(bytes.Buffer)
		errorBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("rerank API returned non-OK status %d: %s, body: %s", resp.StatusCode, resp.Status, errorBody.String())
	}

	var responseBody map[string][]float64

	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("failed to decode rerank response body: %w", err)
	}

	scores, ok := responseBody[rerankScores]
	if !ok || len(scores) != len(docs) {
		return nil, fmt.Errorf("invalid rerank API response format or score count mismatch: expected %d scores, got %d", len(docs), len(scores))
	}

	scoredDocs := make([]struct {
		Doc   *types.SemanticFileItem
		Score float64
		Index int
	}, len(docs))

	for i := range docs {
		scoredDocs[i] = struct {
			Doc   *types.SemanticFileItem
			Score float64
			Index int
		}{
			Doc:   docs[i],
			Score: scores[i],
			Index: i,
		}
	}

	sort.SliceStable(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].Score > scoredDocs[j].Score
	})

	rerankedDocs := make([]*types.SemanticFileItem, len(docs))
	for i, sd := range scoredDocs {
		rerankedDocs[i] = sd.Doc
	}

	return rerankedDocs, nil
}

func NewReranker(c config.RerankerConf) Reranker {
	return &rerank{
		config: c,
	}
}
