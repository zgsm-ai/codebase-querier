package utils

import "path/filepath"

// CleanPath convert to unix path
func CleanPath(p string) string {
	cleaned := filepath.Clean(p)
	return filepath.ToSlash(cleaned)
}
