package e2e_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	scipindex "github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
)

func Test_GenerateCScipIndex(t *testing.T) {
	// run ./fetch_test_projects.sh to clone real projects from github
	t.Run("redis", func(t *testing.T) {
		projectPath := "c/redis"
		codebasePath := filepath.Join(testProjectsBaseDir, projectPath)
		storeConf := config.CodeBaseStoreConf{
			Local: config.LocalStoreConf{
				BasePath: testProjectsBaseDir,
			},
		}
		scipConf := config.MustLoadCodegraphConfig("../../etc/codegraph.yaml")

		localCodebase, err := codebase.NewLocalCodebase(storeConf)
		assert.NoError(t, err)
		generator := scipindex.NewIndexGenerator(scipConf, localCodebase)
		err = generator.Generate(context.Background(), codebasePath)
		assert.NoError(t, err)
		indexFile := filepath.Join(testProjectsBaseDir, projectPath, types.CodebaseIndexDir, indexFileName)
		t.Logf("index file: %s", indexFile)
		assert.FileExists(t, indexFile)

	})
}
