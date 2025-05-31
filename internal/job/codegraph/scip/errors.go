package scip

import "fmt"

// ErrorCode represents a type of error that can occur during SCIP indexing
type ErrorCode string

const (
	ErrCodeConfig     = "CONFIG_ERROR"
	ErrCodeLanguage   = "LANGUAGE_ERROR"
	ErrCodeCommand    = "COMMAND_ERROR"
	ErrCodeResource   = "RESOURCE_ERROR"
	ErrCodeConcurrent = "CONCURRENT_ERROR"
	ErrCodeBuildTool  = "BUILD_TOOL_ERROR"
	ErrCodeTool       = "TOOL_ERROR"
)

// Error represents an error that occurred during SCIP indexing
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error returns the error message
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new error
func NewError(code ErrorCode, message string, err error) error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsErrorCode checks if the error is of the specified error code
func IsErrorCode(err error, code ErrorCode) bool {
	if scipErr, ok := err.(*Error); ok {
		return scipErr.Code == code
	}
	return false
}

// GetErrorCode returns the error code of the error
func GetErrorCode(err error) ErrorCode {
	if scipErr, ok := err.(*Error); ok {
		return scipErr.Code
	}
	return ErrCodeConfig
}
