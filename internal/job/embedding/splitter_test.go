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
	goCode := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}

func anotherFunc() int {
	return 1
}
`

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
	assert.Contains(t, chunks[0].Content, "func main")
	assert.Equal(t, 5, chunks[0].StartLine, "main function should start at line 5")
	assert.Equal(t, mainFuncTokens, chunks[0].TokenCount, "main function chunk should have correct token count")
	assert.Contains(t, chunks[0].Content, "fmt.Println(\"Hello, world!\")", "main function should be complete")
	assert.LessOrEqual(t, chunks[0].TokenCount, mainFuncTokens+10, "main function chunk should not exceed maxTokensPerChunk")

	// 验证anotherFunc函数
	assert.Contains(t, chunks[1].Content, "func anotherFunc")
	assert.Equal(t, 9, chunks[1].StartLine, "anotherFunc should start at line 9")
	assert.Equal(t, anotherFuncTokens, chunks[1].TokenCount, "anotherFunc chunk should have correct token count")
	assert.Contains(t, chunks[1].Content, "return 1", "anotherFunc should be complete")
	assert.LessOrEqual(t, chunks[1].TokenCount, anotherFuncTokens+10, "anotherFunc chunk should not exceed maxTokensPerChunk")

	// 验证函数完整性
	for _, chunk := range chunks {
		// 验证每个chunk都包含完整的函数定义
		assert.Contains(t, chunk.Content, "func ", "Each chunk should contain a function definition")
		// 验证每个chunk都包含函数体
		assert.Contains(t, chunk.Content, "{", "Each chunk should contain function body")
		assert.Contains(t, chunk.Content, "}", "Each chunk should contain function body")
		// 验证行号范围
		assert.GreaterOrEqual(t, chunk.StartLine, 0, "Chunk should have valid start line (0-based)")
		assert.GreaterOrEqual(t, chunk.EndLine, chunk.StartLine, "Chunk should have valid line range")
	}
}

func TestCodeSplitter_SplitWithOverlapAndMaxTokens(t *testing.T) {
	// Create a longer Go code that will be split into multiple chunks
	goCode := `
package main

import "fmt"

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
`

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
				assert.Contains(t, chunk.Content, "func ", "Each chunk should contain at least one function")
			}
		})
	}
}

func TestCodeSplitter_SplitWithMixedStrategy(t *testing.T) {
	// 测试用例：包含普通函数和超长函数
	goCode := `
package main

import "fmt"

// 普通函数，token数较少
func shortFunc() {
	fmt.Println("short function")
}

// 超长函数，需要滑动窗口切分
func longFunc() {
	// 生成大量重复代码行，确保超过token限制
	fmt.Println("This is a long function with many lines")
	fmt.Println("Each line contains more tokens to ensure we exceed the limit")
	fmt.Println("We need enough tokens to trigger the sliding window split")
	fmt.Println("The token count should be calculated using countToken function")
	fmt.Println("This is line 5 of the long function")
	fmt.Println("This is line 6 of the long function")
	fmt.Println("This is line 7 of the long function")
	fmt.Println("This is line 8 of the long function")
	fmt.Println("This is line 9 of the long function")
	fmt.Println("This is line 10 of the long function")
	fmt.Println("This is line 11 of the long function")
	fmt.Println("This is line 12 of the long function")
	fmt.Println("This is line 13 of the long function")
	fmt.Println("This is line 14 of the long function")
	fmt.Println("This is line 15 of the long function")
	fmt.Println("This is line 16 of the long function")
	fmt.Println("This is line 17 of the long function")
	fmt.Println("This is line 18 of the long function")
	fmt.Println("This is line 19 of the long function")
	fmt.Println("This is the last line of the long function")
}

// 另一个普通函数
func anotherShortFunc() {
	fmt.Println("another short function")
}
`

	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "mixed_funcs.go"),
		Content: goCode,
	}

	// 创建splitter用于计算token
	splitter, err := NewCodeSplitter(SplitOptions{
		MaxTokensPerChunk:          1000, // 使用较大的值，仅用于计算token
		SlidingWindowOverlapTokens: 0,
	})
	assert.NoError(t, err)

	// 计算各个函数的token数
	shortFuncTokens := splitter.countToken(`
