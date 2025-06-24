package scip

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const writeBatchSize = 500

// ScipMetadata scip parsed metadata.
type ScipMetadata struct {
	Metadata                 *scip.Metadata
	ExternalSymbolsByName    map[string]*scip.SymbolInformation
	AllDefinitionOccurrences map[string]*codegraphpb.Symbol
	DocCountByPath           map[string]int
}

// IndexParser represents the SCIP index generator
type IndexParser struct {
	codebaseStore codebase.Store
}

// NewIndexParser creates a new SCIP index generator
func NewIndexParser(codebaseStore codebase.Store) *IndexParser {
	return &IndexParser{
		codebaseStore: codebaseStore,
	}
}

// visitDocument handles a single SCIP document during streaming parse.
func (i *IndexParser) visitDocument(ctx context.Context, metadata *ScipMetadata,
	duplicateDocs map[string][]*scip.Document, pendingDocs *[]*codegraphpb.Document) func(d *scip.Document) {
	return func(document *scip.Document) {
		path := document.RelativePath
		if count := metadata.DocCountByPath[path]; count > 1 {
			samePathDocs := append(duplicateDocs[path], document)
			duplicateDocs[path] = samePathDocs
			if len(samePathDocs) == count {
				flattenedDocs := scip.FlattenDocuments(samePathDocs)
				delete(duplicateDocs, path)
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
			tracer.WithTrace(ctx).Errorf("failed to process document %s: %w", path, err)
			return
		}
		*pendingDocs = append(*pendingDocs, doc)

	}
}

// ProcessScipIndexFile parses a SCIP index file and saves its data to graphStore.
func (i *IndexParser) ProcessScipIndexFile(ctx context.Context, codebasePath, scipFilePath string) error {
	start := time.Now()
	metadata, processedDocs, err := i.ParseScipIndexFile(ctx, codebasePath, scipFilePath)
	if err != nil {
		return fmt.Errorf("scip_parser parse index file %s err:%w", scipFilePath, err)
	}

	graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return err
	}

	defer graphStore.Close()

	err = i.saveDocs(ctx, graphStore, processedDocs)
	if err != nil {
		return fmt.Errorf("scip_parser save docs err:%w", err)
	}

	err = i.saveSymbolKeysMap(ctx, graphStore, metadata)
	if err != nil {
		return fmt.Errorf("scip_parser save symbol keys err:%w", err)
	}

	tracer.WithTrace(ctx).Debugf("scip_parser processed %d files successfully, cost %d ms ", len(processedDocs), time.Since(start).Milliseconds())
	return nil
}

