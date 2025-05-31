package scip

import "fmt"

// ErrorCode represents the type of error that occurred
type ErrorCode int

const (
	ErrCodeConfig ErrorCode = iota + 1
	ErrCodeLanguage
	ErrCodeBuildTool
	ErrCodeCommand
	ErrCodeResource
	ErrCodeConcurrent
)

// SCIPError represents a SCIP-specific error
type SCIPError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *SCIPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewError creates a new SCIPError
func NewError(code ErrorCode, message string, err error) error {
	return &SCIPError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsErrorCode checks if the error is of the specified error code
func IsErrorCode(err error, code ErrorCode) bool {
	if scipErr, ok := err.(*SCIPError); ok {
		return scipErr.Code == code
	}
	return false
}

// GetErrorCode returns the error code of the error
func GetErrorCode(err error) ErrorCode {
	if scipErr, ok := err.(*SCIPError); ok {
		return scipErr.Code
	}
	return ErrCodeConfig
}
