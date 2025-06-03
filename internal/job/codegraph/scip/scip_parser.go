package scip

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const parseBatchSize = 1000

// ParsedMetadata scip parsed metadata.
type ParsedMetadata struct {
	Metadata              *scip.Metadata
	ExternalSymbolsByName map[string]*scip.SymbolInformation
	docCountByPath        map[string]int
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

// getOrCreateSymbol returns the symbol from the map or creates it if not present.
func getOrCreateSymbol(symbols map[string]*codegraph.Symbol, name string, contents []string) *codegraph.Symbol {
	sym, ok := symbols[name]
	if !ok {
		sym = &codegraph.Symbol{
			Name:        name,
			Occurrences: make(map[types.SymbolRole][]codegraph.Occurrence),
		}
		symbols[name] = sym
	}
	if len(contents) > 0 {
		sym.Content += strings.Join(contents, types.EmptyString) // 它自带了\n
	}
	return sym
}

// newOccurrence constructs a new Occurrence.
func newOccurrence(filePath string, rng []int32, nodeType types.SymbolRole) codegraph.Occurrence {
	return codegraph.Occurrence{
		FilePath: filePath,
		Range:    rng,
		NodeType: nodeType,
	}
}

// visitDocument handles a single SCIP document during streaming parse.
func (i *IndexParser) visitDocument(metadata *ParsedMetadata,
	duplicateDocs map[string][]*scip.Document,
	pendingDocs *[]*codegraph.Document,
	pendingSymbols *[]*codegraph.Symbol) func(d *scip.Document) {
	return func(document *scip.Document) {
		path := document.RelativePath
		if count := metadata.docCountByPath[path]; count > 1 {
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
		doc, symbols, err := i.processDocument(document, metadata.ExternalSymbolsByName)
		if err != nil {
			i.logger.Errorf("failed to process document %s: %w", path, err)
			return
		}
		*pendingDocs = append(*pendingDocs, doc)
		*pendingSymbols = append(*pendingSymbols, symbols...)
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

	metadata, err := i.parseMetadata(file)
	if err != nil {
		return fmt.Errorf("failed to collect metadata and symbols: %w", err)
	}
	i.logger.Debugf("parsed %s files cnt: %d", codebasePath, len(metadata.docCountByPath))

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	duplicateDocs := make(map[string][]*scip.Document)
	pendingDocs := make([]*codegraph.Document, 0, parseBatchSize)
	pendingSymbols := make([]*codegraph.Symbol, 0, parseBatchSize)

	visitor := scip.IndexVisitor{
		VisitDocument: i.visitDocument(metadata, duplicateDocs, &pendingDocs, &pendingSymbols),
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return fmt.Errorf("failed to parse SCIP file: %w", err)
	}

	if len(pendingDocs) > 0 {
		if err := i.graphStore.BatchWrite(ctx, pendingDocs, pendingSymbols); err != nil {
			return fmt.Errorf("failed to batch write remaining documents: %w", err)
		}
	}

	return nil
}

// parseMetadata performs the first pass to collect metadata and external symbols.
func (i *IndexParser) parseMetadata(file io.Reader) (*ParsedMetadata, error) {
	result := &ParsedMetadata{
		ExternalSymbolsByName: make(map[string]*scip.SymbolInformation),
		docCountByPath:        make(map[string]int),
	}

	visitor := scip.IndexVisitor{
		VisitMetadata: func(m *scip.Metadata) {
			result.Metadata = m
		},
		VisitDocument: func(d *scip.Document) {
			result.docCountByPath[d.RelativePath]++
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
func (i *IndexParser) processDocument(doc *scip.Document, externalSymbolsByName map[string]*scip.SymbolInformation) (*codegraph.Document, []*codegraph.Symbol, error) {
	addMissingExternalSymbols(doc, externalSymbolsByName)
	normalizeDocumentOrder(doc)
	relativePath := doc.RelativePath
	_ = scip.CanonicalizeDocument(doc)
	document := &codegraph.Document{
		Path:    relativePath,
		Symbols: make([]*codegraph.SymbolInDoc, 0, len(doc.Occurrences)),
	}

	symbols := make(map[string]*codegraph.Symbol)

	populateSymbolsFromOccurrences(doc, relativePath, symbols, document)
	populateRelationships(doc, symbols)
	populateSymbolContent(doc, symbols)

	symbolSlice := make([]*codegraph.Symbol, 0, len(symbols))
	for _, symbol := range symbols {
		symbolSlice = append(symbolSlice, symbol)
	}

	return document, symbolSlice, nil
}

// populateSymbolsFromOccurrences processes Occurrences and fills symbols and document.Symbols.
func populateSymbolsFromOccurrences(doc *scip.Document, relativePath string, symbols map[string]*codegraph.Symbol, document *codegraph.Document) {
	for _, occ := range doc.Occurrences {
		if occ.Symbol == "" {
			continue
		}
		sym, ok := symbols[occ.Symbol]
		if !ok {
			sym = &codegraph.Symbol{
				Name:        occ.Symbol,
				Occurrences: make(map[types.SymbolRole][]codegraph.Occurrence),
			}
			symbols[occ.Symbol] = sym
		}
		role := getSymbolRoleFromOccurrence(occ)
		occurrence := codegraph.Occurrence{
			FilePath: relativePath,
			Range:    occ.Range,
			NodeType: role,
		}
		sym.Occurrences[role] = append(sym.Occurrences[role], occurrence)
		document.Symbols = append(document.Symbols, &codegraph.SymbolInDoc{
			Name:  occ.Symbol,
			Role:  role,
			Range: occ.Range,
		})
	}
	// occurrence 排序
	for _, sym := range symbols {
		for _, occs := range sym.Occurrences {
			codegraphSortOccurrencesByRange(occs)
		}
	}
}

// codegraphSortOccurrencesByRange 用于 scip_parser 里 occurrence 排序，避免与 badgerdb.go 重名
func codegraphSortOccurrencesByRange(occurrences []codegraph.Occurrence) {
	sort.Slice(occurrences, func(i, j int) bool {
		a, b := occurrences[i].Range, occurrences[j].Range
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
}

// getSymbolRoleFromOccurrence returns the SymbolRole for an occurrence.
func getSymbolRoleFromOccurrence(occ *scip.Occurrence) types.SymbolRole {
	if scip.SymbolRole_Definition.Matches(occ) {
		return types.SymbolRoleDefinition
	}
	return types.SymbolRoleReference
}

// populateRelationships processes SymbolInformation.Relationships for implementation/type_definition.
func populateRelationships(doc *scip.Document, symbols map[string]*codegraph.Symbol) {
	for _, s := range doc.Symbols {
		sym, ok := symbols[s.Symbol]
		if !ok {
			continue
		}
		defOccs := sym.Occurrences[types.SymbolRoleDefinition]
		if len(defOccs) == 0 {
			continue
		}
		for _, rel := range s.Relationships {
			if rel.IsImplementation {
				target := getOrCreateOrSetSymbol(symbols, rel.Symbol)
				target.Occurrences[types.SymbolRoleImplementation] = append(target.Occurrences[types.SymbolRoleImplementation], defOccs...)
			}
			if rel.IsTypeDefinition {
				target := getOrCreateOrSetSymbol(symbols, rel.Symbol)
				target.Occurrences[types.SymbolRoleTypeDefinition] = append(target.Occurrences[types.SymbolRoleTypeDefinition], defOccs...)
			}
		}
	}
}

// getOrCreateOrSetSymbol returns the symbol from the map or creates it if not present.
func getOrCreateOrSetSymbol(symbols map[string]*codegraph.Symbol, name string) *codegraph.Symbol {
	sym, ok := symbols[name]
	if !ok {
		sym = &codegraph.Symbol{
			Name:        name,
			Occurrences: make(map[types.SymbolRole][]codegraph.Occurrence),
		}
		symbols[name] = sym
	}
	return sym
}

// populateSymbolContent fills Symbol.Content from SymbolInformation.Documentation.
func populateSymbolContent(doc *scip.Document, symbols map[string]*codegraph.Symbol) {
	for _, s := range doc.Symbols {
		sym, ok := symbols[s.Symbol]
		if ok && sym.Content == "" && len(s.Documentation) > 0 {
			sym.Content = strings.Join(s.Documentation, "\n")
		}
	}
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

// sortOccurrencesByRange 对 occurrence 按 range 排序
func sortOccurrencesByRange(occurrences []*scip.Occurrence) {
	sort.Slice(occurrences, func(i, j int) bool {
		a, b := occurrences[i].Range, occurrences[j].Range
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
}

// normalizeDocumentOrder 对 symbol/occurrence/relationship 排序
func normalizeDocumentOrder(doc *scip.Document) {
	sort.Slice(doc.Symbols, func(i, j int) bool {
		return doc.Symbols[i].Symbol < doc.Symbols[j].Symbol
	})
	sort.Slice(doc.Occurrences, func(i, j int) bool {
		a, b := doc.Occurrences[i].Range, doc.Occurrences[j].Range
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	for _, s := range doc.Symbols {
		sort.Slice(s.Relationships, func(i, j int) bool {
			return s.Relationships[i].Symbol < s.Relationships[j].Symbol
		})
	}
}
