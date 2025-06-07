package parser

import (
	"errors"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// IsRealErr prevent *sitter.QueryError(nil)
func IsRealErr(err error) bool {
	if err != nil {
		var qe *sitter.QueryError
		if errors.As(err, &qe) && qe == nil {
			return false
		}
		return true
	}
	return false
}
