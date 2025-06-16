package utils

import (
	"path"
	"path/filepath"
)

// ToUnixPath 将相对路径转换为 Unix 风格（使用 / 分隔符，去除冗余路径元素）
func ToUnixPath(rawPath string) string {
	// path.Clean 会自动处理为 Unix 风格路径，去除多余的 /、. 和 ..
	filePath := path.Clean(rawPath)
	filePath = filepath.ToSlash(filePath)
	return filePath
}

// PathEqual 比较路径是否相等，/ \ 转为 /
func PathEqual(a, b string) bool {
	return filepath.ToSlash(a) == filepath.ToSlash(b)
}
