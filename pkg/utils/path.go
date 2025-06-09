package utils

import (
	"path"
	"strings"
)

// ToUnixPath 将相对路径转换为 Unix 风格（使用 / 分隔符，去除冗余路径元素）
func ToUnixPath(rawPath string) string {
	// path.Clean 会自动处理为 Unix 风格路径，去除多余的 /、. 和 ..
	filePath := path.Clean(rawPath)
	if strings.Contains(filePath, "\\") {
		filePath = strings.ReplaceAll(filePath, "\\", "/")
	}
	return filePath
}
