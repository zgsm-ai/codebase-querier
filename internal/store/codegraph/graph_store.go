package codegraph

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// GraphStore defines the interface for graph storage
type GraphStore interface {
	// Save saves the graph data to storage
	Save(ctx context.Context, codebaseId int64, codebasePath string, nodes []*types.GraphNode) error

	// Query queries the graph data based on the given options
	Query(ctx context.Context, req *types.RelationQueryOptions) ([]*types.GraphNode, error)

	// DeleteAll deletes all data for a codebase
	DeleteAll(ctx context.Context, codebaseId int64, codebasePath string) error

	// Close closes the storage
	Close() error
}
