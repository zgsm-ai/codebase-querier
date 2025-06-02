package scip

import (
	"context"
	"fmt"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type InternalGraph struct {
	Metadata *scip.Metadata

	Nodes map[string]*InternalGraphNode

	Documents   []*scip.Document
	Occurrences []*scip.Occurrence
}

// InternalGraphNode represents a symbol as a node in our internal graph.
type InternalGraphNode struct {
	SymbolName         string
	SymbolInfo         *scip.SymbolInformation
	Relationships      map[types.NodeType][]RelationshipTarget
	DefinitionPosition *types.Position
}

// RelationshipTarget represents the target of a relationship and the position of the occurrence that creates this relationship.
type RelationshipTarget struct {
	TargetSymbolName string
	Position         types.Position
}

func toPosition(rng []int32) types.Position {
	pos := types.Position{}

	// Ensure range has at least line and character
	if len(rng) >= 2 {
		pos.StartLine = int(rng[0]) + 1
		pos.StartColumn = int(rng[1]) + 1
		pos.EndLine = pos.StartLine
		pos.EndColumn = pos.StartColumn
	}

	// If more elements are present, they specify the end position
	if len(rng) >= 4 {
		pos.EndLine = int(rng[2]) + 1
		pos.EndColumn = int(rng[3]) + 1
	} else if len(rng) == 3 {
		pos.EndLine = int(rng[2]) + 1
		pos.EndColumn = pos.StartColumn
	}

	return pos
}

// SymbolNode represents a symbol and all its associated information from SCIP.
type SymbolNode struct {
	SymbolName       string
	SymbolInfo       *scip.SymbolInformation
	DefinitionOcc    *scip.Occurrence
	ReferenceOccs    []*scip.Occurrence
	RelationshipsOut map[types.NodeType][]string
}

// IndexParser represents the SCIP index generator
type IndexParser struct {
	codebaseStore codebase.Store
	graphStore    codegraph.GraphStore
}

// NewIndexParser  creates a new SCIP index generator
func NewIndexParser(codebaseStore codebase.Store, graphStore codegraph.GraphStore) *IndexParser {
	return &IndexParser{
		codebaseStore: codebaseStore,
		graphStore:    graphStore,
	}
}

// ParseSCIPFileForGraph parses a SCIP index file and saves its data to graphStore.
func (i *IndexParser) ParseSCIPFileForGraph(ctx context.Context, codebasePath, scipFilePath string) error {
	// Verify file exists
	fileStat, err := i.codebaseStore.Stat(ctx, codebasePath, scipFilePath)
	if err != nil {
		return err
	}
	if fileStat == nil || fileStat.IsDir {
		return fmt.Errorf("SCIP file does not exist: %s", scipFilePath)
	}
	if fileStat.Size == 0 {
		return fmt.Errorf("empty SCIP file: %s", scipFilePath)
	}

	// Open file
	file, err := i.codebaseStore.Open(ctx, codebasePath, scipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open SCIP file: %w", err)
	}
	defer file.Close()

	// Collect document information
	documentCountByPath := make(map[string]int)
	repeatedDocumentsByPath := make(map[string][]*scip.Document)
	var nodes []*types.GraphNode

	visitor := scip.IndexVisitor{
		VisitDocument: func(d *scip.Document) {
			path := d.RelativePath
			documentCountByPath[path]++

			// Process repeated documents
			if count := documentCountByPath[path]; count > 1 {
				samePathDocs := append(repeatedDocumentsByPath[path], d)
				repeatedDocumentsByPath[path] = samePathDocs

				// When all documents with the same path are collected, merge them
				if len(samePathDocs) == count {
					flattenedDocs := scip.FlattenDocuments(samePathDocs)
					if len(flattenedDocs) != 1 {
						return
					}
					d = flattenedDocs[0]
				} else {
					return
				}
			}

			// Process document symbols
			for _, s := range d.Symbols {
				// Process relationships
				for _, rel := range s.Relationships {
					var nodeType types.NodeType
					switch {
					case rel.IsImplementation:
						nodeType = types.NodeTypeImplementation
					case rel.IsReference:
						nodeType = types.NodeTypeReference
					case rel.IsTypeDefinition:
						nodeType = types.NodeTypeDefinition
					default:
						continue
					}

					node := &types.GraphNode{
						FilePath:   d.RelativePath,
						SymbolName: rel.Symbol,
						Position:   types.Position{}, // No position information
						NodeType:   nodeType,
					}
					nodes = append(nodes, node)
				}
			}

			// Process occurrences
			for _, occ := range d.Occurrences {
				if occ.Symbol == types.EmptyString {
					continue
				}

				nodeType := types.NodeTypeReference
				if scip.SymbolRole_Definition.Matches(occ) {
					nodeType = types.NodeTypeDefinition
				}

				node := &types.GraphNode{
					FilePath:   d.RelativePath,
					SymbolName: occ.Symbol,
					Position:   toPosition(occ.Range),
					NodeType:   nodeType,
				}
				nodes = append(nodes, node)
			}
		},
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in SCIP file")
	}

	return i.graphStore.Save(ctx, 0, codebasePath, nodes)
}
