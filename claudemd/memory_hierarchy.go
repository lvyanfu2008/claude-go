package claudemd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MemoryHierarchy 定义了记忆文件的完整层次结构
type MemoryHierarchy struct {
	// 记忆类型及其优先级顺序（从低到高）
	typesInOrder []MemoryType

	// 每种记忆类型的路径解析函数
	pathResolvers map[MemoryType]func(cwd string) []string

	// 设置源启用检查
	settingSourceCheckers map[MemoryType]func() bool

	// 排除检查器
	excludeChecker *ExcludeChecker

	// 原始工作目录
	originalCwd string
}

// NewMemoryHierarchy 创建新的记忆层次结构管理器
func NewMemoryHierarchy(cwd string, excludePatterns []string) *MemoryHierarchy {
	absCwd, _ := filepath.Abs(cwd)
	excludeChecker := NewExcludeChecker(excludePatterns)

	mh := &MemoryHierarchy{
		typesInOrder: []MemoryType{
			MemoryManaged,  // 最低优先级：托管内存
			MemoryUser,     // 用户内存
			MemoryProject,  // 项目内存
			MemoryLocal,    // 本地内存
			MemoryAutoMem,  // 自动记忆
			MemoryTeamMem,  // 团队记忆（如果启用）
		},
		pathResolvers: make(map[MemoryType]func(cwd string) []string),
		settingSourceCheckers: make(map[MemoryType]func() bool),
		excludeChecker: excludeChecker,
		originalCwd:    absCwd,
	}

	// 初始化路径解析器
	mh.initPathResolvers(absCwd)

	// 初始化设置源检查器
	mh.initSettingSourceCheckers()

	return mh
}

// initPathResolvers 初始化每种记忆类型的路径解析器
func (mh *MemoryHierarchy) initPathResolvers(cwd string) {
	// 托管内存：/etc/claude-code/CLAUDE.md 或等效路径
	mh.pathResolvers[MemoryManaged] = func(_ string) []string {
		return []string{filepath.Join(ManagedFilePath(), "CLAUDE.md")}
	}

	// 用户内存：~/.claude/CLAUDE.md
	mh.pathResolvers[MemoryUser] = func(_ string) []string {
		cfg, err := ClaudeConfigHomeDir()
		if err != nil {
			return []string{}
		}
		path := filepath.Join(cfg, "CLAUDE.md")
		return []string{path}
	}

	// 项目内存：从当前目录向上遍历到根目录
	mh.pathResolvers[MemoryProject] = func(cwd string) []string {
		var paths []string
		dirs := directoryChainUp(cwd)

		// 处理 Git 工作树嵌套情况
		gitRoot := FindGitRoot(cwd)
		canonicalRoot := ResolveCanonicalGitRoot(cwd)
		isNestedWorktree := gitRoot != "" && canonicalRoot != "" &&
			NormalizePathForComparison(gitRoot) != NormalizePathForComparison(canonicalRoot) &&
			PathInWorkingPath(gitRoot, canonicalRoot)

		// 从根目录向下到当前目录
		for i := len(dirs) - 1; i >= 0; i-- {
			dir := dirs[i]

			// 在嵌套工作树中，跳过主仓库中的检查文件
			skipProject := isNestedWorktree &&
				PathInWorkingPath(dir, canonicalRoot) &&
				!PathInWorkingPath(dir, gitRoot)

			if skipProject {
				continue
			}

			// CLAUDE.md 在目录中
			paths = append(paths, filepath.Join(dir, "CLAUDE.md"))

			// .claude/CLAUDE.md 在目录中
			paths = append(paths, filepath.Join(dir, ".claude", "CLAUDE.md"))
		}

		return paths
	}

	// 本地内存：CLAUDE.local.md 从当前目录向上遍历
	mh.pathResolvers[MemoryLocal] = func(cwd string) []string {
		var paths []string
		dirs := directoryChainUp(cwd)

		// 从根目录向下到当前目录
		for i := len(dirs) - 1; i >= 0; i-- {
			dir := dirs[i]
			paths = append(paths, filepath.Join(dir, "CLAUDE.local.md"))
		}

		return paths
	}

	// 自动记忆：<auto-memory-path>/MEMORY.md
	mh.pathResolvers[MemoryAutoMem] = func(cwd string) []string {
		if !IsAutoMemoryEnabled() {
			return []string{}
		}
		autoPath := strings.TrimSuffix(GetAutoMemPath(cwd), string(filepath.Separator))
		return []string{filepath.Join(autoPath, "MEMORY.md")}
	}

	// 团队记忆：<auto-memory-path>/team/MEMORY.md
	mh.pathResolvers[MemoryTeamMem] = func(cwd string) []string {
		if !IsTeamMemoryPromptActive() {
			return []string{}
		}
		autoPath := strings.TrimSuffix(GetAutoMemPath(cwd), string(filepath.Separator))
		return []string{filepath.Join(autoPath, "team", "MEMORY.md")}
	}
}

