package scip

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"io"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const writeBatchSize = 500

// scipMetadata scip parsed metadata.
type scipMetadata struct {
	Metadata              *scip.Metadata
	ExternalSymbolsByName map[string]*scip.SymbolInformation
	SymbolNamePath        map[string]string
	DocCountByPath        map[string]int
}

// IndexParser represents the SCIP index generator
type IndexParser struct {
	codebaseStore codebase.Store
	graphStore    codegraph.GraphStore
	logger        logx.Logger
}

// NewIndexParser creates a new SCIP index generator
func NewIndexParser(ctx context.Context, codebaseStore codebase.Store, graphStore codegraph.GraphStore) *IndexParser {
	return &IndexParser{
		codebaseStore: codebaseStore,
		graphStore:    graphStore,
		logger:        logx.WithContext(ctx),
	}
}

// visitDocument handles a single SCIP document during streaming parse.
func (i *IndexParser) visitDocument(
	ctx context.Context,
	metadata *scipMetadata,
	duplicateDocs map[string][]*scip.Document,
	pendingDocs *[]*codegraphpb.Document) func(d *scip.Document) {
	return func(document *scip.Document) {
		path := document.RelativePath
		if count := metadata.DocCountByPath[path]; count > 1 {
			samePathDocs := append(duplicateDocs[path], document)
			duplicateDocs[path] = samePathDocs
			if len(samePathDocs) == count {
				flattenedDocs := scip.FlattenDocuments(samePathDocs)
				delete(duplicateDocs, path) //TODO 这是干啥的
				if len(flattenedDocs) != 1 {
					return
				}
				document = flattenedDocs[0]
			} else {
				return
			}
		}
		doc, err := i.processDocument(document, metadata)
		if err != nil {
			i.logger.Errorf("failed to process document %s: %w", path, err)
			return
		}
		*pendingDocs = append(*pendingDocs, doc)

		// 到达一定批次，写入数据
		if len(*pendingDocs) >= writeBatchSize {
			if err := i.graphStore.BatchWrite(ctx, *pendingDocs); err != nil {
				logx.Errorf("failed to batch write remaining documents: %w", err)
			}
			// 处理完，清空
			*pendingDocs = (*pendingDocs)[:0]
		}
	}
}