func (i *IndexParser) ParseScipIndexFile(ctx context.Context, codebasePath string, scipFilePath string) (*ScipMetadata,
	[]*codegraphpb.Document, error) {
	tracer.WithTrace(ctx).Infof("scip_parser start to parse scip index file: %s/%s", codebasePath, scipFilePath)
	start := time.Now()
	fileStat, err := i.codebaseStore.Stat(ctx, codebasePath, scipFilePath)
	if err != nil {
		return nil, nil, err
	}
	if fileStat == nil || fileStat.IsDir {
		return nil, nil, fmt.Errorf("SCIP file does not exist: %s", scipFilePath)
	}
	if fileStat.Size == 0 {
		return nil, nil, fmt.Errorf("empty SCIP file: %s", scipFilePath)
	}

	file, err := i.codebaseStore.Open(ctx, codebasePath, scipFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open SCIP file: %w", err)
	}
	defer file.Close()

	// 需要维护全局的 symbolName-> doc_path 的映射

	metadata, err := i.prepareVisit(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect metadata and symbols: %w", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	duplicateDocs := make(map[string][]*scip.Document)
	processedDocs := make([]*codegraphpb.Document, 0, writeBatchSize)

	visitor := scip.IndexVisitor{
		VisitDocument: i.visitDocument(ctx, metadata, duplicateDocs, &processedDocs),
	}

	if err := visitor.ParseStreaming(file); err != nil {
		return nil, nil, fmt.Errorf("failed to parse SCIP file: %w", err)
	}
	tracer.WithTrace(ctx).Infof("scip_parser parsed index %s %d files successfully, cost %d ms ", scipFilePath,
		len(processedDocs), time.Since(start).Milliseconds())
	return metadata, processedDocs, nil
}

func (i *IndexParser) saveSymbolKeysMap(ctx context.Context, graphStore codegraph.GraphStore, metadata *ScipMetadata) error {
	start := time.Now()
	tracer.WithTrace(ctx).Infof("scip_parser start to save symbol keys map")
	symbolKeysMap := make(map[string]*codegraphpb.KeySet, len(metadata.AllDefinitionOccurrences))
	for _, v := range metadata.AllDefinitionOccurrences {
		if v.Role != codegraphpb.RelationType_RELATION_DEFINITION {
			tracer.WithTrace(ctx).Debugf("symbol %s role is not definition, skip", v.Identifier)
			continue
		}
		if ks, ok := symbolKeysMap[v.Name]; ok {
			ks.Keys = append(ks.Keys, &codegraphpb.KeyRange{
				DocKey: codegraph.DocKey(v.Path),
				Range:  v.Range,
			})
		} else {
			symbolKeysMap[v.Name] = &codegraphpb.KeySet{
				Keys: []*codegraphpb.KeyRange{
					{
						DocKey: codegraph.DocKey(v.Path),
						Range:  v.Range,
					}}}
		}
	}

	if len(symbolKeysMap) > 0 {
		if err := graphStore.BatchWriteDefSymbolKeysMap(ctx, symbolKeysMap); err != nil {
			return fmt.Errorf("failed to batch write symbol keys map: %w", err)
		}
	}
	tracer.WithTrace(ctx).Infof("scip_parser save symbol keys map %d symbols successfully, cost %d ms ",
		len(symbolKeysMap), time.Since(start).Milliseconds())
	return nil
}

func (i *IndexParser) saveDocs(ctx context.Context, graphStore codegraph.GraphStore, processedDocs []*codegraphpb.Document) error {
	start := time.Now()
	tracer.WithTrace(ctx).Infof("scip_parser start to save docs")
	// todo 只能等处理完所有docs，才入库。否则信息不全
	if len(processedDocs) > 0 {
		if err := graphStore.BatchWrite(ctx, processedDocs); err != nil {
			return fmt.Errorf("failed to batch write remaining documents: %w", err)
		}
	}
	tracer.WithTrace(ctx).Infof("scip_parser save %d docs successfully, cost %d ms ",
		len(processedDocs), time.Since(start).Milliseconds())
	return nil
}

// prepareVisit performs the first pass to collect metadata and external symbols.
func (i *IndexParser) prepareVisit(file io.Reader) (*ScipMetadata, error) {
	result := &ScipMetadata{
		ExternalSymbolsByName:    make(map[string]*scip.SymbolInformation),
		DocCountByPath:           make(map[string]int),
		AllDefinitionOccurrences: make(map[string]*codegraphpb.Symbol),
	}

	visitor := scip.IndexVisitor{
		VisitMetadata: func(m *scip.Metadata) {
			result.Metadata = m
		},
		VisitDocument: func(d *scip.Document) {
			result.DocCountByPath[d.RelativePath]++
			if strings.Contains(d.RelativePath, "webhook.go") {
				logx.Infof("scip_parser prepareVisit file %s", d.RelativePath)
			}

			// 处理occurrences
			for _, occ := range d.Occurrences {
				// 跳过local
				if scip.IsLocalSymbol(occ.Symbol) {
					continue
				}
				if getSymbolRoleFromOccurrence(occ) != codegraphpb.RelationType_RELATION_DEFINITION {
					continue
				}
				// 只保存定义，symbol 会在多个文件中重复出现，因为有引用的存在
				result.AllDefinitionOccurrences[occ.Symbol] = buildSymbol(occ, d.RelativePath)
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
func (i *IndexParser) processDocument(doc *scip.Document, metadata *ScipMetadata) (*codegraphpb.Document, error) {
	// TODO go package 并不属于某个filePath，需要在内存中处理完所有关系，才能入库。
	// TODO 这种 带#的找不到definition k8s.io/apiserver/pkg/apis/audit`/Event#RequestURI.  role 8 definition not found in allSymbolDefinitions
	addMissingExternalSymbols(doc, metadata.ExternalSymbolsByName)
	_ = scip.CanonicalizeDocument(doc) // 包含了排序
	document := &codegraphpb.Document{
		Path: doc.RelativePath,
	}
	document.Symbols = populateSymbolsAndOccurrences(doc, metadata.AllDefinitionOccurrences)
	return document, nil
}

// populateSymbolsAndOccurrences processes Occurrences and fills symbols and document.Symbols.
func populateSymbolsAndOccurrences(doc *scip.Document, allSymbolDefinitions map[string]*codegraphpb.Symbol) []*codegraphpb.Symbol {
	//TODO 这些symbol ，都是根据名字关联的
	// doc 的 symbols 和 occurrences 的 relative path 都是doc的path，只有relation的path是别的doc
	populatedSymbols := make([]*codegraphpb.Symbol, 0, len(doc.Occurrences))
	relativePath := doc.RelativePath
	// 先遍历 symbols，这里都是definition，提取它的relations 和documentation
	for _, sym := range doc.Symbols {
		symbolName := sym.Symbol
		relations := sym.Relationships
		rels := make([]*codegraphpb.Relation, len(relations))
		if len(relations) > 0 {
			for _, rel := range relations {
				relationSymbolName := rel.Symbol
				defSymbol, ok := allSymbolDefinitions[relationSymbolName]
				relation := &codegraphpb.Relation{
					Identifier:   relationSymbolName,
					RelationType: getRelationShipTypeFromRelation(rel),
				}
				if ok {
					relation.FilePath = defSymbol.Path
					relation.Name = defSymbol.Name
				} else { // TODO 第三方库（包括标准库）
					// logx.Errorf("relation defSymbol %s definition not found in allSymbolDefinitions", relationSymbolName)
				}
				rels = append(rels, relation)
			}
		}
		symbolDef, ok := allSymbolDefinitions[symbolName]
		if ok {
			symbolDef.Relations = append(symbolDef.Relations, rels...)
		} else { //TODO 带冒号的: scip-go gomod k8s.io/kubernetes . `k8s.io/kubernetes/cluster/images/etcd-version-monitor`/gatherer:MetricFamily. 项目内一些奇怪的东西，应该在prepareVisit中被访问的
			//logx.Errorf("symbols defSymbol %s definition not found in allSymbolDefinitions", symbolName)
			//symbolDef = &codegraphpb.Symbol{
			//	Language: symbolName,
			//	Role: codegraphpb.RelationType_RELATION_DEFINITION,
			//	FilePath: relativePath,
			//}
		}
	}

	// 遍历 occurrences，如果是定义，则直接取全局的，填充它的range；如果不是定义，则是引用，当前relation增加一个定义的关系，它的定义的relation增加一个引用指向它。
	for _, occ := range doc.Occurrences {
		occName := occ.Symbol
		// empty or  local variable
		if occName == types.EmptyString || scip.IsLocalSymbol(occName) {
			continue
		}

		symbolRole := getSymbolRoleFromOccurrence(occ)

		symbolDef, ok := allSymbolDefinitions[occName]
		if !ok { //TODO import或使用标准库的位置 not found，属于reference: scip-go gomod github.com/golang/go/src go1.24.0 os/Args.; 带#的：scip-go gomod github.com/golang/go/src go1.24.0 sync/WaitGroup#Done().；
			// preVisit() 已经将所有的occurrence 遍历过了，如果这里找不到，理论上只能是引用，不可能是定义, 而且是第三方库（包括go标准库）的引用。
			// logx.Debugf("occurrence symbol %s  role %d definition not found in allSymbolDefinitions", occName, occ.SymbolRoles)
		}
		var occurSymbol *codegraphpb.Symbol
		// 当前是定义，则直接指向全局即可
		if symbolRole == codegraphpb.RelationType_RELATION_DEFINITION {
			if symbolDef != nil {
				occurSymbol = symbolDef
			} else {
				occurSymbol = buildSymbol(occ, relativePath)
			}
		} else {
			// 当前是引用或未找到定义，创建一个新的，
			occurSymbol = &codegraphpb.Symbol{
				Identifier: occName,
				Role:       symbolRole,
				Path:       relativePath,
				Range:      occ.Range,
			}
			if symbolDef != nil {
				// 引用->定义
				occurSymbol.Relations = append([]*codegraphpb.Relation(nil), &codegraphpb.Relation{
					Name:         symbolDef.Name,
					Identifier:   symbolDef.Identifier,
					FilePath:     symbolDef.Path,
					Range:        symbolDef.Range,
					RelationType: codegraphpb.RelationType_RELATION_DEFINITION,
				})
				// 定义 -> 引用
				symbolDef.Relations = append(symbolDef.Relations, &codegraphpb.Relation{
					Name:         occurSymbol.Identifier,
					Identifier:   occurSymbol.Identifier,
					FilePath:     relativePath,
					Range:        occ.Range,
					RelationType: codegraphpb.RelationType_RELATION_REFERENCE,
				})
			}
		}
		populatedSymbols = append(populatedSymbols, occurSymbol)
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
		logx.Debugf("unknown relation type: %v", rel)
	}
	return relationType
}

// getSymbolRoleFromOccurrence returns the SymbolRole for an occurrence.
func getSymbolRoleFromOccurrence(occ *scip.Occurrence) codegraphpb.RelationType {
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

func DefaultIndexFilePath() string {
	return filepath.Join(types.CodebaseIndexDir, types.IndexFileName)
}
