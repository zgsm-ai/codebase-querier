package structure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScmLoad(t *testing.T) {
	assert.NotPanics(t,
		func() {
			mustLoad()
		})
}

//func TestCTopQuery(t *testing.T) {
//	code, err := os.ReadFile("./testdata/test.c")
//	assert.NoError(t, err)
//
//}