// ParseSCIPFile parses a SCIP index file and saves its data to graphStore.
func (i *IndexParser) ParseSCIPFile(ctx context.Context, codebasePath, scipFilePath string) error {
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

	file, err := i.codebaseStore.Open(ctx, codebasePath, scipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open SCIP file: %w", err)
	}
	defer file.Close()

	// 需要维护全局的 symbolName-> doc_path 的映射

	metadata, err := i.parseMetadata(file)
	if err != nil {
		return fmt.Errorf("failed to collect metadata and symbols: %w", err)
	}
	i.logger.Debugf("parsed %s files cnt: %d", codebasePath, len(metadata.DocCountByPath))

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	duplicateDocs := make(map[string][]*scip.Document)
	pendingDocs := make([]*codegraphpb.Document, 0, writeBatchSize)

	visitor := scip.IndexVisitor{
		VisitDocument: i.visitDocument(ctx, metadata, duplicateDocs, &pendingDocs),
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	if len(pendingDocs) > 0 {
		if err := i.graphStore.BatchWrite(ctx, pendingDocs); err != nil {
			return fmt.Errorf("failed to batch write remaining documents: %w", err)
		}
	}

	return nil
}

// parseMetadata performs the first pass to collect metadata and external symbols.
func (i *IndexParser) parseMetadata(file io.Reader) (*scipMetadata, error) {
	result := &scipMetadata{
		ExternalSymbolsByName: make(map[string]*scip.SymbolInformation),
		DocCountByPath:        make(map[string]int),
		SymbolNamePath:        make(map[string]string),
	}

	visitor := scip.IndexVisitor{
		VisitMetadata: func(m *scip.Metadata) {
			result.Metadata = m
		},
		VisitDocument: func(d *scip.Document) {
			result.DocCountByPath[d.RelativePath]++
			for _, occ := range d.Occurrences {
				result.SymbolNamePath[occ.Symbol] = d.RelativePath
			}
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

// processDocument processes a single document and returns the document and its symbols.
func (i *IndexParser) processDocument(doc *scip.Document, metadata *scipMetadata) (*codegraphpb.Document, error) {

	addMissingExternalSymbols(doc, metadata.ExternalSymbolsByName)
	_ = scip.CanonicalizeDocument(doc) // 包含了排序
	document := &codegraphpb.Document{
		Path: doc.RelativePath,
	}
	document.Symbols = populateSymbolsAndOccurrences(doc, metadata.SymbolNamePath)
	return document, nil
}

// populateSymbolsAndOccurrences processes Occurrences and fills symbols and document.Symbols.
func populateSymbolsAndOccurrences(doc *scip.Document, symbolNamePath map[string]string) []*codegraphpb.Symbol {
	// doc 的 symbols 和 occurrences 的 relative path 都是doc的path，只有relation的path是别的doc
	populatedSymbols := make([]*codegraphpb.Symbol, len(doc.Occurrences))
	symbols := make(map[string]*codegraphpb.Symbol)
	// 先遍历 symbols，放入map
	for _, sym := range doc.Symbols {
		symbolName := sym.Symbol
		relations := sym.Relationships
		rels := make([]*codegraphpb.Relation, len(relations))
		if len(relations) > 0 {
			for _, rel := range relations {
				relationSymbolName := rel.Symbol
				rels = append(rels, &codegraphpb.Relation{
					Name:         relationSymbolName,
					RelationType: getRelationShipTypeFromRelation(rel),
					FilePath:     symbolNamePath[relationSymbolName],
				})
			}
		}
		symbols[symbolName] = &codegraphpb.Symbol{
			Name:      symbolName,
			Content:   strings.Join(sym.Documentation, types.EmptyString),
			Role:      codegraphpb.RelationType_RELATION_TYPE_UNKNOWN,
			Range:     nil,
			Relations: rels,
		}
	}

	// 然后遍历 occurrences，如果symbol存在，填充它的 position，
	for _, occ := range doc.Occurrences {
		if occ.Symbol == "" {
			continue
		}
		sym, ok := symbols[occ.Symbol]
		if !ok {
			// 不存在
			sym = &codegraphpb.Symbol{
				Name:      occ.Symbol,
				Role:      getRelationShipTypeFromOccurrence(occ), // 不是definition,就是refer
				Relations: nil,
			}
			symbols[occ.Symbol] = sym
		} else {
			// TODO 存在，将它删除，用于调试
			delete(symbols, occ.Symbol)
		}
		// 填充range
		sym.Range = occ.Range
		populatedSymbols = append(populatedSymbols, sym)
	}
	if len(symbols) > 0 {
		// TODO
		logx.Errorf("doc %s symbols is not empty after populate.", doc.RelativePath)
	}
	return populatedSymbols
}

func getRelationShipTypeFromRelation(rel *scip.Relationship) codegraphpb.RelationType {
	relationType := codegraphpb.RelationType_RELATION_TYPE_UNKNOWN
	if rel.IsDefinition {
		relationType = codegraphpb.RelationType_RELATION_DEFINITION
	} else if rel.IsReference {
		relationType = codegraphpb.RelationType_RELATION_REFERENCE
	} else if rel.IsImplementation {
		relationType = codegraphpb.RelationType_RELATION_IMPLEMENTATION
	} else if rel.IsTypeDefinition {
		relationType = codegraphpb.RelationType_RELATION_TYPE_DEFINITION
	} else {
		logx.Errorf("unknown relation type: %v", rel)
	}
	return relationType
}

// getRelationShipTypeFromOccurrence returns the SymbolRole for an occurrence.
func getRelationShipTypeFromOccurrence(occ *scip.Occurrence) codegraphpb.RelationType {
	if scip.SymbolRole_Definition.Matches(occ) {
		return codegraphpb.RelationType_RELATION_DEFINITION
	}
	return codegraphpb.RelationType_RELATION_REFERENCE
}

// addMissingExternalSymbols 注入外部符号
func addMissingExternalSymbols(doc *scip.Document, externalSymbolsByName map[string]*scip.SymbolInformation) {
	defSet := make(map[string]struct{}, len(doc.Symbols))
	for _, s := range doc.Symbols {
		defSet[s.Symbol] = struct{}{}
	}
	refSet := make(map[string]struct{})
	for _, occ := range doc.Occurrences {
		if occ.Symbol != "" {
			refSet[occ.Symbol] = struct{}{}
		}
	}
	for _, s := range doc.Symbols {
		for _, rel := range s.Relationships {
			refSet[rel.Symbol] = struct{}{}
		}
	}
	for sym := range refSet {
		if _, ok := defSet[sym]; !ok {
			if ext, ok := externalSymbolsByName[sym]; ok {
				doc.Symbols = append(doc.Symbols, ext)
				defSet[sym] = struct{}{}
			}
		}
	}
}
