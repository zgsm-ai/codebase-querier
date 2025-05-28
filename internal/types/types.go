package types

// Language represents a programming language.
type Language string

const (
	Unknown    Language = "unknown"
	Java       Language = "java"
	Python     Language = "python"
	Go         Language = "go"
	JavaScript Language = "javascript"
	TypeScript Language = "typescript"
	TSX        Language = "tsx"
	Rust       Language = "rust"
	C          Language = "c"
	CPP        Language = "cpp"
	CSharp     Language = "csharp"
	Ruby       Language = "ruby"
	PHP        Language = "php"
	Kotlin     Language = "kotlin"
	Scala      Language = "scala"
)

// CodeBlock represents a chunk of code with associated metadata.
type CodeBlock struct {
	Content      string // The actual code snippet
	FilePath     string // The path to the file this block came from
	StartLine    int    // The 1-indexed starting line number of the block in the original file
	EndLine      int    // The 1-indexed ending line number of the block in the original file
	ParentFunc   string // The name of the parent function (if applicable)
	ParentClass  string // The name of the parent class or type (if applicable)
	OriginalSize int    // The original size in bytes of this block
	TokenCount   int    // The number of tokens in this block
}
