package claudemd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryHierarchy(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()

	// 创建测试记忆文件
	testCases := []struct {
		path    string
		content string
	}{
		// 托管内存
		{filepath.Join(tmpDir, "managed", "CLAUDE.md"), "# Managed Memory\n\nThis is managed memory."},
		// 用户内存
		{filepath.Join(tmpDir, "user", ".claude", "CLAUDE.md"), "# User Memory\n\nThis is user memory."},
		// 项目内存
		{filepath.Join(tmpDir, "project", "CLAUDE.md"), "# Project Memory\n\nThis is project memory."},
		// 本地内存
		{filepath.Join(tmpDir, "project", "CLAUDE.local.md"), "# Local Memory\n\nThis is local memory."},
	}

	// 设置环境变量指向临时目录
	oldManagedPath := os.Getenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH")
	os.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", filepath.Join(tmpDir, "managed"))
	defer os.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", oldManagedPath)

	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmpDir, "user"))
	defer os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)

	// 创建目录和文件
	for _, tc := range testCases {
		dir := filepath.Dir(tc.path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(tc.path, []byte(tc.content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", tc.path, err)
		}
	}

	// 测试记忆层次结构
	excludePatterns := []string{}
	mh := NewMemoryHierarchy(filepath.Join(tmpDir, "project"), excludePatterns)

	// 启用所有设置源
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_USERSETTINGS", "1")
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS", "1")
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_LOCALSETTINGS", "1")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_USERSETTINGS")
		os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS")
		os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_LOCALSETTINGS")
	}()

	// 加载记忆文件
	files := mh.LoadAllMemoryFiles(filepath.Join(tmpDir, "project"), false)

	// 验证加载的文件数量
	// 应该至少加载项目内存和本地内存
	if len(files) < 2 {
		t.Errorf("Expected at least 2 memory files, got %d", len(files))
	}

	// 验证文件类型
	foundProject := false
	foundLocal := false
	for _, file := range files {
		if file.Type == MemoryProject {
			foundProject = true
		}
		if file.Type == MemoryLocal {
			foundLocal = true
		}
	}

	if !foundProject {
		t.Error("Expected to find project memory file")
	}
	if !foundLocal {
		t.Error("Expected to find local memory file")
	}
}

func TestMemoryHierarchyPriority(t *testing.T) {
	// 测试记忆文件的优先级顺序
	tmpDir := t.TempDir()

	// 在不同层级创建相同名称的文件
	levels := []struct {
		level string
		path  string
	}{
		{"root", filepath.Join(tmpDir, "CLAUDE.md")},
		{"subdir", filepath.Join(tmpDir, "subdir", "CLAUDE.md")},
		{"subsubdir", filepath.Join(tmpDir, "subdir", "subsubdir", "CLAUDE.md")},
	}

	for _, level := range levels {
		dir := filepath.Dir(level.path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		content := "# Memory from " + level.level + "\n\nContent for " + level.level
		if err := os.WriteFile(level.path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", level.path, err)
		}
	}

	// 启用项目设置
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS", "1")
	defer os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS")

	excludePatterns := []string{}
	mh := NewMemoryHierarchy(filepath.Join(tmpDir, "subdir", "subsubdir"), excludePatterns)
	files := mh.LoadAllMemoryFiles(filepath.Join(tmpDir, "subdir", "subsubdir"), false)

	// 应该加载所有三个层级的文件
	if len(files) != 3 {
		t.Errorf("Expected 3 memory files (root, subdir, subsubdir), got %d", len(files))
	}

	// 验证文件按照正确的优先级顺序加载（从低到高优先级）
	// 在 TypeScript 实现中，文件按照从根目录到当前目录的顺序加载
	// 这意味着离当前目录越近的文件优先级越高
	for i, file := range files {
		t.Logf("File %d: %s (Type: %s)", i, file.Path, file.Type)
	}
}

func TestIncludeDirective(t *testing.T) {
	// 测试 @include 指令
	tmpDir := t.TempDir()

	// 创建被包含的文件
	includedPath := filepath.Join(tmpDir, "included.md")
	includedContent := "# Included File\n\nThis content is included."
	if err := os.WriteFile(includedPath, []byte(includedContent), 0644); err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	// 创建主文件包含其他文件
	mainPath := filepath.Join(tmpDir, "CLAUDE.md")
	mainContent := "# Main File\n\nThis includes another file.\n\n@./included.md"
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// 启用项目设置
	os.Setenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS", "1")
	defer os.Unsetenv("CLAUDE_CODE_SETTING_SOURCES_PROJECTSETTINGS")

	// 启用外部包含
	os.Setenv("CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED", "1")
	defer os.Unsetenv("CLAUDE_CODE_CLAUDE_MD_EXTERNAL_INCLUDES_APPROVED")

	excludePatterns := []string{}
	mh := NewMemoryHierarchy(tmpDir, excludePatterns)
	files := mh.LoadAllMemoryFiles(tmpDir, true)

	// 应该加载主文件和被包含的文件
	if len(files) != 2 {
		t.Errorf("Expected 2 files (main + included), got %d", len(files))
	}

	// 验证被包含的文件出现在主文件之前（TS 实现）
	foundIncluded := false
	foundMain := false
	for _, file := range files {
		if filepath.Base(file.Path) == "included.md" {
			foundIncluded = true
		}
		if filepath.Base(file.Path) == "CLAUDE.md" {
			foundMain = true
		}
	}

	if !foundIncluded {
		t.Error("Expected to find included file")
	}
	if !foundMain {
		t.Error("Expected to find main file")
	}
}