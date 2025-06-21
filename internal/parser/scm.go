package parser

import (
	"embed"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"path/filepath"
)

//go:embed queries/*/*.scm
var scmFS embed.FS

const queryDir = "queries"
const defSubdir = "def"
const baseSubDir = "base"
const queryExt = ".scm"

var DefinitionQueries map[Language]string
var BaseQueries map[Language]string

func init() {
	if err := loadScm(); err != nil {
		panic(fmt.Errorf("tree_sitter parser load scm queries err:%v", err))
	}
}

func loadScm() error {
	DefinitionQueries = make(map[Language]string)
	configs := GetLanguageConfigs()
	for _, lang := range configs {
		baseQuery := makeQueryPath(lang.Language, baseSubDir)
		defQuery := makeQueryPath(lang.Language, defSubdir)
		baseQueryContent, err := scmFS.ReadFile(baseQuery)
		if err != nil {
			return fmt.Errorf("failed to read base query file %s for %s: %w", baseQuery, lang.Language, err)
		}
		defQueryContent, err := scmFS.ReadFile(defQuery)
		if err != nil {
			return fmt.Errorf("failed to read definition query file %s for %s: %w", defQuery, lang.Language, err)
		}
		// 校验query
		langParser := sitter.NewParser()
		sitterLang := lang.SitterLanguage()
		err = langParser.SetLanguage(sitterLang)
		if err != nil {
			return fmt.Errorf("failed to init language parser %s: %w", lang.Language, err)
		}
		query, queryError := sitter.NewQuery(sitterLang, string(baseQueryContent))
		if queryError != nil && IsRealQueryErr(queryError) {
			return fmt.Errorf("failed to parse base query file %s: %w", baseQuery, queryError)
		}
		query, queryError = sitter.NewQuery(sitterLang, string(defQueryContent))
		if queryError != nil && IsRealQueryErr(queryError) {
			return fmt.Errorf("failed to parse def query file %s : %w", defQuery, queryError)
		}

		query.Close()
		langParser.Close()
		DefinitionQueries[lang.Language] = string(defQueryContent)
		BaseQueries[lang.Language] = string(baseQueryContent)
	}
	return nil
}

func makeQueryPath(lang Language, subdir string) string {
	return filepath.Join(queryDir, subdir, string(lang)+queryExt)
}
