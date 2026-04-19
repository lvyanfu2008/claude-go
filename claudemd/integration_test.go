package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompleteMemoryHierarchy(t *testing.T) {
	// 创建完整的记忆文件层次结构测试
	tmpDir := t.TempDir()

	// 1. 创建托管内存
	managedDir := filepath.Join(tmpDir, "managed")
	os.MkdirAll(managedDir, 0755)
	managedClaudeMd := filepath.Join(managedDir, "CLAUDE.md")
	os.WriteFile(managedClaudeMd, []byte("# Managed Memory\n\nSystem-wide instructions."), 0644)

	// 2. 创建用户内存
	userDir := filepath.Join(tmpDir, "user", ".claude")
	os.MkdirAll(userDir, 0755)
	userClaudeMd := filepath.Join(userDir, "CLAUDE.md")
	os.WriteFile(userClaudeMd, []byte("# User Memory\n\nUser-specific instructions."), 0644)

	// 3. 创建项目结构
	projectRoot := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectRoot, "subdir")
	os.MkdirAll(subDir, 0755)

	// 在根目录创建 CLAUDE.md
	rootClaudeMd := filepath.Join(projectRoot, "CLAUDE.md")
	os.WriteFile(rootClaudeMd, []byte("# Root Project Memory\n\nInstructions at project root."), 0644)

	// 在根目录创建 .claude/CLAUDE.md
	dotClaudeDir := filepath.Join(projectRoot, ".claude")
	os.MkdirAll(dotClaudeDir, 0755)
	dotClaudeMd := filepath.Join(dotClaudeDir, "CLAUDE.md")
	os.WriteFile(dotClaudeMd, []byte("# Dot Claude Memory\n\nInstructions in .claude directory."), 0644)

	// 在根目录创建规则文件
	rulesDir := filepath.Join(projectRoot, ".claude", "rules")
	os.MkdirAll(rulesDir, 0755)
	ruleFile := filepath.Join(rulesDir, "test-rule.md")
	os.WriteFile(ruleFile, []byte("# Test Rule\n\nThis is a rule file."), 0644)

	// 在子目录创建 CLAUDE.md
	subClaudeMd := filepath.Join(subDir, "CLAUDE.md")
	os.WriteFile(subClaudeMd, []byte("# Subdirectory Memory\n\nInstructions in subdirectory."), 0644)

	// 在根目录创建 CLAUDE.local.md
	localClaudeMd := filepath.Join(projectRoot, "CLAUDE.local.md")
	os.WriteFile(localClaudeMd, []byte("# Local Memory\n\nLocal instructions not checked in."), 0644)

	// 设置环境变量
	oldUserType := os.Getenv("USER_TYPE")
	os.Setenv("USER_TYPE", "ant")
	defer os.Setenv("USER_TYPE", oldUserType)

	oldManagedPath := os.Getenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH")
	os.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", managedDir)
	defer os.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", oldManagedPath)

	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmpDir, "user", ".claude"))
	defer os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)

	// 启用所有设置源
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES", "user,project,local")
	defer os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES")

	// 禁用自动记忆和团队记忆以简化测试
	os.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "1")
	defer os.Unsetenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY")

	// 测试从子目录加载
	excludePatterns := []string{}
	mh := NewMemoryHierarchy(subDir, excludePatterns)

	files := mh.LoadAllMemoryFiles(subDir, false)

	// 验证加载的文件
	t.Logf("Loaded %d memory files:", len(files))
	for i, file := range files {
		t.Logf("  %d. %s (Type: %s)", i+1, file.Path, file.Type)
	}

	// 验证至少包含以下文件类型
	expectedTypes := map[MemoryType]bool{
		MemoryManaged: false,
		MemoryUser:    false,
		MemoryProject: false,
		MemoryLocal:   false,
	}

	for _, file := range files {
		if _, ok := expectedTypes[file.Type]; ok {
			expectedTypes[file.Type] = true
		}
	}

	// 检查是否找到了所有预期的记忆类型
	for memType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected to find memory type: %s", memType)
		}
	}

	// 验证项目内存文件的数量（应该包含根目录和子目录的 CLAUDE.md，以及 .claude/CLAUDE.md）
	projectFileCount := 0
	for _, file := range files {
		if file.Type == MemoryProject {
			projectFileCount++
		}
	}

	// 应该至少有3个项目文件：根目录 CLAUDE.md、.claude/CLAUDE.md、子目录 CLAUDE.md
	if projectFileCount < 3 {
		t.Errorf("Expected at least 3 project memory files, got %d", projectFileCount)
	}

	// 验证规则文件被加载
	foundRuleFile := false
	for _, file := range files {
		if strings.Contains(file.Path, "test-rule.md") {
			foundRuleFile = true
			break
		}
	}

	if !foundRuleFile {
		t.Error("Expected to find rule file")
	}
}

func TestBuildClaudeMdStringWithHierarchy(t *testing.T) {
	// 测试 BuildClaudeMdString 使用新的记忆层次结构
	tmpDir := t.TempDir()

	// 创建简单的项目结构
	projectRoot := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectRoot, 0755)

	// 创建 CLAUDE.md 文件
	claudeMdPath := filepath.Join(projectRoot, "CLAUDE.md")
	claudeMdContent := `# Project Instructions

This is a test CLAUDE.md file.

## Rules
- Follow these instructions
- Test the memory hierarchy`
	os.WriteFile(claudeMdPath, []byte(claudeMdContent), 0644)

	// 设置选项
	opts := LoadOptions{
		OriginalCwd: projectRoot,
	}

	// 启用项目设置
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS", "1")
	defer os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS")

	// 构建 CLAUDE.md 字符串
	result := BuildClaudeMdString(opts)

	// 验证结果
	if result == "" {
		t.Error("Expected non-empty CLAUDE.md string")
	}

	// 验证包含预期的内容
	if !strings.Contains(result, "Project Instructions") {
		t.Error("Expected result to contain 'Project Instructions'")
	}

	if !strings.Contains(result, "Follow these instructions") {
		t.Error("Expected result to contain 'Follow these instructions'")
	}

	// 验证包含记忆指令提示
	if !strings.Contains(result, MemoryInstructionPrompt) {
		t.Error("Expected result to contain memory instruction prompt")
	}

	t.Logf("Generated CLAUDE.md string (first 500 chars):\n%s", result[:min(500, len(result))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}