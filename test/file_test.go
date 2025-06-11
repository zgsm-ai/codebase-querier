package test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		CodebasePath  string
		CodebaseName  string
		ExtraMetadata string
	}{
		ClientId:      "test-client-123",
		CodebasePath:  "/tmp/test/test-project",
		CodebaseName:  "test-project",
		ExtraMetadata: `{"language": "go", "version": "1.0.0"}`,
	}

	// Create test directory structure
	err := os.MkdirAll(filepath.Join(testData.CodebasePath, "src", "api"), 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testData.CodebasePath)

	// Create test files with nested structure
	testFiles := map[string]string{
		"src/api/handler.go": "package api\n\nfunc Handler() {}\n",
		"src/main.go":        "package main\n\nfunc main() {}\n",
		"go.mod":             "module test-project\n",
	}

	// Write test files
	for relPath, content := range testFiles {
		fullPath := filepath.Join(testData.CodebasePath, relPath)
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
		CodebasePath:  testData.CodebasePath,
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
		"codebasePath":  testData.CodebasePath,
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

func TestCodebaseCompare(t *testing.T) {
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
	// Send request to local service
	testDir := "/tmp/test/test-project"
	url := fmt.Sprintf("%s/codebase-indexer/api/v1/tree?path=%s", baseURL, testDir)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []types.TreeNode
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify response structure
	assert.NotEmpty(t, result, "Project tree should not be empty")

	// Helper function to find node by name
	var findNode func([]types.TreeNode, string) *types.TreeNode
	findNode = func(nodes []types.TreeNode, name string) *types.TreeNode {
		for i := range nodes {
			if nodes[i].Name == name {
				return &nodes[i]
			}
		}
		return nil
	}

	// Verify root node
	rootNode := findNode(result, filepath.Base(testDir))
	assert.NotNil(t, rootNode, "Root node should exist")
	assert.True(t, rootNode.IsDir, "Root node should be a directory")

	// Verify expected files exist
	expectedFiles := []string{"test.go", "go.mod"}
	for _, file := range expectedFiles {
		found := false
		for _, node := range rootNode.Children {
			if node.Name == file {
				found = true
				assert.False(t, node.IsDir, "%s should be a file", file)
				break
			}
		}
		assert.True(t, found, "Expected file %s in project tree", file)
	}
}
