package vector

import "errors"

var ErrInvalidCodebasePath = errors.New("invalid codebasePath")
var ErrEmptyResponse = errors.New("response is empty")
var ErrInvalidResponse = errors.New("response is invalid")
