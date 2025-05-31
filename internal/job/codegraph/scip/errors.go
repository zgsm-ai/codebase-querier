package scip

import "fmt"

// ErrorCode represents the type of error
type ErrorCode int

const (
	// ErrCodeUnknown represents an unknown error
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeConfig represents configuration related errors
	ErrCodeConfig
	// ErrCodeLanguage represents language detection errors
	ErrCodeLanguage
	// ErrCodeBuildTool represents build tool detection errors
	ErrCodeBuildTool
	// ErrCodeCommand represents command execution errors
	ErrCodeCommand
	// ErrCodeTimeout represents timeout errors
	ErrCodeTimeout
	// ErrCodeConcurrent represents concurrent processing errors
	ErrCodeConcurrent
	// ErrCodeResource represents resource related errors
	ErrCodeResource
)

// SCIPError represents a SCIP indexing error
type SCIPError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *SCIPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
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
	return ErrCodeUnknown
} 