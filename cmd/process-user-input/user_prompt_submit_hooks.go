package main

import (
	"context"
	"os"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/hookexec"
	"goc/sessiontranscript"
	"goc/types"
)

func buildBaseHookInputForPUI(p *processuserinput.ProcessUserInputParams, cwd string) hookexec.BaseHookInput {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "."
		}
	}
	var sessionID, agentID, agentType string
	if p != nil && p.RuntimeContext != nil {
		rc := p.RuntimeContext
		if rc.ConversationID != nil {
			sessionID = strings.TrimSpace(*rc.ConversationID)
		}
		if rc.AgentID != nil {
			agentID = strings.TrimSpace(*rc.AgentID)
		}
		if rc.AgentType != nil {
			agentType = strings.TrimSpace(*rc.AgentType)
		}
	}
	transcriptPath := ""
	if sessionID != "" {
		transcriptPath = sessiontranscript.TranscriptPath(sessionID, cwd, "", sessiontranscript.ConfigHomeDir())
	}
	pm := ""
	if p != nil {
		pm = string(p.PermissionMode)
	}
	return hookexec.BaseHookInput{
		SessionID:       sessionID,
		TranscriptPath:  transcriptPath,
		Cwd:             cwd,
		PermissionMode:  pm,
		AgentID:         agentID,
		AgentType:       agentType,
		HookEventName:   "UserPromptSubmit",
	}
}

// wireUserPromptSubmitHooks 为 ProcessUserInputParams 配置 UserPromptSubmit 钩子执行函数。
//
// 这个函数是钩子系统与 process-user-input 模块的集成点，负责：
// 1. 检查是否有 UserPromptSubmit 钩子配置
// 2. 设置 ProcessUserInputParams.ExecuteUserPromptSubmitHooks 回调函数
// 3. 在用户提交提示时触发钩子执行
//
// 参数说明：
//   - p: ProcessUserInputParams 指针，用于配置钩子执行函数
//   - merged: 已合并的钩子配置表，包含所有加载的钩子配置
//   - cwd: 当前工作目录，用于钩子命令执行环境
//
// 执行流程：
// 1. 安全检查：如果参数为空或没有 UserPromptSubmit 钩子配置，直接返回
// 2. 工作目录处理：确保有有效的工作目录供钩子命令执行
// 3. 闭包创建：创建一个闭包函数，该函数在用户提交提示时被调用
// 4. 钩子执行：闭包内部调用 hookexec.RunUserPromptSubmitHooks 执行实际钩子逻辑
//
// 钩子触发时机：
// 当 process-user-input 模块处理用户输入时，如果配置了 ExecuteUserPromptSubmitHooks，
// 会在适当的时间点调用该函数执行 UserPromptSubmit 钩子。
//
// 与TypeScript的奇偶性：
// 这个函数对应 TypeScript 版本中 processUserInput 对 executeUserPromptSubmitHooks 的调用点，
// 确保 Go 版本在相同的时间点触发相同的钩子逻辑。
func wireUserPromptSubmitHooks(p *processuserinput.ProcessUserInputParams, merged hookexec.HooksTable, cwd string) {
	// 1. 安全检查：如果参数为空或没有 UserPromptSubmit 钩子配置，直接返回
	if p == nil || len(merged["UserPromptSubmit"]) == 0 {
		return
	}
	
	// 2. 工作目录处理：确保有有效的工作目录供钩子命令执行
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "."
		}
	}
	cwdFinal := cwd
	
	// 3. 设置钩子执行函数
	// 这个闭包函数将在用户提交提示时被 process-user-input 模块调用
	p.ExecuteUserPromptSubmitHooks = func(ctx context.Context, pp *processuserinput.ProcessUserInputParams, inputMessage string) ([]types.AggregatedHookResult, error) {
		// 3.1 构建钩子输入基础信息
		base := buildBaseHookInputForPUI(pp, cwdFinal)
		
		// 3.2 执行 UserPromptSubmit 钩子
		// 调用 hookexec 包中的核心钩子执行函数
		// 参数说明：
		//   - ctx: 上下文，用于超时控制和取消
		//   - merged: 钩子配置表
		//   - cwdFinal: 工作目录
		//   - base: 钩子输入基础信息
		//   - inputMessage: 用户提交的提示文本
		//   - hookexec.DefaultHookTimeoutMs: 默认超时时间（10分钟）
		return hookexec.RunUserPromptSubmitHooks(ctx, merged, cwdFinal, base, inputMessage, hookexec.DefaultHookTimeoutMs)
	}
}
