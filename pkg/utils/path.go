package utils

import (
	"path"
	"path/filepath"
	"strings"
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

func IsChild(parent, path string) bool {
	// 确保路径规范化（处理斜杠、相对路径等）
	parent = ToUnixPath(filepath.Clean(parent))
	path = ToUnixPath(filepath.Clean(path))

	// 计算相对路径
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false // 无法计算相对路径（如跨磁盘）
	}

	// 相对路径不能以 ".." 开头，且不能等于 "."（即相同路径）
	return !strings.HasPrefix(rel, "..") && rel != "."
}
