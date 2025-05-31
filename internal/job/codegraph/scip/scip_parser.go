package scip

import (
	"fmt"
	"os"

	"github.com/sourcegraph/scip/bindings/go/scip"
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

// ParseSCIPFileForGraph parses a SCIP index file and returns its relationship graph.
func ParseSCIPFileForGraph(scipFilePath string) (map[string]*SymbolNode, error) {
	// Verify file exists
	if _, err := os.Stat(scipFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SCIP file does not exist: %s", scipFilePath)
	}

	// Open file
	file, err := os.Open(scipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SCIP file: %w", err)
	}
	defer file.Close()

	// Check if file is empty
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("empty SCIP file")
	}

	// Initialize symbol nodes map
	symbolNodes := make(map[string]*SymbolNode)

	// Create visitor
	visitor := scip.IndexVisitor{
		VisitMetadata: func(m *scip.Metadata) {
			// Store metadata if needed
		},
		VisitDocument: func(d *scip.Document) {
			// Process document symbols
			for _, s := range d.Symbols {
				// Create or update symbol node
				node, exists := symbolNodes[s.Symbol]
				if !exists {
					node = &SymbolNode{
						SymbolName:       s.Symbol,
						SymbolInfo:       s,
						ReferenceOccs:    make([]*scip.Occurrence, 0),
						RelationshipsOut: make(map[types.NodeType][]string),
					}
					symbolNodes[s.Symbol] = node
				} else if node.SymbolInfo == nil {
					node.SymbolInfo = s
				}

				// Process relationships
				for _, rel := range s.Relationships {
					var relType types.NodeType
					switch {
					case rel.IsImplementation:
						relType = types.NodeTypeImplementation
					case rel.IsReference:
						relType = types.NodeTypeReference
					case rel.IsTypeDefinition:
						relType = types.NodeTypeDefinition
					default:
						continue
					}

					// Add relationship
					node.RelationshipsOut[relType] = append(node.RelationshipsOut[relType], rel.Symbol)
				}
			}

			// Process occurrences
			for _, occ := range d.Occurrences {
				if occ.Symbol == types.EmptyString {
					continue
				}

				// Create or update symbol node
				node, exists := symbolNodes[occ.Symbol]
				if !exists {
					node = &SymbolNode{
						SymbolName:       occ.Symbol,
						ReferenceOccs:    make([]*scip.Occurrence, 0),
						RelationshipsOut: make(map[types.NodeType][]string),
					}
					symbolNodes[occ.Symbol] = node
				}

				if scip.SymbolRole_Definition.Matches(occ) {
					node.DefinitionOcc = occ
				} else {
					node.ReferenceOccs = append(node.ReferenceOccs, occ)
				}
			}
		},
		VisitExternalSymbol: func(s *scip.SymbolInformation) {
			// Create or update symbol node
			node, exists := symbolNodes[s.Symbol]
			if !exists {
				node = &SymbolNode{
					SymbolName:       s.Symbol,
					SymbolInfo:       s,
					ReferenceOccs:    make([]*scip.Occurrence, 0),
					RelationshipsOut: make(map[types.NodeType][]string),
				}
				symbolNodes[s.Symbol] = node
			} else if node.SymbolInfo == nil {
				node.SymbolInfo = s
			}

			// Process relationships
			for _, rel := range s.Relationships {
				// Determine relationship type
				var relType types.NodeType
				switch {
				case rel.IsImplementation:
					relType = types.NodeTypeImplementation
				case rel.IsReference:
					relType = types.NodeTypeReference
				case rel.IsTypeDefinition:
					relType = types.NodeTypeDefinition
				default:
					continue
				}

				// Add relationship
				node.RelationshipsOut[relType] = append(node.RelationshipsOut[relType], rel.Symbol)
			}
		},
	}

	// Parse SCIP file
	if err := visitor.ParseStreaming(file); err != nil {
		return nil, fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	// Verify that we parsed some symbols
	if len(symbolNodes) == 0 {
		return nil, fmt.Errorf("no symbols found in SCIP file")
	}

	return symbolNodes, nil
}
