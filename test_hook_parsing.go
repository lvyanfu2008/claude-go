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
	tmpDir, err := os.MkdirTemp("", "claude-go-hook-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. 创建 .claude 目录
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		panic(err)
	}

	// 3. 创建 settings.go.json 文件
	settingsGoPath := filepath.Join(claudeDir, "settings.go.json")
	settingsGoContent := `{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "review.*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Reviewing: $PROMPT'",
            "description": "测试审查钩子"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "resume",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Session resumed'",
            "description": "会话恢复钩子"
          }
        ]
      }
    ]
  }
}`

	if err := os.WriteFile(settingsGoPath, []byte(settingsGoContent), 0644); err != nil {
		panic(err)
	}

	// 4. 创建子目录用于测试项目根目录查找
	subDir := filepath.Join(tmpDir, "src", "project")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		panic(err)
	}

	// 5. 测试 MergedHooksForCwd 函数
	fmt.Println("=== 测试 settings.go.json 钩子解析 ===")
	fmt.Printf("临时目录: %s\n", tmpDir)
	fmt.Printf("子目录: %s\n", subDir)

	// 切换到子目录
	originalDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(subDir); err != nil {
		panic(err)
	}

	// 6. 加载合并的钩子
	mergedHooks, err := hookexec.MergedHooksForCwd(subDir)
	if err != nil {
		panic(err)
	}

	// 7. 打印结果
	fmt.Println("\n=== 加载的钩子表 ===")
	for eventName, matcherGroups := range mergedHooks {
		fmt.Printf("\n事件: %s\n", eventName)
		for i, mg := range matcherGroups {
			fmt.Printf("  匹配器组 %d: matcher='%s'\n", i+1, mg.Matcher)
			for j, hook := range mg.Hooks {
				var hookObj map[string]interface{}
				if err := json.Unmarshal(hook, &hookObj); err == nil {
					fmt.Printf("    钩子 %d: type=%s, command=%s\n", 
						j+1, 
						hookObj["type"], 
						hookObj["command"])
				}
			}
		}
	}

	// 8. 测试特定事件的钩子匹配
	fmt.Println("\n=== 测试 UserPromptSubmit 钩子匹配 ===")
	hookInput := map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"prompt": "review this code",
	}

	matchingHooks := hookexec.CommandHooksForHookInput(mergedHooks, hookInput)
	fmt.Printf("匹配的钩子数量: %d\n", len(matchingHooks))
	for i, hook := range matchingHooks {
		fmt.Printf("  钩子 %d: %s\n", i+1, hook.Command)
	}

	// 9. 测试 SessionStart 钩子匹配
	fmt.Println("\n=== 测试 SessionStart 钩子匹配 ===")
	sessionStartInput := map[string]any{
		"hook_event_name": "SessionStart",
		"source": "resume",
	}

	sessionHooks := hookexec.CommandHooksForHookInput(mergedHooks, sessionStartInput)
	fmt.Printf("匹配的钩子数量: %d\n", len(sessionHooks))
	for i, hook := range sessionHooks {
		fmt.Printf("  钩子 %d: %s\n", i+1, hook.Command)
	}

	// 10. 测试 HasInstructionsLoaded 函数
	fmt.Println("\n=== 测试 HasInstructionsLoaded ===")
	hasInstructions := hookexec.HasInstructionsLoaded(mergedHooks)
	fmt.Printf("是否有 InstructionsLoaded 钩子: %v\n", hasInstructions)

	// 11. 测试文件路径解析
	fmt.Println("\n=== 测试文件路径解析 ===")
	fileChangedInput := map[string]any{
		"hook_event_name": "FileChanged",
		"file_path": "/path/to/file.go",
	}
	mq, use := hookexec.DeriveMatchQuery(fileChangedInput)
	fmt.Printf("文件路径匹配查询: matchQuery='%s', applyMatcherFilter=%v\n", mq, use)

	fmt.Println("\n=== 测试完成 ===")
}