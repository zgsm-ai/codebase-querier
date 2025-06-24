package parser

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// 项目配置信息（简化版）
type ProjectConfig struct {
	Language   Language            // 项目语言
	SourceDirs []string            // 项目下所有源文件目录（相对项目根目录）
	Files      []string            // 项目下所有源文件路径（相对于项目根目录）
	DirToFiles map[string][]string // 目录到文件列表的索引
	FileSet    map[string]struct{} // 文件路径集合，便于快速查找
}

// 构建目录和文件索引
func (c *ProjectConfig) BuildIndex() {
	c.DirToFiles = make(map[string][]string)
	c.FileSet = make(map[string]struct{})
	for _, f := range c.Files {
		dir := filepath.Dir(f)
		c.DirToFiles[dir] = append(c.DirToFiles[dir], f)
		c.FileSet[f] = struct{}{}
	}
}

// 导入解析器接口
type ImportResolver interface {
	Resolve(importStmt *Import, currentFilePath string) error
}

// 解析器管理器
type ResolverManager struct {
	resolvers map[Language]ImportResolver
	mu        sync.RWMutex
}

// 新建解析器管理器
func NewResolverManager() *ResolverManager {
	return &ResolverManager{
		resolvers: make(map[Language]ImportResolver),
	}
}

// 注册解析器
func (rm *ResolverManager) Register(language Language, resolver ImportResolver) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.resolvers[language] = resolver
}

// 解析导入语句
func (rm *ResolverManager) ResolveImport(importStmt *Import, currentFilePath string, language Language) error {
	rm.mu.RLock()
	resolver, exists := rm.resolvers[language]
	rm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("不支持的语言: %s", language)
	}

	return resolver.Resolve(importStmt, currentFilePath)
}

// 初始化所有解析器
func InitResolvers(config *ProjectConfig) *ResolverManager {
	manager := NewResolverManager()

	manager.Register(Java, &JavaResolver{Config: config})
	manager.Register(Python, &PythonResolver{Config: config})
	manager.Register(Go, &GoResolver{Config: config})
	manager.Register(C, &CppResolver{Config: config})
	manager.Register(CPP, &CppResolver{Config: config})
	manager.Register(JavaScript, &JavaScriptResolver{Config: config})
	manager.Register(TypeScript, &JavaScriptResolver{Config: config})
	manager.Register(Ruby, &RubyResolver{Config: config})
	manager.Register(Kotlin, &KotlinResolver{Config: config})
	manager.Register(PHP, &PHPResolver{Config: config})
	manager.Register(Scala, &ScalaResolver{Config: config})
	manager.Register(Rust, &RustResolver{Config: config})

	return manager
}

// Java解析器
type JavaResolver struct {
	Config *ProjectConfig
}

func (r *JavaResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理静态导入
	if strings.HasPrefix(source, "static ") {
		source = strings.TrimPrefix(source, "static ")
	}

	// 处理包导入
	if strings.HasSuffix(source, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(source, ".*"), ".", "/")
		files := findFilesInDirIndex(r.Config, pkgPath, ".java")
		importStmt.FilePaths = files
		if len(importStmt.FilePaths) == 0 {
			return fmt.Errorf("未找到包导入对应的文件: %s", source)
		}
		return nil
	}

	// 处理类导入
	classPath := strings.ReplaceAll(source, ".", "/") + ".java"
	importStmt.FilePaths = findMatchingFiles(r.Config, classPath)

	if len(importStmt.FilePaths) == 0 {
		return fmt.Errorf("未找到导入对应的文件: %s", source)
	}

	return nil
}

// Python解析器
type PythonResolver struct {
	Config *ProjectConfig
}

