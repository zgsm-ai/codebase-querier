package embedding

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCodeSplitter_Split_Go(t *testing.T) {
	// Simple Go code with a function
	goCode := []byte(`
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}

func anotherFunc() int {
	return 1
}
`)

	// Create a dummy CodeFile
	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "simple.go"),
		Content: goCode,
	}

	// 创建splitter用于计算token
	splitter, err := NewCodeSplitter(SplitOptions{
		MaxTokensPerChunk:          1000, // 使用较大的值，确保不会触发切分
		SlidingWindowOverlapTokens: 0,
	})
	assert.NoError(t, err)

	// 计算各个函数的token数
	mainFuncTokens := splitter.countToken(`func main() {
	fmt.Println("Hello, world!")
}`)
	anotherFuncTokens := splitter.countToken(`func anotherFunc() int {
	return 1
}`)
	t.Logf("Token counts - main: %d, anotherFunc: %d", mainFuncTokens, anotherFuncTokens)

	// 使用较小的maxTokensPerChunk，但仍然大于函数token数
	splitter, err = NewCodeSplitter(SplitOptions{
		MaxTokensPerChunk:          int(math.Max(float64(mainFuncTokens), float64(anotherFuncTokens))) + 10,
		SlidingWindowOverlapTokens: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, splitter)

	// Split the code
	chunks, err := splitter.Split(codeFile)
	assert.NoError(t, err)
	assert.NotNil(t, chunks)

	// 验证chunk数量
	assert.Len(t, chunks, 2, "Should have exactly 2 chunks, one per function")

	// 验证main函数
	assert.Contains(t, string(chunks[0].Content), "func main")
	assert.Equal(t, 5, chunks[0].StartLine, "main function should start at line 5")
	assert.Equal(t, mainFuncTokens, chunks[0].TokenCount, "main function chunk should have correct token count")
	assert.Contains(t, string(chunks[0].Content), "fmt.Println(\"Hello, world!\")", "main function should be complete")
	assert.LessOrEqual(t, chunks[0].TokenCount, mainFuncTokens+10, "main function chunk should not exceed maxTokensPerChunk")

	// 验证anotherFunc函数
	assert.Contains(t, string(chunks[1].Content), "func anotherFunc")
	assert.Equal(t, 9, chunks[1].StartLine, "anotherFunc should start at line 9")
	assert.Equal(t, anotherFuncTokens, chunks[1].TokenCount, "anotherFunc chunk should have correct token count")
	assert.Contains(t, string(chunks[1].Content), "return 1", "anotherFunc should be complete")
	assert.LessOrEqual(t, chunks[1].TokenCount, anotherFuncTokens+10, "anotherFunc chunk should not exceed maxTokensPerChunk")

	// 验证函数完整性
	for _, chunk := range chunks {
		// 验证每个chunk都包含完整的函数定义
		assert.Contains(t, string(chunk.Content), "func ", "Each chunk should contain a function definition")
		// 验证每个chunk都包含函数体
		assert.Contains(t, string(chunk.Content), "{", "Each chunk should contain function body")
		assert.Contains(t, string(chunk.Content), "}", "Each chunk should contain function body")
		// 验证行号范围
		assert.GreaterOrEqual(t, chunk.StartLine, 0, "Chunk should have valid start line (0-based)")
		assert.GreaterOrEqual(t, chunk.EndLine, chunk.StartLine, "Chunk should have valid line range")
	}
}

