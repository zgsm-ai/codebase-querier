package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/response"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCreateZipFile(t *testing.T) {

	clientLocalPath := "G:\\tmp\\projects\\go\\kubernetes"
	testZip, err := createTestZip(zipOptions{
		ClientId:        clientId,
		ProjectPath:     clientLocalPath,
		CodebaseName:    "kubernetes",
		ExcludePrefixes: []string{".", "vendor"},
		ExcludeSuffixes: []string{".md"},
		IncludeExts:     []string{".go"}, // 只包含 .go
		OutputDir:       zipOutputDir,
		SkipErrorFile:   true,
	})
	assert.NoError(t, err)
	t.Logf("testZip:%s", testZip)
	defer func() {
		// 清理测试文件
		err = os.Remove(testZip)
		assert.NoError(t, err)
	}()

	// 校验文件内容，包括元数据目录、文件列表和文件内容
	reader, err := zip.OpenReader(testZip)
	assert.NoError(t, err)
	defer reader.Close()

	// 用于存储找到的文件
	var metadataFile *zip.File
	foundFiles := make(map[string]bool)

	// 遍历所有文件
	for _, file := range reader.File {
		// 记录找到的文件
		foundFiles[file.Name] = true

		// 检查是否是元数据文件
		if strings.HasPrefix(file.Name, ".shenma_sync/") {
			metadataFile = file
			continue
		}

		// 验证文件后缀
		if !strings.HasPrefix(file.Name, ".shenma_sync/") {
			ext := strings.ToLower(filepath.Ext(file.Name))
			assert.Contains(t, []string{".go", ".proto"}, ext, "File %s should have allowed extension", file.Name)
		}
	}

	// 验证元数据文件存在
	assert.NotNil(t, metadataFile, "Metadata file should exist")

	// 读取并解析元数据文件
	if metadataFile != nil {
		rc, err := metadataFile.Open()
		assert.NoError(t, err)
		defer rc.Close()

		var metadata types.SyncMetadataFile
		err = json.NewDecoder(rc).Decode(&metadata)
		assert.NoError(t, err)

		// 验证元数据内容
		assert.Equal(t, clientId, metadata.ClientID)
		assert.Equal(t, clientLocalPath, metadata.CodebasePath)
		assert.NotEmpty(t, metadata.FileList, "File list should not be empty")
		assert.Greater(t, metadata.Timestamp, int64(0), "Timestamp should be positive")

		// 验证文件列表中的文件都在 zip 中
		for filePath := range metadata.FileList {
			assert.True(t, foundFiles[filePath], "File %s from metadata should exist in zip", filePath)
			if !strings.HasPrefix(filePath, ".shenma_sync/") {
				ext := strings.ToLower(filepath.Ext(filePath))
				assert.Contains(t, []string{".go", ".proto"}, ext, "File %s in metadata should have allowed extension", filePath)
			}
		}
	}

	// 验证排除规则是否生效
	for path := range foundFiles {
		// 验证没有 vendor 目录
		assert.False(t, strings.Contains(path, "vendor/"), "Should not contain vendor directory")
		// 验证没有点开头的文件/目录（除了元数据目录）
		if !strings.HasPrefix(path, ".shenma_sync/") {
			assert.False(t, strings.HasPrefix(filepath.Base(path), "."), "Should not contain dot files")
			// 验证没有 .md 文件
			assert.False(t, strings.HasSuffix(path, ".md"), "Should not contain .md files")
		}
	}
}

// Common test setup for file operations
type fileTestOptions struct {
	ClientId       string
	ClientPath     string
	CodebaseName   string
	ExtraMetadata  string
	DeleteFileList []string // for metadata file list
}

func setupFileTest(t *testing.T, opts fileTestOptions) string {
	testZip, err := createTestZip(zipOptions{
		ProjectPath:     opts.ClientPath,
		ClientId:        opts.ClientId,
		CodebaseName:    opts.CodebaseName,
		OutputDir:       zipOutputDir,
		SkipErrorFile:   true,
		ExcludePrefixes: []string{"."},
		DeleteFileList:  opts.DeleteFileList, // Add file list to metadata
	})
	assert.NoError(t, err)
	return testZip
}

func sendFileUploadRequest(t *testing.T, opts fileTestOptions, zipFile string) map[string]interface{} {
	// Create multipart form request
	body := &bytes.Buffer{}
	formWriter := multipart.NewWriter(body)

	// Add form fields
	formFields := map[string]string{
		"clientId":      opts.ClientId,
		"codebasePath":  opts.ClientPath,
		"codebaseName":  opts.CodebaseName,
		"extraMetadata": opts.ExtraMetadata,
	}

	for field, value := range formFields {
		err := formWriter.WriteField(field, value)
		assert.NoError(t, err)
	}

	// Add ZIP file
	file, err := os.Open(zipFile)
	assert.NoError(t, err)
	defer file.Close()

	part, err := formWriter.CreateFormFile("file", "file.zip")
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
		Timeout: time.Second * 300,
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
	return result
}

func TestFileUpload(t *testing.T) {
	// Prepare test data
	opts := fileTestOptions{
		ClientId:      "test-client-123",
		ClientPath:    "F:\\tmp\\projects\\go\\kubernetes",
		CodebaseName:  "kubernetes",
		ExtraMetadata: `{"language": "go", "version": "1.0.0"}`,
	}

	testZip := setupFileTest(t, opts)
	defer func() {
		// 清理测试文件
		err := os.Remove(testZip)
		assert.NoError(t, err)
	}()

	result := sendFileUploadRequest(t, opts, testZip)
	assert.Equal(t, float64(0), result["code"]) // 假设 0 表示成功
}

func TestFileDelete(t *testing.T) {
	// Prepare test data with delete operation
	opts := fileTestOptions{
		ClientId:      "test-client-123",
		ClientPath:    "F:\\tmp\\projects\\go\\kubernetes",
		CodebaseName:  "kubernetes",
		ExtraMetadata: `{"language": "go", "version": "1.0.0"}`,
		DeleteFileList: []string{
			"cluster/gce/gci/mounter/mounter.go",
			"cluster/gce/gci/apiserver_etcd_test.go",
			"cluster/gce/gci/apiserver_kms_test.go",
		},
	}
	ctx := context.Background()
	svcCtx := InitSvcCtx(ctx, nil)
	// Get the real codebase path from database
	codebase, err := svcCtx.Querier.Codebase.FindByClientIdAndPath(ctx, opts.ClientId, opts.ClientPath)
	assert.NoError(t, err)
	assert.NotNil(t, codebase)

	// Assert files exist before deletion
	for _, file := range opts.DeleteFileList {
		exists, err := svcCtx.CodebaseStore.Exists(context.Background(), codebase.Path, file)
		assert.NoError(t, err)
		assert.True(t, exists, "File %s should exist before deletion", file)
	}

	testZip := setupFileTest(t, opts)
	defer func() {
		// 清理测试文件
		err := os.Remove(testZip)
		assert.NoError(t, err)
	}()

	result := sendFileUploadRequest(t, opts, testZip)
	assert.Equal(t, float64(0), result["code"]) // 假设 0 表示成功

	// Assert files do not exist after deletion
	for _, file := range opts.DeleteFileList {
		exists, err := svcCtx.CodebaseStore.Exists(context.Background(), codebase.Path, file)
		assert.NoError(t, err)
		assert.False(t, exists, "File %s should not exist after deletion", file)
	}
}

func TestFileRead(t *testing.T) {
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
