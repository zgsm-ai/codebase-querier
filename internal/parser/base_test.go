package parser

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
	"testing"
)

func TestBaseParse(t *testing.T) {
	parser := NewParser()
	opts := ParseOptions{
		IncludeContent: true,
	}

	testCases := []struct {
		name       string
		sourceFile *types.SourceFile
		wantErr    error
	}{
		{
			name: "Go",
			sourceFile: &types.SourceFile{
				Path:    "test.go",
				Content: readFile("testdata/test.go"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, opts)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
		})
	}
}

func readFile(path string) []byte {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return bytes
}
