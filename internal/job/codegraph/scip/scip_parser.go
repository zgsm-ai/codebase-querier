package scip

import (
	"fmt"
	"os"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// Graph represents the relationship graph extracted from a SCIP index.
// This will be our internal representation before converting to the user's GraphNode structure.
type InternalGraph struct {
	Metadata *scip.Metadata
	// Nodes map symbol names to their node information
	Nodes map[string]*InternalGraphNode
	// Store documents and occurrences temporarily if needed for detailed processing
	Documents   []*scip.Document
	Occurrences []*scip.Occurrence // Collected from all documents
}

// InternalGraphNode represents a symbol as a node in our internal graph.
type InternalGraphNode struct {
	SymbolName string                  // The full SCIP symbol name
	SymbolInfo *scip.SymbolInformation // The actual symbol information from SCIP
	// Relationships map relationship type to a list of targets
	Relationships map[types.NodeType][]RelationshipTarget // Use user's NodeType
	// DefinitionPosition store the position if this node represents a definition
	DefinitionPosition *types.Position
}

// RelationshipTarget represents the target of a relationship and the position of the occurrence that creates this relationship.
type RelationshipTarget struct {
	TargetSymbolName string         // The full SCIP symbol name of the target
	Position         types.Position // The position of the occurrence causing this relationship
}

// Helper function to convert scip.Range (which is []int32) to types.Position.
// SCIP ranges are 0-indexed (line, character).
// User's Position is 1-indexed (line, column).
// SCIP range formats can be [line, character], [startLine, startChar, endLine, endChar], or [startLine, startChar, endLine].
func toPosition(rng []int32) types.Position {
	pos := types.Position{}

	// Ensure range has at least line and character
	if len(rng) >= 2 {
		pos.StartLine = int(rng[0]) + 1
		pos.StartColumn = int(rng[1]) + 1
		// Assume end is same as start if not full range
		pos.EndLine = pos.StartLine
		pos.EndColumn = pos.StartColumn
	}

	// If more elements are present, they specify the end position
	if len(rng) >= 4 {
		pos.EndLine = int(rng[2]) + 1
		pos.EndColumn = int(rng[3]) + 1
	} else if len(rng) == 3 {
		pos.EndLine = int(rng[2]) + 1
		// If only end line is provided, assume end column is same as start column, or end of line depending on desired behavior.
		// Sticking to matching start column for simplicity if not a full 4-element range.
		pos.EndColumn = pos.StartColumn
	}

	return pos
}

// // ParseSCIPFileForGraph parses a SCIP index file and returns its relationship graph.
// // This version builds an internal graph structure first.
// func ParseSCIPFileForGraph(scipFilePath string) (*InternalGraph, error) {
// 	file, err := os.Open(scipFilePath)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to open SCIP file %s: %w", scipFilePath, err)
// 	}
// 	defer file.Close()

// 	internalGraph := &InternalGraph{
// 		Nodes:       make(map[string]*InternalGraphNode),
// 		Documents:   make([]*scip.Document, 0),
// 		Occurrences: make([]*scip.Occurrence, 0),
// 	}

// 	// First pass to collect basic info, documents, symbols within documents, and external symbols
// 	visitor := scip.IndexVisitor{
// 		VisitMetadata: func(m *scip.Metadata) {
// 			internalGraph.Metadata = m
// 		},
// 		VisitDocument: func(d *scip.Document) {
// 			internalGraph.Documents = append(internalGraph.Documents, d)
// 			// Collect symbols and occurrences nested within the document during this pass
// 			for _, s := range d.Symbols {
// 				// Add symbol to nodes map, initialize relationships
// 				if _, exists := internalGraph.Nodes[s.Symbol]; !exists {
// 					internalGraph.Nodes[s.Symbol] = &InternalGraphNode{
// 						SymbolName:    s.Symbol,
// 						SymbolInfo:    s, // Store symbol info
// 						Relationships: make(map[types.NodeType][]RelationshipTarget),
// 					}
// 				} else {
// 					// If node already exists (e.g., from ExternalSymbols), merge SymbolInfo if missing
// 					if internalGraph.Nodes[s.Symbol].SymbolInfo == nil {
// 						internalGraph.Nodes[s.Symbol].SymbolInfo = s
// 					}
// 				}

// 				// Process relationships defined within this symbol's SymbolInformation
// 				for _, rel := range s.Relationships {
// 					targetNode, exists := internalGraph.Nodes[rel.Symbol]
// 					if !exists {
// 						// Create target node if it doesn't exist yet (e.g., a referenced external symbol)
// 						targetNode = &InternalGraphNode{
// 							SymbolName:    rel.Symbol,
// 							SymbolInfo:    nil, // SymbolInfo will be filled if we encounter its definition or external info
// 							Relationships: make(map[types.NodeType][]RelationshipTarget),
// 						}
// 						internalGraph.Nodes[rel.Symbol] = targetNode
// 					}

// 					// Determine relationship type based on SCIP relationship flags
// 					var relationType types.NodeType
// 					// Map SCIP relationship kinds to our NodeTypes
// 					if rel.IsImplementation {
// 						relationType = types.NodeTypeImplementation
// 					} else if rel.IsReference { // SCIP IsReference is often implicit in occurrences, but can appear here
// 						// This might need refinement based on how SCIP tools populate this
// 						// For now, let's map IsReference in SymbolInfo to NodeTypeReference if it appears
// 						relationType = types.NodeTypeReference
// 					} else if rel.IsTypeDefinition {
// 						// SCIP doesn't have explicit 'inheritance', often implementation or reference + type def
// 						// Let's map IsTypeDefinition to something relevant, maybe related to definitions or references
// 						// For now, skip explicit mapping for IsTypeDefinition unless it clearly means inheritance.
// 						continue // Skip for now
// 					} else {
// 						// Handle other potential relationship flags if needed
// 						continue // Skip unknown relationship types
// 					}

// 					// Add the relationship to the source node (s.Symbol)
// 					sourceNode := internalGraph.Nodes[s.Symbol]
// 					// Note: Relationships in SymbolInfo don't have specific ranges associated with them
// 					// The range is associated with the Occurrence. We might need a second pass for occurrences.
// 					// For now, we'll add the relationship without position information from SymbolInfo.
// 					// We will add position information from Occurrences later.
// 					sourceNode.Relationships[relationType] = append(sourceNode.Relationships[relationType], RelationshipTarget{
// 						TargetSymbolName: rel.Symbol,
// 						Position:         types.Position{}, // Position will be added from occurrences
// 					})
// 				}
// 			}

// 			// Collect occurrences from documents
// 			internalGraph.Occurrences = append(internalGraph.Occurrences, d.Occurrences...)
// 		},
// 		VisitExternalSymbol: func(s *scip.SymbolInformation) {
// 			// Add external symbol to nodes map or merge info if already exists
// 			if node, exists := internalGraph.Nodes[s.Symbol]; exists {
// 				if node.SymbolInfo == nil {
// 					node.SymbolInfo = s // Add symbol info if it wasn't a definition in a document
// 				}
// 			} else {
// 				internalGraph.Nodes[s.Symbol] = &InternalGraphNode{
// 					SymbolName:    s.Symbol,
// 					SymbolInfo:    s, // Store symbol info
// 					Relationships: make(map[types.NodeType][]RelationshipTarget),
// 				}
// 			}
// 		},
// 	}

// 	// Parse the SCIP file using the visitor
// 	if err := visitor.ParseStreaming(file); err != nil {
// 		return nil, fmt.Errorf("failed to parse SCIP file %s: %w", scipFilePath, err)
// 	}

// 	// Second pass: Process occurrences to establish relationships with position information
// 	// and identify definition positions.
// 	for _, occ := range internalGraph.Occurrences {
// 		if occ.Symbol == "" {
// 			continue // Skip occurrences with no symbol
// 		}

// 		sourceNode, exists := internalGraph.Nodes[occ.Symbol]
// 		if !exists {
// 			// This should ideally not happen if symbols from documents/externals are collected,
// 			// but create a node defensively if an occurrence references an uncollected symbol.
// 			sourceNode = &InternalGraphNode{
// 				SymbolName:    occ.Symbol,
// 				SymbolInfo:    nil, // Info might be missing if only referenced
// 				Relationships: make(map[types.NodeType][]RelationshipTarget),
// 			}
// 			internalGraph.Nodes[occ.Symbol] = sourceNode
// 		}

// 		// Check if this occurrence is a definition
// 		if scip.SymbolRole_Definition.Matches(occ) {
// 			// A symbol can have multiple definitions (e.g., interface and implementation)
// 			// We could store a list of definition positions, but for simplicity now, store the first one found.
// 			if sourceNode.DefinitionPosition == nil {
// 				pos := toPosition(occ.Range)
// 				sourceNode.DefinitionPosition = &pos
// 			}
// 			// Also, a definition occurrence implicitly creates a "definition" relationship originating from the symbol itself
// 			// pointing to the symbol's location in the file. While not a relationship *between* symbols,
// 			// it marks the node's primary location. We can add a self-referencing relationship or just use DefinitionPosition.
// 			// Let's rely on DefinitionPosition for now to mark the main definition location.
// 		} else {
// 			// This is a reference occurrence. It creates a "reference" relationship.
// 			// The target is the symbol being referenced (occ.Symbol).
// 			// The source is implicitly the code containing this occurrence.
// 			// In a direct node-edge graph, this is an edge from the occurrence location to the symbol node.
// 			// In our symbol-centric node model, we can think of this as an incoming reference
// 			// to occ.Symbol, or an outgoing reference from the document/scope containing the occurrence.
// 			// To match your GraphNode structure (which implies outgoing edges from a definition),
// 			// we need to determine *which* symbol is making this reference. This is complex.

// 			// Let's simplify: record references originating from the symbol's definition site
// 			// or other significant locations if available. A common approach is to link
// 			// references *to* the definition. The occurrence's range is where the reference *appears*.
// 			// We need to link the symbol *being referenced* (occ.Symbol) from *where* it is referenced.
// 			// The source of the reference is the code location represented by occ.Range.
// 			// This is not a simple "symbol A references symbol B" if A is the file/function containing the reference.

// 			// Let's revisit your GraphNode: FilePath, SymbolName, Position, Content, NodeType, Children, Parent.
// 			// This structure seems to model a tree starting from a "definition" node, with children being related elements.
// 			// This fits well with "Go to Definition" followed by "Find References".
// 			// The definition itself is a NodeTypeDefinition node. Its children could be references *to* it,
// 			// or things it references, or things that inherit/implement it.

// 			// Let's change the approach slightly to align better with the tree-like GraphNode:
// 			// We will build a map of all GraphNodes keyed by SymbolName.
// 			// The primary node for a symbol will represent its definition.
// 			// References, Inheritance, Implementation will be stored as children or linked structures.

// 			// Let's scrap the InternalGraph/InternalGraphNode and directly build a map of types.GraphNode.
// 			// We'll use the SymbolName as the key.
// 		}
// 	}

// 	// We need a third pass or integrated logic to connect nodes based on occurrences and relationships.
// 	// This requires knowing the context of an occurrence (which symbol/scope it's within).
// 	// The SCIP format links occurrences to documents and symbols, but determining the "source" symbol
// 	// of a reference occurrence requires analyzing the document's structure or relying on tool-specific info.

// 	// Given the complexity of mapping SCIP's flattened data to a potentially tree-like or
// 	// specific node-edge graph structure like your types.GraphNode solely from the SCIP file,
// 	// and the limitations of the current IndexVisitor (which doesn't provide scope context for occurrences),
// 	// a full and accurate conversion to your desired GraphNode tree might be challenging with just ParseStreaming.

// 	// Let's try a simplified approach first:
// 	// 1. Create a map of types.GraphNode keyed by SymbolName.
// 	// 2. Populate nodes for all symbols (definitions and externals). Store definition positions.
// 	// 3. Iterate occurrences:
// 	//    - If it's a definition, update the node's DefinitionPosition and NodeType to NodeTypeDefinition.
// 	//    - If it's a reference, find the target symbol's node. This occurrence is a reference *to* that symbol.
// 	//      We can add this occurrence's position to a list of "referenced from" positions on the target node,
// 	//      or create a child node of type NodeTypeReference pointing *back* to the location of the reference.
// 	// 4. Iterate SymbolInformation relationships:
// 	//    - Add children nodes of the appropriate NodeType (Inheritance, Implementation) linking to target symbols.

// 	// Let's restart the ParseSCIPFileForGraph function logic with this simpler mapping to types.GraphNode map.

// 	return nil, errors.New("restarting ParseSCIPFileForGraph logic to build types.GraphNode map") // Placeholder
// }

// New plan:
// 1. Create map[string]*types.GraphNode nodesMap
// 2. First pass with visitor: collect documents, external symbols.
//    For each symbol in documents or external symbols: create a types.GraphNode in nodesMap if not exists. Populate basic info (SymbolName, potentially FilePath from document).
// 3. Second pass (iterate collected documents and occurrences):
//    For each occurrence:
//      Find the corresponding node in nodesMap (using occ.Symbol).
//      If it's a definition, update node's NodeType to Definition, set Position.
//      If it's a reference, This occurrence is a reference *to* occ.Symbol. We need to know *where* this reference is located (file, position) and potentially *what* symbol/scope contains this reference. This context is hard to get from just the occurrence itself in the visitor.

// Alternative simpler map structure:
// map[string]SymbolNode { SymbolInfo, DefinitionPos, ReferencesToThisSymbol []Position, Relationships map[NodeType][]SymbolName }

// Let's go with a map of SymbolNode structs, collecting all relevant info per symbol.
// Then we can decide how to represent this as []*types.GraphNode.

// SymbolNode represents a symbol and all its associated information from SCIP.
type SymbolNode struct {
	SymbolName       string                      // The full SCIP symbol name
	SymbolInfo       *scip.SymbolInformation     // The actual symbol information from SCIP (if available)
	DefinitionOcc    *scip.Occurrence            // The definition occurrence (if any)
	ReferenceOccs    []*scip.Occurrence          // All reference occurrences of this symbol
	RelationshipsOut map[types.NodeType][]string // Outgoing relationships (type -> target symbol names)
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
			}

			// Process occurrences
			for _, occ := range d.Occurrences {
				if occ.Symbol == "" {
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

				// Process occurrence
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
