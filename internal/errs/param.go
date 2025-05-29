package errs

import "fmt"

var ErrorInvalidParamFmt = "invalid request params: %s %v"
var ErrorRecordNotFoundFmt = "%s not found by %s"

func NewInvalidParamErr(name string, value interface{}) error {
	return fmt.Errorf(ErrorInvalidParamFmt, name, value)
}

func NewRecordNotFoundErr(name string, value interface{}) error {
	return fmt.Errorf(ErrorRecordNotFoundFmt, name, value)
}
