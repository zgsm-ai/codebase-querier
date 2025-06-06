package embedding

import (
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/lang"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGoFileStructure(t *testing.T) {
	// 测试用的 Go 代码
	code := []byte(`
package test

// TestStruct 是一个测试结构体
type TestStruct struct {
	Identifier string
	Age  int
}

// TestInterface 是一个测试接口
type TestInterface interface {
	GetName() string
	GetAge() int
}

// TestFunc 是一个测试函数
func TestFunc(name string, age int) (string, error) {
	return name, nil
}

// TestMethod 是一个测试方法
func (s *TestStruct) TestMethod() string {
	return s.Identifier
}

// 常量定义
const TestConst = "test"

// 变量定义
var TestVar = "test"
`)

	// 获取 Go 语言配置
	parser, err := NewStructureParser()
	assert.NoError(t, err)
	// 解析文件结构
	structure, err := parser.Parse(&types.CodeFile{
		Content: code,
		Path:    "test.go",
	})
	if err != nil {
		t.Fatalf("failed to parse file structure: %v", err)
	}

	// 验证结果
	if len(structure.Definitions) == 0 {
		t.Fatal("no definitions found")
	}
	assert.NotEmpty(t, structure.Path)
	assert.NotEmpty(t, structure.Language)
	// 验证结构体定义
	foundStruct := false
	foundInterface := false
	foundFunction := false
	foundMethod := false
	foundConst := false
	foundVar := false

	for _, def := range structure.Definitions {
		switch def.Type {
		case lang.Struct:
			if def.Name == "TestStruct" {
				foundStruct = true
			}
		case lang.Interface:
			if def.Name == "TestInterface" {
				foundInterface = true
			}
		case lang.Function:
			if def.Name == "TestFunc" {
				foundFunction = true
			} else if def.Name == "TestMethod" {
				foundMethod = true
			}
		case lang.Variable:
			if def.Name == "TestConst" {
				foundConst = true
			} else if def.Name == "TestVar" {
				foundVar = true
			}
		}
	}

	// 检查是否找到所有定义
	if !foundStruct {
		t.Error("TestStruct not found")
	}
	if !foundInterface {
		t.Error("TestInterface not found")
	}
	if !foundFunction {
		t.Error("TestFunc not found")
	}
	if !foundMethod {
		t.Error("TestMethod not found")
	}
	if !foundConst {
		t.Error("TestConst not found")
	}
	if !foundVar {
		t.Error("TestVar not found")
	}
}
