package types

type CodeFile struct {
	CodebaseId   int64
	CodebasePath string
	CodebaseName string
	Name         string
	Path         string
	Content      []byte
	Language     string
}

// CodeChunk represents a chunk of code with associated metadata.
type CodeChunk struct {
	CodebaseId   int64
	CodebasePath string
	CodebaseName string
	Language     string
	Content      []byte // The actual code snippet
	FilePath     string // The FullPath to the file this block came from
	Position     []int  // start from zero, startLine, startColumn, endLine, endColumn
	TokenCount   int    // The number of tokens in this block
}
