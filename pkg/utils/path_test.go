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

func TestIsChild(t *testing.T) {
	tests := []struct {
		parent string
		path   string
		want   bool
	}{
		// 基本直接子文件
		{"a", "a/b.txt", true},
		{"a/", "a/b.txt", true},  // 修复：处理末尾斜杠
		{"a", "a/b/c.txt", true}, // 修复：处理多级子目录

		// 边缘情况
		{"a", "a", false},            // 修复：相同路径返回 false
		{"a", "b.txt", false},        // 正确
		{"a/b/", "b/a/b.txt", false}, // 正确
		{"a/b", "a/b/c/d.txt", true}, // 修复：深层子目录

		// 路径规范化
		{"a/b", "a/b/c/../d.txt", true}, // 修复：处理 ".."
		{"/a", "/a/b", true},            // 处理绝对路径
		{"a", "a/./b", true},            // 处理 "."
	}

	for _, tt := range tests {
		t.Run(tt.parent+"_"+tt.path, func(t *testing.T) {
			if got := IsChild(tt.parent, tt.path); got != tt.want {
				t.Errorf("IsChild() = %v, want %v", got, tt.want)
			}
		})
	}
}