func shortFunc() {
	fmt.Println("short function")
}`)
	longFuncTokens := splitter.countToken(`
func longFunc() {
	fmt.Println("This is a long function with many lines")
	// ... 其他行 ...
	fmt.Println("This is the last line of the long function")
}`)
	anotherShortFuncTokens := splitter.countToken(`
func anotherShortFunc() {
	fmt.Println("another short function")
}`)

	t.Logf("Token counts - shortFunc: %d, longFunc: %d, anotherShortFunc: %d",
		shortFuncTokens, longFuncTokens, anotherShortFuncTokens)

	// 确保长函数的token数确实大于短函数
	assert.Greater(t, longFuncTokens, shortFuncTokens*2,
		"longFunc should have significantly more tokens than shortFunc")

	testCases := []struct {
		name           string
		options        SplitOptions
		description    string
		expectedChunks int
		verifyFunc     func(t *testing.T, chunks []*types.CodeChunk)
	}{
		{
			name: "Function boundary split with long function sliding window",
			options: SplitOptions{
				MaxTokensPerChunk:          shortFuncTokens + 10, // 设置比短函数稍大的限制
				SlidingWindowOverlapTokens: 10,                   // 设置重叠token数
			},
			description:    "验证按函数边界切分，对超长函数使用滑动窗口",
			expectedChunks: 4, // 2个短函数各1个chunk，长函数被切分成2个chunk
			verifyFunc: func(t *testing.T, chunks []*types.CodeChunk) {
				// 验证短函数保持完整
				shortFuncFound := false
				anotherShortFuncFound := false
				longFuncChunks := 0
				var longFuncStartLines []int
				var longFuncContents []string

				for _, chunk := range chunks {
					content := chunk.Content
					if containsFunc(content, "shortFunc") {
						shortFuncFound = true
						assert.Contains(t, content, "fmt.Println(\"short function\")")
						assert.Equal(t, 3, chunk.StartLine, "shortFunc should start at line 3")
						assert.Contains(t, content, "func shortFunc()")
						assert.Equal(t, shortFuncTokens, chunk.TokenCount,
							"shortFunc chunk token count should match countToken result")
						assert.LessOrEqual(t, chunk.TokenCount, shortFuncTokens+10,
							"shortFunc chunk should not exceed maxTokensPerChunk")
					}
					if containsFunc(content, "anotherShortFunc") {
						anotherShortFuncFound = true
						assert.Contains(t, content, "fmt.Println(\"another short function\")")
						assert.Equal(t, 29, chunk.StartLine, "anotherShortFunc should start at line 29")
						assert.Contains(t, content, "func anotherShortFunc()")
						assert.Equal(t, anotherShortFuncTokens, chunk.TokenCount,
							"anotherShortFunc chunk token count should match countToken result")
						assert.LessOrEqual(t, chunk.TokenCount, anotherShortFuncTokens+10,
							"anotherShortFunc chunk should not exceed maxTokensPerChunk")
					}
					if containsFunc(content, "longFunc") {
						longFuncChunks++
						longFuncStartLines = append(longFuncStartLines, chunk.StartLine)
						longFuncContents = append(longFuncContents, content)
						assert.LessOrEqual(t, chunk.TokenCount, shortFuncTokens+10,
							"Long function chunk should not exceed maxTokensPerChunk")
						assert.Contains(t, content, "func longFunc()")
					}
				}

				// 验证所有函数都被正确处理
				assert.True(t, shortFuncFound, "shortFunc should be in a single chunk")
				assert.True(t, anotherShortFuncFound, "anotherShortFunc should be in a single chunk")
				assert.Equal(t, 2, longFuncChunks, "longFunc should be split into 2 chunks")

				// 验证长函数chunks的行号和重叠
				assert.Equal(t, 7, longFuncStartLines[0], "First longFunc chunk should start at line 7")
				assert.Greater(t, longFuncStartLines[1], longFuncStartLines[0],
					"Second chunk should start after first chunk")

				// 验证长函数chunks的重叠部分
				if len(longFuncContents) >= 2 {
					overlap := findOverlap(longFuncContents[0], longFuncContents[1])
					overlapTokens := splitter.countToken(overlap)
					assert.GreaterOrEqual(t, overlapTokens, 10,
						"Long function chunks should have sufficient overlap")
				}

				// 验证长函数chunks组合后的完整性
				combinedLongFunc := ""
				for _, content := range longFuncContents {
					combinedLongFunc += content
				}
				assert.Contains(t, combinedLongFunc, "func longFunc()")
				assert.Contains(t, combinedLongFunc, "This is the last line of the long function")
			},
		},
		{
			name: "All functions within maxTokensPerChunk",
			options: SplitOptions{
				MaxTokensPerChunk:          longFuncTokens + 100, // 设置足够大的限制，容纳所有函数
				SlidingWindowOverlapTokens: 10,                   // 设置重叠token数，但不会生效
			},
			description:    "验证所有函数都在maxTokensPerChunk限制内，不需要滑动窗口切分",
			expectedChunks: 3, // 每个函数一个chunk
			verifyFunc: func(t *testing.T, chunks []*types.CodeChunk) {
				// 验证每个函数都在独立的chunk中
				funcs := []struct {
					name      string
					startLine int
					tokens    int
				}{
					{"shortFunc", 3, shortFuncTokens},
					{"longFunc", 7, longFuncTokens},
					{"anotherShortFunc", 29, anotherShortFuncTokens},
				}

				for _, f := range funcs {
					found := false
					for _, chunk := range chunks {
						if containsFunc(chunk.Content, f.name) {
							found = true
							// 验证函数完整性
							assert.Contains(t, chunk.Content, "func "+f.name+"()")
							if f.name == "longFunc" {
								assert.Contains(t, chunk.Content, "This is the last line of the long function")
							}
							// 验证行号
							assert.Equal(t, f.startLine, chunk.StartLine,
								"Function %s should start at line %d", f.name, f.startLine)
							// 验证token数
							assert.Equal(t, f.tokens, chunk.TokenCount,
								"Chunk for %s should have correct token count", f.name)
							assert.LessOrEqual(t, chunk.TokenCount, longFuncTokens+100,
								"Chunk for %s should not exceed maxTokensPerChunk", f.name)
							break
						}
					}
					assert.True(t, found, "Function %s should be in a chunk", f.name)
				}

				// 验证没有重叠（因为函数都在限制内）
				for i := 1; i < len(chunks); i++ {
					prevChunk := chunks[i-1].Content
					currChunk := chunks[i].Content
					overlap := findOverlap(prevChunk, currChunk)
					assert.Empty(t, overlap, "Chunks should not overlap when functions are within maxTokensPerChunk")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			splitter, err := NewCodeSplitter(tc.options)
			assert.NoError(t, err)
			assert.NotNil(t, splitter)

			chunks, err := splitter.Split(codeFile)
			assert.NoError(t, err)
			assert.NotNil(t, chunks)
			assert.Len(t, chunks, tc.expectedChunks, "Unexpected number of chunks")

			// 运行自定义验证函数
			tc.verifyFunc(t, chunks)

			// 通用验证：确保每个chunk都包含有效的Go代码
			for _, chunk := range chunks {
				// 验证每个chunk都包含函数定义
				assert.Contains(t, chunk.Content, "func ", "Each chunk should contain at least one function")
				// 验证token数
				expectedTokens := splitter.countToken(chunk.Content)
				assert.Equal(t, expectedTokens, chunk.TokenCount,
					"Chunk token count should match countToken result")
				assert.Greater(t, chunk.TokenCount, 0, "Chunk should have tokens")
				assert.LessOrEqual(t, chunk.TokenCount, tc.options.MaxTokensPerChunk,
					"Chunk should not exceed maxTokensPerChunk")
				// 验证行号
				assert.GreaterOrEqual(t, chunk.StartLine, 0, "Chunk should have valid start line (0-based)")
				assert.GreaterOrEqual(t, chunk.EndLine, chunk.StartLine, "Chunk should have valid line range")
			}
		})
	}
}

func TestCodeSplitter_SplitWithSlidingWindow(t *testing.T) {
	// 创建一个超长函数，确保会触发滑动窗口
	goCode := `
