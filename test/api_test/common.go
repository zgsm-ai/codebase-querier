package api

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type zipOptions struct {
	// 本地项目路径
	ProjectPath string
	// 客户端ID
	ClientId string
	// 项目名称
	CodebaseName string
	// 额外的元数据，可选
	ExtraMetadata string
	// 要跳过的文件/目录前缀，可选
	ExcludePrefixes []string
	// 要跳过的文件/目录后缀，可选
	ExcludeSuffixes []string
	// 仅包含的文件后缀，可选。为空则包含所有文件
	IncludeExts []string
	// 输出zip文件的目录，如果为空则使用系统临时目录
	OutputDir     string
	SkipErrorFile bool
	// 文件列表及其操作类型（add/delete/modify）
	DeleteFileList []string
}

// createTestZip 创建用于测试的zip文件
// 返回生成的zip文件路径和可能的错误
func createTestZip(opts zipOptions) (string, error) {
	start := time.Now()
	if opts.ProjectPath == "" {
		return "", errors.New("project path cannot be empty")
	}
	if opts.ClientId == "" {
		return "", errors.New("client id cannot be empty")
	}

	// 默认跳过的文件和目录
	defaultExcludePrefixes := []string{".", "_", "node_modules", "target", "build", "dist", "bin"}
	defaultExcludeSuffixes := []string{".exe", ".dll", ".so", ".dylib", ".zip", ".tar", ".gz", ".rar"}

	// 合并默认和用户指定的排除规则
	excludePrefixes := append(defaultExcludePrefixes, opts.ExcludePrefixes...)
	excludeSuffixes := append(defaultExcludeSuffixes, opts.ExcludeSuffixes...)

	// 确定输出目录
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = os.TempDir()
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}
	// 生成zip文件路径
	timestamp := time.Now().UnixMilli()
	zipFileName := fmt.Sprintf("%s_%d.zip", opts.CodebaseName, timestamp)
	zipPath := filepath.Join(outputDir, zipFileName)

	// 创建zip文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 用于存储文件列表的map
	fileList := make(map[string]string)

	// 遍历项目目录
	err = filepath.Walk(opts.ProjectPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(opts.ProjectPath, path)
		if err != nil && opts.SkipErrorFile {
			return nil
		}
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 检查是否应该跳过此文件/目录
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(filepath.Base(relPath), prefix) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		for _, suffix := range excludeSuffixes {
			if strings.HasSuffix(relPath, suffix) {
				return nil
			}
		}

		// 如果是目录，继续遍历
		if info.IsDir() {
			return nil
		}

		// 统一使用斜杠作为路径分隔符
		relPath = filepath.ToSlash(relPath)

		// 根据ext 跳过文件
		if len(opts.IncludeExts) > 0 {
			ext := filepath.Ext(relPath)
			if !slices.Contains(opts.IncludeExts, ext) {
				return nil
			}
		}

		// 创建文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil && opts.SkipErrorFile {
			return nil
		}
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		// 写入文件内容
		writer, err := zipWriter.CreateHeader(header)
		if err != nil && opts.SkipErrorFile {
			return nil
		}
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil && opts.SkipErrorFile {
			return nil
		}
		if err != nil {
			return err
		}

		if _, err := writer.Write(content); err != nil {
			if err != nil && opts.SkipErrorFile {
				return nil
			}
			return err
		}

		// 添加到文件列表

		fileList[relPath] = types.FileOpAdd
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	for _, v := range opts.DeleteFileList {
		fileList[v] = types.FileOpDelete
	}

	// 创建并写入元数据文件
	metadataFileName := fmt.Sprintf(".shenma_sync/%d", timestamp)
	syncMetadata := types.SyncMetadataFile{
		ClientID:      opts.ClientId,
		CodebasePath:  opts.ProjectPath,
		ExtraMetadata: opts.ExtraMetadata,
		FileList:      fileList,
		Timestamp:     timestamp,
	}

	metadataContent, err := json.MarshalIndent(&syncMetadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataHeader := &zip.FileHeader{
		Name:     metadataFileName,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}

	metadataWriter, err := zipWriter.CreateHeader(metadataHeader)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata header: %w", err)
	}

	if _, err := metadataWriter.Write(metadataContent); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}
	fmt.Printf("zip file created: %s, cost: %dms\n", zipPath, time.Now().UnixMilli()-start.UnixMilli())
	return zipPath, nil
}
