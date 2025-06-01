package test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
)
import scipindex "github.com/zgsm-ai/codebase-indexer/internal/job/codegraph/scip"

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
		localCodebase := codebase.NewLocalCodebase(context.Background(), storeConf)
		generator := scipindex.NewIndexGenerator(scipConf, localCodebase)
		err = generator.Generate(context.Background(), codebasePath)
		assert.NoError(t, err)
		indexFile := filepath.Join(testProjectsBaseDir, projectPath, types.CodebaseIndexDir, indexFileName)
		fmt.Println(indexFile)
		assert.FileExists(t, indexFile)

	})
}
