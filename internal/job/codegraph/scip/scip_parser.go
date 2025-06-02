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

			// 为每个文档创建新的事务
			if err := i.graphStore.BeginWrite(ctx); err != nil {
				processErr = fmt.Errorf("failed to begin transaction for document %s: %w", path, err)
				return
			}

			// 处理文档
			if err := i.processDocument(ctx, document, firstPass.ExternalSymbolsByName); err != nil {
				_ = i.graphStore.RollbackWrite(ctx)
				processErr = fmt.Errorf("failed to process document %s: %w", path, err)
				return
			}

			// 提交当前文档的事务
			if err := i.graphStore.CommitWrite(ctx); err != nil {
				_ = i.graphStore.RollbackWrite(ctx)
				processErr = fmt.Errorf("failed to commit transaction for document %s: %w", path, err)
				return
			}
		},
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	if processErr != nil {
		return processErr
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

// processDocument processes a single document and writes it to the store
func (i *IndexParser) processDocument(ctx context.Context, doc *scip.Document, externalSymbolsByName map[string]*scip.SymbolInformation) error {
	// 创建文档
	document := &codegraph.Document{
		Path:    doc.RelativePath,
		Symbols: make([]string, 0, len(doc.Symbols)),
	}

	// 处理符号和关系
	symbols := make(map[string]*codegraph.Symbol)
	for _, s := range doc.Symbols {
		// 创建或获取符号
		if _, ok := symbols[s.Symbol]; !ok {
			symbols[s.Symbol] = &codegraph.Symbol{
				Name: s.Symbol,
			}
		}

		// 处理关系
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

			pos := codegraph.Position{
				FilePath:    doc.RelativePath,
				NodeType:    nodeType,
				StartLine:   0, // 关系没有位置信息
				StartColumn: 0,
				EndLine:     0,
				EndColumn:   0,
			}

			switch nodeType {
			case types.NodeTypeImplementation:
				symbols[s.Symbol].Implementations = append(symbols[s.Symbol].Implementations, pos)
			case types.NodeTypeReference:
				symbols[s.Symbol].References = append(symbols[s.Symbol].References, pos)
			case types.NodeTypeDefinition:
				symbols[s.Symbol].Definitions = append(symbols[s.Symbol].Definitions, pos)
			}
		}

		document.Symbols = append(document.Symbols, s.Symbol)
	}

	// 处理出现
	for _, occ := range doc.Occurrences {
		if occ.Symbol == types.EmptyString {
			continue
		}

		// 创建或获取符号
		if _, ok := symbols[occ.Symbol]; !ok {
			symbols[occ.Symbol] = &codegraph.Symbol{
				Name: occ.Symbol,
			}
		}

		// 确定节点类型
		nodeType := types.NodeTypeReference
		if scip.SymbolRole_Definition.Matches(occ) {
			nodeType = types.NodeTypeDefinition
		}

		// 使用 toPosition 函数安全地处理范围信息
		typesPos := toPosition(occ.Range)
		pos := codegraph.Position{
			FilePath:    doc.RelativePath,
			NodeType:    nodeType,
			StartLine:   typesPos.StartLine,
			StartColumn: typesPos.StartColumn,
			EndLine:     typesPos.EndLine,
			EndColumn:   typesPos.EndColumn,
		}

		// 添加到相应的位置列表
		switch nodeType {
		case types.NodeTypeDefinition:
			symbols[occ.Symbol].Definitions = append(symbols[occ.Symbol].Definitions, pos)
		case types.NodeTypeReference:
			symbols[occ.Symbol].References = append(symbols[occ.Symbol].References, pos)
		}

		document.Symbols = append(document.Symbols, occ.Symbol)
	}

	// 写入文档
	if err := i.graphStore.WriteDocument(ctx, document); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	// 写入符号
	for _, symbol := range symbols {
		if err := i.graphStore.WriteSymbol(ctx, symbol); err != nil {
			return fmt.Errorf("failed to write symbol: %w", err)
		}
	}

	return nil
}
