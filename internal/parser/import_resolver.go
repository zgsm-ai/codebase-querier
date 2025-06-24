package parser

import (
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"path/filepath"
	"strings"
	"sync"
)

// 项目基础配置信息
type ProjectConfig struct {
	language   Language            // 项目语言
	SourceRoot string              // 源码根路径（如 java 的 src/main/java）
	Dirs       []string            // 源文件目录（相对于 SourceRoot）
	dirToFiles map[string][]string // 目录到文件列表的索引（完整路径）
	fileSet    map[string]struct{} // 文件路径集合（完整路径）
}

func NewProjectConfig(language Language, sourceRoot string, files []string) *ProjectConfig {
	pc := &ProjectConfig{
		language:   language,
		SourceRoot: sourceRoot,
	}
	pc.buildIndex(files)
	return pc
}

// 构建目录和文件索引
func (c *ProjectConfig) buildIndex(files []string) {
	c.dirToFiles = make(map[string][]string)
	c.fileSet = make(map[string]struct{})
	dirSet := make(map[string]struct{})

	for _, f := range files {
		dir := filepath.Dir(f)
		c.dirToFiles[dir] = append(c.dirToFiles[dir], f)
		c.fileSet[f] = struct{}{}
		dirSet[dir] = struct{}{}
	}

	// 提取相对于 SourceRoot 的目录
	c.Dirs = make([]string, 0, len(dirSet))
	for dir := range dirSet {
		// 计算相对于 SourceRoot 的路径
		c.Dirs = append(c.Dirs, dir)
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
		return fmt.Errorf("import resolver unsupported language: %s", language)
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
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理静态导入
	if strings.HasPrefix(importName, "static ") {
		importName = strings.TrimPrefix(importName, "static ")
	}

	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(importName, ".*"), ".", "/")
		fullPkgPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, pkgPath))
		files := findFilesInDirIndex(r.Config, fullPkgPath, ".java")
		importStmt.FilePaths = files
		if len(importStmt.FilePaths) == 0 {
			return fmt.Errorf("cannot find file which package belongs to: %s", importName)
		}
		return nil
	}

	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/") + ".java"
	fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, classPath))
	importStmt.FilePaths = findMatchingFiles(r.Config, fullPath)

	if len(importStmt.FilePaths) == 0 {
		return fmt.Errorf("cannot find file which import belongs to: %s", importName)
	}

	return nil
}

// Python解析器
type PythonResolver struct {
	Config *ProjectConfig
}

