package utils

import "strings"

func IsBlank(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}
