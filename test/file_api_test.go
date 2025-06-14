package test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/response"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// Note: to run these tests, start server manually at first.
const (
	baseURL = "http://localhost:8888" // 替换为实际的服务地址和端口
)

func TestFileUpload(t *testing.T) {
	// Prepare test data
	testData := struct {
		ClientId      string
		ClientPath    string
		CodebaseName  string
		ExtraMetadata string
	}{
		ClientId:      "test-client-123",
		ClientPath:    "/tmp/test/test-project",
		CodebaseName:  "test-project",
		ExtraMetadata: `{"language": "go", "version": "1.0.0"}`,
	}

	// Create test files with nested structure
	testFiles := map[string]string{
		"src/api/handler.go":         "package logic  \n  \nimport (  \n\t\"context\"  \n\t\"errors\"  \n\t\"fmt\"  \n  \n\t\"github.com/zgsm-ai/codebase-indexer/internal/errs\"  \n\t\"github.com/zgsm-ai/codebase-indexer/internal/svc\"  \n\t\"github.com/zgsm-ai/codebase-indexer/internal/types\"  \n\t\"github.com/zgsm-ai/codebase-indexer/pkg/utils\"  \n  \n\t\"github.com/zeromicro/go-zero/core/logx\"  \n\t\"gorm.io/gorm\"  \n)  \n  \ntype CodebaseTreeLogic struct {  \n\tlogx.Logger  \n\tctx    context.Context  \n\tsvcCtx *svc.ServiceContext  \n}  \n  \nfunc NewCodebaseTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CodebaseTreeLogic {  \n\treturn &CodebaseTreeLogic{  \n\t\tLogger: logx.WithContext(ctx),  \n\t\tctx:    ctx,  \n\t\tsvcCtx: svcCtx,  \n\t}  \n}  \n  \nfunc (l *CodebaseTreeLogic) CodebaseTree(req *types.CodebaseTreeRequest) (resp *types.CodebaseTreeResponseData, err error) {  \n\t// 1. 从数据库查询 codebase 信息  \n\tclientCodebasePath := req.ClientPath  \n\tclientId := req.ClientId  \n\tcodebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)  \n\tif errors.Is(err, gorm.ErrRecordNotFound) {  \n\t\treturn nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf(\"client_id: %s, clientCodebasePath: %s\", clientId, clientCodebasePath))  \n\t}  \n\tif err != nil {  \n\t\treturn nil, err  \n\t}  \n  \n\tcodebasePath := codebase.Path  \n\tif utils.IsBlank(codebasePath) {  \n\t\treturn nil, errors.New(\"codebase path is empty\")  \n\t}  \n  \n\t// 2. 获取目录树  \n\tstore := l.svcCtx.CodebaseStore  \n\ttreeOpts := types.TreeOptions{  \n\t\tMaxDepth: req.Depth,  \n\t}  \n  \n\tnodes, err := store.Tree(l.ctx, codebasePath, req.SubDir, treeOpts)  \n\tif err != nil {  \n\t\tl.Logger.Errorf(\"failed to get directory tree: %v\", err)  \n\t\treturn nil, err  \n\t}  \n  \n\t// 3. 计算文件统计信息  \n\tvar totalFiles int  \n\tvar totalSize int64  \n\tif len(nodes) > 0 {  \n\t\tcountFilesAndSize(nodes, &totalFiles, &totalSize, req.IncludeFiles == 1)  \n\t}  \n  \n\tresp = &types.CodebaseTreeResponseData{  \n\t\tCodebaseId:    codebase.ID,  \n\t\tName:          codebase.Name,  \n\t\tRootPath:      codebasePath,  \n\t\tTotalFiles:    totalFiles,  \n\t\tTotalSize:     totalSize,  \n\t\tDirectoryTree: nodes,  \n\t}  \n  \n\treturn resp, nil  \n}  \n  \n// countFilesAndSize 统计文件数量和总大小  \nfunc countFilesAndSize(nodes []*types.TreeNode, totalFiles *int, totalSize *int64, includeFiles bool) {  \n\tif len(nodes) == 0 {  \n\t\treturn  \n\t}  \n  \n\tfor _, node := range nodes {  \n\t\tif node == nil {  \n\t\t\tcontinue  \n\t\t}  \n  \n\t\tif !node.IsDir {  \n\t\t\tif includeFiles {  \n\t\t\t\t*totalFiles++  \n\t\t\t\t*totalSize += node.Size  \n\t\t\t}  \n\t\t\tcontinue  \n\t\t}  \n  \n\t\t// 递归处理子节点  \n\t\tcountFilesAndSize(node.Children, totalFiles, totalSize, includeFiles)  \n\t}  \n}  \n",
		"src/main.go":                "package main\n\nimport (\n\t\"context\"\n\t\"flag\"\n\t\"fmt\"\n\t\"github.com/zeromicro/go-zero/core/conf\"\n\t\"github.com/zeromicro/go-zero/core/logx\"\n\t\"github.com/zeromicro/go-zero/rest\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/config\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/handler\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/job\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/svc\"\n\t\"github.com/zgsm-ai/codebase-indexer/migrations\"\n\t\"net/http\"\n)\n\nvar configFile = flag.String(\"f\", \"etc/conf.yaml\", \"the config file\")\n\nfunc main() {\n\tflag.Parse()\n\n\tvar c config.Config\n\tconf.MustLoad(*configFile, &c, conf.UseEnv())\n\n\tlogx.MustSetup(c.Log)\n\n\tif err := migrations.AutoMigrate(c.Database); err != nil {\n\t\tpanic(err)\n\t}\n\n\tserver := rest.MustNewServer(c.RestConf, rest.WithFileServer(\"/swagger/\", http.Dir(\"api/docs/\")))\n\tdefer server.Stop()\n\n\tserverCtx, cancelFunc := context.WithCancel(context.Background())\n\tdefer cancelFunc()\n\tsvcCtx, err := svc.NewServiceContext(serverCtx, c)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tdefer svcCtx.Close()\n\n\tjobScheduler, err := job.NewScheduler(serverCtx, svcCtx)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tjobScheduler.Schedule()\n\tdefer jobScheduler.Close()\n\n\thandler.RegisterHandlers(server, svcCtx)\n\n\tfmt.Printf(\"Started server at %s:%d\n\", c.Host, c.Port)\n\tserver.Start()\n}\n",
		"src/svc/service_context.go": "package svc\n\nimport (\n\t\"context\"\n\t\"github.com/redis/go-redis/v9\"\n\t\"github.com/zeromicro/go-zero/core/logx\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/codegraph/structure\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/config\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/dao/query\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/embedding\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/store/cache\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/store/codebase\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/store/database\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/store/mq\"\n\tredisstore \"github.com/zgsm-ai/codebase-indexer/internal/store/redis\"\n\t\"github.com/zgsm-ai/codebase-indexer/internal/store/vector\"\n\t\"gorm.io/gorm\"\n)\n\ntype ServiceContext struct {\n\tConfig          config.Config\n\tCodegraphConf   *config.CodegraphConfig\n\tdb              *gorm.GormDB\n\tQuerier         *query.Query\n\tCodebaseStore   codebase.Store\n\tMessageQueue    mq.MessageQueue\n\tDistLock        redisstore.DistributedLock\n\tEmbedder        vector.Embedder\n\tVectorStore     vector.Store\n\tCodeSplitter    *embedding.CodeSplitter\n\tCache           cache.Store[any]\n\tredisClient     *redis.Client // 保存Redis客户端引用以便关闭\n\tStructureParser *structure.Parser\n}\n\n// Close closes the shared Redis client and database connection\nfunc (s *ServiceContext) Close() {\n\tvar errs []error\n\tif s.redisClient != nil {\n\t\tif err := s.redisClient.Close(); err != nil {\n\t\t\terrs = append(errs, err)\n\t\t}\n\t}\n\tif s.db != nil {\n\t\tif err := database.CloseDB(s.db); err != nil {\n\t\t\terrs = append(errs, err)\n\t\t}\n\t}\n\tif len(errs) > 0 {\n\t\tlogx.Errorf(\"service_context close err:%v\", errs)\n\t}\n}\n\nfunc NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {\n\tvar err error\n\tsvcCtx := &ServiceContext{\n\t\tConfig: c,\n\t}\n\tsvcCtx.CodegraphConf = config.MustLoadCodegraphConfig(c.IndexTask.GraphTask.ConfFile)\n\n\t// 初始化数据库连接\n\tdb, err := database.NewPostgresDB(c.Database)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tsvcCtx.db = db\n\n\tquerier := query.Use(db)\n\tsvcCtx.Querier = querier\n\n\t// 创建Redis客户端\n\tclient, err := redisstore.NewRedisClient(c.Redis)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tsvcCtx.redisClient = client\n\n\t// 创建各个组件，共用Redis客户端\n\tmessageQueue, err := mq.NewRedisMQ(ctx, client, c.MessageQueue.ConsumerGroup)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tlock, err := redisstore.NewRedisDistributedLock(client)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tcacheStore := cache.NewRedisStore[any](client)\n\n\tcodebaseStore, err := codebase.NewLocalCodebase(ctx, c.CodeBaseStore)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tembedder, err := vector.NewEmbedder(c.VectorStore.Embedder)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treranker := vector.NewReranker(c.VectorStore.Reranker)\n\n\tvectorStore, err := vector.NewVectorStore(ctx, c.VectorStore, embedder, reranker)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tsplitter, err := embedding.NewCodeSplitter(embedding.SplitOptions{\n\t\tMaxTokensPerChunk:          c.IndexTask.EmbeddingTask.MaxTokensPerChunk,\n\t\tSlidingWindowOverlapTokens: c.IndexTask.EmbeddingTask.OverlapTokens})\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tparser, err := structure.NewStructureParser()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tsvcCtx.StructureParser = parser\n\tsvcCtx.CodebaseStore = codebaseStore\n\tsvcCtx.MessageQueue = messageQueue\n\tsvcCtx.VectorStore = vectorStore\n\tsvcCtx.Embedder = embedder\n\tsvcCtx.CodeSplitter = splitter\n\tsvcCtx.DistLock = lock\n\tsvcCtx.Cache = cacheStore\n\n\treturn svcCtx, err\n}\n",
		"go.mod":                     "module test-project\n",
	}

	// Write test files
	for relPath, content := range testFiles {
		fullPath := filepath.Join(testData.ClientPath, relPath)
		err := os.MkdirAll(filepath.Join(testData.ClientPath, filepath.Dir(relPath)), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Create ZIP file
	zipPath := filepath.Join(os.TempDir(), "test-project.zip")
	defer os.Remove(zipPath)

	zipFile, err := os.Create(zipPath)
	assert.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add files to ZIP using relative paths
	for relPath, content := range testFiles {
		// Create ZIP entry with relative path
		zipHeader := &zip.FileHeader{
			Name:     relPath, // Use relative path directly
			Method:   zip.Deflate,
			Modified: time.Now(),
		}

		fileWriter, err := zipWriter.CreateHeader(zipHeader)
		assert.NoError(t, err)

		_, err = fileWriter.Write([]byte(content))
		assert.NoError(t, err)
	}

	// Add metadata file in .shenma_sync directory
	timestamp := time.Now().UnixMilli()
	metadataFileName := fmt.Sprintf(".shenma_sync/%d", timestamp)

	// Create sync metadata
	syncMetadata := types.SyncMetadataFile{
		ClientID:      testData.ClientId,
		CodebasePath:  testData.ClientPath,
		ExtraMetadata: json.RawMessage(testData.ExtraMetadata),
		FileList:      make(map[string]string),
		Timestamp:     timestamp,
	}

	// Add all files as "add" operations
	for relPath := range testFiles {
		syncMetadata.FileList[relPath] = "add"
	}

	// Marshal metadata to JSON
	metadataContent, err := json.MarshalIndent(syncMetadata, "", "  ")
	assert.NoError(t, err)

	// Add metadata file to ZIP
	metadataHeader := &zip.FileHeader{
		Name:     metadataFileName,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}

	metadataWriter, err := zipWriter.CreateHeader(metadataHeader)
	assert.NoError(t, err)

	_, err = metadataWriter.Write(metadataContent)
	assert.NoError(t, err)

	zipWriter.Close()

	// Create multipart form request
	body := &bytes.Buffer{}
	formWriter := multipart.NewWriter(body)

	// Add form fields
	formFields := map[string]string{
		"clientId":      testData.ClientId,
		"codebasePath":  testData.ClientPath,
		"codebaseName":  testData.CodebaseName,
		"extraMetadata": testData.ExtraMetadata,
	}

	for field, value := range formFields {
		err = formWriter.WriteField(field, value)
		assert.NoError(t, err)
	}

	// Add ZIP file
	file, err := os.Open(zipPath)
	assert.NoError(t, err)
	defer file.Close()

	part, err := formWriter.CreateFormFile("file", "test-project.zip")
	assert.NoError(t, err)
	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	formWriter.Close()

	// Send request to local service
	url := fmt.Sprintf("%s/codebase-indexer/api/v1/files/upload", baseURL)
	req, err := http.NewRequest("POST", url, body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	client := &http.Client{
		Timeout: time.Second * 30,
	}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	t.Logf("resp:%+v", result)

	assert.NoError(t, err)
	assert.Equal(t, float64(0), result["code"]) // 假设 0 表示成功
}

func TestFileDownload(t *testing.T) {
	// Prepare test data
	req := types.FileContentRequest{
		ClientId:     "test-client-123",
		CodebasePath: "/tmp/test/test-project",
		FilePath:     "src/main.go",
		StartLine:    1,
		EndLine:      3,
	}

	// Send request to local service
	url := fmt.Sprintf("%s/codebase-indexer/api/v1/files/content?clientId=%s&codebasePath=%s&filePath=%s&startLine=%d&endLine=%d",
		baseURL, req.ClientId, req.CodebasePath, req.FilePath, req.StartLine, req.EndLine)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read and verify response
	content, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "package main")
}

func TestCodebaseHash(t *testing.T) {
	// Prepare test data
	req := types.CodebaseHashRequest{
		ClientId:     "test-client-123",
		CodebasePath: "/tmp/test/test-project",
	}

	// Send request to local service
	url := fmt.Sprintf("%s/codebase-indexer/api/v1/codebases/hash?clientId=%s&codebasePath=%s",
		baseURL, req.ClientId, req.CodebasePath)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result response.Response[types.CodebaseHashResponseData]
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	// Verify response contains expected files
	foundFiles := make(map[string]bool)
	for _, item := range result.Data.CodebaseHash {
		foundFiles[filepath.Base(item.Path)] = true
	}

	expectedFiles := []string{"main.go", "go.mod"}
	for _, file := range expectedFiles {
		assert.True(t, foundFiles[file], "Expected file %s in response", file)
	}
}

func TestProjectTree(t *testing.T) {
	// Test data
	testData := struct {
		clientId     string
		codebasePath string
		depth        int
		includeFiles int
		subDir       string
	}{
		clientId:     "test-client-123",
		codebasePath: "/tmp/test/test-project",
		depth:        3,
		includeFiles: 1,
		subDir:       "",
	}

	// Send request to local service
	url := fmt.Sprintf("%s/codebase-indexer/api/v1/codebases/directory?clientId=%s&codebasePath=%s&depth=%d&includeFiles=%d&subDir=%s",
		baseURL, testData.clientId, testData.codebasePath, testData.depth, testData.includeFiles, testData.subDir)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result response.Response[types.CodebaseTreeResponseData]
	err = json.NewDecoder(resp.Body).Decode(&result)
	t.Logf("resp:%+v", &result)
	assert.NoError(t, err)

	// Verify response structure
	assert.NotEmpty(t, result, "Project tree should not be empty")

	// Helper function to find node by name
	var findNode func([]*types.TreeNode, string) *types.TreeNode
	findNode = func(nodes []*types.TreeNode, name string) *types.TreeNode {
		for i := range nodes {
			if nodes[i].Name == name {
				return nodes[i]
			}
		}
		return nil
	}

	// Verify root node (should be "src" since we specified it as subDir)
	rootNode := findNode(result.Data.DirectoryTree, "src")
	assert.NotNil(t, rootNode, "Root node should exist")
	assert.True(t, rootNode.IsDir, "Root node should be a directory")

	// Verify expected files exist in the src directory
	expectedFiles := []string{"main.go", "api/handler.go"}
	for _, file := range expectedFiles {
		baseName := filepath.Base(file)
		dirName := filepath.Dir(file)

		var node *types.TreeNode
		if dirName == "." {
			node = findNode(rootNode.Children, baseName)
		} else {
			dirNode := findNode(rootNode.Children, dirName)
			if dirNode != nil {
				node = findNode(dirNode.Children, baseName)
			}
		}

		assert.NotNil(t, node, "Expected file %s in project tree", file)
		if node != nil {
			assert.False(t, node.IsDir, "%s should be a file", file)
		}
	}
}
