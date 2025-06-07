package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func TestParse(t *testing.T) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sittergo.Language())
	err := parser.SetLanguage(lang)
	assert.NoError(t, err)
	code := []byte(`package main
func main() { 
fmt.Println(\"Hello, Sitter!\") 
}
`)
	tree := parser.Parse(code, nil)
	assert.NotNil(t, tree)
	defer tree.Close()
	queryScm := `
	(function_declaration
  name: (identifier) @name) @function
`
	query, err := sitter.NewQuery(lang, queryScm)
	if err != nil && IsRealErr(err) {
		t.Fatal(err)
	}
	defer query.Close()
	cursor := sitter.NewQueryCursor()
	matches := cursor.Matches(query, tree.RootNode(), code)
	for {
		next := matches.Next()
		if next == nil {
			break
		}
		for i, capture := range next.Captures {
			fmt.Printf("%d %s\n", i, capture.Node.Utf8Text(code))
		}
	}
}
