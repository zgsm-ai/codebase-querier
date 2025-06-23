package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	scipindex "github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
	"time"
)

const testProjectsBaseDir = "/tmp/projects"
const indexFileName = "index.scip"

func TestParseScipIndex(t *testing.T) {
	start := time.Now()
	projectPath := "go/kubernetes"
	codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
	storeConf := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: testProjectsBaseDir,
		},
	}
	ctx := context.Background()
	localCodebase, err := codebase.NewLocalCodebase(storeConf)
	assert.NoError(t, err)
	//indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	indexFile := filepath.Join(types.CodebaseIndexDir, indexFileName)
	assert.NoError(t, err)
	graph, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	defer graph.Close()
	parser := scipindex.NewIndexParser(localCodebase, graph)
	ctx = context.WithValue(ctx, tracer.Key, tracer.TaskTraceId(1))
	err = parser.ProcessScipIndexFile(ctx, codebasePath, indexFile)
	assert.NoError(t, err)
	t.Logf("time cost: %v seconds", time.Since(start).Seconds())
}