// initSettingSourceCheckers 初始化设置源检查器
func (mh *MemoryHierarchy) initSettingSourceCheckers() {
	// 托管内存：始终启用
	mh.settingSourceCheckers[MemoryManaged] = func() bool { return true }

	// 用户内存：检查用户设置是否启用
	mh.settingSourceCheckers[MemoryUser] = userMemoryEnabled

	// 项目内存：检查项目设置是否启用
	mh.settingSourceCheckers[MemoryProject] = projectMemoryEnabled

	// 本地内存：检查本地设置是否启用
	mh.settingSourceCheckers[MemoryLocal] = localMemoryEnabled

	// 自动记忆：检查自动记忆是否启用
	mh.settingSourceCheckers[MemoryAutoMem] = IsAutoMemoryEnabled

	// 团队记忆：检查团队记忆是否启用
	mh.settingSourceCheckers[MemoryTeamMem] = IsTeamMemoryPromptActive
}

// LoadAllMemoryFiles 加载所有记忆文件，按照正确的优先级顺序
func (mh *MemoryHierarchy) LoadAllMemoryFiles(cwd string, includeExternal bool) []MemoryFileInfo {
	absCwd, _ := filepath.Abs(cwd)
	processedPaths := make(map[string]struct{})
	var allFiles []MemoryFileInfo

	// 按照优先级顺序加载记忆文件
	for _, memoryType := range mh.typesInOrder {
		// 检查该记忆类型是否启用
		if checker, ok := mh.settingSourceCheckers[memoryType]; ok && !checker() {
			continue
		}

		// 获取该记忆类型的所有路径
		pathResolver, ok := mh.pathResolvers[memoryType]
		if !ok {
			continue
		}

		paths := pathResolver(absCwd)
		for _, path := range paths {
			// 处理记忆文件（包括 @include 扩展）
			files := mh.processMemoryFileWithIncludes(path, memoryType, processedPaths, includeExternal, absCwd)
			allFiles = append(allFiles, files...)
		}

		// 处理规则目录（对于 Managed、User、Project 类型）
		if memoryType == MemoryManaged || memoryType == MemoryUser || memoryType == MemoryProject {
			rulesFiles := mh.loadRulesFiles(memoryType, absCwd, processedPaths, includeExternal)
			allFiles = append(allFiles, rulesFiles...)
		}
	}

	return allFiles
}

// processMemoryFileWithIncludes 处理记忆文件，包括 @include 指令
func (mh *MemoryHierarchy) processMemoryFileWithIncludes(
	filePath string,
	memoryType MemoryType,
	processedPaths map[string]struct{},
	includeExternal bool,
	baseCwd string,
) []MemoryFileInfo {
	return mh.processMemoryFileWithIncludesDepth(filePath, memoryType, processedPaths, includeExternal, baseCwd, 0)
}

