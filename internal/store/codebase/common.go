package codebase

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

// generateUniquePath 使用 SHA-256 生成唯一的路径
// clientId 和 codebasePath 使用 & 作为分隔符，确保不同组合生成不同的 hash
func generateUniquePath(clientId, codebasePath string) string {
	input := clientId + "&" + codebasePath
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// getFullPath 获取完整的文件路径
// basePath: 基础路径（本地存储的根目录或 MinIO 的 bucket）
// clientId: 客户端 ID
// codebasePath: 代码库路径
// target: 目标文件或目录路径
func getFullPath(basePath, clientId, codebasePath, target string) string {
	uniquePath := generateUniquePath(clientId, codebasePath)
	return filepath.Join(basePath, uniquePath, target)
}

// getObjectName 获取 MinIO 对象名称
// clientId: 客户端 ID
// codebasePath: 代码库路径
// target: 目标文件或目录路径
func getObjectName(clientId, codebasePath, target string) string {
	uniquePath := generateUniquePath(clientId, codebasePath)
	return filepath.Join(uniquePath, target)
}
