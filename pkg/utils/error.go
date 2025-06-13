package utils

import (
	"errors"
	"strings"
)

func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error())
		b.WriteString(",")
	}
	return errors.New(b.String())
}