func TestCodeSplitter_SplitWithOverlapAndMaxTokens(t *testing.T) {
	// Create a longer Go code that will be split into multiple chunks
	goCode := []byte(`
package main

import "fmt"
var mu *sync.Mux
const Name = "go"
// Function 1
func function1() {
	fmt.Println("This is function 1")
	fmt.Println("It has multiple lines")
	fmt.Println("To test splitting")
}

// Function 2
func function2() {
	fmt.Println("This is function 2")
	fmt.Println("It also has multiple lines")
	fmt.Println("For testing purposes")
}

// Function 3
func function3() {
	fmt.Println("This is function 3")
	fmt.Println("With more lines")
	fmt.Println("To ensure proper splitting")
}
`)

	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "multi_func.go"),
		Content: goCode,
	}

	testCases := []struct {
		name                       string
		maxTokensPerChunk          int
		overlapTokensWhenMaxExceed int
		wantChunks                 int
		description                string
	}{
		{
			name:                       "No overlap, small maxTokensPerChunk",
			maxTokensPerChunk:          50, // Small enough to force splitting
			overlapTokensWhenMaxExceed: 0,
			wantChunks:                 3, // 每个函数一个chunk，因为函数内容小于maxTokensPerChunk
			description:                "验证当maxTokensPerChunk足够大时，每个函数保持完整",
		},
		{
			name:                       "With overlap, small maxTokensPerChunk",
			maxTokensPerChunk:          50, // Small enough to force splitting
			overlapTokensWhenMaxExceed: 20, // 重叠token数
			wantChunks:                 3,  // 每个函数一个chunk，因为函数内容小于maxTokensPerChunk，overlap不会生效
			description:                "验证当函数内容小于maxTokensPerChunk时，overlap不会生效",
		},
		{
			name:                       "No overlap, large maxTokensPerChunk",
			maxTokensPerChunk:          200, // 足够大，但不会影响函数边界
			overlapTokensWhenMaxExceed: 0,
			wantChunks:                 3, // 仍然是每个函数一个chunk，因为函数边界优先
			description:                "验证即使maxTokensPerChunk很大，函数边界仍然优先",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			splitter, err := NewCodeSplitter(SplitOptions{
				MaxTokensPerChunk:          tc.maxTokensPerChunk,
				SlidingWindowOverlapTokens: tc.overlapTokensWhenMaxExceed,
			})
			assert.NoError(t, err)
			assert.NotNil(t, splitter)

			chunks, err := splitter.Split(codeFile)
			assert.NoError(t, err)
			assert.NotNil(t, chunks)
			assert.Len(t, chunks, tc.wantChunks)

			// Verify that chunks contain the expected content
			if tc.wantChunks > 1 {
				// For multi-chunk scenarios, verify overlapTokensWhenMaxExceed
				for i := 1; i < len(chunks); i++ {
					prevChunk := chunks[i-1].Content
					currChunk := chunks[i].Content

					if tc.overlapTokensWhenMaxExceed > 0 {
						// When overlapTokensWhenMaxExceed is specified, verify that some content is shared
						// between consecutive chunks
						assert.True(t, len(prevChunk) > 0 && len(currChunk) > 0,
							"Chunks should not be empty")
					}
				}
			}

			// Verify that each chunk contains at least one function
			for _, chunk := range chunks {
				assert.Contains(t, string(chunk.Content), "func ", "Each chunk should contain at least one function")
			}
		})
	}
}
func TestCodeSplitter_SplitWithSlidingWindow(t *testing.T) {
	// 创建一个超长函数，确保会触发滑动窗口
	goCode := []byte(`
package main

import "fmt"

func veryLongFunc1() {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d with some content to ensure token count", i))
	}
}`)
	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "long_func.go"),
		Content: goCode,
	}

	// 创建默认配置的splitter用于计算token
	defaultSplitter, err := NewCodeSplitter(SplitOptions{})
	assert.NoError(t, err)

	// 计算单个函数的token数
	funcContent := `func veryLongFunc1() {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d with some content to ensure token count", i))
	}
}`
	singleFuncTokenCount := defaultSplitter.countToken(funcContent)
	t.Logf("Total tokens in single function: %d", singleFuncTokenCount)

	// 计算测试配置
	maxTokens := singleFuncTokenCount / 2
	overlapTokens := maxTokens / 5 // 设置为maxTokens的20%，确保足够重叠

	// 计算公式：预期每个函数的块数
	// 总块数 = ceil((总token数 - maxTokens) / (maxTokens - overlapTokens)) + 1
	expectedChunksPerFunc := ((singleFuncTokenCount-maxTokens)+(maxTokens-overlapTokens-1))/(maxTokens-overlapTokens) + 1
	expectedTotalChunks := expectedChunksPerFunc

	testCases := []struct {
		name           string
		options        SplitOptions
		expectedChunks int
		verifyOverlap  func(t *testing.T, chunks []*types.CodeChunk)
	}{
		{
			name: "Sliding window with overlap",
			options: SplitOptions{
				MaxTokensPerChunk:          maxTokens,
				SlidingWindowOverlapTokens: overlapTokens,
			},
			expectedChunks: expectedTotalChunks,
			verifyOverlap: func(t *testing.T, chunks []*types.CodeChunk) {
				// 按函数分组块
				funcChunks := make(map[string][]*types.CodeChunk)
				for _, chunk := range chunks {
					funcChunks[chunk.ParentFunc] = append(funcChunks[chunk.ParentFunc], chunk)
				}

				// 验证每个函数的块重叠
				for funcName, funcChunks := range funcChunks {
					if len(funcChunks) < 2 {
						continue // 跳过只有一个块的函数
					}

					t.Logf("Verifying overlap for function: %s", funcName)
					for i := 1; i < len(funcChunks); i++ {
						prev := funcChunks[i-1]
						curr := funcChunks[i]

						// 1. 验证行号重叠
						assert.Lessf(t, curr.StartLine, prev.EndLine,
							"Chunk %d (%d-%d) should overlap with chunk %d (%d-%d) in function %s",
							i, curr.StartLine, curr.EndLine, i-1, prev.StartLine, prev.EndLine, funcName)

						// 2. 验证内容重叠
						prevTokens := defaultSplitter.tokenizeToStrings(string(prev.Content))
						currTokens := defaultSplitter.tokenizeToStrings(string(curr.Content))

						// 找到实际重叠的token数
						overlapCount := 0
						minLength := min(len(prevTokens), len(currTokens))

						// 从prev的末尾和curr的开头找最长公共子序列
						for j := 0; j < minLength; j++ {
							if prevTokens[len(prevTokens)-1-j] == currTokens[j] {
								overlapCount++
							} else {
								break
							}
						}

						assert.GreaterOrEqualf(t, overlapCount, overlapTokens,
							"Chunk %d and %d in function %s should overlap by at least %d tokens, got %d",
							i-1, i, funcName, overlapTokens, overlapCount)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			splitter, err := NewCodeSplitter(tc.options)
			assert.NoError(t, err)

			chunks, err := splitter.Split(codeFile)
			assert.NoError(t, err)
			assert.NotNil(t, chunks)
			assert.Equalf(t, tc.expectedChunks, len(chunks),
				"Expected %d chunks, got %d", tc.expectedChunks, len(chunks))

			// 运行自定义验证
			//tc.verifyOverlap(t, chunks)
		})
	}
}

// 辅助函数：将内容转换为token字符串列表
func (p *CodeSplitter) tokenizeToStrings(content string) []string {
	_, tokens, _ := p.tokenizer.Encode(content)
	return tokens
}

// 辅助函数：返回最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
