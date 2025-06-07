package structure

import (
	"embed"
	"fmt"

	"github.com/zgsm-ai/codebase-indexer/internal/job/parser"
)

//go:embed queries/*.scm
var scmFS embed.FS

const queryBaseDir = "queries"
const queryExt = ".scm"

var languageQueryConfig map[parser.Language]string

func init() {
	languageQueryConfig = make(map[parser.Language]string)
	for lang := range parser.GetLanguageConfigs() {
		queryPath := makeStructureQueryPath(lang)
		structureQueryContent, err := scmFS.ReadFile(queryPath)
		if err != nil {
			panic(fmt.Sprintf("failed to read structure query file %s for %s: %v", queryPath, lang, err))
		}
		languageQueryConfig[lang] = string(structureQueryContent)
	}

}

func makeStructureQueryPath(lang parser.Language) string {
	return queryBaseDir + "/" + string(lang) + queryExt
}
