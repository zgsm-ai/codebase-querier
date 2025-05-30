package codegraph

type codebaseLanguage struct {
	Language  string
	buildFile string
	buildTool string
}

func inferCodebaseLanguage(codebasePath string) string {
	return ""
}