func (r *PythonResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理相对导入
	if strings.HasPrefix(importName, ".") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(r.Config.SourceRoot, currentFilePath)
		currentDir := filepath.Dir(currentRelPath)
		dots := strings.Count(importName, ".")
		modulePath := strings.TrimPrefix(importName, strings.Repeat(".", dots))

		// 向上移动目录层级
		dir := currentDir
		for i := 0; i < dots-1; i++ {
			dir = filepath.Dir(dir)
		}

		// 构建完整路径
		if modulePath != "" {
			modulePath = strings.ReplaceAll(modulePath, ".", "/")
			dir = utils.ToUnixPath(filepath.Join(dir, modulePath))
		}

		// 检查是否为包或模块
		for _, ext := range []string{"__init__.py", ".py"} {
			fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, dir, ext))
			if containsFileIndex(r.Config, fullPath) {
				importStmt.FilePaths = append(importStmt.FilePaths, fullPath)
			}
		}

		if len(importStmt.FilePaths) > 0 {
			return nil
		}

		return fmt.Errorf("cannot find file which relative import belongs to: %s", importName)
	}

	// 处理绝对导入
	modulePath := strings.ReplaceAll(importName, ".", "/")
	foundPaths := []string{}

	// 检查是否为包或模块
	for _, ext := range []string{"__init__.py", ".py"} {
		for _, relDir := range r.Config.Dirs {
			dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
			fullPath := utils.ToUnixPath(filepath.Join(dir, modulePath, ext))
			if containsFileIndex(r.Config, fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
			fullPath = utils.ToUnixPath(filepath.Join(dir, modulePath+ext))
			if containsFileIndex(r.Config, fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("未找到绝对导入对应的文件: %s", importName)
}

// Go解析器（简化版）
type GoResolver struct {
	Config *ProjectConfig
}

func (r *GoResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 简化处理：假设非相对路径为项目内导入
	relPath := importName
	if !strings.HasPrefix(importName, ".") {
		relPath = strings.TrimPrefix(importName, "/") // 移除绝对路径前缀
	}

	// 尝试匹配 .go 文件
	relPathWithExt := relPath + ".go"
	fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relPathWithExt))
	if containsFileIndex(r.Config, fullPath) {
		importStmt.FilePaths = []string{fullPath}
		return nil
	}

	// 匹配包目录下所有 .go 文件
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		targetDir := utils.ToUnixPath(filepath.Join(dir, relPath))
		filesInDir := findFilesInDirIndex(r.Config, targetDir, ".go")
		if len(filesInDir) > 0 {
			importStmt.FilePaths = append(importStmt.FilePaths, filesInDir...)
		}
	}
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// C/C++解析器
type CppResolver struct {
	Config *ProjectConfig
}

func (r *CppResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return nil // 系统头文件，不映射到项目文件
	}

	// 移除引号
	headerFile := strings.Trim(importName, "\"")
	foundPaths := []string{}

	// 相对路径导入
	if strings.HasPrefix(headerFile, ".") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(r.Config.SourceRoot, currentFilePath)
		currentDir := filepath.Dir(currentRelPath)
		relPath := utils.ToUnixPath(filepath.Join(currentDir, headerFile))
		fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relPath))
		if containsFileIndex(r.Config, fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	// 在源目录中查找
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		fullPath := utils.ToUnixPath(filepath.Join(dir, headerFile))
		if containsFileIndex(r.Config, fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// JavaScript/TypeScript解析器
type JavaScriptResolver struct {
	Config *ProjectConfig
}

func (r *JavaScriptResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理相对路径
	if strings.HasPrefix(importName, "./") || strings.HasPrefix(importName, "../") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(r.Config.SourceRoot, currentFilePath)
		currentDir := filepath.Dir(currentRelPath)
		targetPath := utils.ToUnixPath(filepath.Join(currentDir, importName))
		foundPaths := []string{}

		// 尝试不同的文件扩展名
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, targetPath+ext))
			if containsFileIndex(r.Config, fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}

		importStmt.FilePaths = foundPaths
		if len(importStmt.FilePaths) > 0 {
			return nil
		}

		return fmt.Errorf("cannot find file which relative import belongs to: %s", importName)
	}

	// 处理项目内绝对路径导入
	foundPaths := []string{}
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(dir, importName+ext))
			if containsFileIndex(r.Config, fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// Rust解析器
type RustResolver struct {
	Config *ProjectConfig
}

func (r *RustResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理crate根路径
	if strings.HasPrefix(importName, "crate::") {
		importName = strings.TrimPrefix(importName, "crate::")
	}

	// 将::转换为路径分隔符
	modulePath := strings.ReplaceAll(importName, "::", "/")
	foundPaths := []string{}

	// 尝试查找.rs文件或模块目录
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		relPath := utils.ToUnixPath(filepath.Join(dir, modulePath+".rs"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		modPath := utils.ToUnixPath(filepath.Join(dir, modulePath, "mod.rs"))
		if containsFileIndex(r.Config, modPath) {
			foundPaths = append(foundPaths, modPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// Ruby解析器
type RubyResolver struct {
	Config *ProjectConfig
}

func (r *RubyResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理相对导入
	if strings.HasPrefix(importName, ".") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(r.Config.SourceRoot, currentFilePath)
		currentDir := filepath.Dir(currentRelPath)
		relPath := strings.TrimPrefix(importName, ".")
		if relPath == "" {
			return fmt.Errorf("invalid relative import: %s", importName)
		}

		// 添加.rb扩展名
		if !strings.HasSuffix(relPath, ".rb") {
			relPath += ".rb"
		}

		fullPath := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, currentDir, relPath))
		if containsFileIndex(r.Config, fullPath) {
			importStmt.FilePaths = []string{fullPath}
			return nil
		}

		return fmt.Errorf("canot find file which relative import belongs to: %s", importName)
	}

	// 处理项目内导入
	foundPaths := []string{}
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		relPath := utils.ToUnixPath(filepath.Join(dir, importName+".rb"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		relPath = utils.ToUnixPath(filepath.Join(dir, importName))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// Kotlin解析器
type KotlinResolver struct {
	Config *ProjectConfig
}

func (r *KotlinResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		return nil // 包导入不映射到具体文件
	}

	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/")
	foundPaths := []string{}

	// 尝试Kotlin文件
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		relPath := utils.ToUnixPath(filepath.Join(dir, classPath+".kt"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		// 尝试Java文件
		relPath = utils.ToUnixPath(filepath.Join(dir, classPath+".java"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// PHP解析器（简化版）
type PHPResolver struct {
	Config *ProjectConfig
}

func (r *PHPResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理命名空间导入
	if strings.HasPrefix(importName, "\\") {
		importName = strings.TrimPrefix(importName, "\\")
	}

	// 将命名空间分隔符转换为路径分隔符
	namespacePath := strings.ReplaceAll(importName, "\\", "/")
	foundPaths := []string{}

	// 在源目录中查找
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		fullPath := utils.ToUnixPath(filepath.Join(dir, namespacePath+".php"))
		if containsFileIndex(r.Config, fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

// Scala解析器
type ScalaResolver struct {
	Config *ProjectConfig
}

func (r *ScalaResolver) Resolve(importStmt *Import, currentFilePath string) error {
	if importStmt.Name == "" {
		return fmt.Errorf("import is empty")
	}

	importStmt.FilePaths = []string{}
	importName := importStmt.Name

	// 处理包导入
	if strings.HasSuffix(importName, "._") {
		return nil // 包导入不映射到具体文件
	}

	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/")
	foundPaths := []string{}

	// 尝试Scala文件
	for _, relDir := range r.Config.Dirs {
		dir := utils.ToUnixPath(filepath.Join(r.Config.SourceRoot, relDir))
		relPath := utils.ToUnixPath(filepath.Join(dir, classPath+".scala"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
		// 尝试Java文件
		relPath = utils.ToUnixPath(filepath.Join(dir, classPath+".java"))
		if containsFileIndex(r.Config, relPath) {
			foundPaths = append(foundPaths, relPath)
		}
	}

	importStmt.FilePaths = foundPaths
	if len(importStmt.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
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
	files, ok := config.dirToFiles[dir]
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
	_, ok := config.fileSet[path]
	return ok
}
