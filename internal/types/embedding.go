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
	StartLine    int    // The 0-indexed starting line number of the block in the original file
	StartColumn  int    // The 0-indexed starting line number of the block in the original file
	EndLine      int    // The 0-indexed ending line number of the block in the original file
	EndColumn    int    // The 0-indexed ending line number of the block in the original file
	TokenCount   int    // The number of tokens in this block
}
