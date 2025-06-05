package types

type CodeFile struct {
	CodebaseId   int64
	CodebasePath string
	CodebaseName string
	Name         string
	Path         string
	Content      string
	Language     string
}

// CodeChunk represents a chunk of code with associated metadata.
type CodeChunk struct {
	CodebaseId   int64
	CodebasePath string
	CodebaseName string
	Language     string
	Content      string // The actual code snippet
	FilePath     string // The FullPath to the file this block came from
	StartLine    int    // The 1-indexed starting line number of the block in the original file
	EndLine      int    // The 1-indexed ending line number of the block in the original file
	OriginalSize int    // The original size in bytes of this block
	TokenCount   int    // The number of tokens in this block
	ParentFunc   string
	ParentClass  string
}
