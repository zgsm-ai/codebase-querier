package test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
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
		t.Logf("index file: %s", indexFile)
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
	ctx := context.Background()
	localCodebase, err := codebase.NewLocalCodebase(ctx, storeConf)
	assert.NoError(t, err)
	//indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	graph, err := codegraph.NewBadgerDBGraph(ctx, codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	defer graph.Close()
	assert.NoError(t, err)
	parser := scipindex.NewIndexParser(ctx, localCodebase, graph)
	err = parser.ParseSCIPFile(ctx, codebasePath, indexFile)
	assert.NoError(t, err)
	t.Logf("time cost: %v seconds", time.Since(start).Seconds())
}

func TestQueryBadgerDB(t *testing.T) {
	start := time.Now()
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)

	// 1. 初始化存储
	ctx := context.Background()
	graph, err := codegraph.NewBadgerDBGraph(ctx, codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	defer graph.Close()

	// 2. 初始化 codebase store
	storeConf := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: testProjectsBaseDir,
		},
	}
	localCodebase, err := codebase.NewLocalCodebase(ctx, storeConf)
	assert.NoError(t, err)

	// 3. 解析和写入数据
	indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	parser := scipindex.NewIndexParser(ctx, localCodebase, graph)
	err = parser.ParseSCIPFile(ctx, codebasePath, indexFile)
	assert.NoError(t, err)
	t.Logf("store time: %f seconds", time.Since(start).Seconds())

	// 4. 执行查询
	targetPath := "cmd/kubeadm/app/util/endpoint.go"
	t.Logf("\nQuerying for file: %s\n", targetPath)
	references, err := graph.Query(ctx, &types.RelationQueryOptions{
		FilePath:   targetPath,
		StartLine:  37,
		EndLine:    37,
		SymbolName: "GetControlPlaneEndpoint",
	})
	if err != nil {
		t.Logf("Query error: %v\n", err)
	}
	assert.True(t, len(references) > 0)
	t.Log("-----------------------------------------------")
	for _, v := range references {
		t.Logf("references name: %v", v.SymbolName)
		t.Logf("references content: %v", v.Content)
		t.Logf("references filepath: %v", v.FilePath)
		t.Logf("references nodetype: %v", v.NodeType)
		t.Logf("references position: %v", v.Position)
		t.Logf("references children cnt: %v", len(v.Children))
	}
	t.Log("-----------------------------------------------")
}

func TestDeleteBadgerDB(t *testing.T) {
	start := time.Now()
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
	graph, err := codegraph.NewBadgerDBGraph(context.Background(),
		codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	assert.NotPanics(t, func() {
		err = graph.DeleteAll(context.Background())
	})
	t.Logf("time cost %d ms", time.Since(start).Milliseconds())
}

func TestInspectBadgerDB(t *testing.T) {
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)

	// 初始化 BadgerDB
	graph, err := codegraph.NewBadgerDBGraph(context.Background(), codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		t.Fatalf("Failed to initialize BadgerDB: %v", err)
	}
	defer graph.Close()

	// 获取 BadgerDB 实例
	db := graph.(*codegraph.BadgerDBGraph).DB()

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
			default:
				fmt.Printf(" unexpected Content: %s\n", string(key))
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to inspect BadgerDB: %v", err)
	}
}
