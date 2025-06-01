package codebase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
)

const logicAnd = "&"
const storeTypeLocal = "local"
const storeTypeMinio = "minio"

var ErrStoreTypeNotSupported = errors.New("store type not supported")

// generateUniquePath 使用 SHA-256 生成唯一的路径
// clientId 和 codebasePath 使用 & 作为分隔符，确保不同组合生成不同的 hash
func generateUniquePath(clientId, codebasePath string) string {
	input := clientId + logicAnd + codebasePath
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func generateCodebasePath(basePath, clientId, codebasePath string) (string, error) {
	return filepath.Join(basePath, generateUniquePath(clientId, codebasePath), codebasePath, filepathSlash), nil
}
