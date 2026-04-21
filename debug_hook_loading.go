package main

import (
	"fmt"
	"os"
	"path/filepath"

	"goc/ccb-engine/settingsfile"
	"goc/hookexec"
)

func main() {
	// 创建一个临时目录
	tmpDir, err := os.MkdirTemp("", "claude-go-debug-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// 设置 CLAUDE_CONFIG_DIR 环境变量
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		panic(err)
	}

	originalClaudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", homeDir)
	defer os.Setenv("CLAUDE_CONFIG_DIR", originalClaudeConfigDir)

	// 测试 UserClaudeSettingsPath
	fmt.Println("=== 测试 UserClaudeSettingsPath ===")
	userPath := settingsfile.UserClaudeSettingsPath()
	fmt.Printf("UserClaudeSettingsPath() = %q\n", userPath)
	
	// 检查路径是否存在
	if userPath != "" {
		if _, err := os.Stat(userPath); os.IsNotExist(err) {
			fmt.Printf("路径不存在: %s\n", userPath)
		} else {
			fmt.Printf("路径存在: %s\n", userPath)
		}
	}

	// 测试 FindClaudeProjectRoot
	fmt.Println("\n=== 测试 FindClaudeProjectRoot ===")
	
	// 创建项目结构
	projectDir := filepath.Join(tmpDir, "project")
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		panic(err)
	}

	// 创建 settings.go.json
	goPath := filepath.Join(claudeDir, "settings.go.json")
	if err := os.WriteFile(goPath, []byte(`{"hooks":{}}`), 0644); err != nil {
		panic(err)
	}

	// 创建子目录
	subDir := filepath.Join(projectDir, "src", "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		panic(err)
	}

	// 查找项目根目录
	root, err := settingsfile.FindClaudeProjectRoot(subDir)
	if err != nil {
		fmt.Printf("FindClaudeProjectRoot 错误: %v\n", err)
	} else {
		fmt.Printf("FindClaudeProjectRoot(%q) = %q\n", subDir, root)
	}

	// 测试 MergedHooksFromPaths
	fmt.Println("\n=== 测试 MergedHooksFromPaths ===")
	
	// 创建用户设置文件
	userClaudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(userClaudeDir, 0755); err != nil {
		panic(err)
	}
	
	userSettingsPath := filepath.Join(userClaudeDir, "settings.json")
	userContent := `{"hooks":{"UserPromptSubmit":[{"matcher":".*","hooks":[{"type":"command","command":"echo 'user'"}]}]}}`
	if err := os.WriteFile(userSettingsPath, []byte(userContent), 0644); err != nil {
		panic(err)
	}
	
	fmt.Printf("用户设置文件: %s\n", userSettingsPath)
	fmt.Printf("项目根目录: %s\n", projectDir)
	
	// 加载合并的钩子
	merged, err := hookexec.MergedHooksFromPaths(projectDir)
	if err != nil {
		fmt.Printf("MergedHooksFromPaths 错误: %v\n", err)
	} else {
		fmt.Printf("合并的钩子表: %v\n", merged)
		for event, groups := range merged {
			fmt.Printf("  事件: %s, 匹配器组数: %d\n", event, len(groups))
		}
	}

	// 测试 MergedHooksForCwd
	fmt.Println("\n=== 测试 MergedHooksForCwd ===")
	
	// 切换到子目录
	originalDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defer os.Chdir(originalDir)
	
	if err := os.Chdir(subDir); err != nil {
		panic(err)
	}
	
	merged2, err := hookexec.MergedHooksForCwd(subDir)
	if err != nil {
		fmt.Printf("MergedHooksForCwd 错误: %v\n", err)
	} else {
		fmt.Printf("MergedHooksForCwd 结果: %v\n", merged2)
		for event, groups := range merged2 {
			fmt.Printf("  事件: %s, 匹配器组数: %d\n", event, len(groups))
		}
	}
}