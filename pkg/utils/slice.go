package utils

import (
	"strconv"
	"strings"
)

func SliceContains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// SliceToString 切片转字符串
func SliceToString(slice []int32) string {
	var strs []string
	for _, v := range slice {
		strs = append(strs, strconv.FormatInt(int64(v), 10))
	}
	return strings.Join(strs, ",")
}
