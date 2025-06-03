package scip

import (
	"context"
	"fmt"
	"io"

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

// FirstPassResult stores the results of the first parsing phase
type FirstPassResult struct {
	Metadata              *scip.Metadata
	ExternalSymbolsByName map[string]*scip.SymbolInformation
	DocumentCountByPath   map[string]int
}

// IndexParser represents the SCIP index generator
type IndexParser struct {
	codebaseStore codebase.Store
	graphStore    codegraph.GraphStore
}

// NewIndexParser creates a new SCIP index generator
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

	// 第一阶段：收集元数据和外部符号信息
	firstPass, err := i.collectMetadataAndSymbols(file)
	if err != nil {
		return fmt.Errorf("failed to collect metadata and symbols: %w", err)
	}

	// 重置文件指针
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to reset file pointer: %w", err)
		}
	}

	// 第二阶段：流式处理文档
	repeatedDocumentsByPath := make(map[string][]*scip.Document)
	var processErr error

	// 批量写入的缓冲区
	batchSize := 1000
	pendingDocs := make([]*codegraph.Document, 0, batchSize)
	pendingSymbols := make([]*codegraph.Symbol, 0, batchSize)

	visitor := scip.IndexVisitor{
		VisitDocument: func(d *scip.Document) {
			if processErr != nil {
				return
			}

			path := d.RelativePath
			document := d

			// 处理重复文档
			if count := firstPass.DocumentCountByPath[path]; count > 1 {
				samePathDocs := append(repeatedDocumentsByPath[path], document)
				repeatedDocumentsByPath[path] = samePathDocs

				// 当收集到所有相同路径的文档时，合并它们
				if len(samePathDocs) == count {
					flattenedDocs := scip.FlattenDocuments(samePathDocs)
					if len(flattenedDocs) != 1 {
						return
					}
					document = flattenedDocs[0]
				} else {
					return
				}
			}

			// 处理文档
			doc, symbols, err := i.processDocument(ctx, document, firstPass.ExternalSymbolsByName)
			if err != nil {
				processErr = fmt.Errorf("failed to process document %s: %w", path, err)
				return
			}

			// 添加到待处理列表
			pendingDocs = append(pendingDocs, doc)
			pendingSymbols = append(pendingSymbols, symbols...)

			// 当达到批量大小时执行批量写入
			if len(pendingDocs) >= batchSize {
				if err := i.graphStore.BatchWrite(ctx, pendingDocs, pendingSymbols); err != nil {
					processErr = fmt.Errorf("failed to batch write documents: %w", err)
					return
				}
				pendingDocs = pendingDocs[:0]
				pendingSymbols = pendingSymbols[:0]
			}
		},
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	if processErr != nil {
		return processErr
	}

	// 写入剩余的文档和符号
	if len(pendingDocs) > 0 {
		if err := i.graphStore.BatchWrite(ctx, pendingDocs, pendingSymbols); err != nil {
			return fmt.Errorf("failed to batch write remaining documents: %w", err)
		}
	}

	return nil
}

// collectMetadataAndSymbols performs the first pass to collect metadata and external symbols
func (i *IndexParser) collectMetadataAndSymbols(file io.Reader) (*FirstPassResult, error) {
	result := &FirstPassResult{
		ExternalSymbolsByName: make(map[string]*scip.SymbolInformation),
		DocumentCountByPath:   make(map[string]int),
	}

	visitor := scip.IndexVisitor{
		VisitMetadata: func(m *scip.Metadata) {
			result.Metadata = m
		},
		VisitDocument: func(d *scip.Document) {
			result.DocumentCountByPath[d.RelativePath]++
		},
		VisitExternalSymbol: func(s *scip.SymbolInformation) {
			result.ExternalSymbolsByName[s.Symbol] = s
		},
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return nil, err
	}

	return result, nil
}

// processDocument processes a single document and returns the document and its symbols
func (i *IndexParser) processDocument(ctx context.Context, doc *scip.Document, externalSymbolsByName map[string]*scip.SymbolInformation) (*codegraph.Document, []*codegraph.Symbol, error) {
	document := &codegraph.Document{
		Path:    doc.RelativePath,
		Symbols: make([]codegraph.SymbolInDoc, 0, len(doc.Occurrences)),
	}

	symbols := make(map[string]*codegraph.Symbol)
	for _, s := range doc.Symbols {
		if _, ok := symbols[s.Symbol]; !ok {
			symbols[s.Symbol] = &codegraph.Symbol{
				Name:        s.Symbol,
				Occurrences: make(map[types.NodeType][]codegraph.Occurrence),
			}
		}
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
			occ := codegraph.Occurrence{
				FilePath: doc.RelativePath,
				Range:    []int32{0, 0, 0, 0},
				NodeType: nodeType,
			}
			symbols[s.Symbol].Occurrences[nodeType] = append(symbols[s.Symbol].Occurrences[nodeType], occ)
		}
	}

	for _, occ := range doc.Occurrences {
		if occ.Symbol == types.EmptyString {
			continue
		}
		if _, ok := symbols[occ.Symbol]; !ok {
			symbols[occ.Symbol] = &codegraph.Symbol{
				Name:        occ.Symbol,
				Occurrences: make(map[types.NodeType][]codegraph.Occurrence),
			}
		}
		nodeType := types.NodeTypeReference
		if scip.SymbolRole_Definition.Matches(occ) {
			nodeType = types.NodeTypeDefinition
		}
		occurrence := codegraph.Occurrence{
			FilePath: doc.RelativePath,
			Range:    occ.Range,
			NodeType: nodeType,
		}
		symbols[occ.Symbol].Occurrences[nodeType] = append(symbols[occ.Symbol].Occurrences[nodeType], occurrence)
		document.Symbols = append(document.Symbols, codegraph.SymbolInDoc{
			Name:     occ.Symbol,
			NodeType: nodeType,
			Range:    occ.Range,
		})
	}

	for _, occ := range doc.Occurrences {
		if symbol, ok := symbols[occ.Symbol]; ok && symbol.Content == "" && len(occ.Range) == 4 {
			startLine := int(occ.Range[0])
			endLine := int(occ.Range[2])
			lines := splitLines(doc.Text)
			if startLine >= 0 && endLine < len(lines) && startLine <= endLine {
				content := joinLines(lines[startLine : endLine+1])
				symbol.Content = content
			}
		}
	}

	symbolSlice := make([]*codegraph.Symbol, 0, len(symbols))
	for _, symbol := range symbols {
		symbolSlice = append(symbolSlice, symbol)
	}

	return document, symbolSlice, nil
}

// splitLines splits text into lines
func splitLines(text string) []string {
	lines := make([]string, 0)
	start := 0
	for i, c := range text {
		if c == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// joinLines joins lines with newline
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	res := lines[0]
	for i := 1; i < len(lines); i++ {
		res += "\n" + lines[i]
	}
	return res
}
