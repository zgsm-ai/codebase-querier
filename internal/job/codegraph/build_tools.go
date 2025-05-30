package codegraph

var buildTools = map[string][]string{
	"go":         {"go"},
	"python":     {"python"},
	"java":       {"maven", "gradle"},
	"c++":        {"make", "cmake"},
	"c":          {"make", "cmake"},
	"ruby":       {"ruby"},
	"php":        {"php"},
	"javascript": {"javascript"},
	"typescript": {"typescript"},
	"rust":       {"rust"},
}
