package structure

import (
	"embed"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/zgsm-ai/codebase-indexer/internal/parser"
)

//go:embed queries/*.scm
var scmFS embed.FS

const queryBaseDir = "queries"
const queryExt = ".scm"

var languageQueryConfig map[parser.Language]string

func init() {
	mustLoad()

}

func mustLoad() {
	languageQueryConfig = make(map[parser.Language]string)
	configs := parser.GetLanguageConfigs()
	for _, lang := range configs {
		queryPath := makeStructureQueryPath(lang.Language)
		structureQueryContent, err := scmFS.ReadFile(queryPath)
		if err != nil {
			panic(fmt.Sprintf("failed to read structure query file %s for %s: %v", queryPath, lang.Language, err))
		}
		// 校验query
		langParser := sitter.NewParser()
		sitterLang := lang.SitterLanguage()
		err = langParser.SetLanguage(sitterLang)
		if err != nil {
			panic(fmt.Sprintf("failed to read structure query file %s for %s: %v", queryPath, lang.Language, err))
		}
		query, queryError := sitter.NewQuery(sitterLang, string(structureQueryContent))
		if queryError != nil && parser.IsRealQueryErr(queryError) {
			panic(fmt.Sprintf("failed to parse structure file %s query %s : %v", queryPath, lang.Language, queryError))
		}
		query.Close()
		langParser.Close()
		languageQueryConfig[lang.Language] = string(structureQueryContent)
	}
}

func makeStructureQueryPath(lang parser.Language) string {
	return queryBaseDir + "/" + string(lang) + queryExt
}
