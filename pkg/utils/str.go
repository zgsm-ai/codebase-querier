package utils

import "strings"

func IsBlank(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// CountLines  is a helper to count lines in a byte slice.
// Reuses the existing logic.
func CountLines(data []byte) int {
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	// Add one for the last line if it doesn't end with a newline
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	// If the content is empty, there are 0 lines. If it's not empty but has no newline, it's 1 line.
	if len(data) == 0 {
		return 0
	}
	if lines == 0 && len(data) > 0 {
		return 1
	} // Fix: handle non-empty single line
	if lines == 0 && len(data) == 0 {
		return 0
	} // Explicitly handle empty
	return lines
}