// processMemoryFileWithIncludesDepth 带深度限制的递归处理
func (mh *MemoryHierarchy) processMemoryFileWithIncludesDepth(
	filePath string,
	memoryType MemoryType,
	processedPaths map[string]struct{},
	includeExternal bool,
	baseCwd string,
	depth int,
) []MemoryFileInfo {
	var files []MemoryFileInfo

	// 检查深度限制
	if depth >= maxIncludeDepth {
		return files
	}

	// 规范化路径用于去重
	normPath := NormalizePathForComparison(filePath)
	if _, processed := processedPaths[normPath]; processed {
		return files
	}

	// 检查路径是否被排除
	if mh.excludeChecker.IsExcluded(filePath, memoryType) {
		return files
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		// 文件不存在是正常情况，静默忽略
		return files
	}

	// 解析文件内容，提取 @include 路径
	info, includePaths := ParseMemoryFileContent(string(content), filePath, memoryType, filePath)
	if info == nil {
		return files
	}

	// 标记为已处理
	processedPaths[normPath] = struct{}{}
	files = append(files, *info)

	// 处理 @include 指令
	// 总是处理包含指令，但会检查文件是否在工作路径内
	{
		for _, includePath := range includePaths {
			// 检查包含的文件是否在工作路径内
			if !PathInWorkingPath(includePath, mh.originalCwd) && !includeExternal {
				continue
			}

			// 递归处理包含的文件
			includedFiles := mh.processMemoryFileWithIncludesDepth(
				includePath,
				memoryType, // 包含的文件继承相同的记忆类型
				processedPaths,
				includeExternal,
				baseCwd,
				depth+1,
			)
			// 包含的文件应该出现在包含文件之前（TS 实现）
			files = append(includedFiles, files...)
		}
	}

	return files
}

// loadRulesFiles 加载规则目录中的文件
func (mh *MemoryHierarchy) loadRulesFiles(
	memoryType MemoryType,
	cwd string,
	processedPaths map[string]struct{},
	includeExternal bool,
) []MemoryFileInfo {
	var rulesDir string

	switch memoryType {
	case MemoryManaged:
		rulesDir = managedClaudeRulesDir()
	case MemoryUser:
		dir, err := userClaudeRulesDir()
		if err != nil {
			return []MemoryFileInfo{}
		}
		rulesDir = dir
	case MemoryProject:
		// 项目规则目录从当前目录向上遍历
		return mh.loadProjectRulesFiles(cwd, processedPaths, includeExternal)
	default:
		return []MemoryFileInfo{}
	}

	return ProcessMdRules(rulesDir, memoryType, processedPaths, includeExternal, cwd, false, nil, mh.excludeChecker)
}

// loadProjectRulesFiles 加载项目规则文件
func (mh *MemoryHierarchy) loadProjectRulesFiles(
	cwd string,
	processedPaths map[string]struct{},
	includeExternal bool,
) []MemoryFileInfo {
	var allFiles []MemoryFileInfo
	dirs := directoryChainUp(cwd)

	// 处理 Git 工作树嵌套情况
	gitRoot := FindGitRoot(cwd)
	canonicalRoot := ResolveCanonicalGitRoot(cwd)
	isNestedWorktree := gitRoot != "" && canonicalRoot != "" &&
		NormalizePathForComparison(gitRoot) != NormalizePathForComparison(canonicalRoot) &&
		PathInWorkingPath(gitRoot, canonicalRoot)

	// 从根目录向下到当前目录
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]

		// 在嵌套工作树中，跳过主仓库中的检查文件
		skipProject := isNestedWorktree &&
			PathInWorkingPath(dir, canonicalRoot) &&
			!PathInWorkingPath(dir, gitRoot)

		if skipProject {
			continue
		}

		rulesDir := filepath.Join(dir, ".claude", "rules")
		files := ProcessMdRules(rulesDir, MemoryProject, processedPaths, includeExternal, cwd, false, nil, mh.excludeChecker)
		allFiles = append(allFiles, files...)
	}

	return allFiles
}

// isSettingSourceEnabled 检查设置源是否启用
func isSettingSourceEnabled(source string) bool {
	// 简化实现：检查环境变量
	// 完整的实现应该解析 settings.json 文件
	envVar := fmt.Sprintf("CLAUDE_CODE_SETTING_SOURCES_%s", strings.ToUpper(source))
	if val := strings.TrimSpace(os.Getenv(envVar)); val != "" {
		return truthy(val)
	}

	// 默认启用所有设置源
	return true
}