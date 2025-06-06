package lang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGoFileStructure(t *testing.T) {
	// 测试用的 Go 代码
	code := []byte(`
package test

// TestStruct 是一个测试结构体
type TestStruct struct {
	Name string
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
	return s.Name
}

// 常量定义
const TestConst = "test"

// 变量定义
var TestVar = "test"
`)

	// 获取 Go 语言配置
	configs, err := GetLanguageConfigs()
	assert.NoError(t, err)
	goConfig := GetLanguageConfigByExt(configs, ".go")
	assert.NotNil(t, goConfig, "Go config should not be nil")
	assert.NotEmpty(t, goConfig.structureQueryPath, "structure query path should not be empty")
	assert.NotEmpty(t, goConfig.StructureQuery, "structure query should not be empty")
	// 解析文件结构
	structure, err := ParseFileStructure(code, goConfig)
	if err != nil {
		t.Fatalf("failed to parse file structure: %v", err)
	}

	// 验证结果
	if len(structure.Definitions) == 0 {
		t.Fatal("no definitions found")
	}

	// 验证结构体定义
	foundStruct := false
	foundInterface := false
	foundFunction := false
	foundMethod := false
	foundConst := false
	foundVar := false

	for _, def := range structure.Definitions {
		switch def.Type {
		case Struct:
			if def.Name == "TestStruct" {
				foundStruct = true
			}
		case Interface:
			if def.Name == "TestInterface" {
				foundInterface = true
			}
		case Function:
			if def.Name == "TestFunc" {
				foundFunction = true
			} else if def.Name == "TestMethod" {
				foundMethod = true
			}
		case Variable:
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