func (r *PythonResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理相对导入
	if strings.HasPrefix(source, ".") {
		currentDir := filepath.Dir(currentFilePath)
		dots := strings.Count(source, ".")
		modulePath := strings.TrimPrefix(source, strings.Repeat(".", dots))

		// 向上移动目录层级
		dir := currentDir
		for i := 0; i < dots-1; i++ {
			dir = filepath.Dir(dir)
		}

		// 构建完整路径
		if modulePath != "" {
			modulePath = strings.ReplaceAll(modulePath, ".", "/")
			dir = filepath.Join(dir, modulePath)
		}

		// 检查是否为包或模块
		for _, ext := range []string{"__init__.py", ".py"} {
			relPath := filepath.ToSlash(filepath.Join(dir, ext))
			if containsFileIndex(r.Config, relPath) {
				importStmt.FilePaths = append(importStmt.FilePaths, relPath)
			}
		}

		if len(importStmt.FilePaths) > 0 {
			return nil
		}

		return fmt.Errorf("未找到相对导入对应的文件: %s", source)
	}

	// 处理绝对导入
	modulePath := strings.ReplaceAll(source, ".", "/")
	foundPaths := []string{}

	// 检查是否为包或模块
	for _, ext := range []string{"__init__.py", ".py"} {
		for _, srcDir := range r.Config.SourceDirs {
			relPath := filepath.ToSlash(filepath.Join(srcDir, modulePath, ext))
			if containsFileIndex(r.Config, relPath) {
				foundPaths = append(foundPaths, relPath)
			}
			relPath = filepath.ToSlash(filepath.Join(srcDir, modulePath+ext))
			if containsFileIndex(r.Config, relPath) {
				foundPaths = append(foundPaths, relPath)
			}
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到绝对导入对应的文件: %s", source)
}

// Go解析器（简化版）
type GoResolver struct {
	Config *ProjectConfig
}

func (r *GoResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 简化处理：假设非相对路径为项目内导入
	relPath := source
	if !strings.HasPrefix(source, ".") {
		relPath = strings.TrimPrefix(source, "/") // 移除绝对路径前缀
	}

	// 尝试匹配 .go 文件
	relPathWithExt := relPath + ".go"
	if containsFileIndex(r.Config, relPathWithExt) {
		importStmt.FilePaths = []string{relPathWithExt}
		return nil
	}

	// 匹配包目录下所有 .go 文件
	for _, srcDir := range r.Config.SourceDirs {
		dirPath := filepath.ToSlash(filepath.Join(srcDir, relPath))
		filesInDir := findFilesInDirIndex(r.Config, dirPath, ".go")
		if len(filesInDir) > 0 {
			importStmt.FilePaths = append(importStmt.FilePaths, filesInDir...)
		}
	}
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// C/C++解析器
type CppResolver struct {
	Config *ProjectConfig
}

func (r *CppResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理系统头文件
	if strings.HasPrefix(source, "<") && strings.HasSuffix(source, ">") {
		return nil // 系统头文件，不映射到项目文件
	}

	// 移除引号
	headerFile := strings.Trim(source, "\"")
	foundPaths := []string{}

	// 相对路径导入
	if strings.HasPrefix(headerFile, ".") {
		currentDir := filepath.Dir(currentFilePath)
		relPath := filepath.ToSlash(filepath.Join(currentDir, headerFile))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	// 在源目录中查找
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, headerFile))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// JavaScript/TypeScript解析器
type JavaScriptResolver struct {
	Config *ProjectConfig
}

func (r *JavaScriptResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理相对路径
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		currentDir := filepath.Dir(currentFilePath)
		targetPath := filepath.Join(currentDir, source)
		foundPaths := []string{}

		// 尝试不同的文件扩展名
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			relPath := filepath.ToSlash(targetPath + ext)
			if containsFileIndex(r.Config, relPath) {
				foundPaths = append(foundPaths, relPath)
			}
		}

		importStmt.FilePaths = foundPaths
		if len(importStmt.FilePaths) > 0 {
			return nil
		}

		return fmt.Errorf("未找到相对导入对应的文件: %s", source)
	}

	// 处理项目内绝对路径导入
	foundPaths := []string{}
	for _, srcDir := range r.Config.SourceDirs {
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			relPath := filepath.ToSlash(filepath.Join(srcDir, source+ext))
			if containsFileIndex(r.Config, relPath) {
				foundPaths = append(foundPaths, relPath)
			}
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// Rust解析器
type RustResolver struct {
	Config *ProjectConfig
}

func (r *RustResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理crate根路径
	if strings.HasPrefix(source, "crate::") {
		source = strings.TrimPrefix(source, "crate::")
	}

	// 将::转换为路径分隔符
	modulePath := strings.ReplaceAll(source, "::", "/")
	foundPaths := []string{}

	// 尝试查找.rs文件或模块目录
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, modulePath+".rs"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		relModPath := filepath.ToSlash(filepath.Join(srcDir, modulePath, "mod.rs"))
		if containsFileIndex(r.Config, relModPath) {
			foundPaths = append(foundPaths, relModPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// Ruby解析器
type RubyResolver struct {
	Config *ProjectConfig
}

func (r *RubyResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理相对导入
	if strings.HasPrefix(source, ".") {
		currentDir := filepath.Dir(currentFilePath)
		relPath := strings.TrimPrefix(source, ".")
		if relPath == "" {
			return fmt.Errorf("无效的相对导入: %s", source)
		}

		// 添加.rb扩展名
		if !strings.HasSuffix(relPath, ".rb") {
			relPath += ".rb"
		}

		relPathStr := filepath.ToSlash(filepath.Join(currentDir, relPath))
		if containsFileIndex(r.Config, relPathStr) {
			importStmt.FilePaths = []string{relPathStr}
			return nil
		}

		return fmt.Errorf("未找到相对导入对应的文件: %s", source)
	}

	// 处理项目内导入
	foundPaths := []string{}
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, source+".rb"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		relPath = filepath.ToSlash(filepath.Join(srcDir, source))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// Kotlin解析器
type KotlinResolver struct {
	Config *ProjectConfig
}

func (r *KotlinResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理包导入
	if strings.HasSuffix(source, ".*") {
		return nil // 包导入不映射到具体文件
	}

	// 处理类导入
	classPath := strings.ReplaceAll(source, ".", "/")
	foundPaths := []string{}

	// 尝试Kotlin文件
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, classPath+".kt"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		// 尝试Java文件
		relPath = filepath.ToSlash(filepath.Join(srcDir, classPath+".java"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// PHP解析器（简化版）
type PHPResolver struct {
	Config *ProjectConfig
}

func (r *PHPResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理命名空间导入
	if strings.HasPrefix(source, "\\") {
		source = strings.TrimPrefix(source, "\\")
	}

	// 将命名空间分隔符转换为路径分隔符
	namespacePath := strings.ReplaceAll(source, "\\", "/")
	foundPaths := []string{}

	// 在源目录中查找
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, namespacePath+".php"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// Scala解析器
type ScalaResolver struct {
	Config *ProjectConfig
}

func (r *ScalaResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Source == "" {
		return fmt.Errorf("导入源为空")
	}

	importStmt.FilePaths = []string{}
	source := importStmt.Source

	// 处理包导入
	if strings.HasSuffix(source, "._") {
		return nil // 包导入不映射到具体文件
	}

	// 处理类导入
	classPath := strings.ReplaceAll(source, ".", "/")
	foundPaths := []string{}

	// 尝试Scala文件
	for _, srcDir := range r.Config.SourceDirs {
		relPath := filepath.ToSlash(filepath.Join(srcDir, classPath+".scala"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		// 尝试Java文件
		relPath = filepath.ToSlash(filepath.Join(srcDir, classPath+".java"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到导入对应的文件: %s", source)
}

// 辅助函数：查找匹配的文件路径
func findMatchingFiles(config *ProjectConfig, targetPath string) []string {
	var result []string
	if containsFileIndex(config, targetPath) {
		result = append(result, targetPath)
	}
	return result
}

// 辅助函数：查找目录下所有指定扩展名的文件
func findFilesInDirIndex(config *ProjectConfig, dir string, ext string) []string {
	var result []string
	files, ok := config.DirToFiles[dir]
	if !ok {
		return result
	}
	for _, f := range files {
		if strings.HasSuffix(f, ext) {
			result = append(result, f)
		}
	}
	return result
}

// 辅助函数：检查文件是否存在于项目文件集合中
func containsFileIndex(config *ProjectConfig, path string) bool {
	_, ok := config.FileSet[path]
	return ok
}
