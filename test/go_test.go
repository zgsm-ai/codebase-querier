package test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	scipindex "github.com/zgsm-ai/codebase-indexer/internal/job/codegraph/scip"
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
	references, err := graph.GetSymbolReferences(context.Background(), "ErrPodCompleted")
	assert.NoError(t, err)
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
