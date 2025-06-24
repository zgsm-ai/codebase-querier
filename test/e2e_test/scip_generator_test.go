package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	scipindex "github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"path/filepath"
	"testing"
)

const testProjectsBaseDir = "G:\\work\\scip-test"
const indexFileName = "index.scip"

func TestParseScipIndex(t *testing.T) {
	storeConf := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: testProjectsBaseDir,
		},
	}
	ctx := context.Background()
	localCodebase, err := codebase.NewLocalCodebase(storeConf)
	assert.NoError(t, err)
	assert.NoError(t, err)

	testCases := []struct {
		language string
		project  string
	}{
		{
			language: "c",
			project:  "sqlite",
			// scip-clang
			// symbol:
			//    cxx . . $ `tool/mkkeywordhash.c:150:11`!
			//    cxx . . $ SHA3Context#$anonymous_type_0#
			//    cxx . . $ SHA3Final(ea2187754a463e1e).   (函数)
			// name:
			//   tool/mksourceid.c:96:10
		},
		{
			language: "cpp",
			project:  "whisper.cpp",
			// scip-clang
			// symbol:
			//   cxx . . $ `<file>/build/CMakeFiles/FindOpenMP/OpenMPTryFlag.cpp`/
			//   cxx . . $ ggml_are_same_layout(3bd891636d1661b7).    (method)
			//   cxx . . $ ggml_dyn_tallocr#max_size.
			// name:
			//   <file>/ggml/src/ggml-cpu/amx/common.h
		},
		{
			language: "go",
			project:  "gin",
			// scip-go
			// symbol:
			//     scip-go gomod github.com/gin-gonic/gin . `github.com/gin-gonic/gin`/
			//     scip-go gomod github.com/gin-gonic/gin . `github.com/gin-gonic/gin`/authPairs#
			// name:
			//     authPairs
		},
		{
			language: "java",
			project:  "hadoop",
			// scip-java
			// symbol:
			//      semanticdb maven maven/org.apache.hadoop/hadoop-annotations 3.5.0-SNAPSHOT org/apache/hadoop/classification/InterfaceAudience#
			// name:
		},
		{
			language: "javascript",
			project:  "vue",
			// scip-typescript   TODO namespace 参数有问题 ， relation的name是identifier， 有些name是空的
			// symbol:
			//    scip-typescript npm vue 2.7.16 src/`global.d.ts`/
			//    scip-typescript npm vue 2.7.16 src/`global.d.ts`/DevtoolsHook#Vue.
			// name:
			//    Vue
		},
		{
			language: "typescript",
			project:  "vue-next",
			// scip-typescript  todo name有空的，namespace只有src， relation的name是identifier，得解析下
			// symbol:
			//     scip-typescript npm @vue/shared 3.5.16 src/`looseEqual.ts`/
			//     scip-typescript npm @vue/shared 3.5.16 src/`looseEqual.ts`/looseCompareArrays().(a)
			// name:
			//     looseEqual
		},
		{
			language: "python",
			project:  "django",
			// scip-python
			// symbol:  TODO name 有空的, Namepsce 也是
			//     scip-python python Django 1 django/__init__:
			//     scip-python python Django 1 `django.core.exceptions`/ImproperlyConfigured#
			//     scip-python python python-stdlib 3.11 builtins/RuntimeError#
			//     scip-python python Django 1 `django.apps.registry`/Apps#ready.
			// name:
			//    __init__
			// namespace: django.apps.registry
		},
		{
			language: "ruby",
			project:  "vagrant",
			// scip-ruby
			// symbol:
			//     scip-ruby gem ruby v0.0.1 Kernel#require_relative().
			//     scip-ruby gem ruby v0.0.1 Vagrant#Action#Builtin#
			// name:
			//     initialize
			// namespace: Vagrant
		},
		{
			language: "rust",
			project:  "deno",
		},
		// rust-analyzer    TODO namespace 会解析出错误的东西
		// symbol:
		//       rust-analyzer cargo test_sqlite_extension 0.1.0 crate/
		//       rust-analyzer cargo test_sqlite_extension 0.1.0 SQLITE3_EXTENSION_INIT2!
		//       rust-analyzer cargo deno_runtime 0.215.0 worker_bootstrap/WorkerExecutionMode#
		//       rust-analyzer cargo deno_runtime 0.215.0 worker_bootstrap/BootstrapOptions#enable_op_summary_metrics.
		// name:
		//       SQLITE3_EXTENSION_INIT2!
		//       WorkerExecutionMode
		// namespace:
		//       worker_bootstrap
	}

	for _, tt := range testCases {
		t.Run(tt.language, func(t *testing.T) {
			codebasePath := filepath.Join(testProjectsBaseDir, tt.language, tt.project)
			parser := scipindex.NewIndexParser(localCodebase)
			ctx = context.WithValue(ctx, tracer.Key, tracer.TaskTraceId(1))
			metadata, documents, err := parser.ParseScipIndexFile(ctx, codebasePath, indexFileName)
			assert.NoError(t, err)
			assert.NotNil(t, metadata)
			assert.NotNil(t, documents)
			assert.Greater(t, len(documents), 0)
		})
	}

}
