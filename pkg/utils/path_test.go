package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToUnixPath(t *testing.T) {
	// 测试用例
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "a//b/c", "a/b/c"},
		{"with dot", "a/./b/c", "a/b/c"},
		{"with parent", "a/b/../c", "a/c"},
		{"mixed separators", "a\\b\\c", "a/b/c"}, // 自动转换为 /
		{"root absolute", "/a/b/c", "/a/b/c"},    // 转为相对路径
		{"current dir", ".", "."},
		{"parent dir", "..", ".."},
		{"complex", "../../a/./b//c/..", "../../a/b"},
	}

	for _, tc := range testCases {
		result := ToUnixPath(tc.input)
		assert.Equal(t, tc.expected, result, fmt.Sprintf("Test case: %s", tc.name))
	}
}
