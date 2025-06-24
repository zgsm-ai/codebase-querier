package parser

import (
	"reflect"
	"testing"
)

func TestJavaResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		Files: []string{
			"src/a/b/c/Class1.java",
			"src/a/b/c/Class2.java",
			"src/a/b/d/Class3.java",
		},
	}
	cfg.BuildIndex()
	resolver := &JavaResolver{Config: cfg}

	// 测试包导入
	imp := &Import{Source: "a.b.c.*"}
	err := resolver.Resolve(imp, "src/a/b/c/Any.java")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/a/b/c/Class1.java", "src/a/b/c/Class2.java"}) {
		t.Errorf("Java 包导入失败: %v, %v", err, imp.FilePaths)
	}

	// 测试类导入
	imp = &Import{Source: "a.b.c.Class1"}
	err = resolver.Resolve(imp, "src/a/b/c/Any.java")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/a/b/c/Class1.java"}) {
		t.Errorf("Java 类导入失败: %v, %v", err, imp.FilePaths)
	}
}

func TestGoResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		SourceDirs: []string{"pkg"},
		Files: []string{
			"pkg/foo/bar.go",
			"pkg/foo/baz.go",
			"pkg/other.go",
		},
	}
	cfg.BuildIndex()
	resolver := &GoResolver{Config: cfg}

	// 测试包导入
	imp := &Import{Source: "foo"}
	err := resolver.Resolve(imp, "pkg/other.go")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"pkg/foo/bar.go", "pkg/foo/baz.go"}) {
		t.Errorf("Go 包导入失败: %v, %v", err, imp.FilePaths)
	}

	// 测试文件导入
	imp = &Import{Source: "foo/bar"}
	err = resolver.Resolve(imp, "pkg/other.go")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"pkg/foo/bar.go"}) {
		t.Errorf("Go 文件导入失败: %v, %v", err, imp.FilePaths)
	}
}

func TestPythonResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		SourceDirs: []string{"src"},
		Files: []string{
			"src/foo/__init__.py",
			"src/foo/bar.py",
		},
	}
	cfg.BuildIndex()
	resolver := &PythonResolver{Config: cfg}

	// 测试绝对导入
	imp := &Import{Source: "foo.bar"}
	err := resolver.Resolve(imp, "src/main.py")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo/bar.py"}) {
		t.Errorf("Python 绝对导入失败: %v, %v", err, imp.FilePaths)
	}
}

func TestCppResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		SourceDirs: []string{"include"},
		Files: []string{
			"include/foo.h",
			"include/bar.h",
		},
	}
	cfg.BuildIndex()
	resolver := &CppResolver{Config: cfg}

	// 测试相对路径导入
	imp := &Import{Source: "foo.h"}
	err := resolver.Resolve(imp, "include/main.cpp")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"include/foo.h"}) {
		t.Errorf("C/C++ 头文件导入失败: %v, %v", err, imp.FilePaths)
	}
}

func TestJavaScriptResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		SourceDirs: []string{"src"},
		Files: []string{
			"src/foo.js",
			"src/bar/index.js",
		},
	}
	cfg.BuildIndex()
	resolver := &JavaScriptResolver{Config: cfg}

	// 测试相对路径导入
	imp := &Import{Source: "./foo"}
	err := resolver.Resolve(imp, "src/main.js")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo.js"}) {
		t.Errorf("JS 相对路径导入失败: %v, %v", err, imp.FilePaths)
	}

	// 测试绝对路径导入
	imp = &Import{Source: "bar"}
	err = resolver.Resolve(imp, "src/main.js")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/bar/index.js"}) {
		t.Errorf("JS 绝对路径导入失败: %v, %v", err, imp.FilePaths)
	}
}

func TestRustResolver_Basic(t *testing.T) {
	cfg := &ProjectConfig{
		SourceDirs: []string{"src"},
		Files: []string{
			"src/foo.rs",
			"src/bar/mod.rs",
		},
	}
	cfg.BuildIndex()
	resolver := &RustResolver{Config: cfg}

	// 测试模块导入
	imp := &Import{Source: "foo"}
	err := resolver.Resolve(imp, "src/main.rs")
	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo.rs"}) {
		t.Errorf("Rust 模块导入失败: %v, %v", err, imp.FilePaths)
	}
}
