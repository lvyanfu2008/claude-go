package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"goc/hookexec"
)

func main() {
	// 1. 创建一个临时目录结构
	tmpDir, err := os.MkdirTemp("", "claude-go-hook-merge-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. 创建 .claude 目录
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		panic(err)
	}

	// 3. 创建 settings.go.json 文件（项目级）
	settingsGoPath := filepath.Join(claudeDir, "settings.go.json")
	settingsGoContent := `{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "review.*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Project: Reviewing $PROMPT'",
            "description": "项目级审查钩子"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Project: Pre-compact hook'",
            "description": "项目级压缩前钩子"
          }
        ]
      }
    ]
  }
}`

	if err := os.WriteFile(settingsGoPath, []byte(settingsGoContent), 0644); err != nil {
		panic(err)
	}

	// 4. 创建 settings.local.json 文件（本地覆盖）
	settingsLocalPath := filepath.Join(claudeDir, "settings.local.json")
	settingsLocalContent := `{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "review.*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Local: Additional review hook'",
            "description": "本地额外审查钩子"
          }
        ]
      }
    ],
    "PostCompact": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Local: Post-compact hook'",
            "description": "本地压缩后钩子"
          }
        ]
      }
    ]
  }
}`

	if err := os.WriteFile(settingsLocalPath, []byte(settingsLocalContent), 0644); err != nil {
		panic(err)
	}

	// 5. 模拟用户主目录设置
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		panic(err)
	}

	userClaudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(userClaudeDir, 0755); err != nil {
		panic(err)
	}

	userSettingsPath := filepath.Join(userClaudeDir, "settings.json")
	userSettingsContent := `{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'User: Global hook for all prompts'",
            "description": "用户级全局钩子"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'User: Session started'",
            "description": "用户级会话开始钩子"
          }
        ]
      }
    ]
  }
}`

	if err := os.WriteFile(userSettingsPath, []byte(userSettingsContent), 0644); err != nil {
		panic(err)
	}

	// 6. 设置环境变量模拟用户主目录
	originalClaudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", homeDir)
	defer os.Setenv("CLAUDE_CONFIG_DIR", originalClaudeConfigDir)

	// 7. 创建子目录用于测试
	subDir := filepath.Join(tmpDir, "src", "project")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		panic(err)
	}

	// 切换到子目录
	originalDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(subDir); err != nil {
		panic(err)
	}

	// 8. 测试 MergedHooksForCwd 函数
	fmt.Println("=== 测试多级设置文件合并 ===")
	fmt.Printf("临时目录: %s\n", tmpDir)
	fmt.Printf("用户设置: %s\n", userSettingsPath)
	fmt.Printf("项目设置: %s\n", settingsGoPath)
	fmt.Printf("本地设置: %s\n", settingsLocalPath)

	mergedHooks, err := hookexec.MergedHooksForCwd(subDir)
	if err != nil {
		panic(err)
	}

	// 9. 打印合并结果
	fmt.Println("\n=== 合并后的钩子表 ===")
	for eventName, matcherGroups := range mergedHooks {
		fmt.Printf("\n事件: %s\n", eventName)
		fmt.Printf("  匹配器组数量: %d\n", len(matcherGroups))
		
		totalHooks := 0
		for i, mg := range matcherGroups {
			totalHooks += len(mg.Hooks)
			fmt.Printf("  匹配器组 %d: matcher='%s' (钩子数: %d)\n", 
				i+1, mg.Matcher, len(mg.Hooks))
			
			for j, hook := range mg.Hooks {
				var hookObj map[string]interface{}
				if err := json.Unmarshal(hook, &hookObj); err == nil {
					fmt.Printf("    钩子 %d: %s\n", 
						j+1, hookObj["command"])
				}
			}
		}
		fmt.Printf("  总钩子数: %d\n", totalHooks)
	}

	// 10. 测试 UserPromptSubmit 钩子匹配（应该匹配所有三个来源的钩子）
	fmt.Println("\n=== 测试 UserPromptSubmit 钩子合并匹配 ===")
	hookInput := map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"prompt": "review this code",
	}

	matchingHooks := hookexec.CommandHooksForHookInput(mergedHooks, hookInput)
	fmt.Printf("匹配的钩子数量: %d (预期: 3)\n", len(matchingHooks))
	
	// 检查是否包含所有来源的钩子
	sources := map[string]bool{
		"User: Global hook for all prompts": false,
		"Project: Reviewing $PROMPT": false,
		"Local: Additional review hook": false,
	}
	
	for i, hook := range matchingHooks {
		fmt.Printf("  钩子 %d: %s\n", i+1, hook.Command)
		for source := range sources {
			if hook.Command == source {
				sources[source] = true
			}
		}
	}
	
	// 验证所有来源都存在
	allFound := true
	for source, found := range sources {
		if !found {
			fmt.Printf("  警告: 未找到来源: %s\n", source)
			allFound = false
		}
	}
	
	if allFound {
		fmt.Println("  ✅ 所有来源的钩子都已正确合并")
	}

	// 11. 测试合并顺序（用户 → 项目 → 本地）
	fmt.Println("\n=== 测试合并顺序 ===")
	fmt.Println("预期顺序:")
	fmt.Println("  1. 用户设置 (~/.claude/settings.json)")
	fmt.Println("  2. 项目设置 (.claude/settings.go.json)")
	fmt.Println("  3. 本地设置 (.claude/settings.local.json)")
	
	// 检查 SessionStart 钩子（只应在用户设置中）
	fmt.Println("\nSessionStart 钩子来源检查:")
	sessionStartInput := map[string]any{
		"hook_event_name": "SessionStart",
		"source": "new",
	}
	sessionHooks := hookexec.CommandHooksForHookInput(mergedHooks, sessionStartInput)
	fmt.Printf("  匹配的钩子数量: %d (预期: 1，来自用户设置)\n", len(sessionHooks))
	if len(sessionHooks) == 1 && sessionHooks[0].Command == "echo 'User: Session started'" {
		fmt.Println("  ✅ SessionStart 钩子来自用户设置")
	}

	// 12. 测试其他事件类型
	fmt.Println("\n=== 测试其他事件类型 ===")
	
	// PreCompact（项目设置）
	preCompactInput := map[string]any{
		"hook_event_name": "PreCompact",
		"compactType": "default",
	}
	preCompactHooks := hookexec.CommandHooksForHookInput(mergedHooks, preCompactInput)
	fmt.Printf("PreCompact 钩子数量: %d (预期: 1，来自项目设置)\n", len(preCompactHooks))
	
	// PostCompact（本地设置）
	postCompactInput := map[string]any{
		"hook_event_name": "PostCompact",
		"compactType": "default",
	}
	postCompactHooks := hookexec.CommandHooksForHookInput(mergedHooks, postCompactInput)
	fmt.Printf("PostCompact 钩子数量: %d (预期: 1，来自本地设置)\n", len(postCompactHooks))

	fmt.Println("\n=== 测试完成 ===")
}