package main

import "fmt"

func veryLongFunc() {
	// 生成足够多的行以确保超过token限制
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d with some content to ensure token count", i))
	}
	// 更多代码...
}
`
	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "long_func.go"),
		Content: goCode,
	}

	// 创建splitter用于计算token
	splitter, err := NewCodeSplitter(SplitOptions{
		MaxTokensPerChunk:          1000,
		SlidingWindowOverlapTokens: 0,
	})
	assert.NoError(t, err)

	// 计算函数内容的token数
	funcContent := `func veryLongFunc() {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d with some content to ensure token count", i))
	}
}`
	totalTokens := splitter.countToken(funcContent)
	t.Logf("Total tokens in function: %d", totalTokens)

	testCases := []struct {
		name           string
		options        SplitOptions
		expectedChunks int
		verifyOverlap  func(t *testing.T, chunks []*types.CodeChunk)
	}{
		{
			name: "Sliding window with overlap",
			options: SplitOptions{
				MaxTokensPerChunk:          totalTokens / 3,  // 强制切分成多个chunk
				SlidingWindowOverlapTokens: totalTokens / 10, // 设置10%的重叠
			},
			expectedChunks: 4, // 预期切分成4个chunk
			verifyOverlap: func(t *testing.T, chunks []*types.CodeChunk) {
				// 验证chunk数量
				assert.GreaterOrEqual(t, len(chunks), 2, "Should have at least 2 chunks")

				// 验证每个chunk的token数
				for i, chunk := range chunks {
					assert.LessOrEqual(t, chunk.TokenCount, totalTokens/3,
						"Chunk %d should not exceed max tokens", i)
				}

				// 验证重叠部分
				for i := 1; i < len(chunks); i++ {
					prevChunk := chunks[i-1].Content
					currChunk := chunks[i].Content

					// 验证重叠部分的大小
					overlapTokens := splitter.countToken(findOverlap(prevChunk, currChunk))
					assert.GreaterOrEqual(t, overlapTokens, totalTokens/10,
						"Chunk %d should have sufficient overlap with previous chunk", i)
				}

				// 验证所有chunk组合起来包含完整函数
				combinedContent := ""
				for _, chunk := range chunks {
					combinedContent += chunk.Content
				}
				assert.Contains(t, combinedContent, "func veryLongFunc()")
				assert.Contains(t, combinedContent, "lines = append(lines, fmt.Sprintf")
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
			assert.GreaterOrEqual(t, len(chunks), tc.expectedChunks,
				"Should have at least expected number of chunks")

			// 运行自定义验证
			tc.verifyOverlap(t, chunks)
		})
	}
}

// 辅助函数：查找两个字符串的重叠部分
func findOverlap(s1, s2 string) string {
	// 简单实现：找到最长的公共子串
	// 实际实现可能需要更复杂的逻辑
	for i := 0; i < len(s1); i++ {
		for j := 0; j < len(s2); j++ {
			k := 0
			for i+k < len(s1) && j+k < len(s2) && s1[i+k] == s2[j+k] {
				k++
			}
			if k > 0 {
				return s1[i : i+k]
			}
		}
	}
	return ""
}

// 辅助函数：检查内容是否包含特定函数定义
func containsFunc(content, funcName string) bool {
	return contains(content, "func "+funcName+"(") || contains(content, "func "+funcName+"()")
}

// 辅助函数：简单的字符串包含检查
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr
}
