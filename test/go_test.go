package test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	scipindex "github.com/zgsm-ai/codebase-indexer/internal/job/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const testProjectsBaseDir = "/tmp/projects"
const indexFileName = "index.scip"

func Test_GenerateGoScipIndex(t *testing.T) {
	// run ./fetch_test_projects.sh to clone real projects from github
	t.Run("kubernetes", func(t *testing.T) {
		projectPath := "go/kubernetes"
		codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
		storeConf := config.CodeBaseStoreConf{
			Local: config.LocalStoreConf{
				BasePath: testProjectsBaseDir,
			},
		}
		scipConf, err := scipindex.LoadConfig("../etc/codegraph.yaml")
		assert.NoError(t, err)
		localCodebase, err := codebase.NewLocalCodebase(context.Background(), storeConf)
		assert.NoError(t, err)
		generator := scipindex.NewIndexGenerator(scipConf, localCodebase)
		err = generator.Generate(context.Background(), codebasePath)
		assert.NoError(t, err)
		indexFile := filepath.Join(testProjectsBaseDir, projectPath, types.CodebaseIndexDir, indexFileName)
		fmt.Println(indexFile)
		assert.FileExists(t, indexFile)

	})
}

func TestParseGoScipIndexBadgerDB(t *testing.T) {
	start := time.Now()
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
	storeConf := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: testProjectsBaseDir,
		},
	}
	localCodebase, err := codebase.NewLocalCodebase(context.Background(), storeConf)
	assert.NoError(t, err)
	indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	graph, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	defer graph.Close()
	assert.NoError(t, err)
	parser := scipindex.NewIndexParser(localCodebase, graph)
	err = parser.ParseSCIPFileForGraph(context.Background(), codebasePath, indexFile)
	assert.NoError(t, err)
	//fmt.Printf("graph: %v", graph)
	fmt.Printf("time: %v seconds", time.Since(start).Seconds())
}

func TestQueryBadgerDB(t *testing.T) {
	start := time.Now()
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)

	// 1. 初始化存储
	graph, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	defer graph.Close()

	// 2. 初始化 codebase store
	storeConf := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: testProjectsBaseDir,
		},
	}
	localCodebase, err := codebase.NewLocalCodebase(context.Background(), storeConf)
	assert.NoError(t, err)

	// 3. 解析和写入数据
	indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	parser := scipindex.NewIndexParser(localCodebase, graph)
	err = parser.ParseSCIPFileForGraph(context.Background(), codebasePath, indexFile)
	assert.NoError(t, err)
	fmt.Printf("time: %v seconds", time.Since(start).Seconds())

	// 4. 执行查询
	targetPath := "cmd/kubeadm/app/util/endpoint.go"
	// 调试：检查数据库中的键
	fmt.Println("\nChecking database keys:")
	err = graph.(*codegraph.BadgerDBGraph).DB().View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			if strings.Contains(string(key), targetPath) {
				fmt.Printf("Found key: %s\n", key)
			}
		}
		return nil
	})
	assert.NoError(t, err)

	fmt.Printf("\nQuerying for file: %s\n", targetPath)
	references, err := graph.Query(context.Background(), &types.RelationQueryOptions{
		FilePath:   targetPath,
		StartLine:  36,
		EndLine:    36,
		SymbolName: "GetControlPlaneEndpoint",
	})
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
	}
	fmt.Printf("references: %v", references)
}

func TestDeleteBadgerDB(t *testing.T) {
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
	graph, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	err = graph.DeleteAll(context.Background())
	assert.NoError(t, err)
}

func TestInspectBadgerDB(t *testing.T) {
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)

	// 初始化 BadgerDB
	storage, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		t.Fatalf("Failed to initialize BadgerDB: %v", err)
	}
	defer storage.Close()

	// 获取 BadgerDB 实例
	db := storage.(*codegraph.BadgerDBGraph).DB()

	// 遍历所有数据
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			// 根据 key 前缀判断数据类型
			switch {
			case bytes.HasPrefix(key, []byte("doc:")):
				doc, err := codegraph.DeserializeDocument(val)
				if err != nil {
					return err
				}
				fmt.Printf("Document: %s\n", doc.Path)
				fmt.Printf("  Symbols: %d\n", len(doc.Symbols))
				fmt.Printf("  Content size: %d bytes\n", len(doc.Content))
			case bytes.HasPrefix(key, []byte("sym:")):
				sym, err := codegraph.DeserializeSymbol(val)
				if err != nil {
					return err
				}
				fmt.Printf("Symbol: %s\n", sym.Name)
				fmt.Printf("  Definitions: %d\n", len(sym.Definitions))
				fmt.Printf("  References: %d\n", len(sym.References))
				fmt.Printf("  Implementations: %d\n", len(sym.Implementations))
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to inspect BadgerDB: %v", err)
	}
}
